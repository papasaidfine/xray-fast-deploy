package doctor

import "fmt"

type Status string

const (
	OK   Status = "ok"
	Warn Status = "warn"
	Fail Status = "fail"
)

type Probe struct {
	XrayInstalled bool
	ServiceActive bool
	ConfigExists  bool
	ConfigValid   bool
	ConfigPort    int
	ListeningPort int
	PortListening bool
	Firewall      FirewallProbe
	BBR           BBRProbe
	PublicIPv4    string
	SavedAddress  string
	DiskOK        bool
	TimeOK        bool
	IPForwarding  IPForwardingProbe
	RecentErrors  []string
}

type FirewallProbe struct {
	Status string
	Detail string
}

type BBRProbe struct {
	CongestionControl string
	DefaultQdisc      string
	ModuleLoaded      bool
}

type IPForwardingProbe struct {
	Enabled bool
}

type Result struct {
	Name    string
	Status  Status
	Message string
	Advice  string
}

func Run(probe Probe) []Result {
	return []Result{
		boolResult("Xray binary", probe.XrayInstalled, "installed", "not found", "Install Xray before managing this node."),
		boolResult("Xray service", probe.ServiceActive, "active", "inactive", "Run `systemctl status xray` and `journalctl -u xray -n 50 --no-pager`."),
		boolResult("Config file", probe.ConfigExists, "exists", "missing", "Create /usr/local/etc/xray/config.json or run the installer once available."),
		boolResult("Config test", probe.ConfigValid, "passed", "failed", "Run `xray run -test -config /usr/local/etc/xray/config.json` and fix the reported JSON/config error."),
		portResult(probe),
		firewallResult(probe.Firewall, probe.ConfigPort),
		bbrResult(probe.BBR),
		savedAddressResult(probe.SavedAddress, probe.PublicIPv4),
		recentErrorsResult(probe.RecentErrors),
		boolResult("Log disk space", probe.DiskOK, "available", "low", "Free disk space or reduce journal size with `journalctl --vacuum-size=500M`."),
		boolResult("System time", probe.TimeOK, "sane", "check required", "Run `timedatectl` and enable NTP if the clock is wrong."),
		ipForwardingResult(probe.IPForwarding),
	}
}

func boolResult(name string, ok bool, okMessage, failMessage, advice string) Result {
	if ok {
		return Result{Name: name, Status: OK, Message: okMessage}
	}
	return Result{Name: name, Status: Fail, Message: failMessage, Advice: advice}
}

func portResult(probe Probe) Result {
	if probe.PortListening {
		return Result{Name: "Port listening", Status: OK, Message: fmt.Sprintf("%d/tcp detected", probe.ConfigPort)}
	}
	message := "not detected"
	if probe.ConfigPort != 0 {
		message = fmt.Sprintf("%d/tcp not detected", probe.ConfigPort)
	}
	if probe.ListeningPort != 0 && probe.ListeningPort != probe.ConfigPort {
		message = fmt.Sprintf("configured %d/tcp, observed %d/tcp", probe.ConfigPort, probe.ListeningPort)
	}
	return Result{Name: "Port listening", Status: Fail, Message: message, Advice: "Run `ss -tlnp` and check whether Xray is active and bound to the configured port."}
}

func firewallResult(firewall FirewallProbe, port int) Result {
	message := valueOrUnknown(firewall.Detail)
	if firewall.Status == "allowed" {
		return Result{Name: "Local firewall", Status: OK, Message: message}
	}
	status := Warn
	advice := "Check local firewall rules for the configured TCP port."
	if firewall.Status == "blocked" {
		status = Fail
		if port != 0 {
			advice = fmt.Sprintf("Allow the local firewall port, for example `ufw allow %d/tcp`, or the equivalent firewalld/iptables rule.", port)
		}
	}
	return Result{Name: "Local firewall", Status: status, Message: message, Advice: advice}
}

func bbrResult(bbr BBRProbe) Result {
	message := fmt.Sprintf("congestion=%s qdisc=%s module=%t", valueOrUnknown(bbr.CongestionControl), valueOrUnknown(bbr.DefaultQdisc), bbr.ModuleLoaded)
	if bbr.CongestionControl == "bbr" && bbr.DefaultQdisc == "fq" && bbr.ModuleLoaded {
		return Result{Name: "BBR", Status: OK, Message: message}
	}
	return Result{Name: "BBR", Status: Warn, Message: message, Advice: "Configure /etc/sysctl.d/90-xray-bbr.conf and /etc/modules-load.d/bbr.conf, then run `sysctl --system`."}
}

func savedAddressResult(saved, public string) Result {
	if saved == "" {
		return Result{Name: "Saved address", Status: Warn, Message: "not set", Advice: "Run `xctl server-address --address <ip-or-domain>` before exporting links."}
	}
	if public != "" && public != "UNKNOWN" && saved != public {
		return Result{Name: "Saved address", Status: Warn, Message: fmt.Sprintf("%s (detected IPv4: %s)", saved, public), Advice: "If clients use raw IP links, update saved address and re-export links. If using DNS, update DNS instead."}
	}
	return Result{Name: "Saved address", Status: OK, Message: saved}
}

func recentErrorsResult(errors []string) Result {
	if len(errors) == 0 {
		return Result{Name: "Recent errors", Status: OK, Message: "none"}
	}
	return Result{Name: "Recent errors", Status: Fail, Message: errors[0], Advice: "Run `journalctl -u xray -n 50 --no-pager` for recent service errors."}
}

func ipForwardingResult(ipForwarding IPForwardingProbe) Result {
	if ipForwarding.Enabled {
		return Result{Name: "IP forwarding", Status: Warn, Message: "enabled", Advice: "Enabled forwarding is only needed for router/TUN/NAT use cases, not default Xray proxy mode."}
	}
	return Result{Name: "IP forwarding", Status: OK, Message: "disabled", Advice: "Disabled is normal for the default VLESS REALITY proxy deployment."}
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
