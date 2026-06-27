---
name: pp-unifi-network
description: "Every UniFi Network controller operation as a single static Go binary Trigger phrases: `list my unifi devices`, `block this client on my network`, `what changed on my unifi network`, `show my wifi networks`, `check my firewall rules`, `use unifi-network`, `run unifi-network`."
author: "marcvstvllivs"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - unifi-network-pp-cli
    install:
      - kind: go
        bins: [unifi-network-pp-cli]
        module: github.com/mvanhorn/printing-press-library/library/devices/unifi-network/cmd/unifi-network-pp-cli
---

# UniFi Network — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `unifi-network-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   npx -y @mvanhorn/printing-press-library install unifi-network --cli-only
   ```
2. Verify: `unifi-network-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails before this CLI has a public-library category, install Node or use the category-specific Go fallback after publish.

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

unifi-network-pp-cli drives the full UniFi Network controller API — devices, clients, WLANs/VLANs, firewall, QoS, routing, VPN, DNS, DPI, content filtering, events and stats — over one dependency-free binary. It mirrors your whole controller config into local SQLite so you can run `search`, `sql`, and transcendence commands like `topology`, `audit`, and `since` that the UI and per-call tools cannot.

## When to Use This CLI

Use this CLI to operate and audit a UniFi Network controller from the shell or an agent: inventory devices and clients, manage WLANs/VLANs/firewall/QoS/routing/VPN/DNS, control devices (reboot/adopt/locate/upgrade/PoE), and run cross-resource queries over a local mirror. It is ideal when you want scriptable, offline-capable, agent-native access to the controller that the web UI and per-call MCP tools don't provide.

## Anti-triggers

Do not use this CLI for:
- Do not use this CLI for UniFi Protect cameras, UniFi Access doors, or UniFi Talk — it targets the Network application only.
- Do not use it to reach UniFi devices over the Ubiquiti cloud (ui.com remote access); it talks directly to a controller on your network.
- Do not use it as a real-time event stream/websocket monitor; it polls the event log rather than subscribing to live pushes.

## Unique Capabilities

These capabilities aren't available in any other tool for this API.

### Local joins that compound
- **`topology`** — Render the AP/switch/gateway uplink hierarchy of your whole site as a tree from the local store.

  _Reach for this to understand physical/logical network layout in one shot instead of cross-referencing every device's uplink field by hand._

  ```bash
  unifi-network-pp-cli topology --json
  ```
- **`audit`** — Find dangling references and risky settings across firewall policies, networks, WLANs, groups, and port profiles.

  _Reach for this after config changes to catch broken references the UniFi UI never surfaces._

  ```bash
  unifi-network-pp-cli audit --json
  ```
- **`since`** — Diff current controller state against the last synced snapshot to show added/removed/changed resources.

  _Reach for this to answer 'what changed on my network' without scrubbing the raw event log._

  ```bash
  unifi-network-pp-cli since --json
  ```
- **`poe-budget`** — Sum per-port PoE draw against each switch's power capacity and flag switches near their budget.

  _Reach for this before adding powered devices to avoid tripping a switch's PoE budget._

  ```bash
  unifi-network-pp-cli poe-budget --json
  ```
- **`firewall-coverage`** — Show which firewall zone pairs have policies and which are wide open, joining zones, policies, and groups locally.

  _Reach for this to review firewall posture per zone pair instead of reading policies one by one._

  ```bash
  unifi-network-pp-cli firewall-coverage --json
  ```

### Agent-native operations
- **`client-hunt`** — Locate any client (online or offline) by name, IP, MAC, or vendor with device/network context in one ranked answer.

  _Reach for this when you need to find a device fast and the UI only shows currently-connected clients._

  ```bash
  unifi-network-pp-cli client-hunt 'iphone' --json
  ```

## Command Reference

**acl-rules** — MAC ACL rules (Layer 2 access control)

- `unifi-network-pp-cli acl-rules create` — Create a MAC ACL rule. Provide the rule object via --body-json
- `unifi-network-pp-cli acl-rules delete` — Delete a MAC ACL rule by id
- `unifi-network-pp-cli acl-rules list` — List MAC ACL rules (Policy Engine) for L2 access control within a VLAN
- `unifi-network-pp-cli acl-rules update` — Update a MAC ACL rule by id

