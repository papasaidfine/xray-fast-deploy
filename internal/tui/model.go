package tui

import (
	"fmt"
	"strconv"
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

type mode int

const (
	modeNormal mode = iota
	modeInput
	modeConfirm
	modeBusy
	modeShowLink
)

type pendingKind int

const (
	pendingNone pendingKind = iota
	pendingAddClient
	pendingRemoveClient
	pendingRenameClientOld
	pendingRenameClientNew
	pendingResetUUID
	pendingShowLink
	pendingChangePort
	pendingChangeDisguise
	pendingServerAddress
	pendingRestart
)

type Model struct {
	active  int
	data    ModelData
	svc     Service
	cursor  int
	mode    mode
	pending pendingKind

	prompt string
	input  string

	confirmMsg string

	flash    string
	flashErr bool

	linkName string
	linkText string

	renameOld string
}

var tabs = []string{"Dashboard", "Clients", "Doctor", "Logs", "Server"}

type actionResultMsg struct {
	err   error
	flash string
}

type linkResultMsg struct {
	name string
	link string
	err  error
}

func New(svc Service) Model {
	return Model{svc: svc, data: svc.Data()}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actionResultMsg:
		m.mode = modeNormal
		m.pending = pendingNone
		if msg.err != nil {
			m.flash = "error: " + msg.err.Error()
			m.flashErr = true
		} else {
			m.flash = msg.flash
			m.flashErr = false
		}
		m.data = m.svc.Data()
		if m.cursor >= len(m.data.Clients) && m.cursor > 0 {
			m.cursor = len(m.data.Clients) - 1
		}
		return m, nil
	case linkResultMsg:
		if msg.err != nil {
			m.mode = modeNormal
			m.pending = pendingNone
			m.flash = "error: " + msg.err.Error()
			m.flashErr = true
			return m, nil
		}
		m.mode = modeShowLink
		m.pending = pendingNone
		m.linkName = msg.name
		m.linkText = msg.link
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	switch m.mode {
	case modeInput:
		return m.handleInputKey(key)
	case modeConfirm:
		return m.handleConfirmKey(key)
	case modeShowLink:
		m.mode = modeNormal
		m.linkName = ""
		m.linkText = ""
		return m, nil
	case modeBusy:
		return m, nil
	}

	if key.Type == tea.KeyEsc {
		return m, tea.Quit
	}
	if key.Type == tea.KeyRunes && len(key.Runes) == 1 && key.Runes[0] == 'q' {
		return m, tea.Quit
	}

	m.flash = ""
	m.flashErr = false

	switch key.Type {
	case tea.KeyTab, tea.KeyRight:
		m.active = (m.active + 1) % len(tabs)
		m.cursor = 0
		return m, nil
	case tea.KeyShiftTab, tea.KeyLeft:
		m.active--
		if m.active < 0 {
			m.active = len(tabs) - 1
		}
		m.cursor = 0
		return m, nil
	case tea.KeyUp:
		if tabs[m.active] == "Clients" && m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case tea.KeyDown:
		if tabs[m.active] == "Clients" && m.cursor < len(m.data.Clients)-1 {
			m.cursor++
		}
		return m, nil
	}

	if key.Type != tea.KeyRunes || len(key.Runes) != 1 {
		return m, nil
	}
	r := key.Runes[0]

	switch tabs[m.active] {
	case "Dashboard", "Doctor", "Logs":
		if r == 'r' {
			m.data = m.svc.Data()
			m.flash = "refreshed"
		}
	case "Clients":
		switch r {
		case 'r':
			m.data = m.svc.Data()
			m.flash = "refreshed"
		case 'a':
			m.startInput(pendingAddClient, "new client name: ")
		case 'd':
			if name := m.selectedClient(); name != "" {
				m.startConfirm(pendingRemoveClient, fmt.Sprintf("delete client %q? (y/N): ", name))
			}
		case 'R':
			if name := m.selectedClient(); name != "" {
				m.renameOld = name
				m.startInput(pendingRenameClientNew, fmt.Sprintf("new name for %q: ", name))
			}
		case 'u':
			if name := m.selectedClient(); name != "" {
				m.startConfirm(pendingResetUUID, fmt.Sprintf("reset UUID for %q? (y/N): ", name))
			}
		case 's', '\r':
			if name := m.selectedClient(); name != "" {
				m.pending = pendingShowLink
				m.mode = modeBusy
				return m, m.fetchLinkCmd(name)
			}
		}
	case "Server":
		switch r {
		case 'r':
			m.data = m.svc.Data()
			m.flash = "refreshed"
		case 'p':
			m.startInput(pendingChangePort, fmt.Sprintf("new port (current %d): ", m.data.Port))
		case 'D':
			m.startInput(pendingChangeDisguise, fmt.Sprintf("disguise domain (current %s): ", m.data.SNI))
		case 'A':
			m.startInput(pendingServerAddress, fmt.Sprintf("server address (current %s): ", m.data.Address))
		case 't':
			m.mode = modeBusy
			m.pending = pendingNone
			return m, m.runCmd(m.svc.TestTUI, "config test passed")
		case 'X':
			m.startConfirm(pendingRestart, "restart xray service? (y/N): ")
		}
	}
	return m, nil
}

func (m *Model) startInput(p pendingKind, prompt string) {
	m.mode = modeInput
	m.pending = p
	m.prompt = prompt
	m.input = ""
}

func (m *Model) startConfirm(p pendingKind, msg string) {
	m.mode = modeConfirm
	m.pending = p
	m.confirmMsg = msg
}

func (m Model) handleInputKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc:
		m.mode = modeNormal
		m.pending = pendingNone
		m.input = ""
		m.renameOld = ""
		return m, nil
	case tea.KeyEnter:
		return m.submitInput()
	case tea.KeyBackspace:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
		return m, nil
	case tea.KeySpace:
		m.input += " "
		return m, nil
	case tea.KeyRunes:
		m.input += string(key.Runes)
		return m, nil
	}
	return m, nil
}

