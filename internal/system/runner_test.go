package system

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lonelyrower/xray-fast-deploy/internal/xray"
)

func TestSafeConfigUpdateValidatesBeforeReplaceAndRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := xray.NewRealityConfig(xray.ConfigOptions{
		UUID:       "11111111-1111-4111-8111-111111111111",
		PrivateKey: "private-key",
		Dest:       "www.apple.com:443",
		SNI:        "www.apple.com",
		Port:       443,
		ShortID:    "short-id",
		ClientName: "phone",
	})
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save initial: %v", err)
	}

	runner := &FakeRunner{}
	err := SafeConfigUpdate(path, runner, func(cfg *xray.Config) error {
		return cfg.SetPort(8443)
	})
	if err != nil {
		t.Fatalf("safe update: %v", err)
	}

	updated, err := xray.LoadConfig(path)
	if err != nil {
		t.Fatalf("load updated: %v", err)
	}
	if updated.Port() != 8443 {
		t.Fatalf("port = %d, want 8443", updated.Port())
	}
	if !runner.Tested || !runner.Restarted {
		t.Fatalf("tested=%v restarted=%v, want both true", runner.Tested, runner.Restarted)
	}
}

func TestSafeConfigUpdatePreservesFileMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := xray.NewRealityConfig(xray.ConfigOptions{
		UUID:       "11111111-1111-4111-8111-111111111111",
		PrivateKey: "private-key",
		Dest:       "www.apple.com:443",
		SNI:        "www.apple.com",
		Port:       443,
		ShortID:    "short-id",
		ClientName: "phone",
	})
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save initial: %v", err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatalf("chmod initial: %v", err)
	}

	runner := &FakeRunner{}
	if err := SafeConfigUpdate(path, runner, func(cfg *xray.Config) error {
		return cfg.SetPort(8443)
	}); err != nil {
		t.Fatalf("safe update: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0644 {
		t.Fatalf("mode = %o, want 0644", info.Mode().Perm())
	}
}

func TestSafeConfigUpdateDoesNotReplaceOrRestartWhenValidationFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := xray.NewRealityConfig(xray.ConfigOptions{
		UUID:       "11111111-1111-4111-8111-111111111111",
		PrivateKey: "private-key",
		Dest:       "www.apple.com:443",
		SNI:        "www.apple.com",
		Port:       443,
		ShortID:    "short-id",
		ClientName: "phone",
	})
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save initial: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}

	runner := &FakeRunner{TestErr: errors.New("invalid config")}
	err = SafeConfigUpdate(path, runner, func(cfg *xray.Config) error {
		return cfg.SetPort(8443)
	})
	if err == nil {
		t.Fatal("safe update succeeded, want validation error")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("config changed after failed validation")
	}
	if runner.Restarted {
		t.Fatal("service restarted after failed validation")
	}
}

type FakeRunner struct {
	Tested    bool
	Restarted bool
	TestErr   error
}

func (f *FakeRunner) TestConfig(path string) error {
	f.Tested = true
	return f.TestErr
}

func (f *FakeRunner) RestartService() error {
	f.Restarted = true
	return nil
}
