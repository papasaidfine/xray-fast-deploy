package xray

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestEmptyConfigMutationsReturnError(t *testing.T) {
	cases := []struct {
		name string
		call func(*Config) error
	}{
		{"AddClient", func(c *Config) error {
			return c.AddClient(Client{ID: "11111111-1111-4111-8111-111111111111", Email: "phone"})
		}},
		{"RemoveClient", func(c *Config) error { return c.RemoveClient("phone") }},
		{"RenameClient", func(c *Config) error { return c.RenameClient("phone", "tablet") }},
		{"ResetClientUUID", func(c *Config) error {
			return c.ResetClientUUID("phone", "22222222-2222-4222-8222-222222222222")
		}},
		{"SetPort", func(c *Config) error { return c.SetPort(8443) }},
		{"SetDisguise", func(c *Config) error {
			return c.SetDisguise("www.apple.com:443", "www.apple.com")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call(&Config{})
			if err == nil {
				t.Fatalf("%s on empty config succeeded, want error", tc.name)
			}
			if !strings.Contains(err.Error(), "no inbounds") {
				t.Fatalf("%s error = %q, want mention of missing inbounds", tc.name, err)
			}
		})
	}
}

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

	if err := cfg.SetPort(8443); err != nil {
		t.Fatalf("set port: %v", err)
	}
	if err := cfg.SetDisguise("www.microsoft.com:443", "www.microsoft.com"); err != nil {
		t.Fatalf("set disguise: %v", err)
	}
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
