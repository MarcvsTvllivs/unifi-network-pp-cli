// Copyright 2026 marcvstvllivs and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Hand-authored UniFi action commands. The generated spec covers clean REST
// reads and CRUD; this file adds the controller's cmd/{mgr} actions (block,
// reboot, adopt, archive-alarm, …), read-modify-write toggles, and a few
// grouped reads (switch, vpn) — the surface that does not map to a single
// typed endpoint. registerUnifiActions wires them onto the generated resource
// parents (and two new top-level groups) from one hook in root.go, so no
// generated parent file is edited.
package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// registerUnifiActions attaches hand-authored subcommands to their generated
// resource parents and registers the switch/vpn groups.
func registerUnifiActions(root *cobra.Command, flags *rootFlags) {
	attach := func(parent string, children ...*cobra.Command) {
		p := findSubcommand(root, parent)
		if p == nil {
			return
		}
		p.AddCommand(children...)
	}

	attach("clients",
		newClientsBlockCmd(flags),
		newClientsUnblockCmd(flags),
		newClientsReconnectCmd(flags),
		newClientsForgetCmd(flags),
		newClientsAuthorizeGuestCmd(flags),
		newClientsUnauthorizeGuestCmd(flags),
		newClientsRenameCmd(flags),
		newClientsSetIPCmd(flags),
	)
	attach("devices",
		newDevicesAdoptCmd(flags),
		newDevicesRebootCmd(flags),
		newDevicesLocateCmd(flags),
		newDevicesUpgradeCmd(flags),
		newDevicesProvisionCmd(flags),
		newDevicesSpeedtestCmd(flags),
		newDevicesRFScanCmd(flags),
		newDevicesLEDCmd(flags),
		newDevicesToggleCmd(flags),
		newDevicesSiteLEDsCmd(flags),
	)
	attach("wlans", newToggleCmd(flags, "wlans", unifiV1("/rest/wlanconf"), unifiV1("/rest/wlanconf"), true))
	attach("firewall-policies", newToggleCmd(flags, "firewall-policies", unifiV2("/firewall-policies"), unifiV2("/firewall-policies"), false))
	attach("qos-rules", newToggleCmd(flags, "qos-rules", unifiV2("/qos-rules"), unifiV2("/qos-rules"), false))
	attach("oon-policies", newToggleCmd(flags, "oon-policies", unifiV2("/object-oriented-network-configs"), unifiV2("/object-oriented-network-config"), false))
	attach("traffic-routes", newToggleCmd(flags, "traffic-routes", unifiV2("/trafficroutes"), unifiV2("/trafficroutes"), false))
	attach("port-forwards", newToggleCmd(flags, "port-forwards", unifiV1("/rest/portforward"), unifiV1("/rest/portforward"), true))
	attach("vouchers", newVouchersCreateCmd(flags), newVouchersRevokeCmd(flags))
	attach("events", newEventsArchiveAlarmCmd(flags), newEventsArchiveAllCmd(flags), newEventsListCmd(flags), newEventsAnomaliesCmd(flags))
	attach("system", newSystemBackupCreateCmd(flags), newSystemBackupDeleteCmd(flags), newSystemBackupsCmd(flags))
	attach("stats", newStatsTopClientsCmd(flags))

	root.AddCommand(newSwitchCmd(flags))
	root.AddCommand(newVPNCmd(flags))
}

func findSubcommand(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// macAction builds a "<verb> <mac>" command that POSTs a cmd/{mgr} body of the
// shape {"cmd": cmdName, "mac": <mac>}.
func macAction(flags *rootFlags, use, short, mgrSuffix, cmdName string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a MAC address is required"))
			}
			return unifiPost(cmd, flags, unifiV1(mgrSuffix), map[string]any{"cmd": cmdName, "mac": args[0]})
		},
	}
}

// ---- clients ----

func newClientsBlockCmd(flags *rootFlags) *cobra.Command {
	c := macAction(flags, "block <mac>", "Block a client/device from the network by MAC", "/cmd/stamgr", "block-sta")
	c.Example = "  unifi-network-pp-cli clients block aa:bb:cc:dd:ee:ff"
	return c
}

func newClientsUnblockCmd(flags *rootFlags) *cobra.Command {
	c := macAction(flags, "unblock <mac>", "Unblock a previously blocked client by MAC", "/cmd/stamgr", "unblock-sta")
	c.Annotations = map[string]string{"mcp:read-only": "false"}
	return c
}

