package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListenUnixSocketCreatesSocket(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "agent.sock")
	ln, err := listenUnixSocket(socket)
	if err != nil {
		t.Fatalf("listenUnixSocket returned error: %v", err)
	}
	defer ln.Close()

	if _, err := os.Stat(socket); err != nil {
		t.Fatalf("expected socket file to exist: %v", err)
	}
}
