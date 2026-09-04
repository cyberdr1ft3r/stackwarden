package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestListenUnixSocketCreatesSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix sockets are verified on Unix-like hosts")
	}
	socket := filepath.Join(t.TempDir(), "agent.sock")
	ln, err := listenUnixSocket(socket)
	if err != nil {
		t.Fatalf("listenUnixSocket returned error: %v", err)
	}
	defer ln.Close()

	info, err := os.Stat(socket)
	if err != nil {
		t.Fatalf("expected socket file to exist: %v", err)
	}
	if got := info.Mode() & 0o777; got != 0o660 {
		t.Fatalf("expected socket mode 0660, got %03o", got)
	}
	if dirInfo, err := os.Stat(filepath.Dir(socket)); err != nil {
		t.Fatalf("expected socket dir to exist: %v", err)
	} else if got := dirInfo.Mode() & 0o777; got != 0o750 {
		t.Fatalf("expected socket dir mode 0750, got %03o", got)
	}
}

func TestListenUnixSocketRefusesNonSocketStalePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix sockets are verified on Unix-like hosts")
	}
	socket := filepath.Join(t.TempDir(), "agent.sock")
	if err := os.WriteFile(socket, []byte("not a socket"), 0o644); err != nil {
		t.Fatalf("failed to write stale file: %v", err)
	}

	ln, err := listenUnixSocket(socket)
	if err == nil {
		_ = ln.Close()
		t.Fatal("expected non-socket stale path to be rejected")
	}
	if _, statErr := os.Stat(socket); statErr != nil {
		t.Fatalf("expected stale non-socket file to remain: %v", statErr)
	}
}