func newClientsReconnectCmd(flags *rootFlags) *cobra.Command {
	return macAction(flags, "reconnect <mac>", "Force a client to reconnect (kick) by MAC", "/cmd/stamgr", "kick-sta")
}

func newClientsForgetCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "forget <mac>",
		Short: "Forget a client, deleting its name, notes, fixed IP and history",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a MAC address is required"))
			}
			return unifiPost(cmd, flags, unifiV1("/cmd/stamgr"), map[string]any{"cmd": "forget-sta", "macs": []string{args[0]}})
		},
	}
}

func newClientsAuthorizeGuestCmd(flags *rootFlags) *cobra.Command {
	var minutes, up, down, megabytes int
	cmd := &cobra.Command{
		Use:   "authorize-guest <mac>",
		Short: "Authorize a guest client to access the guest network by MAC",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a MAC address is required"))
			}
			body := map[string]any{"cmd": "authorize-guest", "mac": args[0]}
			if minutes > 0 {
				body["minutes"] = minutes
			}
			if up > 0 {
				body["up"] = up
			}
			if down > 0 {
				body["down"] = down
			}
			if megabytes > 0 {
				body["bytes"] = megabytes
			}
			return unifiPost(cmd, flags, unifiV1("/cmd/stamgr"), body)
		},
	}
	cmd.Flags().IntVar(&minutes, "minutes", 0, "Authorization duration in minutes (0 = controller default)")
	cmd.Flags().IntVar(&up, "up", 0, "Upload speed cap in Kbps")
	cmd.Flags().IntVar(&down, "down", 0, "Download speed cap in Kbps")
	cmd.Flags().IntVar(&megabytes, "megabytes", 0, "Total data quota in MB")
	return cmd
}

func newClientsUnauthorizeGuestCmd(flags *rootFlags) *cobra.Command {
	return macAction(flags, "unauthorize-guest <mac>", "Revoke guest authorization for a client by MAC", "/cmd/stamgr", "unauthorize-guest")
}

func newClientsRenameCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "rename <client-id> <name>",
		Short: "Rename a client by its controller id (see 'clients known' for ids)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 2 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a client id and a new name are required"))
			}
			return unifiPut(cmd, flags, unifiV1("/rest/user/"+args[0]), map[string]any{"name": args[1]})
		},
	}
}

func newClientsSetIPCmd(flags *rootFlags) *cobra.Command {
	var ip string
	var clear bool
	cmd := &cobra.Command{
		Use:   "set-ip <client-id>",
		Short: "Set or clear a client's fixed IP (DHCP reservation) by controller id",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a client id is required"))
			}
			body := map[string]any{}
			if clear {
				body["use_fixedip"] = false
			} else {
				if ip == "" {
					_ = cmd.Usage()
					return usageErr(fmt.Errorf("--ip is required (or pass --clear)"))
				}
				body["use_fixedip"] = true
				body["fixed_ip"] = ip
			}
			return unifiPut(cmd, flags, unifiV1("/rest/user/"+args[0]), body)
		},
	}
	cmd.Flags().StringVar(&ip, "ip", "", "Fixed IP address to assign")
	cmd.Flags().BoolVar(&clear, "clear", false, "Clear the existing fixed IP")
	return cmd
}

// ---- devices ----

func newDevicesAdoptCmd(flags *rootFlags) *cobra.Command {
	return macAction(flags, "adopt <mac>", "Adopt a pending device by MAC", "/cmd/devmgr", "adopt")
}

func newDevicesRebootCmd(flags *rootFlags) *cobra.Command {
	return macAction(flags, "reboot <mac>", "Reboot a device by MAC", "/cmd/devmgr", "restart")
}

func newDevicesUpgradeCmd(flags *rootFlags) *cobra.Command {
	c := macAction(flags, "upgrade <mac>", "Upgrade a device's firmware by MAC (cached firmware)", "/cmd/devmgr", "upgrade")
	c.Example = "  unifi-network-pp-cli devices upgrade aa:bb:cc:dd:ee:ff"
	return c
}

func newDevicesProvisionCmd(flags *rootFlags) *cobra.Command {
	return macAction(flags, "provision <mac>", "Force re-provision a device by MAC (push current config)", "/cmd/devmgr", "force-provision")
}

