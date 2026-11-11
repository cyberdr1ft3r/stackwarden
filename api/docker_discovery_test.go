package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	shared "github.com/m0b3u/stackwarden/pkg"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestAPIDockerDiscoveryMediatesAndReferencesHostListeners(t *testing.T) {
	body := `{"status":"available","containers":[{"id":"docker:0123456789abcdef","name":"web","image_reference":"web@sha256:abc","state":"running","running":true,"published_ports":[{"host_ip":"0.0.0.0","host_port":8080,"container_port":80,"protocol":"tcp"}],"compose_hint":{"source":"compose_labels","project":"site","service":"web"}}]}`
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/docker/apps" {
			t.Fatalf("unexpected agent request: %s %s", req.Method, req.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	rr := httptest.NewRecorder()
	handleDockerDiscovery(rr, httptest.NewRequest(http.MethodGet, "/v1/read/docker/apps", nil), &auditLog{}, client)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var result shared.DockerDiscovery
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != shared.DockerStatusAvailable || result.HostListenersEndpoint != "/v1/read/ports" || len(result.Containers) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestAPIDockerDiscoveryDistinguishesUnavailableFromEmpty(t *testing.T) {
	t.Run("unavailable", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("agent down")
		})}
		rr := httptest.NewRecorder()
		handleDockerDiscovery(rr, httptest.NewRequest(http.MethodGet, "/v1/read/docker/apps", nil), &auditLog{}, client)
		if !strings.Contains(rr.Body.String(), `"status":"unavailable"`) || !strings.Contains(rr.Body.String(), `"reason":"agent_unavailable"`) {
			t.Fatalf("unexpected unavailable response: %s", rr.Body.String())
		}
	})

	t.Run("empty", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"status":"available","containers":[]}`)),
			}, nil
		})}
		rr := httptest.NewRecorder()
		handleDockerDiscovery(rr, httptest.NewRequest(http.MethodGet, "/v1/read/docker/apps", nil), &auditLog{}, client)
		if !strings.Contains(rr.Body.String(), `"status":"available"`) || !strings.Contains(rr.Body.String(), `"containers":[]`) {
			t.Fatalf("unexpected empty response: %s", rr.Body.String())
		}
	})
}

func TestAPIDockerDiscoveryRejectsMalformedAgentData(t *testing.T) {
	body := `{"status":"available","containers":[{"id":"docker:0123456789abcdef","state":"running","compose_hint":{"source":"ownership","project":"site"}}]}`
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	rr := httptest.NewRecorder()
	handleDockerDiscovery(rr, httptest.NewRequest(http.MethodGet, "/v1/read/docker/apps", nil), &auditLog{}, client)
	if !strings.Contains(rr.Body.String(), `"status":"unknown"`) || !strings.Contains(rr.Body.String(), `"reason":"agent_validation_failed"`) {
		t.Fatalf("unexpected validation response: %s", rr.Body.String())
	}
}

func TestAPIDockerDiscoveryIsReadOnly(t *testing.T) {
	rr := httptest.NewRecorder()
	handleDockerDiscovery(rr, httptest.NewRequest(http.MethodPost, "/v1/read/docker/apps", strings.NewReader("{}")), &auditLog{}, &http.Client{})
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}
