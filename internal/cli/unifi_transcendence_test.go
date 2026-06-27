// Copyright 2026 marcvstvllivs and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Tests for the pure logic behind the hand-authored transcendence commands.
package cli

import "testing"

func TestBuildTopology(t *testing.T) {
	devices := []map[string]any{
		{"mac": "gw", "name": "Gateway", "type": "ugw"},
		{"mac": "sw", "name": "Switch", "type": "usw", "uplink": map[string]any{"uplink_mac": "gw"}},
		{"mac": "ap", "name": "AP", "type": "uap", "uplink": map[string]any{"uplink_mac": "sw"}},
		{"mac": "orphan", "name": "Orphan", "type": "uap", "uplink": map[string]any{"uplink_mac": "missing"}},
	}
	roots := buildTopology(devices)
	// Gateway and orphan (parent not in set) are roots.
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
	var gw *topoNode
	for _, r := range roots {
		if r.MAC == "gw" {
			gw = r
		}
	}
	if gw == nil || len(gw.Children) != 1 || gw.Children[0].MAC != "sw" {
		t.Fatalf("gateway should have switch child, got %+v", gw)
	}
	if len(gw.Children[0].Children) != 1 || gw.Children[0].Children[0].MAC != "ap" {
		t.Fatalf("switch should have AP child")
	}
}

func TestHuntClients(t *testing.T) {
	clients := []map[string]any{
		{"mac": "aa:bb", "name": "Jannis iPhone", "ip": "192.168.1.5", "oui": "Apple"},
		{"mac": "cc:dd", "hostname": "printer", "ip": "192.168.1.9", "oui": "HP"},
		{"mac": "ee:ff", "name": "TV", "oui": "Samsung"},
	}
	// Substring name match.
	got := huntClients(clients, "iphone")
	if len(got) != 1 || got[0].MAC != "aa:bb" || got[0].Match != "name" {
		t.Fatalf("expected iphone name match, got %+v", got)
	}
	// Vendor match ranks, but exact IP match should outrank a vendor match.
	got = huntClients(clients, "192.168.1.9")
	if len(got) != 1 || got[0].MAC != "cc:dd" {
		t.Fatalf("expected ip match for printer, got %+v", got)
	}
	// No match returns empty, not nil-panic.
	if got := huntClients(clients, "nonexistent"); len(got) != 0 {
		t.Fatalf("expected no matches, got %d", len(got))
	}
}

func TestSummarizePoE(t *testing.T) {
	devices := []map[string]any{
		{
			"mac": "sw1", "name": "Switch", "model": "US-8-150W", "total_max_power": float64(150),
			"port_table": []any{
				map[string]any{"port_poe": true, "poe_power": "30.0"},
				map[string]any{"port_poe": true, "poe_power": float64(15)},
				map[string]any{"port_poe": false},
			},
		},
		{"mac": "ap1", "name": "AP"}, // no port_table → skipped
	}
	got := summarizePoE(devices)
	if len(got) != 1 {
		t.Fatalf("expected 1 switch report, got %d", len(got))
	}
	if got[0].DrawWatts != 45 {
		t.Fatalf("expected 45W draw, got %v", got[0].DrawWatts)
	}
	if got[0].PoEPorts != 2 {
		t.Fatalf("expected 2 PoE ports, got %d", got[0].PoEPorts)
	}
	if got[0].PercentUsed != 30 || got[0].Status != "ok" {
		t.Fatalf("expected 30%% ok, got %v %q", got[0].PercentUsed, got[0].Status)
	}
}

func TestAuditConfig(t *testing.T) {
	networks := []map[string]any{{"_id": "net1", "name": "LAN"}}
	usergroups := []map[string]any{{"_id": "ug1"}}
	fwGroups := []map[string]any{{"_id": "grp1"}}
	wlans := []map[string]any{
		{"_id": "w1", "name": "Good", "networkconf_id": "net1", "usergroup_id": "ug1"},
		{"_id": "w2", "name": "Dangling", "networkconf_id": "gone"},
		{"_id": "w3", "name": "NoPass", "security": "wpa2-psk", "enabled": true},
	}
	policies := []map[string]any{
		{"_id": "p1", "name": "BadRef", "source": map[string]any{"ip_group_id": "missing-grp"}},
	}
	findings := auditConfig(networks, wlans, policies, fwGroups, usergroups)
	kinds := map[string]int{}
	for _, f := range findings {
		kinds[f.Kind]++
	}
	if kinds["dangling-network-ref"] != 1 {
		t.Fatalf("expected 1 dangling network ref, got %d (%+v)", kinds["dangling-network-ref"], findings)
	}
	if kinds["psk-without-passphrase"] != 1 {
		t.Fatalf("expected 1 psk-without-passphrase, got %d", kinds["psk-without-passphrase"])
	}
	if kinds["dangling-group-ref"] != 1 {
		t.Fatalf("expected 1 dangling group ref, got %d", kinds["dangling-group-ref"])
	}
}

func TestFirewallCoverage(t *testing.T) {
	policies := []map[string]any{
		{"action": "ALLOW", "enabled": true, "source": map[string]any{"zone_id": "z1"}, "destination": map[string]any{"zone_id": "z2"}},
		{"action": "BLOCK", "enabled": true, "source": map[string]any{"zone_id": "z1"}, "destination": map[string]any{"zone_id": "z2"}},
		{"action": "ALLOW", "enabled": false, "source": map[string]any{"zone_id": "z2"}, "destination": map[string]any{"zone_id": "z1"}},
	}
	zoneNames := map[string]string{"z1": "LAN", "z2": "WAN"}
	cov := firewallCoverage(policies, zoneNames)
	if len(cov) != 2 {
		t.Fatalf("expected 2 zone pairs, got %d", len(cov))
	}
	// First (sorted): LAN→WAN with 2 policies (1 allow, 1 block).
	if cov[0].SourceZone != "LAN" || cov[0].DestZone != "WAN" || cov[0].Policies != 2 || cov[0].Allow != 1 || cov[0].Block != 1 {
		t.Fatalf("unexpected LAN→WAN coverage: %+v", cov[0])
	}
}

func TestDiffResources(t *testing.T) {
	prior := []map[string]any{
		{"_id": "a", "name": "A", "enabled": true},
		{"_id": "b", "name": "B", "enabled": true},
	}
	current := []map[string]any{
		{"_id": "a", "name": "A", "enabled": false}, // changed
		{"_id": "c", "name": "C"},                   // added
	}
	d := diffResources(prior, current)
	if len(d.Added) != 1 || d.Added[0] != "C (c)" {
		t.Fatalf("expected C added, got %+v", d.Added)
	}
	if len(d.Removed) != 1 || d.Removed[0] != "B (b)" {
		t.Fatalf("expected B removed, got %+v", d.Removed)
	}
	if len(d.Changed) != 1 || d.Changed[0] != "A (a)" {
		t.Fatalf("expected A changed, got %+v", d.Changed)
	}
}

func TestConfigHashIgnoresVolatile(t *testing.T) {
	a := map[string]any{"name": "X", "enabled": true, "uptime": float64(100)}
	b := map[string]any{"name": "X", "enabled": true, "uptime": float64(999)}
	if configHash(a) != configHash(b) {
		t.Fatalf("volatile field uptime should not affect hash")
	}
	c := map[string]any{"name": "X", "enabled": false, "uptime": float64(100)}
	if configHash(a) == configHash(c) {
		t.Fatalf("enabled change should affect hash")
	}
}
