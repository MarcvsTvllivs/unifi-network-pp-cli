// Copyright 2026 marcvstvllivs and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Hand-authored UniFi session transport. The generated client speaks a simple
// api_key (X-API-KEY) scheme — correct for the Integration API (DPI, firewall
// ordering) but not for the controller API, which is the bulk of the surface.
// The controller API needs: a username/password login handshake, a session
// cookie + rotating CSRF token echoed on mutations, a `/proxy/network` path
// prefix on UniFi OS consoles, per-call site substitution, and tolerance for
// self-signed certs.
//
// Rather than fork the generated client.go (which has no global request
// mutator hook), this installs an http.RoundTripper that wraps every request:
// it substitutes the __SITE__ path sentinel, prepends the UniFi OS prefix,
// logs in lazily, attaches the CSRF header on mutating verbs, and re-auths once
// on 401. Integration API paths pass straight through (the generated X-API-KEY
// header is all they need). The whole thing is a no-op under PRINTING_PRESS_VERIFY
// so the verify/mock harness keeps talking plain HTTP to its httptest server.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"unifi-network-pp-cli/internal/cliutil"
	"unifi-network-pp-cli/internal/config"
)

const (
	siteSentinel      = "__SITE__"
	unifiOSPrefix     = "/proxy/network"
	integrationPrefix = "/proxy/network/integration"
	csrfRequestHeader = "X-CSRF-Token"
	loginTimeout      = 20 * time.Second
	probeTimeout      = 8 * time.Second
)

// InstallUnifiTransport upgrades a generated Client to speak the UniFi
// controller protocol. It overrides the base URL from UNIFI_HOST/UNIFI_PORT,
// installs a cookie jar, and wraps the transport with the session round
// tripper. It is intentionally a no-op in verify/mock mode so the printing-press
// verifier's literal-path expectations (no prefix, literal __SITE__) hold.
func InstallUnifiTransport(c *Client, cfg *config.Config) {
	if c == nil || c.HTTPClient == nil {
		return
	}
	if cliutil.IsVerifyEnv() {
		return
	}
	settings := config.LoadUnifiSettings(cfg)
	if settings.Host != "" {
		c.BaseURL = strings.TrimRight(settings.BaseURL(), "/")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return
	}

	base := http.DefaultTransport.(*http.Transport).Clone()
	// #nosec G402 -- self-signed certs are the default on UniFi controllers; TLS verification is opt-in via UNIFI_VERIFY_SSL (settings.VerifySSL), so the insecure path is gated behind explicit user configuration.
	base.TLSClientConfig = &tls.Config{InsecureSkipVerify: !settings.VerifySSL} //nolint:gosec // gated by UNIFI_VERIFY_SSL

	c.HTTPClient.Transport = &unifiRoundTripper{
		base:     base,
		settings: settings,
		jar:      jar,
	}
	c.HTTPClient.Jar = jar
	// The generated CheckRedirect re-derives an X-API-KEY header on each hop,
	// which is harmless for cookie auth but irrelevant; leave it in place.
}

type unifiRoundTripper struct {
	base     http.RoundTripper
	settings config.UnifiSettings
	jar      http.CookieJar

	mu        sync.Mutex
	loggedIn  bool
	probed    bool
	isUnifiOS bool
	csrfToken string
}

func (rt *unifiRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Integration API: cookie login does not apply. The generated do() already
	// set X-API-KEY from UNIFI_API_KEY; just surface a clear error when it's
	// missing rather than letting an opaque 401 through.
	if strings.HasPrefix(req.URL.Path, integrationPrefix) {
		if rt.settings.APIKey == "" && req.Header.Get("X-API-KEY") == "" {
			return nil, fmt.Errorf("this endpoint uses the UniFi Network Integration API: set UNIFI_API_KEY (create one in the UniFi UI under Control Plane > Integrations)")
		}
		return rt.base.RoundTrip(req)
	}

	if err := rt.ensureSession(req.Context()); err != nil {
		return nil, err
	}
	rt.prepare(req)

	resp, err := rt.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	rt.captureCSRF(resp.Header)

	// One transparent re-auth on 401 (session expired mid-run). Only retry
	// when the body is replayable (GET, or a request that carries GetBody).
	if resp.StatusCode == http.StatusUnauthorized && (req.Body == nil || req.GetBody != nil) {
		_ = resp.Body.Close() // discarding the 401 body before the re-auth retry; close error is not actionable here

		rt.mu.Lock()
		rt.loggedIn = false
		rt.mu.Unlock()
		if err := rt.ensureSession(req.Context()); err != nil {
			return nil, err
		}
		retry := req.Clone(req.Context())
		if req.GetBody != nil {
			body, gerr := req.GetBody()
			if gerr != nil {
				// resp.Body was closed above; cannot return a closed body.
				return nil, fmt.Errorf("re-auth retry: replaying request body: %w", gerr)
			}
			retry.Body = body
		}
		rt.prepare(retry)
		return rt.base.RoundTrip(retry)
	}
	return resp, nil
}

