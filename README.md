# Xray REALITY VPN Deployment Script

A modular bash script for deploying and managing Xray with REALITY protocol on Ubuntu/Debian servers.

## Features

- **One-command installation** - Fully automated Xray REALITY setup
- **Auto-detection** - Detects existing installation and switches to management mode
- **Client management** - Add, remove, list, and export client configurations
- **QR code generation** - Instant QR codes for mobile client setup
- **VLESS links** - Shareable links for easy client import
- **Disguise switching** - Change camouflage domain on the fly
- **BBR enabled** - Automatic TCP BBR congestion control for better performance
- **Modular design** - Clean separation of concerns across library files

## Requirements

- Ubuntu 20.04+ or Debian 11+
- Root access (sudo)
- Open port 443 (or custom port) on firewall

## Quick Start

```bash
# Clone or download the scripts
git clone <repo-url> xray-deploy
cd xray-deploy

# Make executable
chmod +x deploy-xray.sh

# Run installation (interactive)
sudo ./deploy-xray.sh
```

The script will:
1. Install Xray and dependencies
2. Generate keys and UUID
3. Configure REALITY with your chosen disguise domain
4. Enable BBR acceleration
5. Display client config with QR code

## Usage

### Auto Mode (Recommended)

```bash
sudo ./deploy-xray.sh
```

- If Xray is **not installed**: Starts interactive installation
- If Xray is **installed**: Opens management menu

### Commands

| Command | Description |
|---------|-------------|
| `install` | Fresh installation |
| `manage` | Open interactive menu |
| `status` | Show server status |
| `add-client` | Add new client |
| `remove-client` | Remove a client |
| `list-clients` | List all clients |
| `show-client` | Show client config with QR |
| `export` | Export all client VLESS links |
| `change-disguise` | Change camouflage domain |
| `change-port` | Change server port |
| `regen-keys` | Regenerate server keys |
| `logs` | Show recent logs |
| `restart` | Restart Xray service |
| `test` | Test configuration |
| `uninstall` | Remove Xray completely |

### Installation Options

```bash
sudo ./deploy-xray.sh install [options]

Options:
  --port PORT         Server port (default: 443)
  --disguise DOMAIN   Camouflage domain (default: www.apple.com)
  --client NAME       First client name (default: default)
```

### Examples

```bash
# Install with Microsoft disguise
sudo ./deploy-xray.sh install --disguise www.microsoft.com

# Install with custom port and client name
sudo ./deploy-xray.sh install --port 8443 --client my-phone

# Add a new client
sudo ./deploy-xray.sh add-client --name laptop

# Change disguise to Google
sudo ./deploy-xray.sh change-disguise --domain www.google.com

# Export all client configs
sudo ./deploy-xray.sh export

# Show specific client config
sudo ./deploy-xray.sh show-client --name laptop
```

## Project Structure

```
xray/
├── deploy-xray.sh          # Main entry point
├── README.md               # This file
└── lib/
    ├── common.sh           # Shared utilities and constants
    ├── config.sh           # Configuration and key management
    ├── clients.sh          # Client CRUD operations
    ├── server.sh           # Server management functions
    ├── install.sh          # Installation routines
    └── menu.sh             # Interactive menu
```

### Module Details

| Module | Responsibility |
|--------|----------------|
| `common.sh` | Colors, logging, paths, OS checks, IP detection |
| `config.sh` | X25519 keys, UUID, config.json creation, key regeneration |
| `clients.sh` | Add/remove/list clients, VLESS links, QR codes |
| `server.sh` | Status, disguise/port changes, logs, service control |
| `install.sh` | Dependencies, Xray install, BBR, uninstall |
| `menu.sh` | Interactive management interface |

## Firewall Configuration

After installation, open the server port:

### Google Cloud Platform (GCP)

```bash
gcloud compute firewall-rules create allow-xray \
    --allow tcp:443 \
    --source-ranges 0.0.0.0/0 \
    --description "Allow Xray REALITY"
```

### UFW (Ubuntu)

```bash
sudo ufw allow 443/tcp
```

### iptables

```bash
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
```

## Client Setup

### Windows (v2rayN)

1. Download [v2rayN](https://github.com/2dust/v2rayN/releases) (With-Core-SelfContained version)
2. Copy the VLESS link from server output
3. In v2rayN: **Servers** → **Import bulk URLs from clipboard**
4. Right-click tray icon → **System Proxy** → **Set System Proxy**

### Android (v2rayNG)

1. Install [v2rayNG](https://github.com/2dust/v2rayNG) from Play Store or GitHub
2. Scan QR code displayed after installation
3. Tap the connection button

### iOS (Shadowrocket)

1. Install Shadowrocket from App Store
2. Scan QR code or import VLESS link
3. Toggle connection on

### macOS (V2rayU)

1. Download [V2rayU](https://github.com/yanue/V2rayU)
2. Import VLESS link via **Configure** → **Import from clipboard**

## Recommended Disguise Domains

These domains are known to work well with REALITY:

- `www.apple.com` (default)
- `www.microsoft.com`
- `www.google.com`
- `www.cloudflare.com`
- `www.amazon.com`
- `www.tesla.com`
- `www.nvidia.com`

Choose domains that:
- Support TLS 1.3
- Have consistent server behavior
- Are not blocked in your region

## Troubleshooting

### Check service status

```bash
sudo systemctl status xray
```

### View logs

```bash
# Recent logs
sudo journalctl -u xray -n 50

# Follow live
sudo journalctl -u xray -f
```

### Test configuration

```bash
sudo ./deploy-xray.sh test
# or
sudo xray run -test -config /usr/local/etc/xray/config.json
```

### Port not listening

```bash
# Check if port is in use
sudo ss -tlnp | grep 443

# Check for conflicts
sudo lsof -i :443
```

### Connection issues

1. Verify firewall rules allow the port
2. Check client config matches server (UUID, public key, SNI)
3. Try a different disguise domain
4. Ensure server IP is correct

### BBR not enabled

```bash
# Check current congestion control
sysctl net.ipv4.tcp_congestion_control

# Manually enable
sudo modprobe tcp_bbr
echo "net.core.default_qdisc=fq" | sudo tee -a /etc/sysctl.conf
echo "net.ipv4.tcp_congestion_control=bbr" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

## Security Notes

- **Keep keys secure** - Server private key and client configs grant full access
- **Rotate keys periodically** - Use `regen-keys` command (invalidates all clients)
- **Limit SSH access** - Restrict to your IP or use key-only authentication
- **Update regularly** - Keep system and Xray updated

```bash
# Update Xray
sudo bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install

# Update system
sudo apt update && sudo apt upgrade -y
```

## File Locations

| Path | Description |
|------|-------------|
| `/usr/local/bin/xray` | Xray binary |
| `/usr/local/etc/xray/config.json` | Server configuration |
| `/etc/systemd/system/xray.service` | Systemd service file |
| `~/.xray-reality/server.info` | Saved server metadata |

## Contributing

Contributions welcome! Please ensure:
- Scripts remain POSIX-compatible where possible
- New features go in appropriate module
- Test on fresh Ubuntu/Debian install

## License

MIT License

## References

- [Xray-core](https://github.com/XTLS/Xray-core)
- [REALITY Protocol](https://github.com/XTLS/REALITY)
- [v2rayN Client](https://github.com/2dust/v2rayN)
- [Xray Documentation](https://xtls.github.io)
