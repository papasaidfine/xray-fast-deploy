package app

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lonelyrower/xray-fast-deploy/internal/serverinfo"
	"github.com/lonelyrower/xray-fast-deploy/internal/xray"
)

func TestCLIListAddExportClient(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	infoPath := filepath.Join(dir, "server.info")
	cfg := xray.NewRealityConfig(xray.ConfigOptions{
		UUID:       "11111111-1111-4111-8111-111111111111",
		PrivateKey: "private-key",
		Dest:       "www.apple.com:443",
		SNI:        "www.apple.com",
		Port:       443,
		ShortID:    "short-id",
		ClientName: "phone",
	})
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := serverinfo.Save(infoPath, serverinfo.Info{
		PublicKey: "public-key",
		Port:      443,
		SNI:       "www.apple.com",
		Address:   "vpn.example.com",
		Created:   "2026-05-10 00:00:00",
	}); err != nil {
		t.Fatalf("save info: %v", err)
	}

	var out bytes.Buffer
	a := New(Config{ConfigPath: configPath, InfoPath: infoPath, Out: &out, Runner: &FakeRunner{}})

	if err := a.Run([]string{"add-client", "--name", "tablet", "--uuid", "22222222-2222-4222-8222-222222222222"}); err != nil {
		t.Fatalf("add client: %v", err)
	}
	out.Reset()
	if err := a.Run([]string{"list-clients"}); err != nil {
		t.Fatalf("list clients: %v", err)
	}
	if !strings.Contains(out.String(), "tablet") {
		t.Fatalf("list output = %q, want tablet", out.String())
	}
	out.Reset()
	if err := a.Run([]string{"export"}); err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.Contains(out.String(), "vless://22222222-2222-4222-8222-222222222222@vpn.example.com:443") {
		t.Fatalf("export output = %q, want tablet vless link", out.String())
	}
}

func TestHelpUsesXCTLName(t *testing.T) {
	var out bytes.Buffer
	a := New(Config{Out: &out})

	if err := a.Run([]string{"--help"}); err != nil {
		t.Fatalf("help: %v", err)
	}

	if !strings.Contains(out.String(), "xctl") {
		t.Fatalf("help output = %q, want xctl", out.String())
	}
	if strings.Contains(out.String(), "xray-fast-deploy") {
		t.Fatalf("help output = %q, should not use old binary name", out.String())
	}
}

func TestCLIServerAddressUpdatesInfo(t *testing.T) {
	dir := t.TempDir()
	infoPath := filepath.Join(dir, "server.info")
	var out bytes.Buffer
	a := New(Config{InfoPath: infoPath, Out: &out})

	if err := a.Run([]string{"server-address", "--address", "vpn.example.com"}); err != nil {
		t.Fatalf("server-address: %v", err)
	}

	info, err := serverinfo.Load(infoPath)
	if err != nil {
		t.Fatalf("load info: %v", err)
	}
	if info.Address != "vpn.example.com" {
		t.Fatalf("address = %q, want vpn.example.com", info.Address)
	}
}

type FakeRunner struct{}

func (FakeRunner) TestConfig(string) error { return nil }
func (FakeRunner) RestartService() error   { return nil }