func newDevicesSpeedtestCmd(flags *rootFlags) *cobra.Command {
	return macAction(flags, "speedtest <mac>", "Trigger a gateway speedtest by MAC (poll 'stats dashboard' for results)", "/cmd/devmgr", "speedtest")
}

func newDevicesRFScanCmd(flags *rootFlags) *cobra.Command {
	return macAction(flags, "rf-scan <mac>", "Trigger an RF spectrum scan on an AP by MAC (takes 5-10 min)", "/cmd/devmgr", "spectrum-scan")
}

func newDevicesLocateCmd(flags *rootFlags) *cobra.Command {
	var off bool
	cmd := &cobra.Command{
		Use:   "locate <mac>",
		Short: "Toggle device locate mode (LED blink) by MAC; --off to stop",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a MAC address is required"))
			}
			cmdName := "set-locate"
			if off {
				cmdName = "unset-locate"
			}
			return unifiPost(cmd, flags, unifiV1("/cmd/devmgr"), map[string]any{"cmd": cmdName, "mac": args[0]})
		},
	}
	cmd.Flags().BoolVar(&off, "off", false, "Stop locating (turn the blink off)")
	return cmd
}

func newDevicesLEDCmd(flags *rootFlags) *cobra.Command {
	var mode string
	cmd := &cobra.Command{
		Use:   "led <device-id>",
		Short: "Set a device LED override (on|off|default) by controller id",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a device id is required"))
			}
			switch mode {
			case "on", "off", "default":
			default:
				return usageErr(fmt.Errorf("--mode must be on, off, or default"))
			}
			return unifiPut(cmd, flags, unifiV1("/rest/device/"+args[0]), map[string]any{"led_override": mode})
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "default", "LED override: on | off | default")
	return cmd
}

func newDevicesToggleCmd(flags *rootFlags) *cobra.Command {
	var disabled bool
	cmd := &cobra.Command{
		Use:   "toggle <device-id>",
		Short: "Enable or disable a device without unadopting it (--disabled)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a device id is required"))
			}
			return unifiPut(cmd, flags, unifiV1("/rest/device/"+args[0]), map[string]any{"disabled": disabled})
		},
	}
	cmd.Flags().BoolVar(&disabled, "disabled", false, "Disable the device (omit to enable)")
	return cmd
}

func newDevicesSiteLEDsCmd(flags *rootFlags) *cobra.Command {
	var enabled bool
	cmd := &cobra.Command{
		Use:   "site-leds",
		Short: "Toggle all device LEDs site-wide (--enabled)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			obj, err := unifiGetObject(cmd.Context(), c, unifiV1("/get/setting/mgmt"), "")
			if err != nil {
				return err
			}
			obj["led_enabled"] = enabled
			obj["key"] = "mgmt"
			data, _, err := c.Put(cmd.Context(), unifiV1("/set/setting/mgmt"), obj)
			if err != nil {
				return err
			}
			return unifiEmit(cmd, flags, data)
		},
	}
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Turn site-wide LEDs on (false to turn off)")
	return cmd
}

// ---- toggles ----

// newToggleCmd builds a "<resource> toggle <id> --enabled" command. getBase is
// the path to fetch the object (single-id for V1, list path for V2 filtered by
// id); putBase/{id} is where the modified object is written.
func newToggleCmd(flags *rootFlags, resource, getBase, putBase string, v1Single bool) *cobra.Command {
	var enabled bool
	cmd := &cobra.Command{
		Use:   "toggle <id>",
		Short: "Enable or disable this resource by id (--enabled)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("an id is required"))
			}
			id := args[0]
			getPath := getBase
			filterID := id
			if v1Single {
				getPath = getBase + "/" + id
				filterID = ""
			}
			return unifiToggleEnabled(cmd, flags, getPath, putBase+"/"+id, filterID, enabled)
		},
	}
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable (true) or disable (false)")
	return cmd
}

// ---- vouchers ----

