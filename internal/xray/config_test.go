package xray

import (
	"path/filepath"
	"testing"
)

func TestConfigClientLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := NewRealityConfig(ConfigOptions{
		UUID:       "11111111-1111-4111-8111-111111111111",
		PrivateKey: "private-key",
		Dest:       "www.apple.com:443",
		SNI:        "www.apple.com",
		Port:       443,
		ShortID:    "short-id",
		ClientName: "phone",
	})

	if got := cfg.Clients()[0].Email; got != "phone" {
		t.Fatalf("initial client name = %q, want phone", got)
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("save config: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if err := loaded.AddClient(Client{ID: "22222222-2222-4222-8222-222222222222", Email: `phone "alpha"`}); err != nil {
		t.Fatalf("add client: %v", err)
	}
	if err := loaded.RenameClient(`phone "alpha"`, "tablet"); err != nil {
		t.Fatalf("rename client: %v", err)
	}
	if err := loaded.ResetClientUUID("tablet", "33333333-3333-4333-8333-333333333333"); err != nil {
		t.Fatalf("reset uuid: %v", err)
	}
	if err := loaded.RemoveClient("phone"); err != nil {
		t.Fatalf("remove original client: %v", err)
	}

	clients := loaded.Clients()
	if len(clients) != 1 {
		t.Fatalf("client count = %d, want 1", len(clients))
	}
	if clients[0].Email != "tablet" || clients[0].ID != "33333333-3333-4333-8333-333333333333" {
		t.Fatalf("client = %+v, want renamed tablet with reset UUID", clients[0])
	}
}

func TestConfigRejectsDuplicateAndLastClientRemoval(t *testing.T) {
	cfg := NewRealityConfig(ConfigOptions{
		UUID:       "11111111-1111-4111-8111-111111111111",
		PrivateKey: "private-key",
		Dest:       "www.apple.com:443",
		SNI:        "www.apple.com",
		Port:       443,
		ShortID:    "short-id",
		ClientName: "phone",
	})

	if err := cfg.AddClient(Client{ID: "22222222-2222-4222-8222-222222222222", Email: "phone"}); err == nil {
		t.Fatal("AddClient duplicate succeeded, want error")
	}
	if err := cfg.RemoveClient("phone"); err == nil {
		t.Fatal("RemoveClient last client succeeded, want error")
	}
}

func TestServerSettings(t *testing.T) {
	cfg := NewRealityConfig(ConfigOptions{
		UUID:       "11111111-1111-4111-8111-111111111111",
		PrivateKey: "private-key",
		Dest:       "www.apple.com:443",
		SNI:        "www.apple.com",
		Port:       443,
		ShortID:    "short-id",
		ClientName: "phone",
	})

	cfg.SetPort(8443)
	cfg.SetDisguise("www.microsoft.com:443", "www.microsoft.com")
	cfg.SetLogLevel("info")

	if cfg.Port() != 8443 {
		t.Fatalf("port = %d, want 8443", cfg.Port())
	}
	if cfg.SNI() != "www.microsoft.com" {
		t.Fatalf("sni = %q, want www.microsoft.com", cfg.SNI())
	}
	if cfg.LogLevel() != "info" {
		t.Fatalf("log level = %q, want info", cfg.LogLevel())
	}
}
