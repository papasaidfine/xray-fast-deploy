# Xray Fast Deploy TUI Plan

## Goal

Build a local Linux TUI for deploying and managing a bare-metal Xray
`VLESS + TCP + REALITY + Vision` node.

This project is not a web panel, subscription platform, or multi-protocol proxy
orchestrator. The TUI should make common VPS maintenance tasks fast while
keeping the runtime simple:

- no web server
- no database
- no Docker requirement
- no long-running management daemon
- Xray remains the only required service

## Target User

The target user manages one or a few Linux VPS instances and wants a reliable
local control panel over SSH.

Typical workflow:

1. Install Xray on a fresh Debian/Ubuntu VPS.
2. Add, remove, rename, or rotate clients.
3. Show a VLESS link or QR code for a client.
4. Change REALITY disguise settings.
5. Check whether the service, port, firewall, BBR, and logs look healthy.
6. Re-export client links after a VPS public IP change.

## Non-Goals

- Do not become Marzban, 3x-ui, or a general proxy management platform.
- Do not add a public web UI.
- Do not require Docker for the default workflow.
- Do not add a database for MVP.
- Do not manage payments, subscriptions, traffic quotas, or expiry dates.
- Do not support every protocol before the REALITY Vision path is reliable.

Optional protocol profiles such as Hysteria2 or TUIC can be considered later,
but the default product boundary is:

```text
bare-metal Xray + VLESS + TCP + REALITY + xtls-rprx-vision
```

## TUI Definition

A lightweight TUI means:

- starts only when the user runs it
- exits cleanly without leaving a management process behind
- edits local config files and restarts Xray through systemd
- keeps all important actions available as CLI commands
- works over SSH on a headless Linux server

Suitable implementations:

- current Bash menu, improved
- Bash plus `whiptail` or `dialog` fallback
- Go single binary with Bubble Tea/Bubbles
- Rust single binary with Ratatui, if the project moves to Rust

The recommended long-term direction is a Go single binary TUI, but the current
Bash CLI should first be made reliable and scriptable.

## Architecture

Keep a clear split between core operations and UI.

```text
CLI/core layer:
  install
  status
  add-client
  remove-client
  rename-client
  reset-uuid
  show-client
  qr
  export
  change-disguise
  change-port
  test
  logs
  doctor

TUI layer:
  renders menus, forms, status views, logs, and QR output
  calls the same core operations
```

The TUI should not be the only way to operate the server. Every important
operation should remain available as a direct CLI command for automation and
recovery.

## MVP Screens

### Dashboard

Show the current node health at a glance.

Fields:

- Xray service status
- Xray version
- listening port
- REALITY disguise domain/SNI
- public server address used in client links
- client count
- BBR status
- local firewall status
- config test status
- recent restart or error summary

Suggested actions:

- restart Xray
- test config
- open doctor checks
- open logs
- export all links

### Clients

Manage users stored in the Xray inbound client list.

Fields:

- client name/email
- UUID
- optional last-seen timestamp if access logs are enabled

Actions:

- add client
- remove client
- rename client
- reset client UUID
- show client config
- show QR code
- export VLESS link
- show logs for selected client

### Server Settings

Manage Xray settings that affect all clients.

Actions:

- change REALITY disguise domain/SNI
- change listen port
- regenerate REALITY key pair
- change log level
- re-detect public IP
- export all links after IP change
- test and restart Xray

Dangerous actions such as regenerating REALITY keys and uninstalling must show a
clear confirmation prompt because they invalidate existing client configs or
remove the installation.

### Doctor

Run local diagnostics and show actionable results.

Checks:

- Xray binary exists
- Xray service is active
- config file exists
- `xray run -test -config ...` passes
- configured port is listening
- local firewall allows the configured TCP port where detectable
- BBR is active
- public IPv4 can be detected
- saved server address matches the detected public IP
- disk has enough free space for logs
- system time appears sane

Important note:

