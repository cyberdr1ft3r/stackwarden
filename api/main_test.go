package main

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

type countingTransport struct {
	calls int
}

func (t *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.calls++
	body := `{"ok":true}`
	if strings.HasSuffix(req.URL.Path, "/uninstall") {
		body = `{"uninstalled":true}`
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func TestResolveAPIBind_EmptyHostIsTreatedAsLoopback(t *testing.T) {
	t.Setenv("API_BIND", ":8080")
	t.Setenv("ALLOW_NONLOCAL_BIND", "")

	bind, err := resolveAPIBind(6000, false)
	if err != nil {
		t.Fatalf("resolveAPIBind returned error: %v", err)
	}
	if bind != "127.0.0.1:8080" {
		t.Fatalf("expected bind 127.0.0.1:8080, got %q", bind)
	}
}

func TestResolveAPIBind_PortFlagNormalizesEmptyHostFromEnv(t *testing.T) {
	t.Setenv("API_BIND", ":8080")
	t.Setenv("ALLOW_NONLOCAL_BIND", "")

	bind, err := resolveAPIBind(6000, true)
	if err != nil {
		t.Fatalf("resolveAPIBind returned error: %v", err)
	}
	if bind != "127.0.0.1:6000" {
		t.Fatalf("expected bind 127.0.0.1:6000, got %q", bind)
	}
}

func TestResolveAPIBind_PortFlagPreservesHostFromEnv(t *testing.T) {
	t.Setenv("API_BIND", "127.0.0.1:8080")

	bind, err := resolveAPIBind(6000, true)
	if err != nil {
		t.Fatalf("resolveAPIBind returned error: %v", err)
	}
	if bind != "127.0.0.1:6000" {
		t.Fatalf("expected bind 127.0.0.1:6000, got %q", bind)
	}
}

func TestResolveAPIBind_WithoutPortFlagUsesEnvBind(t *testing.T) {
	t.Setenv("API_BIND", "127.0.0.1:8080")

	bind, err := resolveAPIBind(6000, false)
	if err != nil {
		t.Fatalf("resolveAPIBind returned error: %v", err)
	}
	if bind != "127.0.0.1:8080" {
		t.Fatalf("expected bind 127.0.0.1:8080, got %q", bind)
	}
}

func TestResolveAPIBind_DefaultWhenNoEnvAndNoPortFlag(t *testing.T) {
	t.Setenv("API_BIND", "")

	bind, err := resolveAPIBind(6000, false)
	if err != nil {
		t.Fatalf("resolveAPIBind returned error: %v", err)
	}
	if bind != "127.0.0.1:8080" {
		t.Fatalf("expected bind 127.0.0.1:8080, got %q", bind)
	}
}

func TestResolveAPIBind_WithoutPortFlagAllowsEphemeralPortOnSpecificHost(t *testing.T) {
	t.Setenv("API_BIND", "127.0.0.1:0")

	bind, err := resolveAPIBind(6000, false)
	if err != nil {
		t.Fatalf("resolveAPIBind returned error: %v", err)
	}
	if bind != "127.0.0.1:0" {
		t.Fatalf("expected bind 127.0.0.1:0, got %q", bind)
	}
}

func TestResolveAPIBind_WithoutPortFlagRejectsOutOfRangePort(t *testing.T) {
	t.Setenv("API_BIND", ":65536")

	if _, err := resolveAPIBind(6000, false); err == nil {
		t.Fatal("expected error for API_BIND=:65536")
	}
}

func TestResolveAPIBind_WithoutPortFlagRejectsNonLocalByDefault(t *testing.T) {
	t.Setenv("API_BIND", "0.0.0.0:8080")
	t.Setenv("ALLOW_NONLOCAL_BIND", "")

	if _, err := resolveAPIBind(6000, false); err == nil {
		t.Fatal("expected non-local bind to be rejected")
	}
}

func TestResolveAPIBind_WithoutPortFlagAllowsNonLocalWhenOverridden(t *testing.T) {
	t.Setenv("API_BIND", "0.0.0.0:8080")
	t.Setenv("ALLOW_NONLOCAL_BIND", "1")

	bind, err := resolveAPIBind(6000, false)
	if err != nil {
		t.Fatalf("resolveAPIBind returned error: %v", err)
	}
	if bind != "0.0.0.0:8080" {
		t.Fatalf("expected bind 0.0.0.0:8080, got %q", bind)
	}
}

func TestWriteAuthMiddleware_Disabled(t *testing.T) {
	h := writeAuthMiddleware(securityConfig{WriteEnabled: false}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/write/tools/portainer/install", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload["error"] != "write_disabled" {
		t.Fatalf("unexpected payload: %v", payload)
	}
}

func TestWriteAuthMiddleware_RequiresBearerToken(t *testing.T) {
	h := writeAuthMiddleware(securityConfig{WriteEnabled: true, Token: "secret"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/write/tools/portainer/install", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestWriteAuthMiddleware_RejectsInvalidBearerToken(t *testing.T) {
	h := writeAuthMiddleware(securityConfig{WriteEnabled: true, Token: "secret"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/write/tools/portainer/install", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestWriteAuthMiddleware_AllowsCorrectBearerToken(t *testing.T) {
	h := writeAuthMiddleware(securityConfig{WriteEnabled: true, Token: "secret"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/write/tools/portainer/install", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestNewAgentClientUsesUnixSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket transport is verified on Unix-like hosts")
	}

	socketPath := filepath.Join(t.TempDir(), "agent.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen on Unix socket: %v", err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})}
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = server.Close()
	})

	response, err := callAgent(newAgentClient(socketPath), http.MethodGet, "/health", nil, time.Second)
	if err != nil {
		t.Fatalf("callAgent over Unix socket: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", response.StatusCode)
	}
}

func TestLoadSecurityConfig_DefaultsWritesOff(t *testing.T) {
	t.Setenv("STACKWARDEN_WRITE_ENABLED", "")
	t.Setenv("STACKWARDEN_TOKEN", "")
	t.Setenv("AGENT_SOCKET", "")

	cfg, err := loadSecurityConfig()
	if err != nil {
		t.Fatalf("loadSecurityConfig returned error: %v", err)
	}
	if cfg.WriteEnabled {
		t.Fatal("writes must be disabled by default")
	}
	if cfg.Token != "" {
		t.Fatal("default token must be empty")
	}
	if cfg.AgentSocket != "/run/stackwarden/agent.sock" {
		t.Fatalf("unexpected default agent socket: %q", cfg.AgentSocket)
	}
}

func TestLoadSecurityConfig_EnabledWritesRequireToken(t *testing.T) {
	t.Setenv("STACKWARDEN_WRITE_ENABLED", "true")
	t.Setenv("STACKWARDEN_TOKEN", "")

	if _, err := loadSecurityConfig(); err == nil {
		t.Fatal("expected enabled writes without a token to fail")
	}
}

func TestRegisterWriteRoutes_ProtectsEverySupportedAction(t *testing.T) {
	for _, action := range []string{"install", "uninstall"} {
		path := "/v1/write/tools/portainer/" + action

		t.Run(action+"/disabled", func(t *testing.T) {
			transport := &countingTransport{}
			mux := http.NewServeMux()
			registerWriteRoutes(mux, securityConfig{}, &auditLog{}, &http.Client{Transport: transport})

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, path, nil))
			if rr.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d", rr.Code)
			}
			if transport.calls != 0 {
				t.Fatal("disabled write reached the agent")
			}
		})

		t.Run(action+"/unauthorized", func(t *testing.T) {
			transport := &countingTransport{}
			mux := http.NewServeMux()
			cfg := securityConfig{WriteEnabled: true, Token: "test-token"}
			registerWriteRoutes(mux, cfg, &auditLog{}, &http.Client{Transport: transport})

			for _, auth := range []string{"", "Bearer invalid"} {
				req := httptest.NewRequest(http.MethodPost, path, nil)
				if auth != "" {
					req.Header.Set("Authorization", auth)
				}
				rr := httptest.NewRecorder()
				mux.ServeHTTP(rr, req)
				if rr.Code != http.StatusUnauthorized {
					t.Fatalf("expected 401 for %q, got %d", auth, rr.Code)
				}
			}
			if transport.calls != 0 {
				t.Fatal("unauthorized write reached the agent")
			}
		})

		t.Run(action+"/authorized", func(t *testing.T) {
			transport := &countingTransport{}
			mux := http.NewServeMux()
			cfg := securityConfig{WriteEnabled: true, Token: "test-token"}
			registerWriteRoutes(mux, cfg, &auditLog{}, &http.Client{Transport: transport})

			req := httptest.NewRequest(http.MethodPost, path, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
			}
			if transport.calls != 1 {
				t.Fatalf("expected one agent call, got %d", transport.calls)
			}
		})
	}
}

