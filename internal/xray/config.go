package xray

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const VisionFlow = "xtls-rprx-vision"

type Config struct {
	Log       LogConfig        `json:"log"`
	Inbounds  []Inbound        `json:"inbounds"`
	Outbounds []map[string]any `json:"outbounds"`
}

type LogConfig struct {
	LogLevel string `json:"loglevel"`
}

type Inbound struct {
	Port           int             `json:"port"`
	Protocol       string          `json:"protocol"`
	Settings       InboundSettings `json:"settings"`
	StreamSettings StreamSettings  `json:"streamSettings"`
	Sniffing       Sniffing        `json:"sniffing"`
}

type InboundSettings struct {
	Clients    []Client `json:"clients"`
	Decryption string   `json:"decryption"`
}

type Client struct {
	ID    string `json:"id"`
	Flow  string `json:"flow"`
	Email string `json:"email"`
}

type StreamSettings struct {
	Network         string          `json:"network"`
	Security        string          `json:"security"`
	RealitySettings RealitySettings `json:"realitySettings"`
}

type RealitySettings struct {
	Show        bool     `json:"show"`
	Dest        string   `json:"dest"`
	ServerNames []string `json:"serverNames"`
	PrivateKey  string   `json:"privateKey"`
	ShortIDs    []string `json:"shortIds"`
}

type Sniffing struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
}

type ConfigOptions struct {
	UUID       string
	PrivateKey string
	Dest       string
	SNI        string
	Port       int
	ShortID    string
	ClientName string
}

func NewRealityConfig(opts ConfigOptions) *Config {
	if opts.ClientName == "" {
		opts.ClientName = "default"
	}
	return &Config{
		Log: LogConfig{LogLevel: "warning"},
		Inbounds: []Inbound{
			{
				Port:     opts.Port,
				Protocol: "vless",
				Settings: InboundSettings{
					Clients: []Client{
						{
							ID:    opts.UUID,
							Flow:  VisionFlow,
							Email: opts.ClientName,
						},
					},
					Decryption: "none",
				},
				StreamSettings: StreamSettings{
					Network:  "tcp",
					Security: "reality",
					RealitySettings: RealitySettings{
						Show:        false,
						Dest:        opts.Dest,
						ServerNames: []string{opts.SNI},
						PrivateKey:  opts.PrivateKey,
						ShortIDs:    []string{"", opts.ShortID},
					},
				},
				Sniffing: Sniffing{
					Enabled:      true,
					DestOverride: []string{"http", "tls"},
				},
			},
		},
		Outbounds: []map[string]any{
			{"protocol": "freedom", "tag": "direct"},
			{"protocol": "blackhole", "tag": "block"},
		},
	}
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func (c *Config) Clients() []Client {
	if len(c.Inbounds) == 0 {
		return nil
	}
	clients := c.Inbounds[0].Settings.Clients
	out := make([]Client, len(clients))
	copy(out, clients)
	return out
}

// requireInbound guards every method that writes to Inbounds[0]. A config
// deployed by the official Xray installer is `{}`, so Inbounds can be empty
// until `xctl init` has run.
func (c *Config) requireInbound() error {
	if len(c.Inbounds) == 0 {
		return errors.New(`config has no inbounds — run "sudo xctl init" first`)
	}
	return nil
}

func (c *Config) AddClient(client Client) error {
	if err := c.requireInbound(); err != nil {
		return err
	}
	if client.Email == "" {
		return errors.New("client name is required")
	}
	if client.ID == "" {
		return errors.New("client uuid is required")
	}
	if client.Flow == "" {
		client.Flow = VisionFlow
	}
	if c.findClient(client.Email) >= 0 {
		return fmt.Errorf("client %q already exists", client.Email)
	}
	c.Inbounds[0].Settings.Clients = append(c.Inbounds[0].Settings.Clients, client)
	return nil
}

func (c *Config) RemoveClient(name string) error {
	if err := c.requireInbound(); err != nil {
		return err
	}
	clients := c.Inbounds[0].Settings.Clients
	if len(clients) <= 1 {
		return errors.New("cannot remove the last client")
	}
	idx := c.findClient(name)
	if idx < 0 {
		return fmt.Errorf("client %q not found", name)
	}
	c.Inbounds[0].Settings.Clients = append(clients[:idx], clients[idx+1:]...)
	return nil
}

func (c *Config) RenameClient(oldName, newName string) error {
	if err := c.requireInbound(); err != nil {
		return err
	}
	if newName == "" {
		return errors.New("new client name is required")
	}
	if c.findClient(newName) >= 0 {
		return fmt.Errorf("client %q already exists", newName)
	}
	idx := c.findClient(oldName)
	if idx < 0 {
		return fmt.Errorf("client %q not found", oldName)
	}
	c.Inbounds[0].Settings.Clients[idx].Email = newName
	return nil
}

func (c *Config) ResetClientUUID(name, uuid string) error {
	if err := c.requireInbound(); err != nil {
		return err
	}
	if uuid == "" {
		return errors.New("uuid is required")
	}
	idx := c.findClient(name)
	if idx < 0 {
		return fmt.Errorf("client %q not found", name)
	}
	c.Inbounds[0].Settings.Clients[idx].ID = uuid
	return nil
}

func (c *Config) Port() int {
	if len(c.Inbounds) == 0 {
		return 0
	}
	return c.Inbounds[0].Port
}

func (c *Config) SetPort(port int) error {
	if err := c.requireInbound(); err != nil {
		return err
	}
	c.Inbounds[0].Port = port
	return nil
}

func (c *Config) SNI() string {
	if len(c.Inbounds) == 0 || len(c.Inbounds[0].StreamSettings.RealitySettings.ServerNames) == 0 {
		return ""
	}
	return c.Inbounds[0].StreamSettings.RealitySettings.ServerNames[0]
}

func (c *Config) Dest() string {
	if len(c.Inbounds) == 0 {
		return ""
	}
	return c.Inbounds[0].StreamSettings.RealitySettings.Dest
}

func (c *Config) SetDisguise(dest, sni string) error {
	if err := c.requireInbound(); err != nil {
		return err
	}
	c.Inbounds[0].StreamSettings.RealitySettings.Dest = dest
	c.Inbounds[0].StreamSettings.RealitySettings.ServerNames = []string{sni}
	return nil
}

func (c *Config) LogLevel() string {
	return c.Log.LogLevel
}

func (c *Config) SetLogLevel(level string) {
	c.Log.LogLevel = level
}

func (c *Config) ShortID() string {
	if len(c.Inbounds) == 0 {
		return ""
	}
	shortIDs := c.Inbounds[0].StreamSettings.RealitySettings.ShortIDs
	if len(shortIDs) > 1 {
		return shortIDs[1]
	}
	if len(shortIDs) == 1 {
		return shortIDs[0]
	}
	return ""
}

func (c *Config) PrivateKey() string {
	if len(c.Inbounds) == 0 {
		return ""
	}
	return c.Inbounds[0].StreamSettings.RealitySettings.PrivateKey
}

func (c *Config) findClient(name string) int {
	if len(c.Inbounds) == 0 {
		return -1
	}
	for i, client := range c.Inbounds[0].Settings.Clients {
		if client.Email == name {
			return i
		}
	}
	return -1
}