Cloud provider security groups cannot always be verified from inside the VPS.
The TUI can check local listeners and local firewall rules, but external port
reachability requires a remote probe.

### Logs

Show systemd logs and optional Xray access logs.

Views:

- last 50 Xray service logs
- follow live service logs
- access logs by client name/email, if access logging is enabled
- recent errors and warnings

Access logs are useful for per-client troubleshooting, but they have privacy and
disk usage costs. They should be disabled by default or clearly opt-in.

## VPS IP Changes

If the VPS public IP changes, the Xray server config usually does not need to
change.

What changes is the client link address:

```text
vless://UUID@SERVER_ADDRESS:PORT?...#client
```

If clients use a raw IP address, re-export links and QR codes after the IP
change. If clients use a domain name, update DNS and keep the same client links.

The TUI should include:

- detect current public IP
- show saved server address
- update saved server address
- re-export all client links and QR codes

## BBR

BBR should be treated as a system tuning feature, separate from Xray config.

Checks:

```bash
sysctl net.ipv4.tcp_congestion_control
sysctl net.core.default_qdisc
lsmod | grep tcp_bbr
```

Preferred configuration:

```text
/etc/sysctl.d/90-xray-bbr.conf
/etc/modules-load.d/bbr.conf
```

Avoid repeatedly appending duplicate lines to `/etc/sysctl.conf`.

## IP Forwarding

Plain Xray proxy usage does not require Linux IP forwarding.

`net.ipv4.ip_forward` is only needed when the VPS is being used as a router,
TUN gateway, or NAT forwarding host. The doctor screen can display the value,
but it should not mark disabled IP forwarding as an error for the default Xray
REALITY Vision deployment.

## Reliability Work Before TUI Expansion

Before investing in a richer TUI, fix the core command behavior.

High-priority fixes:

1. Make the initial client name passed with `--client` actually persist in
   `config.json`.
2. Use the saved `SERVER_IP` or configured server address before falling back to
   online IP detection.
3. Use `jq --arg` instead of interpolating user input directly into jq filters.
4. Parse `xray x25519` output defensively in case output labels differ across
   Xray versions.
5. Move BBR settings from direct `/etc/sysctl.conf` appends to a dedicated
   `/etc/sysctl.d/90-xray-bbr.conf`.
6. Make install and repair operations idempotent.
7. Make config test mandatory before restart after config-changing operations.

## Suggested Implementation Phases

### Phase 1: Harden Current Bash CLI

- fix the reliability items above
- add a `doctor` command
- improve `status` output
- improve `export` and QR generation after IP changes
- make destructive operations require typed confirmation

### Phase 2: Improve Current Menu

- reorganize menu around Dashboard, Clients, Server, Doctor, Logs, System
- keep fallback text prompts
- optionally use `whiptail` or `dialog` if available
- show concise result screens after actions

### Phase 3: Go Single Binary TUI

Use Go plus Bubble Tea/Bubbles if a richer terminal UI is worth maintaining.

Design rules:

- no daemon
- no database
- no open network port
- no Docker requirement
- invoke or share the same core logic as CLI commands
- keep JSON config manipulation structured and testable

Possible package layout:

```text
cmd/xray-fast-deploy/
internal/xray/
internal/client/
internal/server/
internal/doctor/
internal/tui/
internal/systemd/
internal/firewall/
```

## Future Optional Profiles

After the default REALITY Vision path is stable, optional profiles can be
considered:

- Hysteria2 as a UDP/QUIC backup path
- TUIC as another UDP/QUIC backup path
- VLESS + TLS + WebSocket/gRPC + CDN for CDN-fronted deployments

These should be explicitly marked as optional or experimental. They should not
complicate the default install path.

## Positioning

Suggested project description:

```text
A lightweight local CLI/TUI for deploying and managing bare-metal Xray
VLESS REALITY Vision nodes on Linux VPS instances.
```

Short version:

```text
Local TUI for Xray REALITY Vision. No web panel. No database. No Docker required.
```
