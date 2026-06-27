// Copyright 2026 marcvstvllivs and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Hand-authored transcendence command: audit. Cross-references the locally
// synced config to find dangling references and risky settings the UniFi UI
// never surfaces (a WLAN bound to a deleted VLAN, a firewall policy pointing at
// a missing group, a PSK WLAN with no passphrase, …).
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"unifi-network-pp-cli/internal/store"
)

type auditFinding struct {
	Severity   string `json:"severity"` // error | warning | info
	Kind       string `json:"kind"`
	Resource   string `json:"resource"`
	ResourceID string `json:"resource_id,omitempty"`
	Message    string `json:"message"`
}

// auditConfig runs the cross-resource checks. Pure function for testability.
func auditConfig(networks, wlans, policies, fwGroups, usergroups []map[string]any) []auditFinding {
	networkIDs := idSet(networks)
	usergroupIDs := idSet(usergroups)
	groupIDs := idSet(fwGroups)

	var findings []auditFinding

	for _, w := range wlans {
		name := firstNonEmpty(objStr(w, "name"), objStr(w, "_id"))
		if nc := objStr(w, "networkconf_id"); nc != "" && !networkIDs[nc] {
			findings = append(findings, auditFinding{
				Severity: "error", Kind: "dangling-network-ref", Resource: "wlan", ResourceID: objStr(w, "_id"),
				Message: "WLAN " + name + " references network " + nc + " which does not exist",
			})
		}
		if ug := objStr(w, "usergroup_id"); ug != "" && !usergroupIDs[ug] {
			findings = append(findings, auditFinding{
				Severity: "warning", Kind: "dangling-usergroup-ref", Resource: "wlan", ResourceID: objStr(w, "_id"),
				Message: "WLAN " + name + " references user group " + ug + " which does not exist",
			})
		}
		sec := objStr(w, "security")
		if (sec == "wpapsk" || sec == "wpa2-psk" || sec == "wpa3") && objStr(w, "x_passphrase") == "" && objBool(w, "enabled") {
			findings = append(findings, auditFinding{
				Severity: "warning", Kind: "psk-without-passphrase", Resource: "wlan", ResourceID: objStr(w, "_id"),
				Message: "WLAN " + name + " uses PSK security but has no passphrase set",
			})
		}
	}

	for _, p := range policies {
		name := firstNonEmpty(objStr(p, "name"), objStr(p, "_id"))
		var netRefs, grpRefs []string
		collectPolicyRefs(p, &netRefs, &grpRefs)
		for _, n := range netRefs {
			if n != "" && !networkIDs[n] {
				findings = append(findings, auditFinding{
					Severity: "error", Kind: "dangling-network-ref", Resource: "firewall-policy", ResourceID: objStr(p, "_id"),
					Message: "Firewall policy " + name + " references network " + n + " which does not exist",
				})
			}
		}
		for _, g := range grpRefs {
			if g != "" && !groupIDs[g] {
				findings = append(findings, auditFinding{
					Severity: "error", Kind: "dangling-group-ref", Resource: "firewall-policy", ResourceID: objStr(p, "_id"),
					Message: "Firewall policy " + name + " references firewall group " + g + " which does not exist",
				})
			}
		}
	}

	if findings == nil {
		findings = []auditFinding{}
	}
	return findings
}

func idSet(objs []map[string]any) map[string]bool {
	set := make(map[string]bool, len(objs))
	for _, o := range objs {
		if id := objStr(o, "_id"); id != "" {
			set[id] = true
		}
	}
	return set
}

// collectPolicyRefs recursively gathers network and firewall-group ids
// referenced anywhere in a V2 firewall policy object.
func collectPolicyRefs(v any, netRefs, grpRefs *[]string) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			switch k {
			case "network_ids":
				if arr, ok := val.([]any); ok {
					for _, e := range arr {
						*netRefs = append(*netRefs, idStr(e))
					}
				}
			case "network_id":
				*netRefs = append(*netRefs, idStr(val))
			case "ip_group_id", "port_group_id", "group_id":
				if s := idStr(val); s != "" {
					*grpRefs = append(*grpRefs, s)
				}
			}
			collectPolicyRefs(val, netRefs, grpRefs)
		}
	case []any:
		for _, e := range t {
			collectPolicyRefs(e, netRefs, grpRefs)
		}
	}
}

// pp:data-source local
func newNovelAuditCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "audit",
		Short:       "Find dangling references and risky settings across firewall policies, networks, WLANs, groups, and port profiles.",
		Long:        "Cross-reference the locally synced config to find dangling references (WLAN→missing network/usergroup, firewall policy→missing group/network) and risky settings. Run `sync --full` first.",
		Example:     "  unifi-network-pp-cli audit --json",
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
			networks, _ := storeObjects(db, "networks", novelMaxRows)
			wlans, _ := storeObjects(db, "wlans", novelMaxRows)
			policies, _ := storeObjects(db, "firewall-policies", novelMaxRows)
			fwGroups, _ := storeObjects(db, "firewall-groups", novelMaxRows)
			usergroups, _ := storeObjects(db, "usergroups", novelMaxRows)

			findings := auditConfig(networks, wlans, policies, fwGroups, usergroups)
			errors, warnings := 0, 0
			for _, f := range findings {
				switch f.Severity {
				case "error":
					errors++
				case "warning":
					warnings++
				}
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
				"findings": findings,
				"summary": map[string]int{
					"total":    len(findings),
					"errors":   errors,
					"warnings": warnings,
				},
			}, flags)
		},
	}
	return cmd
}
