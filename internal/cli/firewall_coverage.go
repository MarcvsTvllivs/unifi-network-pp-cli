// Copyright 2026 marcvstvllivs and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Hand-authored transcendence command: firewall-coverage. Joins synced
// firewall policies by zone pair to show which source→destination zone pairs
// have rules and which are wide open — a posture view no single call returns.
package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"unifi-network-pp-cli/internal/store"
)

type zonePairCoverage struct {
	SourceZone string `json:"source_zone"`
	DestZone   string `json:"destination_zone"`
	Policies   int    `json:"policies"`
	Allow      int    `json:"allow"`
	Block      int    `json:"block"`
	Reject     int    `json:"reject"`
	Enabled    int    `json:"enabled"`
}

// firewallCoverage groups policies by (source zone, destination zone) and
// tallies actions. zoneNames maps zone id → display name when available. Pure
// function for testability.
func firewallCoverage(policies []map[string]any, zoneNames map[string]string) []zonePairCoverage {
	type key struct{ src, dst string }
	agg := map[key]*zonePairCoverage{}

	name := func(id string) string {
		if n, ok := zoneNames[id]; ok && n != "" {
			return n
		}
		if id == "" {
			return "(any)"
		}
		return id
	}

	for _, p := range policies {
		src := policyZone(p, "source")
		dst := policyZone(p, "destination")
		k := key{src, dst}
		c := agg[k]
		if c == nil {
			c = &zonePairCoverage{SourceZone: name(src), DestZone: name(dst)}
			agg[k] = c
		}
		c.Policies++
		if objBool(p, "enabled") {
			c.Enabled++
		}
		switch objStr(p, "action") {
		case "ALLOW", "allow":
			c.Allow++
		case "BLOCK", "block":
			c.Block++
		case "REJECT", "reject":
			c.Reject++
		}
	}

	out := make([]zonePairCoverage, 0, len(agg))
	for _, c := range agg {
		out = append(out, *c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SourceZone != out[j].SourceZone {
			return out[i].SourceZone < out[j].SourceZone
		}
		return out[i].DestZone < out[j].DestZone
	})
	return out
}

// policyZone extracts the zone id from a policy's source/destination object.
func policyZone(p map[string]any, side string) string {
	if obj, ok := p[side].(map[string]any); ok {
		return objStr(obj, "zone_id")
	}
	return ""
}

// pp:data-source local
func newNovelFirewallCoverageCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "firewall-coverage",
		Short:       "Show which firewall zone pairs have policies and which are wide open, joining zones, policies, and groups locally.",
		Long:        "Group synced firewall policies by source→destination zone pair and tally actions to review posture. Run `sync --full` first. Do NOT use this to edit rules; use 'firewall-policies'.",
		Example:     "  unifi-network-pp-cli firewall-coverage --json",
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
			policies, err := storeObjects(db, "firewall-policies", novelMaxRows)
			if err != nil {
				return err
			}
			zones, _ := storeObjects(db, "firewall-zones", novelMaxRows)
			zoneNames := make(map[string]string, len(zones))
			for _, z := range zones {
				if id := objStr(z, "_id"); id != "" {
					zoneNames[id] = objStr(z, "name")
				}
			}
			coverage := firewallCoverage(policies, zoneNames)
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
				"zone_pairs":   coverage,
				"policy_count": len(policies),
			}, flags)
		},
	}
	return cmd
}
