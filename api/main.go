package main

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
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

type Tool struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type VersionInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
	BuiltAt string `json:"builtAt,omitempty"`
}

var tools = []Tool{
	{
		ID:          "portainer",
		Name:        "Portainer CE (LTS)",
		Description: "Docker Compose stack for Portainer CE LTS.",
		Tags:        []string{"docker", "management"},
	},
	{
		ID:          "ddev",
		Name:        "DDEV",
		Description: "Installer script and starter config for DDEV.",
		Tags:        []string{"php", "web", "local-dev"},
	},
}

const projectName = "StackWarden"

var (
	version = "dev"
	commit  string
	builtAt string
)

//go:embed tools/templates/** tools/templates/portainer/.env.example tools/templates/ddev/starter/.ddev/**
var templateFS embed.FS

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

func findTool(id string) (Tool, error) {
	for _, t := range tools {
		if t.ID == id {
			return t, nil
		}
	}
	return Tool{}, fmt.Errorf("tool %s not found", id)
}

func buildBundle(toolID string) (*bytes.Buffer, error) {
	base := path.Join("tools", "templates", toolID)

	_, err := fs.Stat(templateFS, base)
	if err != nil {
		return nil, fmt.Errorf("template path missing: %w", err)
	}

	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)

	walkErr := fs.WalkDir(templateFS, base, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			return nil
		}

		rel := strings.TrimPrefix(p, base+"/")
		if rel == "" || strings.Contains(rel, "..") {
			return errors.New("invalid template path")
		}

		data, err := fs.ReadFile(templateFS, p)
		if err != nil {
			return err
		}

		fw, err := zw.Create(path.Clean(rel))
		if err != nil {
			return err
		}

		if _, err := fw.Write(data); err != nil {
			return err
		}

		return nil
	})

	if err := zw.Close(); err != nil {
		return nil, err
	}

	if walkErr != nil {
		return nil, walkErr
	}

	return buf, nil
}

func main() {
	mux := http.NewServeMux()
	audit := &auditLog{}

	// API health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Health{Status: "ok", Service: "api"})
	})

	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		info := VersionInfo{
			Name:    projectName,
			Version: version,
			Commit:  commit,
			BuiltAt: builtAt,
		}

		_ = json.NewEncoder(w).Encode(info)
	})

	// Proxy health check to agent
	mux.HandleFunc("/agent/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		agentURL := buildAgentURL("/health")

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

		agentURL := buildAgentURL("/ports")

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

		statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
		bodyOK := false
		var payload struct {
			OK bool `json:"ok"`
		}
		if err := json.Unmarshal(body, &payload); err == nil {
			bodyOK = payload.OK
		}

		ok = statusOK && bodyOK
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ok := false
		defer func() {
			audit.recordAudit("metrics.read", ok)
		}()

		agentURL := buildAgentURL("/metrics")

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

	mux.HandleFunc("/tools", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tools)
	})

	mux.HandleFunc("/tools/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/tools/"), "/")
		if len(parts) != 2 || parts[1] != "bundle" {
			http.NotFound(w, r)
			return
		}

		toolID := parts[0]
		action := "tool.download." + toolID
		ok := false
		defer func() {
			audit.recordAudit(action, ok)
		}()

		tool, err := findTool(toolID)
		if err != nil {
			http.Error(w, "tool not found", http.StatusNotFound)
			return
		}

		buf, err := buildBundle(tool.ID)
		if err != nil {
			log.Printf("bundle error for %s: %v", tool.ID, err)
			http.Error(w, "failed to build bundle", http.StatusInternalServerError)
			return
		}

		ok = true
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-bundle.zip\"", tool.ID))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	})

	// Serve UI (static)
	uiDir := resolveUIDir()
	fs := http.FileServer(http.Dir(uiDir))
	mux.Handle("/", fs)

	addr := getenv("API_BIND", ":8080")
	log.Printf("Listening on %s", addr)
	log.Printf("serving ui from %s", uiDir)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func buildAgentURL(p string) string {
	base := getenv("AGENT_BASE", getenv("AGENT_URL", "http://127.0.0.1:9091"))
	base = strings.TrimRight(base, "/")
	if base == "" {
		return p
	}
	return base + p
}

func resolveUIDir() string {
	candidates := []string{
		filepath.Join(".", "ui"),
		filepath.Join("..", "ui"),
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			if abs, err := filepath.Abs(c); err == nil {
				return abs
			}
			return filepath.Clean(c)
		}
	}

	return filepath.Clean(filepath.Join(".", "ui"))
}
