package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInitialViewShowsDashboard(t *testing.T) {
	model := New(ModelData{
		Service:      "active",
		Version:      "Xray 1.8.0",
		Port:         443,
		SNI:          "www.apple.com",
		Address:      "vpn.example.com",
		ClientCount:  2,
		BBR:          "enabled",
		ConfigStatus: "valid",
	})

	view := model.View()
	for _, want := range []string{"Dashboard", "Service", "vpn.example.com", "Clients"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func TestTabNavigation(t *testing.T) {
	model := New(ModelData{})
	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	m := next.(Model)
	if m.active != 1 {
		t.Fatalf("active tab = %d, want 1", m.active)
	}
	view := m.View()
	if !strings.Contains(view, "Clients") {
		t.Fatalf("clients view missing Clients:\n%s", view)
	}
}
