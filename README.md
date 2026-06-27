# UniFi Network CLI

**Every UniFi Network controller operation as a single static Go binary, plus a local SQLite mirror and cross-resource queries no UniFi tool has.**

unifi-network-pp-cli drives the full UniFi Network controller API — devices, clients, WLANs/VLANs, firewall, QoS, routing, VPN, DNS, DPI, content filtering, events and stats — over one dependency-free binary. It mirrors your whole controller config into local SQLite so you can run `search`, `sql`, and transcendence commands like `topology`, `audit`, and `since` that the UI and per-call tools cannot.

## Install

The recommended path installs both the `unifi-network-pp-cli` binary and the `pp-unifi-network` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press-library install unifi-network
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press-library install unifi-network --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press-library install unifi-network --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press-library install unifi-network --agent claude-code
npx -y @mvanhorn/printing-press-library install unifi-network --agent claude-code --agent codex
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.3 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/devices/unifi-network/cmd/unifi-network-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/unifi-network-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-unifi-network --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-unifi-network --force
```

## Install for OpenClaw

Tell your OpenClaw agent (copy this):

```
Install the pp-unifi-network skill from https://github.com/mvanhorn/printing-press-library/tree/main/cli-skills/pp-unifi-network. The skill defines how its required CLI can be installed.
```

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/unifi-network-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.
3. Fill in `UNIFI_HOST`, `UNIFI_USERNAME` and `UNIFI_PASSWORD` (a local controller admin, not a UI.com SSO login) when Claude Desktop prompts you. `UNIFI_API_KEY` is optional — only the `dpi` commands and firewall-ordering use it.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


```bash
go install github.com/mvanhorn/printing-press-library/library/devices/unifi-network/cmd/unifi-network-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "unifi-network": {
      "command": "unifi-network-pp-mcp",
      "env": {
        "UNIFI_HOST": "<controller-ip>",
        "UNIFI_USERNAME": "<local-admin>",
        "UNIFI_PASSWORD": "<password>",
        "UNIFI_API_KEY": "<optional-integration-key-for-dpi/firewall-ordering>"
      }
    }
  }
}
```

</details>

## Authentication

Point the CLI at your controller with UNIFI_HOST (and UNIFI_PORT, default 443). Primary auth is your controller login: set UNIFI_USERNAME and UNIFI_PASSWORD (a local admin account is recommended over a UI.com SSO account). The CLI auto-detects UniFi OS vs legacy controllers, performs the login handshake, manages the session cookie and CSRF token, and tolerates self-signed certs (UNIFI_VERIFY_SSL=false by default). A few endpoints (DPI applications/categories, firewall policy ordering) use the Network Integration API instead — set UNIFI_API_KEY (create one in the UniFi UI under Control Plane > Integrations) for those. Pick the site with UNIFI_SITE (default 'default').

## Quick Start

```bash
# Health check: confirms host, auth env vars, and reachability wiring without sending a request.
unifi-network-pp-cli doctor --dry-run

# List adopted devices (APs, switches, gateways, PDUs) with status and firmware.
unifi-network-pp-cli devices list --json

# List connected clients (wired and wireless) with signal and SSID.
unifi-network-pp-cli clients list --json

# Mirror the whole controller config into local SQLite for offline search and cross-resource queries.
unifi-network-pp-cli sync --full

# Render the device uplink/LLDP hierarchy from the synced store.
unifi-network-pp-cli topology --json

