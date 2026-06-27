// Copyright 2026 marcvstvllivs and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Hand-authored transcendence command: poe-budget. Sums per-port PoE draw from
// the locally synced device store against each switch's reported power budget
// and flags switches near capacity.
package cli

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/spf13/cobra"
	"unifi-network-pp-cli/internal/store"
)

type poeReport struct {
	Name        string  `json:"name"`
	MAC         string  `json:"mac"`
	Model       string  `json:"model"`
	PoEPorts    int     `json:"poe_ports"`
	DrawWatts   float64 `json:"draw_watts"`
	BudgetWatts float64 `json:"budget_watts,omitempty"`
	PercentUsed float64 `json:"percent_used,omitempty"`
	Status      string  `json:"status"`
}

// poeFloat reads a numeric field that UniFi sometimes encodes as a JSON string
// (e.g. "12.95" watts) or a number.
func poeFloat(o map[string]any, key string) float64 {
	switch t := o[key].(type) {
	case float64:
		return t
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	}
	return 0
}

// summarizePoE computes a PoE report per switch-like device. Pure function for
// testability. A device is considered a switch if it has a port_table; the
// budget is read from system-stats fields when the controller reports them.
func summarizePoE(devices []map[string]any) []poeReport {
	var reports []poeReport
	for _, d := range devices {
		ports, ok := d["port_table"].([]any)
		if !ok || len(ports) == 0 {
			continue
		}
		var draw float64
		poePorts := 0
		hasPoE := false
		for _, p := range ports {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if objBool(pm, "port_poe") || pm["poe_power"] != nil {
				hasPoE = true
			}
			pw := poeFloat(pm, "poe_power")
			if pw > 0 {
				draw += pw
				poePorts++
			}
		}
		if !hasPoE {
			continue
		}
		budget := firstPositive(
			poeFloat(d, "total_max_power"),
			poeFloat(d, "max_power"),
		)
		rep := poeReport{
			Name:      firstNonEmpty(objStr(d, "name"), objStr(d, "model"), objStr(d, "mac")),
			MAC:       objStr(d, "mac"),
			Model:     objStr(d, "model"),
			PoEPorts:  poePorts,
			DrawWatts: round1(draw),
			Status:    "ok",
		}
		if budget > 0 {
			rep.BudgetWatts = round1(budget)
			rep.PercentUsed = round1(draw / budget * 100)
			switch {
			case rep.PercentUsed >= 90:
				rep.Status = "critical"
			case rep.PercentUsed >= 75:
				rep.Status = "warning"
			}
		} else {
			rep.Status = "unknown-budget"
		}
		reports = append(reports, rep)
	}
	sort.SliceStable(reports, func(i, j int) bool { return reports[i].DrawWatts > reports[j].DrawWatts })
	return reports
}

func firstPositive(vals ...float64) float64 {
	for _, v := range vals {
		if v > 0 {
			return v
		}
	}
	return 0
}

func round1(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}

// pp:data-source local
func newNovelPoeBudgetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "poe-budget",
		Short:       "Sum per-port PoE draw against each switch's power capacity and flag switches near their budget.",
		Long:        "Summarize PoE power draw per switch from the locally synced device store and flag switches near their reported power budget. Run `sync --full` first.",
		Example:     "  unifi-network-pp-cli poe-budget --json",
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
			reports := summarizePoE(devices)
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
				"switches": reports,
				"count":    len(reports),
			}, flags)
		},
	}
	return cmd
}
