package doctor

import (
	"strings"
	"testing"
)

func TestRunReportsCoreChecks(t *testing.T) {
	probe := Probe{
		XrayInstalled: true,
		ServiceActive: true,
		ConfigExists:  true,
		ConfigValid:   true,
		PortListening: true,
		Firewall: FirewallProbe{
			Status: "unknown",
		},
		BBR: BBRProbe{
			CongestionControl: "bbr",
			DefaultQdisc:      "fq",
			ModuleLoaded:      true,
		},
		PublicIPv4:   "203.0.113.10",
		SavedAddress: "vpn.example.com",
		DiskOK:       true,
		TimeOK:       true,
		IPForwarding: IPForwardingProbe{
			Enabled: false,
		},
		RecentErrors: []string{"xray started"},
	}

	results := Run(probe)
	if len(results) != 12 {
		t.Fatalf("result count = %d, want 12", len(results))
	}
	if results[0].Name != "Xray binary" || results[0].Status != OK {
		t.Fatalf("first result = %+v, want xray binary OK", results[0])
	}
	if results[7].Name != "Saved address" || results[7].Status != Warn {
		t.Fatalf("saved address result = %+v, want warning when different from public IP", results[7])
	}
	ipForwarding := findResult(t, results, "IP forwarding")
	if ipForwarding.Status != OK {
		t.Fatalf("ip forwarding result = %+v, want OK because disabled is normal", ipForwarding)
	}
	if ipForwarding.Advice == "" {
		t.Fatalf("ip forwarding advice is empty")
	}
}

func TestRunReportsActionableFailures(t *testing.T) {
	results := Run(Probe{
		XrayInstalled: true,
		ServiceActive: false,
		ConfigExists:  true,
		ConfigValid:   false,
		ConfigPort:    443,
		ListeningPort: 8443,
		PortListening: false,
		Firewall: FirewallProbe{
			Status: "blocked",
			Detail: "ufw active, 443/tcp not allowed",
		},
		BBR: BBRProbe{
			CongestionControl: "cubic",
			DefaultQdisc:      "pfifo_fast",
			ModuleLoaded:      false,
		},
		PublicIPv4:   "UNKNOWN",
		SavedAddress: "",
		DiskOK:       false,
		TimeOK:       false,
		IPForwarding: IPForwardingProbe{Enabled: true},
		RecentErrors: []string{"failed to parse config"},
	})

	assertResult := func(name string, status Status, adviceContains string) {
		t.Helper()
		for _, result := range results {
			if result.Name == name {
				if result.Status != status {
					t.Fatalf("%s status = %s, want %s", name, result.Status, status)
				}
				if adviceContains != "" && !strings.Contains(result.Advice, adviceContains) {
					t.Fatalf("%s advice = %q, want contains %q", name, result.Advice, adviceContains)
				}
				return
			}
		}
		t.Fatalf("result %q not found in %+v", name, results)
	}

	assertResult("Xray service", Fail, "systemctl status xray")
	assertResult("Config test", Fail, "xray run -test")
	assertResult("Port listening", Fail, "ss -tlnp")
	assertResult("Local firewall", Fail, "ufw allow 443/tcp")
	assertResult("BBR", Warn, "/etc/sysctl.d/90-xray-bbr.conf")
	assertResult("Recent errors", Fail, "journalctl -u xray")
	assertResult("Log disk space", Fail, "journalctl")
	assertResult("System time", Fail, "timedatectl")
}

func findResult(t *testing.T, results []Result, name string) Result {
	t.Helper()
	for _, result := range results {
		if result.Name == name {
			return result
		}
	}
	t.Fatalf("result %q not found in %+v", name, results)
	return Result{}
}
