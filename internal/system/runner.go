package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lonelyrower/xray-fast-deploy/internal/xray"
)

type ConfigRunner interface {
	TestConfig(path string) error
	RestartService() error
}

type Runner struct {
	XrayPath string
	Service  string
}

func (r Runner) TestConfig(path string) error {
	xrayPath := r.XrayPath
	if xrayPath == "" {
		xrayPath = "/usr/local/bin/xray"
	}
	cmd := exec.Command(xrayPath, "run", "-test", "-config", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("xray config test failed: %w: %s", err, string(out))
	}
	return nil
}

func (r Runner) RestartService() error {
	service := r.Service
	if service == "" {
		service = "xray"
	}
	cmd := exec.Command("systemctl", "restart", service)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restart %s failed: %w: %s", service, err, string(out))
	}
	return nil
}

func SafeConfigUpdate(path string, runner ConfigRunner, mutate func(*xray.Config) error) error {
	cfg, err := xray.LoadConfig(path)
	if err != nil {
		return err
	}
	if err := mutate(cfg); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	candidate, err := os.CreateTemp(dir, ".config-*.json")
	if err != nil {
		return err
	}
	candidatePath := candidate.Name()
	if err := candidate.Close(); err != nil {
		return err
	}
	defer os.Remove(candidatePath)

	if err := cfg.Save(candidatePath); err != nil {
		return err
	}
	if err := runner.TestConfig(candidatePath); err != nil {
		return err
	}
	if err := os.Rename(candidatePath, path); err != nil {
		return err
	}
	if err := runner.RestartService(); err != nil {
		return err
	}
	return nil
}