**ap-groups** — AP groups (which APs broadcast which SSIDs)

- `unifi-network-pp-cli ap-groups create` — Create an AP group
- `unifi-network-pp-cli ap-groups delete` — Delete an AP group by id
- `unifi-network-pp-cli ap-groups list` — List AP groups with member APs and WLAN memberships
- `unifi-network-pp-cli ap-groups update` — Update an AP group by id

**client-groups** — Client groups (network member groups for OON/firewall)

- `unifi-network-pp-cli client-groups create` — Create a client group (members = list of MAC addresses)
- `unifi-network-pp-cli client-groups delete` — Delete a client group by id
- `unifi-network-pp-cli client-groups get` — Get a client group by id
- `unifi-network-pp-cli client-groups list` — List client groups (MAC member groups used by OON policies and firewall rules)
- `unifi-network-pp-cli client-groups update` — Update a client group by id

**clients** — Wired and wireless clients (connected and known)

- `unifi-network-pp-cli clients known` — List all known clients in controller history (including offline), with fixed-IP and note settings
- `unifi-network-pp-cli clients list` — List currently connected clients (wired and wireless) with mac, ip, status, signal and SSID

**content-filters** — Content filtering profiles (DNS-based blocking, safe search)

- `unifi-network-pp-cli content-filters delete` — Delete a content filtering profile by id
- `unifi-network-pp-cli content-filters list` — List content filtering profiles (category blocking, safe search, allow/block lists).
- `unifi-network-pp-cli content-filters update` — Update a content filtering profile by id

**devices** — Adopted UniFi devices (APs, switches, gateways, PDUs)

- `unifi-network-pp-cli devices channels` — List allowed RF channels for the site's regulatory domain (per band, with DFS and max power)
- `unifi-network-pp-cli devices get` — Get one device's full object by MAC (radio/port/outlet tables, system stats, WAN info)
- `unifi-network-pp-cli devices list` — List adopted devices with mac, name, model, ip, version, uptime, state and uplink

**dns-records** — Static DNS records (V2)

- `unifi-network-pp-cli dns-records create` — Create a static DNS record
- `unifi-network-pp-cli dns-records delete` — Delete a static DNS record by id
- `unifi-network-pp-cli dns-records list` — List static DNS records (hostname, value, type, enabled, ttl)
- `unifi-network-pp-cli dns-records update` — Update a static DNS record by id

**dpi** — DPI applications and categories (Integration API — requires UNIFI_API_KEY)

- `unifi-network-pp-cli dpi applications` — List DPI applications (Integration API; requires UNIFI_API_KEY)
- `unifi-network-pp-cli dpi categories` — List DPI application categories (Integration API; requires UNIFI_API_KEY)

**events** — Event log and alarms

- `unifi-network-pp-cli events alarms` — List active alarms (security alerts, connectivity issues, firmware warnings)
- `unifi-network-pp-cli events alarms-archived` — List archived (historical) alarms

**firewall-groups** — Firewall address and port groups (reusable objects)

- `unifi-network-pp-cli firewall-groups create` — Create a firewall group (group_type is immutable after creation)
- `unifi-network-pp-cli firewall-groups delete` — Delete a firewall group by id
- `unifi-network-pp-cli firewall-groups get` — Get a firewall group by id
- `unifi-network-pp-cli firewall-groups list` — List firewall groups (address-group, ipv6-address-group, port-group)
- `unifi-network-pp-cli firewall-groups update` — Update a firewall group by id (PUT replaces the full object)

**firewall-policies** — Zone-based firewall policies (V2)

- `unifi-network-pp-cli firewall-policies create` — Create a V2 firewall policy. Provide the full policy object via --body-json
- `unifi-network-pp-cli firewall-policies delete` — Delete a firewall policy by id
- `unifi-network-pp-cli firewall-policies list` — List V2 zone-based firewall policies (zone targeting, action, enabled, index)
- `unifi-network-pp-cli firewall-policies update` — Update a firewall policy by id

**firewall-zones** — Firewall zones (V2)

- `unifi-network-pp-cli firewall-zones` — List firewall zones and the zone matrix

**networks** — Networks / VLANs (LAN, WAN, VLAN-only)

