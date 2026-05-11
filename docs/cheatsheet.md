# xctl / Xray cheatsheet

Copy-pasteable commands for the settings `xctl doctor` checks and other
common operations. Everything assumes a normal Xray install (binary at
`/usr/local/bin/xray`, config at `/usr/local/etc/xray/config.json`,
service `xray` running as user `xray`).

---

## Xray service

```bash
sudo systemctl status xray --no-pager
sudo systemctl start xray
sudo systemctl stop xray
sudo systemctl restart xray
sudo systemctl enable xray         # start on boot
sudo systemctl disable xray
sudo journalctl -u xray -n 100 --no-pager
sudo journalctl -u xray -f         # tail
```

## xctl quick reference

```bash
sudo xctl                          # TUI
sudo xctl status
sudo xctl doctor
sudo xctl list-clients
sudo xctl add-client --name phone
sudo xctl remove-client --name phone
sudo xctl rename-client --name phone --new-name tablet
sudo xctl reset-uuid --name tablet
sudo xctl show-client --name tablet
sudo xctl export
sudo xctl change-port --port 8443
sudo xctl change-disguise --domain www.apple.com
sudo xctl server-address --address vpn.example.com
sudo xctl test
sudo xctl restart
sudo xctl logs --lines 100
```

---

## BBR (TCP congestion control)

xctl shortcut:

```bash
sudo xctl bbr enable
sudo xctl bbr disable
sudo xctl bbr status
```

Manual equivalents below.

Check current state:

```bash
sysctl net.ipv4.tcp_congestion_control      # want: bbr
sysctl net.core.default_qdisc               # want: fq
lsmod | grep bbr                            # want: tcp_bbr loaded
```

Enable BBR (persistent across reboots):

```bash
sudo tee /etc/sysctl.d/99-bbr.conf >/dev/null <<'EOF'
net.core.default_qdisc=fq
net.ipv4.tcp_congestion_control=bbr
EOF
sudo sysctl --system
```

Verify:

```bash
sysctl net.ipv4.tcp_congestion_control net.core.default_qdisc
```

Disable BBR (revert to default `cubic`):

```bash
sudo rm /etc/sysctl.d/99-bbr.conf
sudo sysctl -w net.ipv4.tcp_congestion_control=cubic
sudo sysctl -w net.core.default_qdisc=pfifo_fast
```

---

## IP forwarding

For this `VLESS + REALITY + Vision` proxy mode you do **not** need IP
forwarding — `xctl doctor` only reports it for visibility.

xctl shortcut:

```bash
sudo xctl forward enable
sudo xctl forward disable
sudo xctl forward status
```

Manual:

Check:

```bash
sysctl net.ipv4.ip_forward                  # 0 = off, 1 = on
```

Enable (only needed if you also run a VPN/NAT on this host):

```bash
sudo tee /etc/sysctl.d/99-ip-forward.conf >/dev/null <<'EOF'
net.ipv4.ip_forward=1
EOF
sudo sysctl --system
```

Disable:

```bash
sudo rm -f /etc/sysctl.d/99-ip-forward.conf
sudo sysctl -w net.ipv4.ip_forward=0
```

---

## Firewall — open the Xray port

xctl shortcut (auto-detects ufw/firewalld/iptables, uses the current Xray port):

```bash
sudo xctl firewall open
sudo xctl firewall close
sudo xctl firewall status
sudo xctl firewall open --port 8443         # override
```

Manual equivalents below. Replace `443` with your actual port (`sudo xctl status` shows it).

### ufw (Ubuntu/Debian default)

```bash
sudo ufw status
sudo ufw allow 443/tcp
sudo ufw reload
sudo ufw delete allow 443/tcp        # close again
```

### firewalld (RHEL/Fedora/CentOS default)

```bash
sudo firewall-cmd --list-ports
sudo firewall-cmd --permanent --add-port=443/tcp
sudo firewall-cmd --reload
sudo firewall-cmd --permanent --remove-port=443/tcp
sudo firewall-cmd --reload
```

### iptables (raw)

```bash
sudo iptables -L INPUT -n --line-numbers
sudo iptables -I INPUT -p tcp --dport 443 -j ACCEPT
# persist on Debian/Ubuntu:
sudo apt install -y iptables-persistent && sudo netfilter-persistent save
```

### Cloud security groups

Most VPS providers also have an external firewall (AWS Security Group,
GCP firewall, Oracle Cloud security list, etc.). `xctl doctor` cannot
see those — check the provider's web console if the port is open
locally but still unreachable.

---

## System time

```bash
timedatectl
sudo timedatectl set-ntp true                # enable NTP sync
sudo systemctl restart systemd-timesyncd     # if drifted
```

---

## Disk space (logs)

```bash
df -h /var/log
sudo journalctl --vacuum-time=7d             # drop logs older than 7 days
sudo journalctl --vacuum-size=200M           # cap journal size
```

---

## Xray config / state

```bash
sudo cat /usr/local/etc/xray/config.json
sudo ls -l /usr/local/etc/xray/config.json   # should be xray:xray 0644
sudo /usr/local/bin/xray run -test -config /usr/local/etc/xray/config.json
sudo cat /root/.xray-reality/server.info
```

If you ever see `xray.service` fail with "failed to read config" after
manual file edits, check ownership: it must be readable by the `xray`
user (default `xray:xray 0644`).

---

## Reinstall / update Xray itself

xctl shortcut:

```bash
sudo xctl update
```

Manual:

```bash
bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install
bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ remove
```

## First-time setup

```bash
sudo xctl init                                        # defaults: SNI www.microsoft.com, port 443
sudo xctl init --sni www.apple.com --port 443 --name phone
sudo xctl init --force                                # overwrite existing config
```

## Fix config permissions

If you ever see xray fail with "failed to load config files" after a manual edit:

```bash
sudo xctl fix-perms
```

## Reinstall / update xctl

```bash
curl -fsSL https://raw.githubusercontent.com/papasaidfine/xray-fast-deploy/main/scripts/install.sh | sudo bash
```
