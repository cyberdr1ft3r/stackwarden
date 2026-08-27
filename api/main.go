package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	shared "github.com/m0b3u/stackwarden/pkg"
	"github.com/m0b3u/stackwarden/pkg/tools"
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

type installResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Output  string `json:"output,omitempty"`
	Path    string `json:"path,omitempty"`
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

type PortEntry struct {
	Protocol     string `json:"protocol"`
	LocalAddress string `json:"local_address"`
	LocalPort    int    `json:"local_port"`
	PID          int    `json:"pid,omitempty"`
	State        string `json:"state,omitempty"`
	Process      string `json:"process,omitempty"`
}

type PortsPayload struct {
	OK    bool        `json:"ok"`
	Ports []PortEntry `json:"ports,omitempty"`
	Data  []PortEntry `json:"data,omitempty"`
}

type PortRef struct {
	Protocol     string `json:"protocol"`
	LocalAddress string `json:"local_address"`
	LocalPort    int    `json:"local_port"`
}

type PortEvent struct {
	Time    string  `json:"time"`
	Kind    string  `json:"kind"`
	Port    PortRef `json:"port"`
	Details string  `json:"details,omitempty"`
}

type portEventLog struct {
	mu     sync.Mutex
	events []PortEvent
	limit  int
}

type portSnapshot struct {
	mu          sync.Mutex
	last        []PortEntry
	initialized bool
}

type VersionInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
	BuiltAt string `json:"builtAt,omitempty"`
}

type securityConfig struct {
	WriteEnabled bool
	Token        string
	AgentSocket  string
}

const projectName = "StackWarden"

var (
	version = "dev"
	commit  string
	builtAt string
)

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

func (l *portEventLog) append(events ...PortEvent) {
	if len(events) == 0 {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = append(l.events, events...)
	if l.limit > 0 && len(l.events) > l.limit {
		l.events = l.events[len(l.events)-l.limit:]
	}
}

func (l *portEventLog) list() []PortEvent {
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]PortEvent, len(l.events))
	copy(out, l.events)
	return out
}

func (s *portSnapshot) diffAndStore(current []PortEntry) []PortEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	currCopy := clonePorts(current)
	if !s.initialized {
		s.last = currCopy
		s.initialized = true
		return nil
	}

	events := diffPorts(s.last, currCopy)
	s.last = currCopy
	s.initialized = true
	return events
}

func clonePorts(in []PortEntry) []PortEntry {
	if len(in) == 0 {
		return nil
	}
	out := make([]PortEntry, len(in))
	copy(out, in)
	return out
}

func portKey(p PortEntry) string {
	return strings.ToLower(strings.TrimSpace(p.Protocol)) + "|" + strings.TrimSpace(p.LocalAddress) + "|" + fmt.Sprint(p.LocalPort)
}

func isPublicBind(addr string) bool {
	switch strings.TrimSpace(strings.ToLower(addr)) {
	case "0.0.0.0", "::", "*":
		return true
	default:
		return false
	}
}

