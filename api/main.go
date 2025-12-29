package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	shared "github.com/m0b3u/stackwarden/pkg"
)

type Health struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type AgentHealth struct {
	OK      bool   `json:"ok"`
	Status  string `json:"status,omitempty"`
	Service string `json:"service,omitempty"`
	Error   string `json:"error,omitempty"`
}

type AuditEvent struct {
	Time   string `json:"time"`
	Action string `json:"action"`
	Result bool   `json:"result"`
}

type auditLog struct {
	mu     sync.Mutex
	events []AuditEvent
}

func (a *auditLog) recordAudit(action string, ok bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.events = append(a.events, AuditEvent{
		Time:   time.Now().UTC().Format(time.RFC3339Nano),
		Action: action,
		Result: ok,
	})

	if len(a.events) > 50 {
		a.events = a.events[len(a.events)-50:]
	}
}

func (a *auditLog) list() []AuditEvent {
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make([]AuditEvent, len(a.events))
	copy(out, a.events)
	return out
}

func main() {
	mux := http.NewServeMux()
	audit := &auditLog{}

	// API health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Health{Status: "ok", Service: "api"})
	})

	// Proxy health check to agent
	mux.HandleFunc("/agent/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		agentURL := getenv("AGENT_URL", "http://127.0.0.1:9091/health")

		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(agentURL)
		if err != nil {
			_ = json.NewEncoder(w).Encode(AgentHealth{
				OK:    false,
				Error: err.Error(),
			})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			_ = json.NewEncoder(w).Encode(AgentHealth{
				OK:    false,
				Error: "agent returned non-200",
			})
			return
		}

		var h Health
		if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
			_ = json.NewEncoder(w).Encode(AgentHealth{
				OK:    false,
				Error: "invalid agent JSON: " + err.Error(),
			})
			return
		}

		_ = json.NewEncoder(w).Encode(AgentHealth{
			OK:      true,
			Status:  h.Status,
			Service: h.Service,
		})
	})

	mux.HandleFunc("/ports", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ok := false
		defer func() {
			audit.recordAudit("ports.read", ok)
		}()

		agentBase := getenv("AGENT_BASE", "http://127.0.0.1:9091")
		agentURL := agentBase + "/ports"

		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(agentURL)
		if err != nil {
			_ = json.NewEncoder(w).Encode(shared.OperationResult{OK: false, Error: err.Error()})
			return
		}
		defer resp.Body.Close()

		w.WriteHeader(resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			_ = json.NewEncoder(w).Encode(shared.OperationResult{OK: false, Error: "failed to read agent response: " + err.Error()})
			return
		}

		if len(body) == 0 {
			_ = json.NewEncoder(w).Encode(shared.OperationResult{OK: false, Error: "empty response from agent"})
			return
		}

		ok = resp.StatusCode >= 200 && resp.StatusCode < 300
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})

	mux.HandleFunc("/audit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		events := audit.list()
		_ = json.NewEncoder(w).Encode(events)
	})

	// Serve UI (static)
	uiDir := filepath.Clean(filepath.Join("..", "ui"))
	fs := http.FileServer(http.Dir(uiDir))
	mux.Handle("/", fs)

	addr := "127.0.0.1:8080"
	log.Printf("api listening on http://%s", addr)
	log.Printf("serving ui from %s", uiDir)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
