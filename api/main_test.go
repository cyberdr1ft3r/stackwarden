package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveAPIBind_EmptyHostIsTreatedAsLoopback(t *testing.T) {
	t.Setenv("API_BIND", ":8080")
	t.Setenv("ALLOW_NONLOCAL_BIND", "")

	bind, err := resolveAPIBind(6000, false)
	if err != nil {
		t.Fatalf("resolveAPIBind returned error: %v", err)
	}
	if bind != ":8080" {
		t.Fatalf("expected bind :8080, got %q", bind)
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

func TestRegisterLegacyReadRoutes_CompatPathsReachReadMux(t *testing.T) {
	readMux := http.NewServeMux()
	readMux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	readMux.HandleFunc("/port-events", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
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
}