func isLocalBind(addr string) bool {
	switch strings.TrimSpace(strings.ToLower(addr)) {
	case "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func exposureCategory(addr string) string {
	if isPublicBind(addr) {
		return "public"
	}
	if isLocalBind(addr) {
		return "local"
	}
	return ""
}

func diffPorts(prev, curr []PortEntry) []PortEvent {
	events := make([]PortEvent, 0)
	prevByKey := make(map[string]PortEntry, len(prev))
	prevByProtoPort := make(map[string][]PortEntry)
	currByKey := make(map[string]PortEntry, len(curr))
	now := time.Now().UTC().Format(time.RFC3339Nano)

	for _, p := range prev {
		prevByKey[portKey(p)] = p
		ppKey := protoPortKey(p)
		prevByProtoPort[ppKey] = append(prevByProtoPort[ppKey], p)
	}
	for _, p := range curr {
		currByKey[portKey(p)] = p
	}

	consumedPrev := make(map[string]bool)

	for key, currPort := range currByKey {
		if prevPort, ok := prevByKey[key]; ok {
			prevExposure := exposureCategory(prevPort.LocalAddress)
			currExposure := exposureCategory(currPort.LocalAddress)
			if prevExposure != "" && currExposure != "" && prevExposure != currExposure {
				events = append(events, PortEvent{
					Time: now,
					Kind: "exposure",
					Port: PortRef{
						Protocol:     currPort.Protocol,
						LocalAddress: currPort.LocalAddress,
						LocalPort:    currPort.LocalPort,
					},
					Details: fmt.Sprintf("address changed: %s -> %s", prevPort.LocalAddress, currPort.LocalAddress),
				})
			}
			continue
		}

		ppKey := protoPortKey(currPort)
		if prevList, ok := prevByProtoPort[ppKey]; ok {
			currExposure := exposureCategory(currPort.LocalAddress)
			for _, prevPort := range prevList {
				prevExposure := exposureCategory(prevPort.LocalAddress)
				if prevExposure != "" && currExposure != "" && prevExposure != currExposure {
					events = append(events, PortEvent{
						Time: now,
						Kind: "exposure",
						Port: PortRef{
							Protocol:     currPort.Protocol,
							LocalAddress: currPort.LocalAddress,
							LocalPort:    currPort.LocalPort,
						},
						Details: fmt.Sprintf("address changed: %s -> %s", prevPort.LocalAddress, currPort.LocalAddress),
					})
					consumedPrev[portKey(prevPort)] = true
					goto nextPort
				}
			}
		}

		events = append(events, PortEvent{
			Time: now,
			Kind: "opened",
			Port: PortRef{
				Protocol:     currPort.Protocol,
				LocalAddress: currPort.LocalAddress,
				LocalPort:    currPort.LocalPort,
			},
			Details: portDetails(currPort, "opened"),
		})

	nextPort:
	}

	for key, prevPort := range prevByKey {
		if _, ok := currByKey[key]; ok {
			continue
		}
		if consumedPrev[key] {
			continue
		}

		events = append(events, PortEvent{
			Time: now,
			Kind: "closed",
			Port: PortRef{
				Protocol:     prevPort.Protocol,
				LocalAddress: prevPort.LocalAddress,
				LocalPort:    prevPort.LocalPort,
			},
			Details: portDetails(prevPort, "closed"),
		})
	}

	return events
}

func protoPortKey(p PortEntry) string {
	return strings.ToLower(strings.TrimSpace(p.Protocol)) + "|" + fmt.Sprint(p.LocalPort)
}

func portDetails(p PortEntry, action string) string {
	var extras []string
	if p.PID > 0 {
		extras = append(extras, fmt.Sprintf("pid=%d", p.PID))
	}
	if p.Process != "" {
		extras = append(extras, p.Process)
	}
	parts := []string{strings.ToUpper(p.Protocol), fmt.Sprintf("%s:%d", p.LocalAddress, p.LocalPort), action}
	if len(extras) > 0 {
		parts = append(parts, strings.Join(extras, " "))
	}
	return strings.Join(parts, " ")
}

func parsePorts(body []byte) ([]PortEntry, error) {
	var payload PortsPayload
	if err := json.Unmarshal(body, &payload); err == nil {
		if payload.OK || len(payload.Ports) > 0 || len(payload.Data) > 0 {
			if !payload.OK && len(payload.Ports) == 0 && len(payload.Data) == 0 {
				return nil, fmt.Errorf("ports response not OK")
			}
			if len(payload.Ports) > 0 {
				return payload.Ports, nil
			}
			if len(payload.Data) > 0 {
				return payload.Data, nil
			}
			return []PortEntry{}, nil
		}
	}

	var direct []PortEntry
	if err := json.Unmarshal(body, &direct); err == nil {
		return direct, nil
	}

	return nil, fmt.Errorf("unable to parse ports response")
}

func buildBundle(toolID string) (*bytes.Buffer, error) {
	base, err := tools.TemplateBasePath(toolID)
	if err != nil {
		return nil, fmt.Errorf("template path missing: %w", err)
	}

	_, err = fs.Stat(tools.TemplateFS(), base)
	if err != nil {
		return nil, fmt.Errorf("template path missing: %w", err)
	}

	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)

	walkErr := fs.WalkDir(tools.TemplateFS(), base, func(p string, d fs.DirEntry, walkErr error) error {
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

		data, err := fs.ReadFile(tools.TemplateFS(), p)
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

func handleToolInstall(w http.ResponseWriter, r *http.Request, audit *auditLog, agentClient *http.Client, toolID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	action := "tool.install." + toolID
	ok := false
	defer func() {
		audit.recordAudit(action, ok)
	}()

	if _, err := tools.Find(toolID); err != nil {
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}

	resp, err := callAgent(agentClient, http.MethodPost, "/tools/"+toolID+"/install", nil, 65*time.Second)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(installResponse{
			OK:      false,
			Message: "failed to reach agent",
			Output:  err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(installResponse{
			OK:      false,
			Message: "failed to read agent response",
			Output:  err.Error(),
		})
		return
	}

	var payload installResponse
	if err := json.Unmarshal(body, &payload); err == nil {
		ok = payload.OK
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)

	if len(body) == 0 {
		_ = json.NewEncoder(w).Encode(installResponse{
			OK:      false,
			Message: "empty response from agent",
		})
		return
	}

	_, _ = w.Write(body)
}

func handleToolStatus(w http.ResponseWriter, r *http.Request, audit *auditLog, agentClient *http.Client, toolID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	action := "tool.status." + toolID
	ok := false
	defer func() {
		audit.recordAudit(action, ok)
	}()

	if _, err := tools.Find(toolID); err != nil {
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}

	resp, err := callAgent(agentClient, http.MethodGet, "/tools/"+toolID+"/status", nil, 35*time.Second)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(shared.OperationResult{OK: false, Error: "failed to reach agent: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(shared.OperationResult{OK: false, Error: "failed to read agent response: " + err.Error()})
		return
	}

	if len(body) == 0 {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(shared.OperationResult{OK: false, Error: "empty response from agent"})
		return
	}

	ok = resp.StatusCode >= 200 && resp.StatusCode < 300
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func handleToolUninstall(w http.ResponseWriter, r *http.Request, audit *auditLog, agentClient *http.Client, toolID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	action := "tool.uninstall." + toolID
	ok := false
	defer func() {
		audit.recordAudit(action, ok)
	}()

	if _, err := tools.Find(toolID); err != nil {
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}

	resp, err := callAgent(agentClient, http.MethodPost, "/tools/"+toolID+"/uninstall", nil, 65*time.Second)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(shared.OperationResult{OK: false, Error: "failed to reach agent: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(shared.OperationResult{OK: false, Error: "failed to read agent response: " + err.Error()})
		return
	}

	if len(body) == 0 {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(shared.OperationResult{OK: false, Error: "empty response from agent"})
		return
	}

	ok = resp.StatusCode >= 200 && resp.StatusCode < 300
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func main() {
	mux := http.NewServeMux()
	readMux := http.NewServeMux()
	writeMux := http.NewServeMux()
	audit := &auditLog{}
	portEvents := &portEventLog{limit: 100}
	portsSnapshot := &portSnapshot{}

	cfg, err := loadSecurityConfig()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.WriteEnabled {
		log.Printf("WARNING: write endpoints are enabled")
	}

	agentClient := newAgentClient(cfg.AgentSocket)

	// API health
	readMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Health{Status: "ok", Service: "api"})
	})

	readMux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
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
	readMux.HandleFunc("/agent/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		resp, err := callAgent(agentClient, http.MethodGet, "/health", nil, 2*time.Second)
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

	readMux.HandleFunc("/ports", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ok := false
		defer func() {
			audit.recordAudit("ports.read", ok)
		}()

		resp, err := callAgent(agentClient, http.MethodGet, "/ports", nil, 3*time.Second)
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

		if ok {
			if ports, err := parsePorts(body); err == nil {
				if events := portsSnapshot.diffAndStore(ports); len(events) > 0 {
					portEvents.append(events...)
				}
			}
		}
	})

	readMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ok := false
		defer func() {
			audit.recordAudit("metrics.read", ok)
		}()

		resp, err := callAgent(agentClient, http.MethodGet, "/metrics", nil, 3*time.Second)
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

	readMux.HandleFunc("/audit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		events := audit.list()
		_ = json.NewEncoder(w).Encode(events)
	})

	readMux.HandleFunc("/port-events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		events := portEvents.list()
		_ = json.NewEncoder(w).Encode(events)
	})

	readMux.HandleFunc("/tools", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tools.Catalog)
	})

	readMux.HandleFunc("/tools/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/tools/"), "/")
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}

		toolID := parts[0]
		if !isValidToolID(toolID) {
			http.Error(w, "tool not found", http.StatusNotFound)
			return
		}
		action := parts[1]
		switch action {
		case "status":
			handleToolStatus(w, r, audit, agentClient, toolID)
			return
		case "bundle":
		default:
			http.NotFound(w, r)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		actionName := "tool.download." + toolID
		ok := false
		defer func() {
			audit.recordAudit(actionName, ok)
		}()

		tool, err := tools.Find(toolID)
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

	writeMux.HandleFunc("/tools/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/tools/"), "/")
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		toolID := parts[0]
		if !isValidToolID(toolID) {
			http.Error(w, "tool not found", http.StatusNotFound)
			return
		}
		action := parts[1]
		switch action {
		case "install":
			handleToolInstall(w, r, audit, agentClient, toolID)
		case "uninstall":
			handleToolUninstall(w, r, audit, agentClient, toolID)
		default:
			http.NotFound(w, r)
		}
	})

	mux.Handle("/v1/read/", http.StripPrefix("/v1/read", readMux))
	mux.Handle("/v1/write/", writeAuthMiddleware(cfg, http.StripPrefix("/v1/write", writeMux)))
	registerLegacyReadRoutes(mux, readMux)

	// Serve UI (static)
	uiDir := resolveUIDir()
	fs := http.FileServer(http.Dir(uiDir))
	mux.Handle("/", fs)

	portFlagUsed := portFlagProvided(os.Args[1:])
	port := flag.Int("port", 8080, "API listen port")
	flag.Parse()

	addr, err := resolveAPIBind(*port, portFlagUsed)
	if err != nil {
		log.Fatal(err)
	}
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