// ensureSession resolves the controller type (once) and logs in (when needed).
func (rt *unifiRoundTripper) ensureSession(ctx context.Context) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.loggedIn {
		return nil
	}
	if !rt.settings.HasCookieCredentials() {
		return fmt.Errorf("UniFi controller auth not configured: set UNIFI_HOST, UNIFI_USERNAME and UNIFI_PASSWORD (run `unifi-network-pp-cli doctor` for a checklist)")
	}
	if !rt.probed {
		rt.isUnifiOS = rt.detectUnifiOS(ctx)
		rt.probed = true
	}
	return rt.loginLocked(ctx)
}

// prepare rewrites the outgoing request: site substitution, UniFi OS prefix,
// and CSRF echo on mutating verbs. It is idempotent (safe to call on a retry).
func (rt *unifiRoundTripper) prepare(req *http.Request) {
	if strings.Contains(req.URL.Path, siteSentinel) {
		req.URL.Path = strings.ReplaceAll(req.URL.Path, siteSentinel, rt.settings.Site)
		if req.URL.RawPath != "" {
			req.URL.RawPath = strings.ReplaceAll(req.URL.RawPath, siteSentinel, rt.settings.Site)
		}
	}
	rt.mu.Lock()
	isUnifiOS := rt.isUnifiOS
	csrf := rt.csrfToken
	rt.mu.Unlock()

	if isUnifiOS && !strings.HasPrefix(req.URL.Path, unifiOSPrefix) {
		req.URL.Path = unifiOSPrefix + req.URL.Path
		if req.URL.RawPath != "" {
			req.URL.RawPath = unifiOSPrefix + req.URL.RawPath
		}
	}
	if csrf != "" && unifiIsMutating(req.Method) {
		req.Header.Set(csrfRequestHeader, csrf)
	}
}

// detectUnifiOS probes the controller to pick the path prefix and login
// endpoint. UNIFI_CONTROLLER_TYPE forces the answer; auto mode hits the root
// and reads the response: a UniFi OS console answers 200 (and a CSRF header),
// while a legacy Network application redirects to its login page.
func (rt *unifiRoundTripper) detectUnifiOS(ctx context.Context) bool {
	switch rt.settings.ControllerType {
	case "proxy":
		return true
	case "direct":
		return false
	}
	reqCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rt.settings.BaseURL()+"/", nil)
	if err != nil {
		return true
	}
	probe := &http.Client{
		Transport: rt.base,
		Timeout:   probeTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := probe.Do(req)
	if err != nil {
		return true // modern controllers are UniFi OS; default there on probe failure
	}
	defer resp.Body.Close()
	if resp.Header.Get("x-csrf-token") != "" || resp.Header.Get("x-updated-csrf-token") != "" {
		return true
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return false // legacy controllers redirect to /manage
	}
	return resp.StatusCode == http.StatusOK
}

// loginLocked performs the login handshake. Caller must hold rt.mu.
func (rt *unifiRoundTripper) loginLocked(ctx context.Context) error {
	loginPath := "/api/auth/login"
	if !rt.isUnifiOS {
		loginPath = "/api/login"
	}
	payload, _ := json.Marshal(map[string]any{
		"username": rt.settings.Username,
		"password": rt.settings.Password,
		"remember": true,
	})
	reqCtx, cancel := context.WithTimeout(ctx, loginTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, rt.settings.BaseURL()+loginPath, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building UniFi login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	loginClient := &http.Client{Transport: rt.base, Jar: rt.jar, Timeout: loginTimeout}
	resp, err := loginClient.Do(req)
	if err != nil {
		return fmt.Errorf("UniFi login to %s failed: %w", rt.settings.Host, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		hint := "check UNIFI_USERNAME/UNIFI_PASSWORD; use a local controller admin account, not a UI.com cloud (SSO/MFA) login"
		if resp.StatusCode == http.StatusBadRequest && rt.isUnifiOS {
			hint += "; if this is a self-hosted Network application set UNIFI_CONTROLLER_TYPE=direct"
		}
		return fmt.Errorf("UniFi login failed (HTTP %d): %s — %s", resp.StatusCode, strings.TrimSpace(string(body)), hint)
	}
	// Capture the CSRF token directly (we already hold rt.mu).
	if v := firstHeader(resp.Header, "x-updated-csrf-token", "x-csrf-token"); v != "" {
		rt.csrfToken = v
	}
	rt.loggedIn = true
	return nil
}

// captureCSRF stores the rotating CSRF token from any response that carries one.
func (rt *unifiRoundTripper) captureCSRF(h http.Header) {
	v := firstHeader(h, "x-updated-csrf-token", "x-csrf-token")
	if v == "" {
		return
	}
	rt.mu.Lock()
	rt.csrfToken = v
	rt.mu.Unlock()
}

func firstHeader(h http.Header, names ...string) string {
	for _, n := range names {
		if v := h.Get(n); v != "" {
			return v
		}
	}
	return ""
}

func unifiIsMutating(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		return true
	}
	return false
}
