package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPortainerInstallDeniedBeforeAnySideEffect(t *testing.T) {
	restoreDir := withTempToolsDir(t)
	defer restoreDir()

	sentinelDir := t.TempDir()
	sentinel := filepath.Join(sentinelDir, "docker-called")
	binDir := t.TempDir()
	dockerPath := filepath.Join(binDir, "docker")
	if err := os.WriteFile(dockerPath, []byte("#!/bin/sh\ntouch \""+sentinel+"\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	req := httptest.NewRequest(http.MethodPost, "/tools/portainer/install", nil)
	rr := httptest.NewRecorder()
	handleToolInstall(rr, req, "portainer")

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
	var payload installResult
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.OK || !strings.Contains(strings.ToLower(payload.Message), "disabled") {
		t.Fatalf("unexpected response: %#v", payload)
	}
	if _, err := os.Stat(filepath.Join(toolsBaseDir, "portainer")); !os.IsNotExist(err) {
		t.Fatalf("denied install staged files: %v", err)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("denied install executed Docker: %v", err)
	}
}

func TestPortainerInstallPolicyDoesNotAffectStatus(t *testing.T) {
	restoreDir := withTempToolsDir(t)
	defer restoreDir()

	req := httptest.NewRequest(http.MethodGet, "/tools/portainer/status", nil)
	rr := httptest.NewRecorder()
	handleToolStatus(rr, req, "portainer")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status compatibility, got %d: %s", rr.Code, rr.Body.String())
	}
}