func newAgentClient(socketPath string) *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &http.Client{Transport: transport}
}

func callAgent(client *http.Client, method, endpoint string, body io.Reader, timeout time.Duration) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	url := "http://stackwarden-agent" + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
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

func resolveAPIBind(port int, portFlagProvided bool) (string, error) {
	bindFromEnv := os.Getenv("API_BIND")

	if portFlagProvided {
		if port < 1 || port > 65535 {
			return "", fmt.Errorf("invalid port: %d", port)
		}

		if bindFromEnv == "" {
			return validateBind(fmt.Sprintf("127.0.0.1:%d", port))
		}

		host, _, err := splitBindHostPort(bindFromEnv)
		if err != nil {
			return "", err
		}

		return validateBind(net.JoinHostPort(host, strconv.Itoa(port)))
	}

	if bindFromEnv != "" {
		return validateBind(bindFromEnv)
	}

	return validateBind("127.0.0.1:8080")
}

func validateBind(bind string) (string, error) {
	host, portStr, err := splitBindHostPort(bind)
	if err != nil {
		return "", err
	}

	port, err := strconv.Atoi(portStr)
	// Allow port 0 for API_BIND so the OS can choose an ephemeral free port.
	if err != nil || port < 0 || port > 65535 {
		return "", fmt.Errorf("invalid bind port: %q", bind)
	}

	if !allowNonLocalBind() && !isLoopbackHost(host) {
		return "", fmt.Errorf("refusing non-local API bind %q; set ALLOW_NONLOCAL_BIND=1 to override", bind)
	}

	if host == "" {
		return net.JoinHostPort("127.0.0.1", portStr), nil
	}

	return bind, nil
}