- `unifi-network-pp-cli networks create` — Create a network/VLAN. Pass the full network object as --body-json or individual fields
- `unifi-network-pp-cli networks delete` — Delete a network by id
- `unifi-network-pp-cli networks get` — Get a network's full configuration by id
- `unifi-network-pp-cli networks list` — List configured networks (LAN/WAN/VLAN) with name, purpose, subnet, VLAN id and DHCP settings
- `unifi-network-pp-cli networks update` — Update a network by id (PUT replaces the resource; fetch+merge with 'get' first)

**oon-policies** — OON policies (internet scheduling, app blocking, QoS, routing)

- `unifi-network-pp-cli oon-policies create` — Create an OON policy. Provide the full nested policy object via --body-json
- `unifi-network-pp-cli oon-policies delete` — Delete an OON policy by id
- `unifi-network-pp-cli oon-policies list` — List OON policies (internet access schedules, app blocking, QoS, policy routing)
- `unifi-network-pp-cli oon-policies update` — Update an OON policy by id

**port-forwards** — Port forwarding rules (V1)

- `unifi-network-pp-cli port-forwards create` — Create a port forwarding rule
- `unifi-network-pp-cli port-forwards delete` — Delete a port forwarding rule by id
- `unifi-network-pp-cli port-forwards get` — Get a port forwarding rule by id
- `unifi-network-pp-cli port-forwards list` — List port forwarding rules
- `unifi-network-pp-cli port-forwards update` — Update a port forwarding rule by id

**port-profiles** — Switch port profiles

- `unifi-network-pp-cli port-profiles create` — Create a port profile (forward: native | all | customize | disabled)
- `unifi-network-pp-cli port-profiles delete` — Delete a port profile by id (system profiles cannot be deleted)
- `unifi-network-pp-cli port-profiles get` — Get a port profile by id
- `unifi-network-pp-cli port-profiles list` — List port profiles (VLAN, isolation, PoE, STP, 802.1X, storm control)
- `unifi-network-pp-cli port-profiles update` — Update a port profile by id (system profiles with attr_no_edit cannot be modified)

**qos-rules** — QoS / traffic-shaping rules (V2)

- `unifi-network-pp-cli qos-rules create` — Create a QoS rule. Provide the rule object via --body-json
- `unifi-network-pp-cli qos-rules delete` — Delete a QoS rule by id
- `unifi-network-pp-cli qos-rules list` — List QoS rules for the current site
- `unifi-network-pp-cli qos-rules update` — Update a QoS rule by id

**routes** — User-defined static routes (V1)

- `unifi-network-pp-cli routes active` — List the active routing table on the gateway (user-defined and system routes; may be empty on some firmware)
- `unifi-network-pp-cli routes create` — Create a static route (destination CIDR + next-hop IP)
- `unifi-network-pp-cli routes delete` — Delete a static route by id
- `unifi-network-pp-cli routes get` — Get a static route by id
- `unifi-network-pp-cli routes list` — List user-defined static routes (name, destination, next-hop, status)
- `unifi-network-pp-cli routes update` — Update a static route by id

**stats** — Statistics, dashboard and DPI

- `unifi-network-pp-cli stats dashboard` — Pre-aggregated site dashboard (health, device/client counts, ISP status)
- `unifi-network-pp-cli stats dpi` — Deep Packet Inspection stats (applications and categories) for the site

**system** — Controller system info, health and settings

- `unifi-network-pp-cli system health` — Per-subsystem health for WAN, LAN, WLAN and VPN with device and user counts
- `unifi-network-pp-cli system get` — Controller version, uptime, hostname, CPU/memory and update availability
- `unifi-network-pp-cli system settings` — Get a settings section (e.g. mgmt, snmp, super_mgmt, guest_access, country)
- `unifi-network-pp-cli system sites` — List all sites visible to the authenticated account (with their site keys)
- `unifi-network-pp-cli system status` — Controller status (server version, up flag)

**traffic-routes** — Traffic routes (policy-based routing, V2)

- `unifi-network-pp-cli traffic-routes create` — Create a traffic route. Provide the route object via --body-json
- `unifi-network-pp-cli traffic-routes delete` — Delete a traffic route by id
- `unifi-network-pp-cli traffic-routes list` — List traffic routes (policy-based routing by domain/IP/region/device, often for VPN)
- `unifi-network-pp-cli traffic-routes update` — Update a traffic route by id

