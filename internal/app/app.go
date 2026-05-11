package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lonelyrower/xray-fast-deploy/internal/doctor"
	"github.com/lonelyrower/xray-fast-deploy/internal/link"
	"github.com/lonelyrower/xray-fast-deploy/internal/serverinfo"
	"github.com/lonelyrower/xray-fast-deploy/internal/system"
	"github.com/lonelyrower/xray-fast-deploy/internal/tui"
	"github.com/lonelyrower/xray-fast-deploy/internal/xray"
)

const (
	DefaultConfigPath = "/usr/local/etc/xray/config.json"
	DefaultInfoPath   = "/root/.xray-reality/server.info"
)

type Config struct {
	ConfigPath string
	InfoPath   string
	Out        io.Writer
	Runner     system.ConfigRunner
}

type App struct {
	configPath string
	infoPath   string
	out        io.Writer
	runner     system.ConfigRunner
}

func New(cfg Config) *App {
	if cfg.ConfigPath == "" {
		cfg.ConfigPath = DefaultConfigPath
	}
	if cfg.InfoPath == "" {
		cfg.InfoPath = DefaultInfoPath
	}
	if cfg.Out == nil {
		cfg.Out = os.Stdout
	}
	if cfg.Runner == nil {
		cfg.Runner = system.Runner{}
	}
	return &App{configPath: cfg.ConfigPath, infoPath: cfg.InfoPath, out: cfg.Out, runner: cfg.Runner}
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		args = []string{"tui"}
	}
	switch args[0] {
	case "help", "--help", "-h":
		a.printHelp()
	case "list-clients":
		return a.listClients()
	case "add-client":
		return a.addClient(args[1:])
	case "remove-client":
		return a.removeClient(args[1:])
	case "rename-client":
		return a.renameClient(args[1:])
	case "reset-uuid":
		return a.resetUUID(args[1:])
	case "show-client":
		return a.showClient(args[1:])
	case "export":
		return a.export()
	case "status":
		return a.status()
	case "doctor":
		return a.doctor()
	case "change-port":
		return a.changePort(args[1:])
	case "change-disguise":
		return a.changeDisguise(args[1:])
	case "server-address":
		return a.serverAddress(args[1:])
	case "test":
		return a.runner.TestConfig(a.configPath)
	case "restart":
		return a.runner.RestartService()
	case "logs":
		return a.logs(args[1:])
	case "init":
		return a.initConfig(args[1:])
	case "bbr":
		return a.bbrCmd(args[1:])
	case "forward":
		return a.forwardCmd(args[1:])
	case "firewall":
		return a.firewallCmd(args[1:])
	case "fix-perms":
		return a.fixPerms()
	case "update":
		return a.updateXray()
	case "tui":
		return ErrTUIRequested
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
	return nil
}

var ErrTUIRequested = errors.New("tui requested")

func (a *App) Data() tui.ModelData {
	return a.TUIData()
}

func (a *App) AddClientTUI(name string) error {
	if name == "" {
		return errors.New("name is required")
	}
	id := newUUID()
	return system.SafeConfigUpdate(a.configPath, a.runner, func(cfg *xray.Config) error {
		return cfg.AddClient(xray.Client{ID: id, Email: name})
	})
}

func (a *App) RemoveClientTUI(name string) error {
	if name == "" {
		return errors.New("name is required")
	}
	return system.SafeConfigUpdate(a.configPath, a.runner, func(cfg *xray.Config) error {
		return cfg.RemoveClient(name)
	})
}

func (a *App) RenameClientTUI(oldName, newName string) error {
	if oldName == "" || newName == "" {
		return errors.New("both names are required")
	}
	return system.SafeConfigUpdate(a.configPath, a.runner, func(cfg *xray.Config) error {
		return cfg.RenameClient(oldName, newName)
	})
}

func (a *App) ResetUUIDTUI(name string) error {
	if name == "" {
		return errors.New("name is required")
	}
	id := newUUID()
	return system.SafeConfigUpdate(a.configPath, a.runner, func(cfg *xray.Config) error {
		return cfg.ResetClientUUID(name, id)
	})
}