func registerLegacyReadRoutes(mux *http.ServeMux, readMux *http.ServeMux) {
	legacyReadPaths := []string{
		"/health",
		"/version",
		"/agent/health",
		"/tools",
		"/tools/",
		"/ports",
		"/audit",
		"/port-events",
	}

	for _, legacyPath := range legacyReadPaths {
		mux.Handle(legacyPath, readMux)
	}
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return true
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func allowNonLocalBind() bool {
	return strings.TrimSpace(os.Getenv("ALLOW_NONLOCAL_BIND")) == "1"
}

func isValidToolID(toolID string) bool {
	if toolID == "" || strings.Contains(toolID, "..") || filepath.IsAbs(toolID) {
		return false
	}
	for _, c := range toolID {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			continue
		}
		return false
	}
	return true
}

func writeAuthMiddleware(cfg securityConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !cfg.WriteEnabled {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "write_disabled"})
			return
		}

		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		provided := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if subtle.ConstantTimeCompare([]byte(provided), []byte(cfg.Token)) != 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loadSecurityConfig() (securityConfig, error) {
	writeEnabledRaw := strings.TrimSpace(strings.ToLower(os.Getenv("STACKWARDEN_WRITE_ENABLED")))
	writeEnabled := writeEnabledRaw == "1" || writeEnabledRaw == "true" || writeEnabledRaw == "yes"
	token := os.Getenv("STACKWARDEN_TOKEN")
	if writeEnabled && strings.TrimSpace(token) == "" {
		return securityConfig{}, errors.New("STACKWARDEN_TOKEN is required when STACKWARDEN_WRITE_ENABLED=true")
	}
	agentSocket := strings.TrimSpace(getenv("AGENT_SOCKET", "/run/stackwarden/agent.sock"))
	if agentSocket == "" {
		return securityConfig{}, errors.New("AGENT_SOCKET cannot be empty")
	}
	return securityConfig{WriteEnabled: writeEnabled, Token: token, AgentSocket: agentSocket}, nil
}

func splitBindHostPort(bind string) (string, string, error) {
	if !strings.Contains(bind, ":") {
		return "", "", fmt.Errorf("invalid bind address: %q", bind)
	}

	host, portStr, err := net.SplitHostPort(bind)
	if err != nil {
		return "", "", fmt.Errorf("invalid bind address: %q", bind)
	}

	return host, portStr, nil
}

func portFlagProvided(args []string) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--port" || arg == "-port" {
			return true
		}
		if strings.HasPrefix(arg, "--port=") || strings.HasPrefix(arg, "-port=") {
			return true
		}
	}
	return false
}
