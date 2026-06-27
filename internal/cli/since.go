// Copyright 2026 marcvstvllivs and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Hand-authored transcendence command: since. Diffs the live controller config
// against the last synced snapshot in the local store to show what was added,
// removed, or changed — a time-windowed config diff the controller does not
// expose. Requires a reachable controller (live) and a prior `sync --full`.
package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"unifi-network-pp-cli/internal/store"
)

// sinceResource pairs a syncable config resource with its live list path.
type sinceResource struct {
	name string
	path string // controller path with __SITE__ sentinel
}

func sinceResources() []sinceResource {
	return []sinceResource{
		{"networks", unifiV1("/rest/networkconf")},
		{"wlans", unifiV1("/rest/wlanconf")},
		{"firewall-policies", unifiV2("/firewall-policies")},
		{"firewall-groups", unifiV1("/rest/firewallgroup")},
		{"port-forwards", unifiV1("/rest/portforward")},
		{"routes", unifiV1("/rest/routing")},
	}
}

type resourceDiff struct {
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
	Changed []string `json:"changed"`
}

// volatileFields are excluded from the change hash so live-only telemetry does
// not register as a config change.
var volatileFields = map[string]bool{
	"last_seen": true, "uptime": true, "_uptime": true, "rx_bytes": true,
	"tx_bytes": true, "num_sta": true, "latency": true, "speedtest_status": true,
	"_id": true,
}

// configHash produces a stable hash of an object's non-volatile fields. Pure
// function for testability.
func configHash(obj map[string]any) string {
	cleaned := make(map[string]any, len(obj))
	for k, v := range obj {
		if volatileFields[k] {
			continue
		}
		cleaned[k] = v
	}
	b, _ := json.Marshal(cleaned)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// diffResources compares a prior snapshot to current state by _id and content
// hash. Pure function for testability.
func diffResources(prior, current []map[string]any) resourceDiff {
	priorByID := indexByID(prior)
	currentByID := indexByID(current)
	d := resourceDiff{Added: []string{}, Removed: []string{}, Changed: []string{}}

	for id, cur := range currentByID {
		old, ok := priorByID[id]
		if !ok {
			d.Added = append(d.Added, displayID(cur, id))
			continue
		}
		if configHash(cur) != configHash(old) {
			d.Changed = append(d.Changed, displayID(cur, id))
		}
	}
	for id, old := range priorByID {
		if _, ok := currentByID[id]; !ok {
			d.Removed = append(d.Removed, displayID(old, id))
		}
	}
	sort.Strings(d.Added)
	sort.Strings(d.Removed)
	sort.Strings(d.Changed)
	return d
}

func indexByID(objs []map[string]any) map[string]map[string]any {
	out := make(map[string]map[string]any, len(objs))
	for _, o := range objs {
		id := objStr(o, "_id")
		if id == "" {
			id = objStr(o, "mac")
		}
		if id != "" {
			out[id] = o
		}
	}
	return out
}

func displayID(obj map[string]any, id string) string {
	if name := objStr(obj, "name"); name != "" {
		return name + " (" + id + ")"
	}
	return id
}

// pp:data-source local
func newNovelSinceCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "since",
		Short:       "Diff current controller state against the last synced snapshot to show added/removed/changed resources.",
		Long:        "Diff the live controller config against the last `sync --full` snapshot in the local store, per resource. Requires a reachable controller. Do NOT use this for the live event log; use 'events list'.",
		Example:     "  unifi-network-pp-cli since --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			db, err := store.OpenReadOnly(defaultDBPath("unifi-network-pp-cli"))
			if err != nil {
				return fmt.Errorf("opening local store: %w (run `unifi-network-pp-cli sync --full` first)", err)
			}
			defer db.Close()
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			diffs := map[string]resourceDiff{}
			totalChanges := 0
			for _, r := range sinceResources() {
				prior, _ := storeObjects(db, r.name, novelMaxRows)
				raw, err := c.Get(cmd.Context(), r.path, nil)
				if err != nil {
					return err
				}
				var current []map[string]any
				if err := json.Unmarshal(unifiUnwrap(raw), &current); err != nil {
					continue
				}
				d := diffResources(prior, current)
				if len(d.Added)+len(d.Removed)+len(d.Changed) > 0 {
					diffs[r.name] = d
					totalChanges += len(d.Added) + len(d.Removed) + len(d.Changed)
				}
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
				"changes":       diffs,
				"total_changes": totalChanges,
			}, flags)
		},
	}
	return cmd
}
