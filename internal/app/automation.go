package app

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lonelyrower/xray-fast-deploy/internal/link"
	"github.com/lonelyrower/xray-fast-deploy/internal/serverinfo"
	"github.com/lonelyrower/xray-fast-deploy/internal/xray"
)

const (
	xrayInstallURL  = "https://github.com/XTLS/Xray-install/raw/main/install-release.sh"
	bbrSysctlFile   = "/etc/sysctl.d/99-xctl-bbr.conf"
	fwdSysctlFile   = "/etc/sysctl.d/99-xctl-ip-forward.conf"
	defaultSNI      = "www.microsoft.com"
	defaultPort     = 443
	defaultName     = "default"
	xrayServiceName = "xray"
)

// --------- init ---------

func (a *App) initConfig(args []string) error {
	if err := requireRoot("init"); err != nil {
		return err
	}
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	sni := fs.String("sni", defaultSNI, "TLS SNI / disguise domain")
	port := fs.Int("port", defaultPort, "listen port")
	name := fs.String("name", defaultName, "initial client name")
	force := fs.Bool("force", false, "overwrite existing config")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if _, err := os.Stat(a.configPath); err == nil && !*force {
		return fmt.Errorf("%s already exists; pass --force to overwrite", a.configPath)
	}

	if !commandExists("xray") && !fileExists("/usr/local/bin/xray") {
		fmt.Fprintln(a.out, "Xray not found, installing via XTLS official script...")
		if err := installXray(); err != nil {
			return err
		}
	}

	pub, priv, err := generateRealityKeypair()
	if err != nil {
		return fmt.Errorf("generate reality keypair: %w", err)
	}
	uuid := newUUID()
	shortID := newShortID()

	cfg := xray.NewRealityConfig(xray.ConfigOptions{
		UUID:       uuid,
		PrivateKey: priv,
		Dest:       *sni + ":443",
		SNI:        *sni,
		Port:       *port,
		ShortID:    shortID,
		ClientName: *name,
	})
	if err := cfg.Save(a.configPath); err != nil {
		return err
	}
	if err := applyServiceOwnership(a.configPath); err != nil {
		return fmt.Errorf("set config ownership: %w", err)
	}

	addr, _ := detectPublicIPv4()
	info := serverinfo.Info{
		PublicKey: pub,
		Port:      *port,
		SNI:       *sni,
		Address:   addr,
	}
	if err := serverinfo.Save(a.infoPath, info); err != nil {
		return err
	}

	if err := exec.Command("systemctl", "enable", xrayServiceName).Run(); err != nil {
		fmt.Fprintf(a.out, "warning: systemctl enable %s failed: %v\n", xrayServiceName, err)
	}
	if err := a.runner.RestartService(); err != nil {
		return err
	}

	fmt.Fprintln(a.out, "Xray initialized.")
	fmt.Fprintf(a.out, "SNI: %s\nPort: %d\nClient: %s\n", *sni, *port, *name)
	resolved := serverinfo.ResolveAddress(info, detectPublicIPv4)
	fmt.Fprintln(a.out, link.GenerateVLESS(link.Link{
		UUID:      uuid,
		Address:   resolved,
		Port:      *port,
		PublicKey: pub,
		SNI:       *sni,
		Name:      *name,
		ShortID:   shortID,
	}))
	return nil
}

func installXray() error {
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf(`bash -c "$(curl -L %s)" @ install`, xrayInstallURL))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func generateRealityKeypair() (pub, priv string, err error) {
	path, err := exec.LookPath("xray")
	if err != nil {
		path = "/usr/local/bin/xray"
	}
	out, err := exec.Command(path, "x25519").CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("xray x25519: %w: %s", err, string(out))
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if v, ok := cutPrefix(line, "Private key:"); ok {
			priv = strings.TrimSpace(v)
		}
		if v, ok := cutPrefix(line, "PrivateKey:"); ok {
			priv = strings.TrimSpace(v)
		}
		if v, ok := cutPrefix(line, "Public key:"); ok {
			pub = strings.TrimSpace(v)
		}
		if v, ok := cutPrefix(line, "Password:"); ok {
			pub = strings.TrimSpace(v)
		}
	}
	if pub == "" || priv == "" {
		return "", "", fmt.Errorf("could not parse xray x25519 output:\n%s", string(out))
	}
	return pub, priv, nil
}

func cutPrefix(s, prefix string) (string, bool) {
	if strings.HasPrefix(s, prefix) {
		return s[len(prefix):], true
	}
	return "", false
}

func newShortID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%08x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// --------- bbr ---------