func (a *App) ChangePortTUI(port int) error {
	if port < 1 || port > 65535 {
		return errors.New("port must be 1-65535")
	}
	return system.SafeConfigUpdate(a.configPath, a.runner, func(cfg *xray.Config) error {
		cfg.SetPort(port)
		return nil
	})
}

func (a *App) ChangeDisguiseTUI(domain string) error {
	if domain == "" {
		return errors.New("domain is required")
	}
	return system.SafeConfigUpdate(a.configPath, a.runner, func(cfg *xray.Config) error {
		cfg.SetDisguise(domain+":443", domain)
		return nil
	})
}

func (a *App) SetServerAddressTUI(address string) error {
	if address == "" {
		return errors.New("address is required")
	}
	info, _ := serverinfo.Load(a.infoPath)
	info.Address = address
	if info.Port == 0 {
		info.Port = 443
	}
	return serverinfo.Save(a.infoPath, info)
}

func (a *App) TestTUI() error {
	return a.runner.TestConfig(a.configPath)
}

func (a *App) RestartTUI() error {
	return a.runner.RestartService()
}

func (a *App) ClientLinkTUI(name string) (string, error) {
	if name == "" {
		return "", errors.New("name is required")
	}
	cfg, err := xray.LoadConfig(a.configPath)
	if err != nil {
		return "", err
	}
	info, _ := serverinfo.Load(a.infoPath)
	address := serverinfo.ResolveAddress(info, detectPublicIPv4)
	for _, client := range cfg.Clients() {
		if client.Email == name {
			return link.GenerateVLESS(link.Link{
				UUID:      client.ID,
				Address:   address,
				Port:      cfg.Port(),
				PublicKey: info.PublicKey,
				SNI:       cfg.SNI(),
				Name:      client.Email,
				ShortID:   cfg.ShortID(),
			}), nil
		}
	}
	return "", fmt.Errorf("client %q not found", name)
}

func (a *App) TUIData() tui.ModelData {
	data := tui.ModelData{
		Service:      systemctlActive("xray"),
		Version:      xrayVersion(),
		BBR:          bbrStatus(),
		ConfigStatus: "unknown",
	}
	cfg, err := xray.LoadConfig(a.configPath)
	if err == nil {
		data.Port = cfg.Port()
		data.SNI = cfg.SNI()
		data.ClientCount = len(cfg.Clients())
		for _, client := range cfg.Clients() {
			data.Clients = append(data.Clients, tui.Client{Name: client.Email, UUID: client.ID})
		}
		if a.runner.TestConfig(a.configPath) == nil {
			data.ConfigStatus = "valid"
		} else {
			data.ConfigStatus = "failed"
		}
	} else {
		data.LoadError = wrapPermErr(err, a.configPath).Error()
	}
	if info, err := serverinfo.Load(a.infoPath); err == nil {
		data.Address = info.Address
	}
	return data
}

func (a *App) printHelp() {
	fmt.Fprintln(a.out, `xctl

Commands:
  tui
  status
  doctor
  list-clients
  add-client --name NAME [--uuid UUID]
  remove-client --name NAME
  rename-client --name OLD --new-name NEW
  reset-uuid --name NAME [--uuid UUID]
  show-client --name NAME
  export
  change-port --port PORT
  change-disguise --domain DOMAIN
  server-address --address ADDRESS
  test
  restart
  logs [--lines N]
  init [--sni DOMAIN] [--port N] [--name NAME] [--force]
  bbr <enable|disable|status>
  forward <enable|disable|status>
  firewall <open|close|status> [--port N]
  fix-perms
  update`)
}

func (a *App) listClients() error {
	cfg, err := xray.LoadConfig(a.configPath)
	if err != nil {
		return err
	}
	for _, client := range cfg.Clients() {
		fmt.Fprintf(a.out, "%s\t%s\n", client.Email, client.ID)
	}
	return nil
}

func (a *App) addClient(args []string) error {
	fs := flag.NewFlagSet("add-client", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "client name")
	uuid := fs.String("uuid", "", "client uuid")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return errors.New("--name is required")
	}
	id := *uuid
	if id == "" {
		id = newUUID()
	}
	return system.SafeConfigUpdate(a.configPath, a.runner, func(cfg *xray.Config) error {
		return cfg.AddClient(xray.Client{ID: id, Email: *name})
	})
}