func (m Model) submitInput() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.input)
	pending := m.pending
	old := m.renameOld
	m.input = ""
	if value == "" {
		m.mode = modeNormal
		m.pending = pendingNone
		m.renameOld = ""
		return m, nil
	}
	m.mode = modeBusy
	switch pending {
	case pendingAddClient:
		return m, m.runCmd(func() error { return m.svc.AddClientTUI(value) }, "client added")
	case pendingRenameClientNew:
		m.renameOld = ""
		return m, m.runCmd(func() error { return m.svc.RenameClientTUI(old, value) }, "client renamed")
	case pendingChangePort:
		port, err := strconv.Atoi(value)
		if err != nil {
			return m, instantErr(fmt.Errorf("invalid port %q", value))
		}
		return m, m.runCmd(func() error { return m.svc.ChangePortTUI(port) }, "port changed")
	case pendingChangeDisguise:
		return m, m.runCmd(func() error { return m.svc.ChangeDisguiseTUI(value) }, "disguise changed")
	case pendingServerAddress:
		return m, m.runCmd(func() error { return m.svc.SetServerAddressTUI(value) }, "address saved")
	}
	m.mode = modeNormal
	m.pending = pendingNone
	return m, nil
}

func (m Model) handleConfirmKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Type == tea.KeyEsc {
		m.mode = modeNormal
		m.pending = pendingNone
		return m, nil
	}
	if key.Type != tea.KeyRunes || len(key.Runes) != 1 {
		return m, nil
	}
	r := key.Runes[0]
	if r != 'y' && r != 'Y' {
		m.mode = modeNormal
		m.pending = pendingNone
		return m, nil
	}
	pending := m.pending
	name := m.selectedClient()
	m.mode = modeBusy
	switch pending {
	case pendingRemoveClient:
		return m, m.runCmd(func() error { return m.svc.RemoveClientTUI(name) }, "client removed")
	case pendingResetUUID:
		return m, m.runCmd(func() error { return m.svc.ResetUUIDTUI(name) }, "uuid reset")
	case pendingRestart:
		return m, m.runCmd(m.svc.RestartTUI, "xray restarted")
	}
	m.mode = modeNormal
	m.pending = pendingNone
	return m, nil
}

func (m Model) runCmd(fn func() error, success string) tea.Cmd {
	return func() tea.Msg {
		return actionResultMsg{err: fn(), flash: success}
	}
}

func instantErr(err error) tea.Cmd {
	return func() tea.Msg { return actionResultMsg{err: err} }
}

func (m Model) fetchLinkCmd(name string) tea.Cmd {
	return func() tea.Msg {
		link, err := m.svc.ClientLinkTUI(name)
		return linkResultMsg{name: name, link: link, err: err}
	}
}

func (m Model) selectedClient() string {
	if m.cursor < 0 || m.cursor >= len(m.data.Clients) {
		return ""
	}
	return m.data.Clients[m.cursor].Name
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
	b.WriteString("\n\n")
	b.WriteString(m.footer())
	return b.String()
}

func (m Model) footer() string {
	if m.mode == modeShowLink {
		return fmt.Sprintf("%s\n\n%s\n\nPress any key to continue.", m.linkName, m.linkText)
	}
	if m.mode == modeInput {
		return m.prompt + m.input + "_  (Enter: submit  Esc: cancel)"
	}
	if m.mode == modeConfirm {
		return m.confirmMsg
	}
	if m.mode == modeBusy {
		return "working..."
	}

	help := m.tabHelp()
	if m.flash != "" {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		if m.flashErr {
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		}
		return style.Render(m.flash) + "\n" + help
	}
	return help
}

func (m Model) tabHelp() string {
	common := "Tab/←→: switch  q: quit"
	switch tabs[m.active] {
	case "Clients":
		return "↑↓: select  a: add  d: delete  R: rename  u: reset-uuid  s: show-link  r: refresh  " + common
	case "Server":
		return "p: port  D: disguise  A: address  t: test  X: restart  r: refresh  " + common
	default:
		return "r: refresh  " + common
	}
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
		b.WriteString("No clients. Press 'a' to add one.\n")
		return b.String()
	}
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	for i, client := range m.data.Clients {
		marker := "  "
		if i == m.cursor {
			marker = cursorStyle.Render("> ")
		}
		fmt.Fprintf(&b, "%s%-20s %s\n", marker, client.Name, client.UUID)
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
Saved Address: %s`,
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
