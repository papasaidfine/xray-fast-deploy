package tui

type Service interface {
	Data() ModelData
	AddClientTUI(name string) error
	RemoveClientTUI(name string) error
	RenameClientTUI(oldName, newName string) error
	ResetUUIDTUI(name string) error
	ChangePortTUI(port int) error
	ChangeDisguiseTUI(domain string) error
	SetServerAddressTUI(address string) error
	TestTUI() error
	RestartTUI() error
	ClientLinkTUI(name string) (string, error)
}