func (a *App) removeClient(args []string) error {
	name, err := requiredName("remove-client", args)
	if err != nil {
		return err
	}
	return system.SafeConfigUpdate(a.configPath, a.runner, func(cfg *xray.Config) error {
		return cfg.RemoveClient(name)
	})
}

func (a *App) renameClient(args []string) error {
	fs := flag.NewFlagSet("rename-client", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "client name")
	newName := fs.String("new-name", "", "new client name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" || *newName == "" {
		return errors.New("--name and --new-name are required")
	}
	return system.SafeConfigUpdate(a.configPath, a.runner, func(cfg *xray.Config) error {
		return cfg.RenameClient(*name, *newName)
	})
}

func (a *App) resetUUID(args []string) error {
	fs := flag.NewFlagSet("reset-uuid", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "client name")
	uuid := fs.String("uuid", "", "client uuid")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return errors.New("--name is required")
	}
	id := *uuid
	if id == "" {
		id = newUUID()
	}
	return system.SafeConfigUpdate(a.configPath, a.runner, func(cfg *xray.Config) error {
		return cfg.ResetClientUUID(*name, id)
	})
}

func (a *App) showClient(args []string) error {
	name, err := requiredName("show-client", args)
	if err != nil {
		return err
	}
	cfg, err := xray.LoadConfig(a.configPath)
	if err != nil {
		return err
	}
	info, _ := serverinfo.Load(a.infoPath)
	address := serverinfo.ResolveAddress(info, detectPublicIPv4)
	for _, client := range cfg.Clients() {
		if client.Email == name {
			fmt.Fprintln(a.out, link.GenerateVLESS(link.Link{
				UUID:      client.ID,
				Address:   address,
				Port:      cfg.Port(),
				PublicKey: info.PublicKey,
				SNI:       cfg.SNI(),
				Name:      client.Email,
				ShortID:   cfg.ShortID(),
			}))
			return nil
		}
	}
	return fmt.Errorf("client %q not found", name)
}

func (a *App) export() error {
	cfg, err := xray.LoadConfig(a.configPath)
	if err != nil {
		return err
	}
	info, _ := serverinfo.Load(a.infoPath)
	address := serverinfo.ResolveAddress(info, detectPublicIPv4)
	for _, client := range cfg.Clients() {
		fmt.Fprintf(a.out, "%s\n%s\n", client.Email, link.GenerateVLESS(link.Link{
			UUID:      client.ID,
			Address:   address,
			Port:      cfg.Port(),
			PublicKey: info.PublicKey,
			SNI:       cfg.SNI(),
			Name:      client.Email,
			ShortID:   cfg.ShortID(),
		}))
	}
	return nil
}

func (a *App) status() error {
	cfg, err := xray.LoadConfig(a.configPath)
	if err != nil {
		return wrapPermErr(err, a.configPath)
	}
	info, _ := serverinfo.Load(a.infoPath)
	fmt.Fprintf(a.out, "Service: %s\n", systemctlActive("xray"))
	fmt.Fprintf(a.out, "Port: %d\n", cfg.Port())
	fmt.Fprintf(a.out, "SNI: %s\n", cfg.SNI())
	fmt.Fprintf(a.out, "Saved Address: %s\n", info.Address)
	fmt.Fprintf(a.out, "Clients: %d\n", len(cfg.Clients()))
	return nil
}

