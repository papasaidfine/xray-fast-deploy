package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ModelData struct {
	Service      string
	Version      string
	Port         int
	SNI          string
	Address      string
	ClientCount  int
	BBR          string
	ConfigStatus string
	Clients      []Client
	Doctor       []string
	Logs         []string
}

type Client struct {
	Name string
	UUID string
}

type Model struct {
	active int
	data   ModelData
}

var tabs = []string{"Dashboard", "Clients", "Doctor", "Logs", "Server"}

func New(data ModelData) Model {
	return Model{data: data}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyTab, tea.KeyRight:
		m.active = (m.active + 1) % len(tabs)
	case tea.KeyShiftTab, tea.KeyLeft:
		m.active--
		if m.active < 0 {
			m.active = len(tabs) - 1
		}
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")
	switch tabs[m.active] {
	case "Dashboard":
		b.WriteString(m.dashboard())
	case "Clients":
		b.WriteString(m.clients())
	case "Doctor":
		b.WriteString(m.doctor())
	case "Logs":
		b.WriteString(m.logs())
	case "Server":
		b.WriteString(m.server())
	}
	b.WriteString("\n\nTab/Arrow: switch  Esc/Ctrl+C: quit\n")
	return b.String()
}

func (m Model) renderTabs() string {
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("62")).Padding(0, 1)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1)
	parts := make([]string, 0, len(tabs))
	for i, tab := range tabs {
		if i == m.active {
			parts = append(parts, activeStyle.Render(tab))
		} else {
			parts = append(parts, inactiveStyle.Render(tab))
		}
	}
	return strings.Join(parts, " ")
}

func (m Model) dashboard() string {
	return fmt.Sprintf(`Dashboard

Service: %s
Xray Version: %s
Port: %d
SNI: %s
Saved Address: %s
Clients: %d
BBR: %s
Config Test: %s`,
		value(m.data.Service),
		value(m.data.Version),
		m.data.Port,
		value(m.data.SNI),
		value(m.data.Address),
		m.data.ClientCount,
		value(m.data.BBR),
		value(m.data.ConfigStatus),
	)
}

func (m Model) clients() string {
	var b strings.Builder
	b.WriteString("Clients\n\n")
	if len(m.data.Clients) == 0 {
		b.WriteString("No clients loaded.\n")
		return b.String()
	}
	for _, client := range m.data.Clients {
		fmt.Fprintf(&b, "%s\t%s\n", client.Name, client.UUID)
	}
	return b.String()
}

func (m Model) doctor() string {
	if len(m.data.Doctor) == 0 {
		return "Doctor\n\nRun `xctl doctor` for full diagnostics with advice."
	}
	return "Doctor\n\n" + strings.Join(m.data.Doctor, "\n")
}

func (m Model) logs() string {
	if len(m.data.Logs) == 0 {
		return "Logs\n\nRun `xctl logs` for systemd logs."
	}
	return "Logs\n\n" + strings.Join(m.data.Logs, "\n")
}

func (m Model) server() string {
	return fmt.Sprintf(`Server Settings

Port: %d
SNI: %s
Saved Address: %s

Use CLI commands for mutations:
  change-port
  change-disguise
  server-address
  test
  restart`,
		m.data.Port,
		value(m.data.SNI),
		value(m.data.Address),
	)
}

func value(v string) string {
	if v == "" {
		return "unknown"
	}
	return v
}