func TestRegisterWriteRoutes_RejectsMaliciousToolIDs(t *testing.T) {
	transport := &countingTransport{}
	mux := http.NewServeMux()
	cfg := securityConfig{WriteEnabled: true, Token: "test-token"}
	registerWriteRoutes(mux, cfg, &auditLog{}, &http.Client{Transport: transport})

	for _, path := range []string{
		"/v1/write/tools/%2e%2e/install",
		"/v1/write/tools/portainer%2f..%2fddev/install",
		"/v1/write/tools/Portainer/install",
	} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected malicious path %q to return 404, got %d", path, rr.Code)
		}
	}
	if transport.calls != 0 {
		t.Fatal("malicious tool ID reached the agent")
	}
}

func TestUIUsesVersionedRoutesAndEphemeralWriteToken(t *testing.T) {
	uiPath := filepath.Join("..", "ui", "index.html")
	data, err := os.ReadFile(uiPath)
	if err != nil {
		t.Fatalf("read UI: %v", err)
	}
	source := string(data)

	fetchPattern := regexp.MustCompile(`fetch\((?:\"|` + "`" + `)(/[^\"` + "`" + `]+)`)
	matches := fetchPattern.FindAllStringSubmatch(source, -1)
	if len(matches) == 0 {
		t.Fatal("expected UI fetch calls")
	}
	for _, match := range matches {
		route := match[1]
		if !strings.HasPrefix(route, "/v1/read/") && !strings.HasPrefix(route, "/v1/write/") {
			t.Errorf("UI uses unversioned API route %q", route)
		}
	}

	for _, line := range strings.Split(source, "\n") {
		if strings.Contains(line, `fetch(`+"`"+`/v1/write/`) && !strings.Contains(line, "writeFetchOptions()") {
			t.Errorf("UI write does not apply Bearer options: %s", strings.TrimSpace(line))
		}
	}
	for _, forbidden := range []string{"localStorage", "sessionStorage", "console.log", "console.debug"} {
		if strings.Contains(source, forbidden) {
			t.Errorf("UI must not persist or log the write token; found %q", forbidden)
		}
	}
}