func (a *App) doctor() error {
	cfg, _ := xray.LoadConfig(a.configPath)
	info, _ := serverinfo.Load(a.infoPath)
	probe := doctor.Probe{
		XrayInstalled: fileExists("/usr/local/bin/xray") || commandExists("xray"),
		ServiceActive: systemctlActive("xray") == "active",
		ConfigExists:  fileExists(a.configPath),
		ConfigValid:   a.runner.TestConfig(a.configPath) == nil,
		Firewall:      firewallProbe(0),
		BBR:           bbrProbe(),
		PublicIPv4:    mustDetectPublicIPv4(),
		SavedAddress:  info.Address,
		DiskOK:        diskSpaceOK("/var/log"),
		TimeOK:        time.Now().Year() >= 2020,
		IPForwarding:  ipForwardingProbe(),
		RecentErrors:  recentXrayErrors(),
	}
	if cfg != nil {
		probe.ConfigPort = cfg.Port()
		probe.PortListening = portListening(cfg.Port())
		probe.ListeningPort = observedListeningPort(cfg.Port())
		probe.Firewall = firewallProbe(cfg.Port())
	}
	for _, result := range doctor.Run(probe) {
		if result.Advice == "" {
			fmt.Fprintf(a.out, "%s\t%s\t%s\n", result.Status, result.Name, result.Message)
		} else {
			fmt.Fprintf(a.out, "%s\t%s\t%s\t%s\n", result.Status, result.Name, result.Message, result.Advice)
		}
	}
	return nil
}

