package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type fakeService struct {
	data         ModelData
	addCalled    string
	removeCalled string
	renamePair   [2]string
	resetCalled  string
	portCalled   int
	linkResult   string
	linkErr      error
	addErr       error
	removeErr    error
}

func (f *fakeService) Data() ModelData { return f.data }
func (f *fakeService) AddClientTUI(name string) error {
	f.addCalled = name
	return f.addErr
}
func (f *fakeService) RemoveClientTUI(name string) error {
	f.removeCalled = name
	return f.removeErr
}
func (f *fakeService) RenameClientTUI(oldName, newName string) error {
	f.renamePair = [2]string{oldName, newName}
	return nil
}
func (f *fakeService) ResetUUIDTUI(name string) error {
	f.resetCalled = name
	return nil
}
func (f *fakeService) ChangePortTUI(port int) error {
	f.portCalled = port
	return nil
}
func (f *fakeService) ChangeDisguiseTUI(string) error   { return nil }
func (f *fakeService) SetServerAddressTUI(string) error { return nil }
func (f *fakeService) TestTUI() error                   { return nil }
func (f *fakeService) RestartTUI() error                { return nil }
func (f *fakeService) ClientLinkTUI(name string) (string, error) {
	return f.linkResult, f.linkErr
}
func (f *fakeService) BBREnableTUI() error      { return nil }
func (f *fakeService) BBRDisableTUI() error     { return nil }
func (f *fakeService) ForwardEnableTUI() error  { return nil }
func (f *fakeService) ForwardDisableTUI() error { return nil }
func (f *fakeService) FirewallOpenTUI() error   { return nil }
func (f *fakeService) FirewallCloseTUI() error  { return nil }
func (f *fakeService) FixPermsTUI() error       { return nil }
func (f *fakeService) CheckUpdateTUI() (string, string) {
	return "dev", "dev"
}

func newModel(svc Service) Model { return New(svc) }

func TestInitialViewShowsDashboard(t *testing.T) {
	svc := &fakeService{data: ModelData{
		Service:      "active",
		Version:      "Xray 1.8.0",
		Port:         443,
		SNI:          "www.apple.com",
		Address:      "vpn.example.com",
		ClientCount:  2,
		BBR:          "enabled",
		ConfigStatus: "valid",
	}}

	view := newModel(svc).View()
	for _, want := range []string{"Dashboard", "Service", "vpn.example.com", "Clients"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func TestDashboardShowsNotInitializedHint(t *testing.T) {
	svc := &fakeService{data: ModelData{NotInitialized: true}}
	view := newModel(svc).View()
	if !strings.Contains(view, "config not initialized — run: sudo xctl init") {
		t.Fatalf("view missing not-initialized hint:\n%s", view)
	}
}

func TestTabNavigation(t *testing.T) {
	svc := &fakeService{}
	next, _ := newModel(svc).Update(tea.KeyMsg{Type: tea.KeyTab})
	m := next.(Model)
	if m.active != 1 {
		t.Fatalf("active tab = %d, want 1", m.active)
	}
	if !strings.Contains(m.View(), "Clients") {
		t.Fatalf("clients view missing Clients:\n%s", m.View())
	}
}

func TestAddClientFlow(t *testing.T) {
	svc := &fakeService{data: ModelData{Clients: []Client{{Name: "phone", UUID: "u"}}}}
	m := newModel(svc)

	step := func(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return model.Update(msg)
	}

	model, _ := step(m, tea.KeyMsg{Type: tea.KeyTab})
	model, _ = step(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if model.(Model).mode != modeInput {
		t.Fatalf("expected input mode after 'a'")
	}
	model, _ = step(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t', 'a', 'b'}})
	model, _ = step(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l', 'e', 't'}})
	model, cmd := step(model, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a tea.Cmd from Enter")
	}
	msg := cmd()
	model, _ = step(model, msg)
	if svc.addCalled != "tablet" {
		t.Fatalf("AddClientTUI got %q, want %q", svc.addCalled, "tablet")
	}
	if !strings.Contains(model.(Model).flash, "client added") {
		t.Fatalf("missing flash, got %q", model.(Model).flash)
	}
}

func TestDeleteClientRequiresConfirm(t *testing.T) {
	svc := &fakeService{data: ModelData{Clients: []Client{{Name: "phone", UUID: "u"}}}}
	m := newModel(svc)
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if model.(Model).mode != modeConfirm {
		t.Fatalf("expected confirm mode")
	}
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if svc.removeCalled != "" {
		t.Fatalf("RemoveClientTUI should not have been called after 'n'")
	}
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected delete cmd after 'y'")
	}
	cmd()
	if svc.removeCalled != "phone" {
		t.Fatalf("RemoveClientTUI got %q, want phone", svc.removeCalled)
	}
	_ = model
}

func TestActionErrorShowsFlash(t *testing.T) {
	svc := &fakeService{
		data:   ModelData{Clients: []Client{{Name: "phone"}}},
		addErr: errors.New("boom"),
	}
	m := newModel(svc)
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model, _ = model.Update(cmd())
	if !model.(Model).flashErr {
		t.Fatalf("expected flashErr true")
	}
	if !strings.Contains(model.(Model).flash, "boom") {
		t.Fatalf("flash missing error: %q", model.(Model).flash)
	}
}
