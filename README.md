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

Installs xctl to `/usr/local/bin/xctl`:

```bash
curl -fsSL https://raw.githubusercontent.com/papasaidfine/xray-fast-deploy/main/scripts/install.sh | sudo bash
```

Build from source:

```bash
go build -o xctl ./cmd/xctl
sudo install -m 0755 xctl /usr/local/bin/xctl
```

## Sudo

`xctl` reads and writes root-owned Xray files (`/usr/local/etc/xray/config.json` and `/root/.xray-reality/server.info`) and calls `systemctl restart xray`, so all real use needs root:

```bash
sudo xctl
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

The TUI is interactive. Keybindings:

- `Tab` / `←` `→` / `h` `l` — switch tabs (vim-style hjkl supported)
- `↑` `↓` / `j` `k` — move cursor on the Clients tab; `g`/`G` jump to top/bottom
- Clients tab: `a` add, `d` delete, `R` rename, `u` reset UUID, `s` show VLESS link (with QR), `r` refresh
- Server tab: `p` change port, `D` change disguise domain, `A` change saved address, `t` test config, `X` restart xray, `r` refresh
- Tools tab: `b` toggle BBR, `f` toggle IP forwarding, `w` toggle firewall (current Xray port), `P` fix config perms, `r` refresh
- `Esc` cancels an input or confirm prompt; `q` or `Ctrl+C` quits

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
xctl show-client --name tablet --qr        # also render QR for phone import
xctl export
xctl export --qr                           # QR per client
xctl change-port --port 8443
xctl change-disguise --domain www.apple.com
xctl server-address --address vpn.example.com
xctl test
xctl restart
xctl logs --lines 50
xctl logs -f                                # follow (tail -f) the xray journal
```

## First-time setup

On a fresh VPS:

```bash
sudo xctl init                              # installs Xray if missing, generates config, restarts
# or with explicit options:
sudo xctl init --sni www.apple.com --port 443 --name phone
```

`init` prints the initial VLESS link. Defaults: SNI `www.microsoft.com`, port `443`, client name `default`. Refuses to overwrite an existing config unless `--force` is passed.

## System automation

xctl wraps the common system-level tweaks so you don't have to remember the commands:

```bash
sudo xctl bbr enable           # enable BBR + fq qdisc, persistent
sudo xctl bbr disable
sudo xctl bbr status

sudo xctl forward enable       # net.ipv4.ip_forward=1
sudo xctl forward disable
sudo xctl forward status

sudo xctl firewall open        # open current Xray port (auto-detects ufw/firewalld/iptables)
sudo xctl firewall close
sudo xctl firewall status
sudo xctl firewall open --port 8443   # override port

sudo xctl fix-perms            # restore <xray-user>:<xray-group> 0644 on the config

xctl version                   # print current xctl version + check for newer release
sudo xctl install              # download latest xctl release and replace /usr/local/bin/xctl
sudo xctl xray-update          # update Xray itself (runs XTLS official install-release.sh)
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

## Cheatsheet

See [docs/cheatsheet.md](docs/cheatsheet.md) for copy-pasteable commands covering everything `doctor` checks: enable/disable BBR, IP forwarding, open firewall ports (ufw/firewalld/iptables), inspect logs and disk space, restart Xray, etc.

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