func newVouchersCreateCmd(flags *rootFlags) *cobra.Command {
	var minutes, count, quota, up, down, megabytes int
	var note string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create hotspot voucher(s) for guest access",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() == 0 && len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			body := map[string]any{"cmd": "create-voucher", "expire": minutes, "n": count, "quota": quota}
			if note != "" {
				body["note"] = note
			}
			if up > 0 {
				body["up"] = up
			}
			if down > 0 {
				body["down"] = down
			}
			if megabytes > 0 {
				body["bytes"] = megabytes
			}
			return unifiPost(cmd, flags, unifiV1("/cmd/hotspot"), body)
		},
	}
	cmd.Flags().IntVar(&minutes, "minutes", 1440, "Validity in minutes after activation")
	cmd.Flags().IntVar(&count, "count", 1, "Number of vouchers to create")
	cmd.Flags().IntVar(&quota, "quota", 1, "Uses per voucher (0 = multi-use, 1 = single-use, n = n times)")
	cmd.Flags().StringVar(&note, "note", "", "Note attached to the voucher(s)")
	cmd.Flags().IntVar(&up, "up", 0, "Upload cap in Kbps")
	cmd.Flags().IntVar(&down, "down", 0, "Download cap in Kbps")
	cmd.Flags().IntVar(&megabytes, "megabytes", 0, "Data cap in MB")
	return cmd
}

func newVouchersRevokeCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <voucher-id>",
		Short: "Revoke a hotspot voucher by its id",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a voucher id is required"))
			}
			return unifiPost(cmd, flags, unifiV1("/cmd/hotspot"), map[string]any{"cmd": "delete-voucher", "_id": args[0]})
		},
	}
}

// ---- events ----

func newEventsArchiveAlarmCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "archive-alarm <alarm-id>",
		Short: "Archive (resolve/dismiss) a specific alarm by id",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("an alarm id is required"))
			}
			return unifiPost(cmd, flags, unifiV1("/cmd/evtmgr"), map[string]any{"cmd": "archive-alarm", "_id": args[0]})
		},
	}
}

func newEventsArchiveAllCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "archive-all",
		Short: "Archive (resolve/dismiss) all active alarms",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			return unifiPost(cmd, flags, unifiV1("/cmd/evtmgr"), map[string]any{"cmd": "archive-all-alarms"})
		},
	}
}

func newEventsListCmd(flags *rootFlags) *cobra.Command {
	var withinHours, limit int
	var eventType string
	cmd := &cobra.Command{
		Use:         "list",
		Short:       "List event log entries (connects, state changes, config changes)",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			body := map[string]any{"within": withinHours, "_limit": limit, "_start": 0}
			if eventType != "" {
				body["type"] = eventType
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, _, err := c.PostQueryWithParams(cmd.Context(), unifiV1("/stat/event"), nil, body)
			if err != nil {
				return err
			}
			return unifiEmit(cmd, flags, data)
		},
	}
	cmd.Flags().IntVar(&withinHours, "within-hours", 24, "Look-back window in hours")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum events to return")
	cmd.Flags().StringVar(&eventType, "type", "", "Filter by event type prefix (e.g. EVT_AP_)")
	return cmd
}

func newEventsAnomaliesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "anomalies",
		Short: "List network anomaly detection events (last 24h)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, _, err := c.PostQueryWithParams(cmd.Context(), unifiV1("/stat/anomalies"), nil, map[string]any{})
			if err != nil {
				return err
			}
			return unifiEmit(cmd, flags, data)
		},
	}
	cmd.Annotations = map[string]string{"mcp:read-only": "true"}
	return cmd
}

// ---- system backups ----

func newSystemBackupCreateCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "backup-create",
		Short: "Create a backup of the controller configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			return unifiPost(cmd, flags, unifiV1("/cmd/backup"), map[string]any{"cmd": "backup"})
		},
	}
}

func newSystemBackupDeleteCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "backup-delete <filename>",
		Short: "Delete a backup file by filename (see 'system backups')",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a backup filename is required"))
			}
			return unifiPost(cmd, flags, unifiV1("/cmd/backup"), map[string]any{"cmd": "delete-backup", "filename": args[0]})
		},
	}
}

func newSystemBackupsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backups",
		Short: "List available controller backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, _, err := c.PostQueryWithParams(cmd.Context(), unifiV1("/cmd/backup"), nil, map[string]any{"cmd": "list-backups"})
			if err != nil {
				return err
			}
			return unifiEmit(cmd, flags, data)
		},
	}
	cmd.Annotations = map[string]string{"mcp:read-only": "true"}
	return cmd
}

// ---- stats top-clients ----