func (a *App) bbrCmd(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: bbr <enable|disable|status>")
	}
	switch args[0] {
	case "enable":
		if err := requireRoot("bbr enable"); err != nil {
			return err
		}
		return a.bbrEnable()
	case "disable":
		if err := requireRoot("bbr disable"); err != nil {
			return err
		}
		return a.bbrDisable()
	case "status":
		return a.bbrStatus()
	}
	return fmt.Errorf("unknown bbr subcommand %q", args[0])
}

func (a *App) bbrEnable() error {
	_ = exec.Command("modprobe", "tcp_bbr").Run()
	content := "net.core.default_qdisc=fq\nnet.ipv4.tcp_congestion_control=bbr\n"
	if err := os.WriteFile(bbrSysctlFile, []byte(content), 0o644); err != nil {
		return err
	}
	if out, err := exec.Command("sysctl", "--system").CombinedOutput(); err != nil {
		return fmt.Errorf("sysctl --system: %w: %s", err, out)
	}
	return a.bbrStatus()
}

func (a *App) bbrDisable() error {
	_ = os.Remove(bbrSysctlFile)
	_ = exec.Command("sysctl", "-w", "net.ipv4.tcp_congestion_control=cubic").Run()
	_ = exec.Command("sysctl", "-w", "net.core.default_qdisc=pfifo_fast").Run()
	return a.bbrStatus()
}

func (a *App) bbrStatus() error {
	cc := sysctlGet("net.ipv4.tcp_congestion_control")
	qdisc := sysctlGet("net.core.default_qdisc")
	fmt.Fprintf(a.out, "congestion_control: %s\ndefault_qdisc: %s\ntcp_bbr_module: %v\n", cc, qdisc, moduleLoaded("tcp_bbr"))
	return nil
}

// --------- ip forwarding ---------

func (a *App) forwardCmd(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: forward <enable|disable|status>")
	}
	switch args[0] {
	case "enable":
		if err := requireRoot("forward enable"); err != nil {
			return err
		}
		return a.forwardEnable()
	case "disable":
		if err := requireRoot("forward disable"); err != nil {
			return err
		}
		return a.forwardDisable()
	case "status":
		return a.forwardStatus()
	}
	return fmt.Errorf("unknown forward subcommand %q", args[0])
}

func (a *App) forwardEnable() error {
	if err := os.WriteFile(fwdSysctlFile, []byte("net.ipv4.ip_forward=1\n"), 0o644); err != nil {
		return err
	}
	if out, err := exec.Command("sysctl", "--system").CombinedOutput(); err != nil {
		return fmt.Errorf("sysctl --system: %w: %s", err, out)
	}
	return a.forwardStatus()
}

func (a *App) forwardDisable() error {
	_ = os.Remove(fwdSysctlFile)
	_ = exec.Command("sysctl", "-w", "net.ipv4.ip_forward=0").Run()
	return a.forwardStatus()
}

func (a *App) forwardStatus() error {
	fmt.Fprintf(a.out, "ip_forward: %s\n", sysctlGet("net.ipv4.ip_forward"))
	return nil
}

// --------- firewall ---------

func (a *App) firewallCmd(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: firewall <open|close|status> [--port N]")
	}
	sub := args[0]
	fs := flag.NewFlagSet("firewall "+sub, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	port := fs.Int("port", 0, "port (defaults to current Xray port)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	p := *port
	if p == 0 {
		cfg, err := xray.LoadConfig(a.configPath)
		if err != nil {
			return fmt.Errorf("read config to detect port: %w", err)
		}
		p = cfg.Port()
	}
	if p < 1 || p > 65535 {
		return fmt.Errorf("invalid port %d", p)
	}

	switch sub {
	case "open":
		if err := requireRoot("firewall open"); err != nil {
			return err
		}
		return a.firewallOpen(p)
	case "close":
		if err := requireRoot("firewall close"); err != nil {
			return err
		}
		return a.firewallClose(p)
	case "status":
		return a.firewallStatusPort(p)
	}
	return fmt.Errorf("unknown firewall subcommand %q", sub)
}

func (a *App) firewallOpen(port int) error {
	if commandExists("ufw") {
		if out, err := exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", port)).CombinedOutput(); err != nil {
			return fmt.Errorf("ufw: %w: %s", err, out)
		}
		fmt.Fprintf(a.out, "ufw: allowed %d/tcp\n", port)
		return nil
	}
	if exec.Command("systemctl", "is-active", "--quiet", "firewalld").Run() == nil && commandExists("firewall-cmd") {
		args := []string{"--permanent", "--add-port", fmt.Sprintf("%d/tcp", port)}
		if out, err := exec.Command("firewall-cmd", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("firewall-cmd: %w: %s", err, out)
		}
		_ = exec.Command("firewall-cmd", "--reload").Run()
		fmt.Fprintf(a.out, "firewalld: allowed %d/tcp\n", port)
		return nil
	}
	if commandExists("iptables") {
		if out, err := exec.Command("iptables", "-I", "INPUT", "-p", "tcp", "--dport", fmt.Sprintf("%d", port), "-j", "ACCEPT").CombinedOutput(); err != nil {
			return fmt.Errorf("iptables: %w: %s", err, out)
		}
		fmt.Fprintf(a.out, "iptables: inserted ACCEPT for %d/tcp (not persistent — see docs/cheatsheet.md)\n", port)
		return nil
	}
	return errors.New("no supported firewall found (ufw, firewalld, iptables)")
}

