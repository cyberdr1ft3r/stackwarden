package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m0b3u/stackwarden/pkg/tools"
)

type runResponse struct {
	stdout string
	stderr string
	code   int
	err    error
}

type fakeRunner struct {
	responses map[string]runResponse
	calls     []string
}

func (f *fakeRunner) Run(ctx context.Context, cmd []string, workdir string) (string, string, int, error) {
	key := strings.Join(cmd, " ")
	f.calls = append(f.calls, key)
	resp, ok := f.responses[key]
	if !ok {
		return "", "", -1, errors.New("unexpected command: " + key)
	}
	return resp.stdout, resp.stderr, resp.code, resp.err
}

func addDummyBinary(t *testing.T, name string) func() {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("failed to write dummy binary: %v", err)
	}
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	return func() {
		_ = os.Setenv("PATH", oldPath)
	}
}

func withTempToolsDir(t *testing.T) func() {
	t.Helper()
	old := toolsBaseDir
	dir := t.TempDir()
	toolsBaseDir = dir
	return func() {
		toolsBaseDir = old
	}
}

func TestComposeStatusRunning(t *testing.T) {
	restoreDir := withTempToolsDir(t)
	defer restoreDir()
	restorePath := addDummyBinary(t, "docker")
	defer restorePath()

	tool := tools.Tool{ID: "portainer", InstallKind: tools.InstallKindCompose}
	toolDir := filepath.Join(toolsBaseDir, tool.ID)
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}

	composeFile := filepath.Join(toolDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte("services:\n"), 0o644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	runner := &fakeRunner{
		responses: map[string]runResponse{
			"docker compose -f " + composeFile + " ps": {
				stdout: "NAME COMMAND STATE\nportainer Up 3 minutes\n",
				code:   0,
			},
		},
	}

	status := collectToolStatus(context.Background(), tool, runner)
	if !status.Installed {
		t.Fatalf("expected installed true")
	}
	if !status.Running {
		t.Fatalf("expected running true")
	}
	if status.Details == nil || status.Details.Compose == nil {
		t.Fatalf("expected compose details")
	}
	if len(status.Details.Compose.Containers) != 1 || status.Details.Compose.Containers[0] != "portainer" {
		t.Fatalf("unexpected containers: %#v", status.Details.Compose.Containers)
	}
}

func TestLinuxCLIStatusVersion(t *testing.T) {
	restoreDir := withTempToolsDir(t)
	defer restoreDir()

	tool := tools.Tool{
		ID:          "ddev",
		InstallKind: tools.InstallKindLinuxCLI,
		Status: tools.StatusSpec{
			Binary:       "ddev",
			VersionCmd:   []string{"ddev", "version"},
			VersionRegex: `v?(\d+\.\d+\.\d+)`,
		},
	}

	runner := &fakeRunner{
		responses: map[string]runResponse{
			"which ddev": {
				stdout: "/usr/bin/ddev\n",
				code:   0,
			},
			"ddev version": {
				stdout: "DDEV version v1.23.4\n",
				code:   0,
			},
		},
	}

	status := collectToolStatus(context.Background(), tool, runner)
	if !status.Installed {
		t.Fatalf("expected installed true")
	}
	if status.Version != "1.23.4" {
		t.Fatalf("unexpected version: %s", status.Version)
	}
	if status.Details == nil || status.Details.CLI == nil || status.Details.CLI.Path != "/usr/bin/ddev" {
		t.Fatalf("expected cli details with path")
	}
}

func TestBundleOnlyStatus(t *testing.T) {
	restoreDir := withTempToolsDir(t)
	defer restoreDir()

	tool := tools.Tool{ID: "bundle-only", InstallKind: tools.InstallKindBundleOnly}
	toolDir := filepath.Join(toolsBaseDir, tool.ID)
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}

	status := collectToolStatus(context.Background(), tool, &fakeRunner{responses: map[string]runResponse{}})
	if !status.Staged || !status.Installed {
		t.Fatalf("expected staged/installed true")
	}
}