func TestRegisterLegacyReadRoutes_CompatPathsReachReadMux(t *testing.T) {
	readMux := http.NewServeMux()
	readMux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	readMux.HandleFunc("/port-events", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	readMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPartialContent)
	})

	mux := http.NewServeMux()
	registerLegacyReadRoutes(mux, readMux)

	versionReq := httptest.NewRequest(http.MethodGet, "/version", nil)
	versionRR := httptest.NewRecorder()
	mux.ServeHTTP(versionRR, versionReq)
	if versionRR.Code != http.StatusNoContent {
		t.Fatalf("expected /version to be routed to read mux, got %d", versionRR.Code)
	}

	portEventsReq := httptest.NewRequest(http.MethodGet, "/port-events", nil)
	portEventsRR := httptest.NewRecorder()
	mux.ServeHTTP(portEventsRR, portEventsReq)
	if portEventsRR.Code != http.StatusAccepted {
		t.Fatalf("expected /port-events to be routed to read mux, got %d", portEventsRR.Code)
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRR := httptest.NewRecorder()
	mux.ServeHTTP(metricsRR, metricsReq)
	if metricsRR.Code != http.StatusPartialContent {
		t.Fatalf("expected /metrics to be routed to read mux, got %d", metricsRR.Code)
	}
}

func TestRegisterLegacyReadRoutes_DoesNotExposeLegacyWriteActions(t *testing.T) {
	readMux := http.NewServeMux()
	readMux.HandleFunc("/tools/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/tools/"), "/")
		if len(parts) == 2 && parts[1] == "status" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	})

	mux := http.NewServeMux()
	registerLegacyReadRoutes(mux, readMux)

	for _, path := range []string{"/tools/portainer/install", "/tools/portainer/uninstall"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected %s to return 404, got %d", path, rr.Code)
		}
	}
}
