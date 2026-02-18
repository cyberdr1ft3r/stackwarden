package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Health struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Health{Status: "ok", Service: "agent"})
	})

	mux.HandleFunc("/ports", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		resp := collectPorts(ctx)
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		resp := collectMetrics(ctx)
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/tools/", toolActionHandler)

	socketPath := getenv("AGENT_SOCKET", "/run/stackwarden/agent.sock")
	ln, err := listenUnixSocket(socketPath)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	log.Printf("agent listening on unix socket %s", socketPath)
	log.Fatal(http.Serve(ln, mux))
}

func listenUnixSocket(socketPath string) (net.Listener, error) {
	if socketPath == "" {
		return nil, errors.New("agent socket path cannot be empty")
	}
	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	_ = os.Remove(socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(socketPath, 0o660); err != nil {
		_ = ln.Close()
		return nil, err
	}
	return ln, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
