package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/m0b3u/stackwarden/pkg/tools"
)

func TestUninstallComposeRemovesStagedDir(t *testing.T) {
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
			"docker compose -f " + composeFile + " down": {
				stdout: "Removed\n",
				code:   0,
			},
		},
	}

	result := uninstallTool(context.Background(), tool, runner)
	if !result.Uninstalled {
		t.Fatalf("expected uninstalled true")
	}
	if !result.RemovedStagedDir {
		t.Fatalf("expected staged dir removed")
	}
	if _, err := os.Stat(toolDir); !os.IsNotExist(err) {
		t.Fatalf("expected tool dir removed")
	}
}

func TestUninstallComposeFailureRetainsStagedDir(t *testing.T) {
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
			"docker compose -f " + composeFile + " down": {
				stderr: "failed\n",
				code:   1,
				err:    errors.New("compose down failed"),
			},
		},
	}

	result := uninstallTool(context.Background(), tool, runner)
	if result.Uninstalled {
		t.Fatal("expected uninstall failure")
	}
	if result.RemovedStagedDir {
		t.Fatal("staged directory must remain after failed teardown")
	}
	if _, err := os.Stat(composeFile); err != nil {
		t.Fatalf("compose file must remain available for recovery: %v", err)
	}
}

func TestFindComposeFileRejectsSymlink(t *testing.T) {
	toolDir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "compose.yml")
	if err := os.WriteFile(outside, []byte("services:\n"), 0o600); err != nil {
		t.Fatalf("failed to create outside compose file: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(toolDir, "compose.yml")); err != nil {
		t.Fatalf("failed to create compose symlink: %v", err)
	}

	if path, ok := findComposeFile(toolDir); ok {
		t.Fatalf("expected compose symlink to be rejected, got %q", path)
	}
}

func TestUninstallLinuxCLICommands(t *testing.T) {
	restoreDir := withTempToolsDir(t)
	defer restoreDir()

	tool := tools.Tool{
		ID:          "ddev",
		InstallKind: tools.InstallKindLinuxCLI,
		Uninstall: tools.Uninstall{
			UninstallCmds: [][]string{{"apt-get", "remove", "-y", "ddev"}},
		},
	}

	toolDir := filepath.Join(toolsBaseDir, tool.ID)
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}

	runner := &fakeRunner{
		responses: map[string]runResponse{
			"apt-get remove -y ddev": {
				stdout: "Removed ddev\n",
				code:   0,
			},
		},
	}

	result := uninstallTool(context.Background(), tool, runner)
	if !result.Uninstalled {
		t.Fatalf("expected uninstalled true")
	}
	if len(runner.calls) != 1 || runner.calls[0] != "apt-get remove -y ddev" {
		t.Fatalf("unexpected uninstall calls: %#v", runner.calls)
	}
	if _, err := os.Stat(toolDir); !os.IsNotExist(err) {
		t.Fatalf("expected tool dir removed")
	}
}
