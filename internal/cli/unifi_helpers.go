// Copyright 2026 marcvstvllivs and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Shared helpers for hand-authored UniFi action commands (cmd/{mgr} actions,
// read-modify-write toggles) and transcendence commands. The session transport
// (unifi_session.go) handles auth/prefix/site; these helpers build paths and
// shape output so the action commands stay small and uniform.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"unifi-network-pp-cli/internal/store"
)

// unifiV1 builds a controller V1 path with the __SITE__ sentinel the session
// transport substitutes at request time, e.g. unifiV1("/cmd/stamgr").
func unifiV1(suffix string) string {
	return "/api/s/__SITE__" + suffix
}

// unifiV2 builds a controller V2 path with the __SITE__ sentinel.
func unifiV2(suffix string) string {
	return "/v2/api/site/__SITE__" + suffix
}

// unifiUnwrap strips the V1 {"meta":{...},"data":[...]} envelope for display.
// V2/integration responses (bare list/object) are returned unchanged.
func unifiUnwrap(data json.RawMessage) json.RawMessage {
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(data, &env) == nil && len(env.Data) > 0 {
		return env.Data
	}
	return data
}

// unifiEmit prints an API result, unwrapping the V1 envelope and honoring
// --json/--select/--compact/--csv via the generated output helper.
func unifiEmit(cmd *cobra.Command, flags *rootFlags, data json.RawMessage) error {
	return printJSONFiltered(cmd.OutOrStdout(), unifiUnwrap(data), flags)
}

// unifiPost POSTs a JSON body to a controller path and prints the result.
// Used by every cmd/{mgr} action (block, reboot, archive-alarm, …).
func unifiPost(cmd *cobra.Command, flags *rootFlags, path string, body map[string]any) error {
	c, err := flags.newClient()
	if err != nil {
		return err
	}
	data, _, err := c.Post(cmd.Context(), path, body)
	if err != nil {
		return err
	}
	return unifiEmit(cmd, flags, data)
}

// unifiPut PUTs a JSON body to a controller path and prints the result.
func unifiPut(cmd *cobra.Command, flags *rootFlags, path string, body any) error {
	c, err := flags.newClient()
	if err != nil {
		return err
	}
	data, _, err := c.Put(cmd.Context(), path, body)
	if err != nil {
		return err
	}
	return unifiEmit(cmd, flags, data)
}

// unifiGetObject fetches one object. For V1 REST paths the controller returns
// {"meta":..,"data":[obj]}; this unwraps and returns the first element. For V2
// list-and-filter, pass the list path and a non-empty id to filter by _id.
func unifiGetObject(ctx context.Context, c interface {
	Get(context.Context, string, map[string]string) (json.RawMessage, error)
}, path, filterID string) (map[string]any, error) {
	raw, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	raw = unifiUnwrap(raw)

	// Try object first.
	var obj map[string]any
	if json.Unmarshal(raw, &obj) == nil && obj != nil {
		if _, hasID := obj["_id"]; hasID || filterID == "" {
			return obj, nil
		}
	}
	// Fall back to a list and filter by _id.
	var list []map[string]any
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("unexpected response shape: %w", err)
	}
	for _, o := range list {
		if filterID == "" {
			return o, nil
		}
		if idStr(o["_id"]) == filterID {
			return o, nil
		}
	}
	return nil, fmt.Errorf("no object found for id %q", filterID)
}

// unifiToggleEnabled performs a read-modify-write that sets the "enabled" field
// on a resource and PUTs it back. getPath fetches the object (V1 single or V2
// list+filter via filterID); putPath is where the merged object is written.
func unifiToggleEnabled(cmd *cobra.Command, flags *rootFlags, getPath, putPath, filterID string, enabled bool) error {
	c, err := flags.newClient()
	if err != nil {
		return err
	}
	obj, err := unifiGetObject(cmd.Context(), c, getPath, filterID)
	if err != nil {
		return err
	}
	obj["enabled"] = enabled
	data, _, err := c.Put(cmd.Context(), putPath, obj)
	if err != nil {
		return err
	}
	return unifiEmit(cmd, flags, data)
}

// idStr renders an arbitrary JSON scalar id as a string.
func idStr(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}

// storeObjects lists a synced resource type and decodes each row to a map.
func storeObjects(db *store.Store, resourceType string, limit int) ([]map[string]any, error) {
	rows, err := db.List(resourceType, limit)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		var o map[string]any
		if json.Unmarshal(r, &o) == nil && o != nil {
			out = append(out, o)
		}
	}
	return out, nil
}

// objStr reads a string field from a decoded JSON object (missing → "").
func objStr(o map[string]any, key string) string {
	if v, ok := o[key]; ok {
		return idStr(v)
	}
	return ""
}

// objNum reads a numeric field as float64 (missing/non-numeric → 0).
func objNum(o map[string]any, key string) float64 {
	switch t := o[key].(type) {
	case float64:
		return t
	case json.Number:
		f, _ := t.Float64()
		return f
	case int:
		return float64(t)
	}
	return 0
}

// objContainsFold reports whether any of the named string fields contains the
// (case-insensitive) needle. Used by client-hunt ranking.
func objContainsFold(o map[string]any, needle string, keys ...string) bool {
	needle = strings.ToLower(needle)
	for _, k := range keys {
		if strings.Contains(strings.ToLower(objStr(o, k)), needle) {
			return true
		}
	}
	return false
}
