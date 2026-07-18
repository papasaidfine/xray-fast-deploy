# xctl

`xctl` is a local Go CLI/TUI for managing a bare-metal Xray `VLESS + TCP + REALITY + Vision` node over SSH.

![xctl TUI dashboard](docs/screenshots/dashboard.png)

## Install

Installs `xctl` to `/usr/local/bin/xctl`:

```bash
curl -fsSL https://raw.githubusercontent.com/papasaidfine/xray-fast-deploy/main/scripts/install.sh | sudo bash
```

If GitHub is blocked, route the download through a proxy:

```bash
curl -fsSL https://raw.githubusercontent.com/papasaidfine/xray-fast-deploy/main/scripts/install.sh \
  | sudo bash -s -- --proxy socks5://127.0.0.1:1080
```

If `raw.githubusercontent.com` is blocked too, the outer `curl` is your own command — prefix it with `-x socks5://…` or use a mirror.

Every command except `xctl version` needs `sudo` (it edits root-owned Xray files and restarts the service).

## First-time setup

```bash
sudo xctl init                                    # generate config, install Xray if missing, start
sudo xctl init --proxy socks5://127.0.0.1:1080    # same, but pull Xray through a proxy (GitHub blocked)
sudo xctl init --sni www.icloud.com --port 8443   # override defaults (www.apple.com:443)
```

`init` installs Xray if it's missing, generates the Reality keypair and config, starts the service, and prints your first VLESS link.

**On a host where GitHub is blocked, `--proxy` is required** — without it `init` can't download Xray. It has no effect once Xray is already installed. The same flag works on `xctl xray-update`.

## Everything else

Run `sudo xctl` for the TUI, or `xctl --help` for the full command list — clients, ports/SNI, export + QR, firewall, BBR, logs, `doctor`, and `xray-update [--proxy …]`.

- **[`docs/cheatsheet.md`](docs/cheatsheet.md)** — copy-pasteable manual equivalents (raw `sysctl` / `ufw` / `firewalld` / `iptables` / `journalctl`) plus a few extras.
- Port open locally but still unreachable? Check your cloud security group (AWS / GCP / Oracle …) — it's invisible from inside the VPS.

## Safety

Every config change writes a candidate, runs `xray -test`, then atomically swaps it in and restarts. If the test fails, the live config is left untouched and the service is not restarted.
