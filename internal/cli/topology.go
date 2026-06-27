// Copyright 2026 marcvstvllivs and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Hand-authored transcendence command: topology. Builds the AP/switch/gateway
// uplink hierarchy from the locally synced device store — a cross-device join
// no single controller call returns.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"unifi-network-pp-cli/internal/store"
)

const novelMaxRows = 100000

// topoNode is one device in the rendered uplink tree.
type topoNode struct {
	Name     string      `json:"name"`
	MAC      string      `json:"mac"`
	Model    string      `json:"model"`
	Type     string      `json:"type"`
	IP       string      `json:"ip,omitempty"`
	Children []*topoNode `json:"children,omitempty"`
}

// buildTopology turns a flat device list into uplink-rooted trees. Devices
// whose uplink MAC is unknown (gateways, or anything whose parent isn't in the
// set) become roots. Pure function for testability.
func buildTopology(devices []map[string]any) []*topoNode {
	byMAC := make(map[string]*topoNode, len(devices))
	uplink := make(map[string]string, len(devices))
	order := make([]string, 0, len(devices))

	for _, d := range devices {
		mac := objStr(d, "mac")
		if mac == "" {
			continue
		}
		byMAC[mac] = &topoNode{
			Name:  firstNonEmpty(objStr(d, "name"), objStr(d, "model"), mac),
			MAC:   mac,
			Model: objStr(d, "model"),
			Type:  objStr(d, "type"),
			IP:    objStr(d, "ip"),
		}
		order = append(order, mac)
		if up, ok := d["uplink"].(map[string]any); ok {
			uplink[mac] = objStr(up, "uplink_mac")
		}
	}

	var roots []*topoNode
	for _, mac := range order {
		node := byMAC[mac]
		parentMAC := uplink[mac]
		if parent, ok := byMAC[parentMAC]; ok && parentMAC != mac {
			parent.Children = append(parent.Children, node)
		} else {
			roots = append(roots, node)
		}
	}
	return roots
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// pp:data-source local
func newNovelTopologyCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "topology",
		Short:       "Render the AP/switch/gateway uplink hierarchy of your whole site as a tree from the local store.",
		Long:        "Render the device uplink/LLDP hierarchy of your whole site as a tree, joined from the locally synced device store. Run `sync --full` first. Do NOT use this for live port statistics; use 'switch ports'.",
		Example:     "  unifi-network-pp-cli topology --json",
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
			devices, err := storeObjects(db, "devices", novelMaxRows)
			if err != nil {
				return err
			}
			roots := buildTopology(devices)
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
				"site_roots":   roots,
				"device_count": len(devices),
			}, flags)
		},
	}
	return cmd
}
