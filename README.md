# xctl

`xctl` is a local Go CLI/TUI for managing a bare-metal Xray `VLESS + TCP + REALITY + Vision` node over SSH.

## Current Scope

`xctl` manages an existing Xray installation. It does not install or auto-start Xray yet.

Expected system state:

- Xray binary exists, usually `/usr/local/bin/xray`
- systemd service is named `xray`
- server config exists at `/usr/local/etc/xray/config.json`
- server metadata exists at `/root/.xray-reality/server.info`

## Install

Download a Linux release asset and install it:

```bash
sudo install -m 0755 xctl-linux-amd64 /usr/local/bin/xctl
```

Build from source:

```bash
go build -o xctl ./cmd/xctl
```

## Sudo

Use `sudo` on a VPS. The default config paths are root-owned and service changes go through systemd.

Recommended:

```bash
sudo xctl tui
sudo xctl doctor
sudo xctl add-client --name phone
sudo xctl export
```

Write/service commands:

- `add-client`
- `remove-client`
- `rename-client`
- `reset-uuid`
- `change-port`
- `change-disguise`
- `server-address`
- `test`
- `restart`

## Usage

Open the TUI:

```bash
sudo xctl
# or
sudo xctl tui
```

CLI:

```bash
xctl status
xctl doctor
xctl list-clients
xctl add-client --name phone
xctl remove-client --name phone
xctl rename-client --name phone --new-name tablet
xctl reset-uuid --name tablet
xctl show-client --name tablet
xctl export
xctl change-port --port 8443
xctl change-disguise --domain www.apple.com
xctl server-address --address vpn.example.com
xctl test
xctl restart
xctl logs --lines 50
```

## Doctor

`xctl doctor` runs local diagnostics and prints `ok`, `warn`, or `fail` results with short repair advice.

Checks:

- Xray binary exists
- Xray systemd service is active
- config file exists
- `xray run -test -config ...` passes
- configured port is listening
- local firewall state where detectable: `ufw`, `firewalld`, `iptables`
- BBR congestion control, default qdisc, and `tcp_bbr` module
- saved server address compared with detected public IPv4
- recent Xray service errors from `journalctl`
- log disk space
- system time sanity
- Linux IP forwarding state

Disabled IP forwarding is normal for this proxy mode. Cloud security groups cannot be verified from inside the VPS.

## Safety Model

Config-changing operations follow the same pipeline:

1. Read `/usr/local/etc/xray/config.json`
2. Write a temporary candidate config
3. Run `xray run -test -config <candidate>`
4. Atomically replace the active config
5. Restart Xray through systemd

If validation fails, the active config is left untouched and Xray is not restarted.

## VPS IP Changes

After a VPS IP change, update the saved address and re-export client links.

```bash
sudo xctl server-address --address vpn.example.com
sudo xctl export
```

## Server Metadata

`xctl` reads `/root/.xray-reality/server.info` for values used in client links:

```text
PUBLIC_KEY="..."
PORT="443"
SNI="www.apple.com"
SERVER_IP="vpn.example.com"
CREATED="2026-05-11 00:00:00"
```

Use `server-address` to update the address embedded in exported VLESS links:

```bash
sudo xctl server-address --address vpn.example.com
```

## Project Layout

```text
cmd/xctl/               binary entrypoint
internal/app/           CLI dispatcher and app orchestration
internal/xray/          structured Xray config operations
internal/link/          VLESS link generation
internal/serverinfo/    saved server metadata
internal/system/        system command runner and safe config update pipeline
internal/doctor/        diagnostic result model
internal/tui/           Bubble Tea TUI model
```

## Development

```bash
go test ./...
go build ./cmd/xctl
```

Tagged releases publish Linux `amd64` and `arm64` binaries through GitHub Actions.