```

## Unique Features

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

## Usage

Run `unifi-network-pp-cli --help` for the full command reference and flag list.

## Commands

### acl-rules

MAC ACL rules (Layer 2 access control)

- **`unifi-network-pp-cli acl-rules create`** - Create a MAC ACL rule. Provide the rule object via --body-json
- **`unifi-network-pp-cli acl-rules delete`** - Delete a MAC ACL rule by id
- **`unifi-network-pp-cli acl-rules list`** - List MAC ACL rules (Policy Engine) for L2 access control within a VLAN
- **`unifi-network-pp-cli acl-rules update`** - Update a MAC ACL rule by id

### ap-groups

AP groups (which APs broadcast which SSIDs)

- **`unifi-network-pp-cli ap-groups create`** - Create an AP group
- **`unifi-network-pp-cli ap-groups delete`** - Delete an AP group by id
- **`unifi-network-pp-cli ap-groups list`** - List AP groups with member APs and WLAN memberships
- **`unifi-network-pp-cli ap-groups update`** - Update an AP group by id

### client-groups

Client groups (network member groups for OON/firewall)

- **`unifi-network-pp-cli client-groups create`** - Create a client group (members = list of MAC addresses)
- **`unifi-network-pp-cli client-groups delete`** - Delete a client group by id
- **`unifi-network-pp-cli client-groups get`** - Get a client group by id
- **`unifi-network-pp-cli client-groups list`** - List client groups (MAC member groups used by OON policies and firewall rules)
- **`unifi-network-pp-cli client-groups update`** - Update a client group by id

### clients

Wired and wireless clients (connected and known)

- **`unifi-network-pp-cli clients known`** - List all known clients in controller history (including offline), with fixed-IP and note settings
- **`unifi-network-pp-cli clients list`** - List currently connected clients (wired and wireless) with mac, ip, status, signal and SSID

### content-filters

Content filtering profiles (DNS-based blocking, safe search)

- **`unifi-network-pp-cli content-filters delete`** - Delete a content filtering profile by id
- **`unifi-network-pp-cli content-filters list`** - List content filtering profiles (category blocking, safe search, allow/block lists). Note: profiles are created in the UI
- **`unifi-network-pp-cli content-filters update`** - Update a content filtering profile by id

### devices

Adopted UniFi devices (APs, switches, gateways, PDUs)

- **`unifi-network-pp-cli devices channels`** - List allowed RF channels for the site's regulatory domain (per band, with DFS and max power)
- **`unifi-network-pp-cli devices get`** - Get one device's full object by MAC (radio/port/outlet tables, system stats, WAN info)
- **`unifi-network-pp-cli devices list`** - List adopted devices with mac, name, model, ip, version, uptime, state and uplink

### dns-records

Static DNS records (V2)

- **`unifi-network-pp-cli dns-records create`** - Create a static DNS record
- **`unifi-network-pp-cli dns-records delete`** - Delete a static DNS record by id
- **`unifi-network-pp-cli dns-records list`** - List static DNS records (hostname, value, type, enabled, ttl)
- **`unifi-network-pp-cli dns-records update`** - Update a static DNS record by id

### dpi

DPI applications and categories (Integration API — requires UNIFI_API_KEY)

- **`unifi-network-pp-cli dpi applications`** - List DPI applications (Integration API; requires UNIFI_API_KEY)
- **`unifi-network-pp-cli dpi categories`** - List DPI application categories (Integration API; requires UNIFI_API_KEY)

### events

Event log and alarms

- **`unifi-network-pp-cli events alarms`** - List active alarms (security alerts, connectivity issues, firmware warnings)
- **`unifi-network-pp-cli events alarms-archived`** - List archived (historical) alarms

### firewall-groups

Firewall address and port groups (reusable objects)

- **`unifi-network-pp-cli firewall-groups create`** - Create a firewall group (group_type is immutable after creation)
- **`unifi-network-pp-cli firewall-groups delete`** - Delete a firewall group by id
- **`unifi-network-pp-cli firewall-groups get`** - Get a firewall group by id
- **`unifi-network-pp-cli firewall-groups list`** - List firewall groups (address-group, ipv6-address-group, port-group)
- **`unifi-network-pp-cli firewall-groups update`** - Update a firewall group by id (PUT replaces the full object)

### firewall-policies

Zone-based firewall policies (V2)

- **`unifi-network-pp-cli firewall-policies create`** - Create a V2 firewall policy. Provide the full policy object via --body-json
- **`unifi-network-pp-cli firewall-policies delete`** - Delete a firewall policy by id
- **`unifi-network-pp-cli firewall-policies list`** - List V2 zone-based firewall policies (zone targeting, action, enabled, index)
- **`unifi-network-pp-cli firewall-policies update`** - Update a firewall policy by id

### firewall-zones

Firewall zones (V2)

- **`unifi-network-pp-cli firewall-zones`** - List firewall zones and the zone matrix

### networks

Networks / VLANs (LAN, WAN, VLAN-only)

- **`unifi-network-pp-cli networks create`** - Create a network/VLAN. Pass the full network object as --body-json or individual fields
- **`unifi-network-pp-cli networks delete`** - Delete a network by id
- **`unifi-network-pp-cli networks get`** - Get a network's full configuration by id
- **`unifi-network-pp-cli networks list`** - List configured networks (LAN/WAN/VLAN) with name, purpose, subnet, VLAN id and DHCP settings
- **`unifi-network-pp-cli networks update`** - Update a network by id (PUT replaces the resource; fetch+merge with 'get' first)

### oon-policies

OON policies (internet scheduling, app blocking, QoS, routing)

- **`unifi-network-pp-cli oon-policies create`** - Create an OON policy. Provide the full nested policy object via --body-json
- **`unifi-network-pp-cli oon-policies delete`** - Delete an OON policy by id
- **`unifi-network-pp-cli oon-policies list`** - List OON policies (internet access schedules, app blocking, QoS, policy routing)
- **`unifi-network-pp-cli oon-policies update`** - Update an OON policy by id

### port-forwards

Port forwarding rules (V1)

- **`unifi-network-pp-cli port-forwards create`** - Create a port forwarding rule
- **`unifi-network-pp-cli port-forwards delete`** - Delete a port forwarding rule by id
- **`unifi-network-pp-cli port-forwards get`** - Get a port forwarding rule by id
- **`unifi-network-pp-cli port-forwards list`** - List port forwarding rules
- **`unifi-network-pp-cli port-forwards update`** - Update a port forwarding rule by id

### port-profiles

Switch port profiles

- **`unifi-network-pp-cli port-profiles create`** - Create a port profile (forward: native | all | customize | disabled)
- **`unifi-network-pp-cli port-profiles delete`** - Delete a port profile by id (system profiles cannot be deleted)
- **`unifi-network-pp-cli port-profiles get`** - Get a port profile by id
- **`unifi-network-pp-cli port-profiles list`** - List port profiles (VLAN, isolation, PoE, STP, 802.1X, storm control)
- **`unifi-network-pp-cli port-profiles update`** - Update a port profile by id (system profiles with attr_no_edit cannot be modified)

### qos-rules

QoS / traffic-shaping rules (V2)

- **`unifi-network-pp-cli qos-rules create`** - Create a QoS rule. Provide the rule object via --body-json
- **`unifi-network-pp-cli qos-rules delete`** - Delete a QoS rule by id
- **`unifi-network-pp-cli qos-rules list`** - List QoS rules for the current site
- **`unifi-network-pp-cli qos-rules update`** - Update a QoS rule by id

### routes

User-defined static routes (V1)

- **`unifi-network-pp-cli routes active`** - List the active routing table on the gateway (user-defined and system routes; may be empty on some firmware)
- **`unifi-network-pp-cli routes create`** - Create a static route (destination CIDR + next-hop IP)
- **`unifi-network-pp-cli routes delete`** - Delete a static route by id
- **`unifi-network-pp-cli routes get`** - Get a static route by id
- **`unifi-network-pp-cli routes list`** - List user-defined static routes (name, destination, next-hop, status)
- **`unifi-network-pp-cli routes update`** - Update a static route by id

### stats

Statistics, dashboard and DPI

- **`unifi-network-pp-cli stats dashboard`** - Pre-aggregated site dashboard (health, device/client counts, ISP status)
- **`unifi-network-pp-cli stats dpi`** - Deep Packet Inspection stats (applications and categories) for the site

### system

Controller system info, health and settings

- **`unifi-network-pp-cli system health`** - Per-subsystem health for WAN, LAN, WLAN and VPN with device and user counts
- **`unifi-network-pp-cli system get`** - Controller version, uptime, hostname, CPU/memory and update availability
- **`unifi-network-pp-cli system settings`** - Get a settings section (e.g. mgmt, snmp, super_mgmt, guest_access, country)
- **`unifi-network-pp-cli system sites`** - List all sites visible to the authenticated account (with their site keys)
- **`unifi-network-pp-cli system status`** - Controller status (server version, up flag)

### traffic-routes

Traffic routes (policy-based routing, V2)

- **`unifi-network-pp-cli traffic-routes create`** - Create a traffic route. Provide the route object via --body-json
- **`unifi-network-pp-cli traffic-routes delete`** - Delete a traffic route by id
- **`unifi-network-pp-cli traffic-routes list`** - List traffic routes (policy-based routing by domain/IP/region/device, often for VPN)
- **`unifi-network-pp-cli traffic-routes update`** - Update a traffic route by id

### usergroups

User groups (bandwidth profiles)

- **`unifi-network-pp-cli usergroups create`** - Create a user group (bandwidth profile). Use -1 for unlimited
- **`unifi-network-pp-cli usergroups delete`** - Delete a user group by id
- **`unifi-network-pp-cli usergroups get`** - Get a user group by id
- **`unifi-network-pp-cli usergroups list`** - List user groups (bandwidth profiles) with up/down speed caps in Kbps
- **`unifi-network-pp-cli usergroups update`** - Update a user group by id

### vouchers

Hotspot vouchers (guest network access)

- **`unifi-network-pp-cli vouchers`** - List hotspot vouchers (codes, expiry, usage quota, bandwidth limits)

### wlans

Wireless LANs / SSIDs

- **`unifi-network-pp-cli wlans create`** - Create a WLAN/SSID
- **`unifi-network-pp-cli wlans delete`** - Delete a WLAN/SSID by id
- **`unifi-network-pp-cli wlans get`** - Get a WLAN's full configuration by id
- **`unifi-network-pp-cli wlans list`** - List configured WLANs (SSIDs) with security, band, enabled state and network binding
- **`unifi-network-pp-cli wlans update`** - Update a WLAN by id (PUT replaces; fetch+merge with 'get' first)


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
unifi-network-pp-cli acl-rules list

# JSON for scripting and agents
unifi-network-pp-cli acl-rules list --json

# Filter to specific fields
unifi-network-pp-cli acl-rules list --json --select id,name,status

# Dry run — show the request without sending
unifi-network-pp-cli acl-rules list --dry-run

# Agent mode — JSON + compact + no prompts in one flag
unifi-network-pp-cli acl-rules list --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Explicit retries** - add `--idempotent` to create retries and `--ignore-missing` to delete retries when a no-op success is acceptable
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
unifi-network-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Config file: `~/.config/unifi-network-pp-cli/config.json`

Static request headers can be configured under `headers`; per-command header overrides take precedence.

Environment variables:

| Name | Kind | Required | Description |
| --- | --- | --- | --- |
| `UNIFI_HOST` | per_call | Yes | Controller IP/hostname (UDM, Cloud Key, or self-hosted Network application). |
| `UNIFI_USERNAME` | per_call | Yes | Local controller admin username (not a UI.com SSO login). |
| `UNIFI_PASSWORD` | per_call | Yes | Local controller admin password. |
| `UNIFI_SITE` | per_call | No | Controller site key (default `default`). |
| `UNIFI_PORT` | per_call | No | Controller HTTPS port (default `443`). |
| `UNIFI_VERIFY_SSL` | per_call | No | Verify TLS cert (default `false` — self-signed controllers). |
| `UNIFI_CONTROLLER_TYPE` | per_call | No | `auto` (default), `proxy` (UniFi OS), or `direct` (legacy). |
| `UNIFI_API_KEY` | per_call | No | Integration API key — `dpi` commands & firewall-ordering only; does **not** authenticate controller stat/rest/cmd endpoints. |

### agentcookie (optional)

If you use agentcookie to sync secrets across machines, this CLI auto-adopts agentcookie-managed credentials with no extra setup. When the daemon writes to this CLI's config, `unifi-network-pp-cli doctor` reports `agentcookie: detected` and `auth-status` labels the source as `agentcookie`. Skip this section if you don't use agentcookie - the CLI works the same as any other.

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `unifi-network-pp-cli doctor` to check credentials and the effective controller host
- Verify the controller login is set: `echo $UNIFI_HOST $UNIFI_USERNAME` (and that `UNIFI_PASSWORD` is exported)
- Use a **local** controller admin, not a UI.com cloud/SSO login
- The Integration API key (`UNIFI_API_KEY`) authenticates only the `dpi` and firewall-ordering commands — it cannot authenticate site-scoped `stat`/`rest`/`cmd` endpoints
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific
- **401 / login failed against a UniFi OS console** — Use a local controller admin account (UNIFI_USERNAME/UNIFI_PASSWORD), not a UI.com cloud SSO login; cloud SSO + MFA accounts cannot complete the local login handshake.
- **x509: certificate signed by unknown authority** — Self-signed controllers are expected; keep UNIFI_VERIFY_SSL=false (the default) or set it explicitly.
- **DPI or firewall-ordering commands return 'API key required'** — Create a Network Integration API key in the UniFi UI (Control Plane > Integrations) and export UNIFI_API_KEY.
- **Wrong controller type detected / 404 on every call** — Force it with UNIFI_CONTROLLER_TYPE=proxy (UniFi OS: UDM/UDR/Cloud Key Gen2+) or =direct (self-hosted Network application).
- **Commands target the wrong site** — Set UNIFI_SITE to the controller site key (not the display name); the default site key is 'default'.

## Acknowledgements

This CLI was modeled on the **UniFi Network MCP** — [MarcvsTvllivs/unifi-mcp](https://github.com/MarcvsTvllivs/unifi-mcp). Its resource coverage, command surface, and auth model (a username/password cookie-session controller login plus the optional Integration API key for DPI and firewall-ordering) were mapped from that project's Network application so this CLI delivers materially the same functionality as a single, dependency-free Go binary.
