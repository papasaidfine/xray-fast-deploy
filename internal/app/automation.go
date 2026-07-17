package app

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lonelyrower/xray-fast-deploy/internal/link"
	"github.com/lonelyrower/xray-fast-deploy/internal/serverinfo"
	"github.com/lonelyrower/xray-fast-deploy/internal/xray"
)

const xctlReleaseAPI = "https://api.github.com/repos/papasaidfine/xray-fast-deploy/releases/latest"

const (
	xrayInstallURL  = "https://github.com/XTLS/Xray-install/raw/main/install-release.sh"
	bbrSysctlFile   = "/etc/sysctl.d/99-xctl-bbr.conf"
	fwdSysctlFile   = "/etc/sysctl.d/99-xctl-ip-forward.conf"
	defaultSNI      = "www.apple.com"
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
	proxy := fs.String("proxy", "", "route the Xray installer download through this proxy (http:// or socks5://)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if _, err := os.Stat(a.configPath); err == nil && !*force {
		return fmt.Errorf("%s already exists; pass --force to overwrite", a.configPath)
	}

	if !commandExists("xray") && !fileExists("/usr/local/bin/xray") {
		fmt.Fprintln(a.out, "Xray not found, installing via XTLS official script...")
		if err := runXrayInstaller(a.out, *proxy); err != nil {
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

// runXrayInstaller fetches the XTLS official install-release.sh and runs its
// "install" action. When proxy is non-empty it is used both for fetching the
// script (curl -x) and for the script's own downloads (--proxy), so it works
// on hosts where GitHub is blocked. The script is piped in via stdin and the
// proxy is passed as a discrete argv element, never spliced into a shell
// string, so a user-supplied value cannot be interpreted by the shell.
func runXrayInstaller(out io.Writer, proxy string) error {
	script, err := exec.Command("curl", installerCurlArgs(proxy)...).Output()
	if err != nil {
		return fmt.Errorf("download xray installer: %w", err)
	}
	cmd := exec.Command("bash", installerBashArgs(proxy)...)
	cmd.Stdin = bytes.NewReader(script)
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}

// installerCurlArgs builds the curl argv that downloads the installer script,
// routing the fetch through proxy when one is set.
func installerCurlArgs(proxy string) []string {
	args := []string{"-L"}
	if proxy != "" {
		args = append(args, "-x", proxy)
	}
	return append(args, xrayInstallURL)
}

// installerBashArgs builds the bash argv that runs the installer from stdin
// (bash -s: positional params start at $1, matching the script's "install"
// action), appending --proxy so the script proxies its own core download too.
func installerBashArgs(proxy string) []string {
	args := []string{"-s", "install"}
	if proxy != "" {
		args = append(args, "--proxy", proxy)
	}
	return args
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
	return parseX25519Output(string(out))
}

// parseX25519Output handles every label `xray x25519` has used across
// versions: "Private key:"/"Public key:", "PrivateKey:"/"Password:", and
// the 26.x "PrivateKey:"/"Password (PublicKey):" (plus an ignored Hash32
// line). Labels are matched by keyword, not exact prefix, so minor future
// renames keep working.
func parseX25519Output(out string) (pub, priv string, err error) {
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		label, value, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			continue
		}
		label = strings.ToLower(strings.ReplaceAll(label, " ", ""))
		value = strings.TrimSpace(value)
		switch {
		case strings.Contains(label, "private"):
			priv = value
		case strings.Contains(label, "public") || label == "password":
			pub = value
		}
	}
	if pub == "" || priv == "" {
		return "", "", fmt.Errorf("could not parse xray x25519 output:\n%s", out)
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

func (a *App) updateXray(args []string) error {
	if err := requireRoot("update"); err != nil {
		return err
	}
	fs := flag.NewFlagSet("xray-update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	proxy := fs.String("proxy", "", "route the Xray installer download through this proxy (http:// or socks5://)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fmt.Fprintln(a.out, "Updating Xray via XTLS official installer...")
	if err := runXrayInstaller(a.out, *proxy); err != nil {
		return err
	}
	// Re-apply service ownership in case the installer reset perms.
	if _, err := os.Stat(a.configPath); err == nil {
		_ = applyServiceOwnership(a.configPath)
	}
	return nil
}

// --------- helpers ---------

func describeConfigPerms(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "unknown"
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Sprintf("%#o", info.Mode().Perm())
	}
	owner := strconv.Itoa(int(stat.Uid))
	group := strconv.Itoa(int(stat.Gid))
	if u, err := user.LookupId(owner); err == nil {
		owner = u.Username
	}
	if g, err := user.LookupGroupId(group); err == nil {
		group = g.Name
	}
	return fmt.Sprintf("%s:%s %#o", owner, group, info.Mode().Perm())
}

// TUI wrappers — these are what tui.Service calls. They delegate to the
// CLI implementations (which include root checks).

func (a *App) BBREnableTUI() error      { return a.bbrEnable() }
func (a *App) BBRDisableTUI() error     { return a.bbrDisable() }
func (a *App) ForwardEnableTUI() error  { return a.forwardEnable() }
func (a *App) ForwardDisableTUI() error { return a.forwardDisable() }
func (a *App) FirewallOpenTUI() error {
	cfg, err := xray.LoadConfig(a.configPath)
	if err != nil {
		return err
	}
	return a.firewallOpen(cfg.Port())
}
func (a *App) FirewallCloseTUI() error {
	cfg, err := xray.LoadConfig(a.configPath)
	if err != nil {
		return err
	}
	return a.firewallClose(cfg.Port())
}
func (a *App) FixPermsTUI() error { return a.fixPerms() }

func (a *App) CheckUpdateTUI() (current, latest string) {
	latestTag, _, err := fetchLatestRelease()
	if err != nil {
		return a.version, ""
	}
	if compareVersions(a.version, latestTag) >= 0 {
		return a.version, a.version
	}
	return a.version, latestTag
}

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

// --------- xctl version / self-update ---------

func (a *App) printVersion() error {
	fmt.Fprintf(a.out, "xctl %s\n", a.version)
	latest, _, err := fetchLatestRelease()
	if err != nil {
		fmt.Fprintf(a.out, "  (could not check for updates: %v)\n", err)
		return nil
	}
	switch compareVersions(a.version, latest) {
	case 0:
		fmt.Fprintln(a.out, "  up to date")
	case -1:
		fmt.Fprintf(a.out, "  new version available: %s — run: sudo xctl install\n", latest)
	default:
		fmt.Fprintf(a.out, "  ahead of latest release %s\n", latest)
	}
	return nil
}

func (a *App) selfUpdate() error {
	if err := requireRoot("install"); err != nil {
		return err
	}
	latest, url, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("check latest release: %w", err)
	}
	if compareVersions(a.version, latest) == 0 {
		fmt.Fprintf(a.out, "xctl is already at %s\n", latest)
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}

	dir := filepath.Dir(exe)
	candidate, err := os.CreateTemp(dir, ".xctl-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	candidatePath := candidate.Name()
	defer os.Remove(candidatePath)

	fmt.Fprintf(a.out, "downloading %s ...\n", url)
	if err := download(url, candidate); err != nil {
		candidate.Close()
		return err
	}
	if err := candidate.Close(); err != nil {
		return err
	}
	if err := os.Chmod(candidatePath, 0o755); err != nil {
		return err
	}
	if err := os.Rename(candidatePath, exe); err != nil {
		return fmt.Errorf("replace %s: %w", exe, err)
	}
	fmt.Fprintf(a.out, "xctl updated to %s\n", latest)
	return nil
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

func fetchLatestRelease() (tag, assetURL string, err error) {
	req, err := http.NewRequest(http.MethodGet, xctlReleaseAPI, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", "", fmt.Errorf("github API returned %s", resp.Status)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", "", err
	}
	if rel.TagName == "" {
		return "", "", errors.New("no tag_name in release response")
	}
	want := fmt.Sprintf("xctl-linux-%s", runtime.GOARCH)
	for _, a := range rel.Assets {
		if a.Name == want {
			return rel.TagName, a.URL, nil
		}
	}
	return rel.TagName, "", fmt.Errorf("no asset named %s in release %s", want, rel.TagName)
}

func download(url string, dst io.Writer) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}
	_, err = io.Copy(dst, resp.Body)
	return err
}

// compareVersions returns -1 if a<b, 0 if equal, 1 if a>b. "dev" sorts as
// the lowest possible version so dev builds always see a new release as
// newer. Numeric prerelease parts (e.g. "beta.10") are compared
// numerically so "beta.10" > "beta.2".
func compareVersions(a, b string) int {
	if a == b {
		return 0
	}
	if a == "dev" {
		return -1
	}
	if b == "dev" {
		return 1
	}
	aNums, aPre := splitVersion(a)
	bNums, bPre := splitVersion(b)
	for i := 0; i < len(aNums) || i < len(bNums); i++ {
		var av, bv int
		if i < len(aNums) {
			av = aNums[i]
		}
		if i < len(bNums) {
			bv = bNums[i]
		}
		if av != bv {
			if av < bv {
				return -1
			}
			return 1
		}
	}
	if aPre == "" && bPre == "" {
		return 0
	}
	if aPre == "" {
		return 1
	}
	if bPre == "" {
		return -1
	}
	aParts := strings.Split(aPre, ".")
	bParts := strings.Split(bPre, ".")
	for i := 0; i < len(aParts) || i < len(bParts); i++ {
		if i >= len(aParts) {
			return -1
		}
		if i >= len(bParts) {
			return 1
		}
		ap, bp := aParts[i], bParts[i]
		an, aErr := strconv.Atoi(ap)
		bn, bErr := strconv.Atoi(bp)
		if aErr == nil && bErr == nil {
			if an != bn {
				if an < bn {
					return -1
				}
				return 1
			}
			continue
		}
		if ap != bp {
			if ap < bp {
				return -1
			}
			return 1
		}
	}
	return 0
}

func splitVersion(v string) (nums []int, pre string) {
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		pre = v[i+1:]
		v = v[:i]
	}
	for _, part := range strings.Split(v, ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nums, pre
		}
		nums = append(nums, n)
	}
	return nums, pre
}
