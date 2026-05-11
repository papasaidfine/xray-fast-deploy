package serverinfo

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadAndAddressPrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.info")
	info := Info{
		PublicKey: "public-key",
		Port:      443,
		SNI:       "www.apple.com",
		Address:   "vpn.example.com",
		Created:   "2026-05-10 00:00:00",
	}

	if err := Save(path, info); err != nil {
		t.Fatalf("save info: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load info: %v", err)
	}

	if loaded.Address != "vpn.example.com" {
		t.Fatalf("address = %q, want vpn.example.com", loaded.Address)
	}

	if got := ResolveAddress(loaded, func() (string, error) { return "203.0.113.10", nil }); got != "vpn.example.com" {
		t.Fatalf("resolved address = %q, want saved address", got)
	}
}

func TestResolveAddressFallsBackToDetector(t *testing.T) {
	got := ResolveAddress(Info{}, func() (string, error) { return "203.0.113.10", nil })
	if got != "203.0.113.10" {
		t.Fatalf("resolved address = %q, want detected ip", got)
	}
}