**usergroups** — User groups (bandwidth profiles)

- `unifi-network-pp-cli usergroups create` — Create a user group (bandwidth profile). Use -1 for unlimited
- `unifi-network-pp-cli usergroups delete` — Delete a user group by id
- `unifi-network-pp-cli usergroups get` — Get a user group by id
- `unifi-network-pp-cli usergroups list` — List user groups (bandwidth profiles) with up/down speed caps in Kbps
- `unifi-network-pp-cli usergroups update` — Update a user group by id

**vouchers** — Hotspot vouchers (guest network access)

- `unifi-network-pp-cli vouchers` — List hotspot vouchers (codes, expiry, usage quota, bandwidth limits)

**wlans** — Wireless LANs / SSIDs

- `unifi-network-pp-cli wlans create` — Create a WLAN/SSID
- `unifi-network-pp-cli wlans delete` — Delete a WLAN/SSID by id
- `unifi-network-pp-cli wlans get` — Get a WLAN's full configuration by id
- `unifi-network-pp-cli wlans list` — List configured WLANs (SSIDs) with security, band, enabled state and network binding
- `unifi-network-pp-cli wlans update` — Update a WLAN by id (PUT replaces; fetch+merge with 'get' first)


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
unifi-network-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes

### Find and block a noisy client

```bash
unifi-network-pp-cli client-hunt 'amazon' --json && unifi-network-pp-cli clients block <mac>
```

Locate a client by vendor/name from the local store, then block it by MAC.

### Audit firewall posture

```bash
unifi-network-pp-cli sync --full && unifi-network-pp-cli firewall-coverage --json
```

Mirror config locally, then map which firewall zone pairs are covered vs wide open.

### Narrow a verbose device list

```bash
unifi-network-pp-cli devices list --json --select 'mac,name,model,state,version'
```

Device objects are tens of KB each; --select with dotted paths returns only the fields you need.

### Pre-flight a config change

```bash
unifi-network-pp-cli wlans toggle <id> --enabled false --dry-run
```

Preview the exact request and body before mutating; drop --dry-run to apply.

### Check PoE headroom before adding a camera

```bash
unifi-network-pp-cli poe-budget --json
```

Flags switches near their PoE power budget from synced per-port draw.

## Auth Setup

Point the CLI at your controller with UNIFI_HOST (and UNIFI_PORT, default 443). Primary auth is your controller login: set UNIFI_USERNAME and UNIFI_PASSWORD (a local admin account is recommended over a UI.com SSO account). The CLI auto-detects UniFi OS vs legacy controllers, performs the login handshake, manages the session cookie and CSRF token, and tolerates self-signed certs (UNIFI_VERIFY_SSL=false by default). A few endpoints (DPI applications/categories, firewall policy ordering) use the Network Integration API instead — set UNIFI_API_KEY (create one in the UniFi UI under Control Plane > Integrations) for those. Pick the site with UNIFI_SITE (default 'default').

Run `unifi-network-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  unifi-network-pp-cli acl-rules list --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Explicit retries** — use `--idempotent` only when an already-existing create should count as success, and `--ignore-missing` only when a missing delete target should count as success

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal AND no machine-format flag (`--json`, `--csv`, `--compact`, `--quiet`, `--plain`, `--select`) is set — piped/agent consumers and explicit-format runs get pure JSON on stdout.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
unifi-network-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
unifi-network-pp-cli feedback --stdin < notes.txt
unifi-network-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.local/share/unifi-network-pp-cli/feedback.jsonl`. They are never POSTed unless `UNIFI_NETWORK_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `UNIFI_NETWORK_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
unifi-network-pp-cli profile save briefing --json
unifi-network-pp-cli --profile briefing acl-rules list
unifi-network-pp-cli profile list --json
unifi-network-pp-cli profile show briefing
unifi-network-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 4 | Authentication required |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `unifi-network-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/devices/unifi-network/cmd/unifi-network-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add unifi-network-pp-mcp -- unifi-network-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which unifi-network-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   unifi-network-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `unifi-network-pp-cli <command> --help`.