func (a *App) firewallClose(port int) error {
	if commandExists("ufw") {
		if out, err := exec.Command("ufw", "delete", "allow", fmt.Sprintf("%d/tcp", port)).CombinedOutput(); err != nil {
			return fmt.Errorf("ufw: %w: %s", err, out)
		}
		fmt.Fprintf(a.out, "ufw: removed %d/tcp\n", port)
		return nil
	}
	if exec.Command("systemctl", "is-active", "--quiet", "firewalld").Run() == nil && commandExists("firewall-cmd") {
		args := []string{"--permanent", "--remove-port", fmt.Sprintf("%d/tcp", port)}
		if out, err := exec.Command("firewall-cmd", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("firewall-cmd: %w: %s", err, out)
		}
		_ = exec.Command("firewall-cmd", "--reload").Run()
		fmt.Fprintf(a.out, "firewalld: removed %d/tcp\n", port)
		return nil
	}
	if commandExists("iptables") {
		_, _ = exec.Command("iptables", "-D", "INPUT", "-p", "tcp", "--dport", fmt.Sprintf("%d", port), "-j", "ACCEPT").CombinedOutput()
		fmt.Fprintf(a.out, "iptables: attempted delete for %d/tcp\n", port)
		return nil
	}
	return errors.New("no supported firewall found")
}

func (a *App) firewallStatusPort(port int) error {
	probe := firewallProbe(port)
	fmt.Fprintf(a.out, "%s\t%s\n", probe.Status, probe.Detail)
	return nil
}

// --------- fix-perms ---------

func (a *App) fixPerms() error {
	if err := requireRoot("fix-perms"); err != nil {
		return err
	}
	user, group := serviceUserGroup(xrayServiceName)
	if user == "" {
		user = "xray"
		group = "xray"
	}
	if err := chownByName(a.configPath, user, group); err != nil {
		return err
	}
	if err := os.Chmod(a.configPath, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(a.out, "set %s to %s:%s 0644\n", a.configPath, user, group)
	return a.runner.RestartService()
}

func applyServiceOwnership(path string) error {
	user, group := serviceUserGroup(xrayServiceName)
	if user == "" {
		user = "xray"
		group = "xray"
	}
	if err := os.Chmod(path, 0o644); err != nil {
		return err
	}
	return chownByName(path, user, group)
}

func serviceUserGroup(service string) (user, group string) {
	out, err := exec.Command("systemctl", "show", service, "-p", "User", "-p", "Group").Output()
	if err != nil {
		return "", ""
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if v, ok := cutPrefix(line, "User="); ok {
			user = strings.TrimSpace(v)
		}
		if v, ok := cutPrefix(line, "Group="); ok {
			group = strings.TrimSpace(v)
		}
	}
	return user, group
}

func chownByName(path, user, group string) error {
	cmd := exec.Command("chown", fmt.Sprintf("%s:%s", user, group), path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("chown %s: %w: %s", path, err, out)
	}
	return nil
}

// --------- update xray ---------

func (a *App) updateXray() error {
	if err := requireRoot("update"); err != nil {
		return err
	}
	fmt.Fprintln(a.out, "Updating Xray via XTLS official installer...")
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf(`bash -c "$(curl -L %s)" @ install`, xrayInstallURL))
	cmd.Stdout = a.out
	cmd.Stderr = a.out
	if err := cmd.Run(); err != nil {
		return err
	}
	// Re-apply service ownership in case the installer reset perms.
	if _, err := os.Stat(a.configPath); err == nil {
		_ = applyServiceOwnership(a.configPath)
	}
	return nil
}

// --------- helpers ---------

func requireRoot(cmd string) error {
	if os.Geteuid() == 0 {
		return nil
	}
	return fmt.Errorf("%s must run as root — try: sudo xctl %s", cmd, cmd)
}

func sysctlGet(key string) string {
	out, err := exec.Command("sysctl", "-n", key).Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