func (a *App) changePort(args []string) error {
	fs := flag.NewFlagSet("change-port", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	port := fs.Int("port", 0, "port")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *port < 1 || *port > 65535 {
		return errors.New("--port must be 1-65535")
	}
	return system.SafeConfigUpdate(a.configPath, a.runner, func(cfg *xray.Config) error {
		cfg.SetPort(*port)
		return nil
	})
}

func (a *App) changeDisguise(args []string) error {
	fs := flag.NewFlagSet("change-disguise", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	domain := fs.String("domain", "", "domain")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *domain == "" {
		return errors.New("--domain is required")
	}
	return system.SafeConfigUpdate(a.configPath, a.runner, func(cfg *xray.Config) error {
		cfg.SetDisguise(*domain+":443", *domain)
		return nil
	})
}

func (a *App) serverAddress(args []string) error {
	fs := flag.NewFlagSet("server-address", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	address := fs.String("address", "", "address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *address == "" {
		return errors.New("--address is required")
	}
	info, _ := serverinfo.Load(a.infoPath)
	info.Address = *address
	if info.Port == 0 {
		info.Port = 443
	}
	return serverinfo.Save(a.infoPath, info)
}

func (a *App) logs(args []string) error {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	lines := fs.String("lines", "50", "lines")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cmd := exec.Command("journalctl", "-u", "xray", "-n", *lines, "--no-pager")
	cmd.Stdout = a.out
	cmd.Stderr = a.out
	return cmd.Run()
}

func wrapPermErr(err error, path string) error {
	if os.IsPermission(err) && os.Geteuid() != 0 {
		return fmt.Errorf("%w (try running with sudo — %s is root-owned)", err, path)
	}
	return err
}

func requiredName(command string, args []string) (string, error) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "client name")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if *name == "" {
		return "", errors.New("--name is required")
	}
	return *name, nil
}

func newUUID() string {
	data, err := os.ReadFile("/proc/sys/kernel/random/uuid")
	if err == nil {
		return strings.TrimSpace(string(data))
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func detectPublicIPv4() (string, error) {
	for _, endpoint := range []string{"https://ifconfig.me", "https://ipinfo.io/ip", "https://icanhazip.com"} {
		client := http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(endpoint)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			ip := strings.TrimSpace(string(body))
			if ip != "" {
				return ip, nil
			}
		}
	}
	return "", errors.New("public IPv4 detection failed")
}

func mustDetectPublicIPv4() string {
	ip, err := detectPublicIPv4()
	if err != nil {
		return "UNKNOWN"
	}
	return ip
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func systemctlActive(service string) string {
	if err := exec.Command("systemctl", "is-active", "--quiet", service).Run(); err != nil {
		return "inactive"
	}
	return "active"
}

func xrayVersion() string {
	path, err := exec.LookPath("xray")
	if err != nil {
		if fileExists("/usr/local/bin/xray") {
			path = "/usr/local/bin/xray"
		} else {
			return "not installed"
		}
	}
	out, err := exec.Command(path, "version").Output()
	if err != nil {
		return "unknown"
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	return line
}

func bbrStatus() string {
	probe := bbrProbe()
	if probe.CongestionControl == "bbr" && probe.DefaultQdisc == "fq" && probe.ModuleLoaded {
		return "enabled"
	}
	if probe.CongestionControl == "" {
		return "unknown"
	}
	return "disabled"
}

func bbrProbe() doctor.BBRProbe {
	data, err := os.ReadFile("/proc/sys/net/ipv4/tcp_congestion_control")
	if err != nil {
		return doctor.BBRProbe{}
	}
	qdisc, _ := os.ReadFile("/proc/sys/net/core/default_qdisc")
	return doctor.BBRProbe{
		CongestionControl: strings.TrimSpace(string(data)),
		DefaultQdisc:      strings.TrimSpace(string(qdisc)),
		ModuleLoaded:      moduleLoaded("tcp_bbr"),
	}
}

func portListening(port int) bool {
	return observedListeningPort(port) == port
}

func observedListeningPort(port int) int {
	out, err := exec.Command("ss", "-tln").Output()
	if err != nil {
		return 0
	}
	needle := fmt.Sprintf(":%d ", port)
	if strings.Contains(string(out), needle) {
		return port
	}
	return 0
}

func ipForwardingProbe() doctor.IPForwardingProbe {
	data, err := os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	if err != nil {
		return doctor.IPForwardingProbe{}
	}
	return doctor.IPForwardingProbe{Enabled: strings.TrimSpace(string(data)) == "1"}
}

func firewallProbe(port int) doctor.FirewallProbe {
	if port == 0 {
		return doctor.FirewallProbe{Status: "unknown", Detail: "configured port unknown"}
	}
	if commandExists("ufw") {
		out, err := exec.Command("ufw", "status").CombinedOutput()
		text := string(out)
		if err == nil && strings.Contains(text, "Status: active") {
			if strings.Contains(text, fmt.Sprintf("%d/tcp", port)) || strings.Contains(text, fmt.Sprintf("%d ", port)) {
				return doctor.FirewallProbe{Status: "allowed", Detail: fmt.Sprintf("ufw allows %d/tcp", port)}
			}
			return doctor.FirewallProbe{Status: "blocked", Detail: fmt.Sprintf("ufw active, %d/tcp not allowed", port)}
		}
	}
	if exec.Command("systemctl", "is-active", "--quiet", "firewalld").Run() == nil && commandExists("firewall-cmd") {
		out, err := exec.Command("firewall-cmd", "--list-ports").Output()
		if err == nil {
			if strings.Contains(string(out), fmt.Sprintf("%d/tcp", port)) {
				return doctor.FirewallProbe{Status: "allowed", Detail: fmt.Sprintf("firewalld allows %d/tcp", port)}
			}
			return doctor.FirewallProbe{Status: "blocked", Detail: fmt.Sprintf("firewalld active, %d/tcp not allowed", port)}
		}
	}
	if commandExists("iptables") {
		out, err := exec.Command("iptables", "-L", "INPUT", "-n").Output()
		if err == nil && strings.Contains(string(out), fmt.Sprintf("dpt:%d", port)) {
			return doctor.FirewallProbe{Status: "allowed", Detail: fmt.Sprintf("iptables has a rule for %d/tcp", port)}
		}
	}
	return doctor.FirewallProbe{Status: "unknown", Detail: "no active local firewall rule detected"}
}

func moduleLoaded(module string) bool {
	data, err := os.ReadFile("/proc/modules")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, module+" ") {
			return true
		}
	}
	return false
}

func diskSpaceOK(path string) bool {
	out, err := exec.Command("df", "-P", path).Output()
	if err != nil {
		return true
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return true
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 5 {
		return true
	}
	used := strings.TrimSuffix(fields[4], "%")
	var percent int
	if _, err := fmt.Sscanf(used, "%d", &percent); err != nil {
		return true
	}
	return percent < 95
}

func recentXrayErrors() []string {
	out, err := exec.Command("journalctl", "-u", "xray", "-n", "50", "--no-pager").Output()
	if err != nil {
		return nil
	}
	var errors []string
	for _, line := range strings.Split(string(out), "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") || strings.Contains(lower, "failed") || strings.Contains(lower, "panic") {
			errors = append(errors, strings.TrimSpace(line))
		}
	}
	if len(errors) > 3 {
		return errors[len(errors)-3:]
	}
	return errors
}
