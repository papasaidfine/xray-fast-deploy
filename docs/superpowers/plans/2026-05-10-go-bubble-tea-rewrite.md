# Go Bubble Tea Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the Bash runtime with a Go single binary that provides CLI commands, a Bubble Tea TUI, tests, and GitHub Actions CI/release.

**Architecture:** The Go binary owns the core logic: structured Xray config edits, client management, VLESS link generation, server metadata, local doctor checks, system command execution, and TUI rendering. Config-changing operations use a safety pipeline: load JSON, write a temporary candidate, run `xray run -test`, atomically replace, then restart through systemd.

**Tech Stack:** Go 1.24+, Bubble Tea/Bubbles/Lip Gloss for TUI, standard library JSON/CLI parsing, GitHub Actions for CI and tagged releases.

---

### Task 1: Go Core Foundation

**Files:**
- Create: `go.mod`
- Create: `internal/xray/config.go`
- Create: `internal/xray/config_test.go`
- Create: `internal/link/link.go`
- Create: `internal/link/link_test.go`
- Create: `internal/serverinfo/serverinfo.go`
- Create: `internal/serverinfo/serverinfo_test.go`

- [ ] Write failing tests for loading/saving Xray config, preserving install client names, adding/renaming/removing/resetting clients, VLESS link encoding, and server info address precedence.
- [ ] Run `go test ./...` and confirm failures are missing packages/types.
- [ ] Implement the minimal Go core to pass these tests.
- [ ] Run `go test ./...` and confirm pass.

### Task 2: Safe Config Operations and Doctor

**Files:**
- Create: `internal/system/runner.go`
- Create: `internal/system/runner_test.go`
- Create: `internal/doctor/doctor.go`
- Create: `internal/doctor/doctor_test.go`
- Modify: `internal/xray/config.go`

- [ ] Write failing tests for candidate config validation before replace/restart and doctor result classification.
- [ ] Run `go test ./...` and confirm failures.
- [ ] Implement command runner abstraction, safe config update pipeline, and doctor checks.
- [ ] Run `go test ./...` and confirm pass.

### Task 3: CLI

**Files:**
- Create: `cmd/xctl/main.go`
- Create: `internal/app/app.go`
- Create: `internal/app/app_test.go`

- [ ] Write failing tests for CLI argument dispatch and non-root-readable pure commands where practical.
- [ ] Run `go test ./...` and confirm failures.
- [ ] Implement commands: `tui`, `status`, `doctor`, `list-clients`, `add-client`, `remove-client`, `rename-client`, `reset-uuid`, `show-client`, `export`, `change-port`, `change-disguise`, `server-address`, `test`, `restart`, `logs`.
- [ ] Run `go test ./...` and `go build ./cmd/xctl`.

### Task 4: Bubble Tea TUI

**Files:**
- Create: `internal/tui/model.go`
- Create: `internal/tui/model_test.go`
- Modify: `cmd/xctl/main.go`

- [ ] Write failing tests for initial view rendering and key navigation.
- [ ] Run `go test ./...` and confirm failures.
- [ ] Implement Dashboard, Clients, Doctor, Logs, and Server Settings views.
- [ ] Run `go test ./...` and `go build ./cmd/xctl`.

### Task 5: Remove Bash Runtime and Update Docs

**Files:**
- Delete: `deploy-xray.sh`
- Delete: `lib/common.sh`
- Delete: `lib/config.sh`
- Delete: `lib/clients.sh`
- Delete: `lib/server.sh`
- Delete: `lib/install.sh`
- Delete: `lib/menu.sh`
- Delete: `tests/reliability_tests.sh`
- Modify: `README.md`
- Modify: `TUI_PLAN.md`

- [ ] Remove Bash runtime files and Bash tests.
- [ ] Update README with Go binary install, CLI, TUI, and release usage.
- [ ] Run `go test ./...` and `go build ./cmd/xctl`.

### Task 6: GitHub Actions CI and Release

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/release.yml`

- [ ] Add CI workflow for `go test ./...` and `go build ./cmd/xctl`.
- [ ] Add release workflow for `v*` tags, building Linux `amd64` and `arm64` binaries.
- [ ] Run local verification: `go test ./...` and `go build ./cmd/xctl`.

### Self-Review

- Covers approved scope: Go core migration, Bubble Tea TUI, Bash removal, CI, release.
- No Bash runtime compatibility path remains.
- Tests are required before implementation for core behavior.
- System-changing operations are behind runner abstractions so unit tests do not mutate the host.