func newStatsTopClientsCmd(flags *rootFlags) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "top-clients",
		Short: "Top clients by total traffic (rx+tx bytes)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			raw, err := c.Get(cmd.Context(), unifiV1("/stat/sta"), nil)
			if err != nil {
				return err
			}
			var clients []map[string]any
			if err := json.Unmarshal(unifiUnwrap(raw), &clients); err != nil {
				return fmt.Errorf("parsing clients: %w", err)
			}
			sort.Slice(clients, func(i, j int) bool {
				return objNum(clients[i], "rx_bytes")+objNum(clients[i], "tx_bytes") >
					objNum(clients[j], "rx_bytes")+objNum(clients[j], "tx_bytes")
			})
			if limit > 0 && len(clients) > limit {
				clients = clients[:limit]
			}
			out, _ := json.Marshal(clients)
			return unifiEmit(cmd, flags, out)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Number of top clients to return")
	cmd.Annotations = map[string]string{"mcp:read-only": "true"}
	return cmd
}

// ---- switch group ----

func newSwitchCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch",
		Short: "Switch operations (PoE power-cycle, port tables)",
	}
	cmd.AddCommand(newSwitchPowerCycleCmd(flags), newSwitchPortsCmd(flags))
	return cmd
}

func newSwitchPowerCycleCmd(flags *rootFlags) *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:   "power-cycle <switch-mac>",
		Short: "Power-cycle PoE on a switch port (reboots the powered device)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a switch MAC address is required"))
			}
			if port <= 0 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--port is required (1-based port index)"))
			}
			return unifiPost(cmd, flags, unifiV1("/cmd/devmgr"), map[string]any{"cmd": "power-cycle", "mac": args[0], "port_idx": port})
		},
	}
	cmd.Flags().IntVar(&port, "port", 0, "1-based switch port index to power-cycle")
	return cmd
}

func newSwitchPortsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ports <switch-mac>",
		Short: "Show the live port table for a switch by MAC",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a switch MAC address is required"))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			raw, err := c.Get(cmd.Context(), unifiV1("/stat/device/"+args[0]), nil)
			if err != nil {
				return err
			}
			var devs []map[string]any
			if err := json.Unmarshal(unifiUnwrap(raw), &devs); err != nil || len(devs) == 0 {
				return fmt.Errorf("device %s not found", args[0])
			}
			ports := devs[0]["port_table"]
			out, _ := json.Marshal(ports)
			return unifiEmit(cmd, flags, out)
		},
	}
	cmd.Annotations = map[string]string{"mcp:read-only": "true"}
	return cmd
}

// ---- vpn group ----

func newVPNCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vpn",
		Short: "VPN servers and clients (stored in network configs)",
	}
	cmd.AddCommand(newVPNListCmd(flags, "servers", []string{"vpn-server", "remote-user-vpn", "site-vpn"}, "server"),
		newVPNListCmd(flags, "clients", []string{"vpn-client"}, "client"),
		newVPNToggleCmd(flags))
	return cmd
}

func newVPNListCmd(flags *rootFlags, use string, purposes []string, vpnTypeToken string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: "List VPN " + use + " configured on the controller",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			raw, err := c.Get(cmd.Context(), unifiV1("/rest/networkconf"), nil)
			if err != nil {
				return err
			}
			var nets []map[string]any
			if err := json.Unmarshal(unifiUnwrap(raw), &nets); err != nil {
				return fmt.Errorf("parsing networks: %w", err)
			}
			matched := make([]map[string]any, 0)
			for _, n := range nets {
				purpose := strings.ToLower(objStr(n, "purpose"))
				vpnType := strings.ToLower(objStr(n, "vpn_type"))
				isMatch := strings.Contains(vpnType, vpnTypeToken)
				for _, p := range purposes {
					if purpose == p {
						isMatch = true
						break
					}
				}
				if isMatch {
					matched = append(matched, n)
				}
			}
			out, _ := json.Marshal(matched)
			return unifiEmit(cmd, flags, out)
		},
	}
	cmd.Annotations = map[string]string{"mcp:read-only": "true"}
	return cmd
}

func newVPNToggleCmd(flags *rootFlags) *cobra.Command {
	var enabled bool
	cmd := &cobra.Command{
		Use:   "toggle <network-id>",
		Short: "Enable or disable a VPN config by its network id (--enabled)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a network id is required"))
			}
			id := args[0]
			return unifiToggleEnabled(cmd, flags, unifiV1("/rest/networkconf/"+id), unifiV1("/rest/networkconf/"+id), "", enabled)
		},
	}
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable (true) or disable (false)")
	return cmd
}
