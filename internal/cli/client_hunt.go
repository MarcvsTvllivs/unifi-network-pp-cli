// Copyright 2026 marcvstvllivs and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Hand-authored transcendence command: client-hunt. Ranks synced clients by a
// query match across name/hostname/ip/mac/vendor in one shot — covering both
// online and historically-known clients the UI hides once they go offline.
package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"unifi-network-pp-cli/internal/store"
)

type huntResult struct {
	Score    int            `json:"score"`
	Match    string         `json:"matched_on"`
	Name     string         `json:"name"`
	Hostname string         `json:"hostname,omitempty"`
	IP       string         `json:"ip,omitempty"`
	MAC      string         `json:"mac"`
	Vendor   string         `json:"vendor,omitempty"`
	Network  string         `json:"network,omitempty"`
	Wired    bool           `json:"wired"`
	Raw      map[string]any `json:"-"`
}

// huntClients ranks clients against a query. Exact field equality scores
// highest, then prefix, then substring; vendor/oui matches score lowest. Pure
// function for testability.
func huntClients(clients []map[string]any, query string) []huntResult {
	q := strings.ToLower(strings.TrimSpace(query))
	var out []huntResult
	for _, c := range clients {
		fields := []struct {
			label  string
			value  string
			weight int
		}{
			{"mac", strings.ToLower(objStr(c, "mac")), 100},
			{"ip", strings.ToLower(firstNonEmpty(objStr(c, "ip"), objStr(c, "fixed_ip"))), 90},
			{"name", strings.ToLower(firstNonEmpty(objStr(c, "name"), objStr(c, "unifi_device_name"))), 80},
			{"hostname", strings.ToLower(objStr(c, "hostname")), 70},
			{"vendor", strings.ToLower(firstNonEmpty(objStr(c, "oui"), objStr(c, "dev_vendor"))), 40},
		}
		best := 0
		matchedOn := ""
		for _, f := range fields {
			if f.value == "" {
				continue
			}
			score := 0
			switch {
			case f.value == q:
				score = f.weight + 50
			case strings.HasPrefix(f.value, q):
				score = f.weight + 20
			case strings.Contains(f.value, q):
				score = f.weight
			}
			if score > best {
				best = score
				matchedOn = f.label
			}
		}
		if best == 0 {
			continue
		}
		out = append(out, huntResult{
			Score:    best,
			Match:    matchedOn,
			Name:     firstNonEmpty(objStr(c, "name"), objStr(c, "hostname"), objStr(c, "mac")),
			Hostname: objStr(c, "hostname"),
			IP:       firstNonEmpty(objStr(c, "ip"), objStr(c, "fixed_ip")),
			MAC:      objStr(c, "mac"),
			Vendor:   firstNonEmpty(objStr(c, "oui"), objStr(c, "dev_vendor")),
			Network:  objStr(c, "network"),
			Wired:    objBool(c, "is_wired"),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

func objBool(o map[string]any, key string) bool {
	b, _ := o[key].(bool)
	return b
}

// pp:data-source local
func newNovelClientHuntCmd(flags *rootFlags) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:         "client-hunt <query>",
		Short:       "Locate any client (online or offline) by name, IP, MAC, or vendor with device/network context in one ranked answer.",
		Long:        "Search the locally synced client store by name, hostname, IP, MAC, or vendor and return ranked matches. Run `sync --full` first. Finds offline/known clients the UI hides.",
		Example:     "  unifi-network-pp-cli client-hunt iphone --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a search query is required"))
			}
			db, err := store.OpenReadOnly(defaultDBPath("unifi-network-pp-cli"))
			if err != nil {
				return fmt.Errorf("opening local store: %w (run `unifi-network-pp-cli sync --full` first)", err)
			}
			defer db.Close()
			clients, err := storeObjects(db, "clients", novelMaxRows)
			if err != nil {
				return err
			}
			results := huntClients(clients, args[0])
			if limit > 0 && len(results) > limit {
				results = results[:limit]
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
				"query":   args[0],
				"matches": results,
				"count":   len(results),
			}, flags)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum matches to return")
	return cmd
}
