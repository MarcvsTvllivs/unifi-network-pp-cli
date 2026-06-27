// Copyright 2026 marcvstvllivs and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Hand-authored UniFi controller settings. The generated Config struct models
// only the Integration API key (X-API-KEY / UNIFI_API_KEY); the controller's
// primary auth is a username/password cookie+CSRF session against a LAN host.
// Those settings live here, read from the environment, so the generated config
// surface stays untouched and regen-safe.
package config

import (
	"os"
	"strconv"
	"strings"
)

// UnifiSettings holds everything the session transport needs to reach a UniFi
// Network controller. Cookie-session credentials (host/username/password) come
// from the environment; the Integration API key is shared with the generated
// Config (UNIFI_API_KEY).
type UnifiSettings struct {
	Host           string // controller host or IP (no scheme)
	Port           string // controller port (default 443)
	Site           string // controller site key (default "default")
	Username       string // local controller admin username
	Password       string // local controller admin password
	APIKey         string // Integration API key (X-API-KEY), optional
	VerifySSL      bool   // verify TLS cert (default false — self-signed controllers)
	ControllerType string // auto | proxy (UniFi OS) | direct (legacy)
}

// LoadUnifiSettings resolves controller settings from the environment, falling
// back to the generated Config for the API key. Host scheme/port embedded in
// UNIFI_HOST are normalized out so BaseURL can compose them deterministically.
func LoadUnifiSettings(cfg *Config) UnifiSettings {
	host := strings.TrimSpace(os.Getenv("UNIFI_HOST"))
	port := strings.TrimSpace(os.Getenv("UNIFI_PORT"))

	// Tolerate a full URL in UNIFI_HOST (https://1.2.3.4:8443) by splitting
	// scheme and any embedded port back out.
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimSuffix(host, "/")
	if h, p, ok := strings.Cut(host, ":"); ok {
		host = h
		if port == "" {
			port = p
		}
	}
	if port == "" {
		port = "443"
	}

	site := strings.TrimSpace(os.Getenv("UNIFI_SITE"))
	if site == "" {
		site = "default"
	}

	apiKey := ""
	if cfg != nil {
		apiKey = cfg.UnifiApiKey
	}
	if v := strings.TrimSpace(os.Getenv("UNIFI_API_KEY")); v != "" {
		apiKey = v
	}

	controllerType := strings.ToLower(strings.TrimSpace(os.Getenv("UNIFI_CONTROLLER_TYPE")))
	switch controllerType {
	case "proxy", "direct":
		// explicit
	default:
		controllerType = "auto"
	}

	return UnifiSettings{
		Host:           host,
		Port:           port,
		Site:           site,
		Username:       strings.TrimSpace(os.Getenv("UNIFI_USERNAME")),
		Password:       os.Getenv("UNIFI_PASSWORD"),
		APIKey:         apiKey,
		VerifySSL:      envBool("UNIFI_VERIFY_SSL", false),
		ControllerType: controllerType,
	}
}

// BaseURL composes the controller origin (no trailing slash).
func (u UnifiSettings) BaseURL() string {
	if u.Host == "" {
		return ""
	}
	return "https://" + u.Host + ":" + u.Port
}

// HasCookieCredentials reports whether username/password are set for the
// cookie-session login flow.
func (u UnifiSettings) HasCookieCredentials() bool {
	return u.Host != "" && u.Username != "" && u.Password != ""
}

func envBool(name string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
