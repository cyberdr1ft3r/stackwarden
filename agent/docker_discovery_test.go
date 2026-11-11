package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	shared "github.com/m0b3u/stackwarden/pkg"
)

type dockerDoerFunc func(*http.Request) (*http.Response, error)

func (f dockerDoerFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func dockerJSONResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestCollectDockerDiscoveryUsesOneExactReadOnlyRequest(t *testing.T) {
	fixture := `[
		{
			"Id":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			"Names":["/web"],
			"Image":"registry.example/web@sha256:abc",
			"State":"running",
			"Status":"Up 2 minutes",
			"Ports":[
				{"IP":"127.0.0.1","PrivatePort":8080,"PublicPort":18080,"Type":"tcp"},
				{"PrivatePort":9000,"Type":"tcp"}
			],
			"Labels":{
				"com.docker.compose.project":"site",
				"com.docker.compose.service":"web",
				"private.secret":"must-not-leak"
			},
			"Mounts":[{"Source":"/secret/path"}]
		}
	]`
	calls := 0
	client := dockerDoerFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1.41/containers/json" || req.URL.RawQuery != "all=1" {
			t.Fatalf("unexpected Engine target: %s", req.URL.String())
		}
		if req.Body != nil {
			t.Fatal("read-only request must not have a body")
		}
		return dockerJSONResponse(fixture), nil
	})

	result := collectDockerDiscovery(context.Background(), client)
	if calls != 1 {
		t.Fatalf("expected one Engine request, got %d", calls)
	}
	if result.Status != shared.DockerStatusAvailable || len(result.Containers) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	container := result.Containers[0]
	if container.ID != "docker:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected scoped id: %q", container.ID)
	}
	if container.Running == nil || !*container.Running || container.State != "running" {
		t.Fatalf("unexpected running state: %#v", container)
	}
	if len(container.PublishedPorts) != 1 || container.PublishedPorts[0].HostPort != 18080 {
		t.Fatalf("only published ports should be returned: %#v", container.PublishedPorts)
	}
	if container.ComposeHint == nil || container.ComposeHint.Source != "compose_labels" || container.ComposeHint.Project != "site" {
		t.Fatalf("unexpected grouping hint: %#v", container.ComposeHint)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"must-not-leak", "/secret/path", "private.secret", "Mounts", "Labels"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("response leaked forbidden Docker data %q: %s", forbidden, encoded)
		}
	}
}

func TestCollectDockerDiscoveryPreservesUnavailableReasons(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		reason string
	}{
		{name: "missing", err: os.ErrNotExist, reason: "not_found"},
		{name: "permission", err: os.ErrPermission, reason: "permission_denied"},
		{name: "timeout", err: context.DeadlineExceeded, reason: "timeout"},
		{name: "other", err: errors.New("daemon down"), reason: "engine_unavailable"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := collectDockerDiscovery(context.Background(), dockerDoerFunc(func(*http.Request) (*http.Response, error) {
				return nil, tc.err
			}))
			if result.Status != shared.DockerStatusUnavailable || result.Reason != tc.reason || result.Containers == nil {
				t.Fatalf("unexpected result: %#v", result)
			}
		})
	}
}

func TestCollectDockerDiscoveryRejectsMalformedAndOversizedResponses(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		reason string
	}{
		{name: "malformed", body: "[", reason: "invalid_response"},
		{name: "oversized", body: strings.Repeat("x", maxDockerResponseBytes+1), reason: "response_too_large"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := collectDockerDiscovery(context.Background(), dockerDoerFunc(func(*http.Request) (*http.Response, error) {
				return dockerJSONResponse(tc.body), nil
			}))
			if result.Status != shared.DockerStatusUnknown || result.Reason != tc.reason {
				t.Fatalf("unexpected result: %#v", result)
			}
		})
	}
}

func TestCollectDockerDiscoveryMarksPartialWithoutInventingState(t *testing.T) {
	fixture := `[
		{"Id":"not-a-docker-id","Names":["bad"],"Image":"bad","State":"running"},
		{"Id":"abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd","Names":["/worker"],"Image":"worker:latest","State":"mystery","Ports":[{"IP":"0.0.0.0","PrivatePort":80,"PublicPort":8080,"Type":"invalid"}]}
	]`
	result := collectDockerDiscovery(context.Background(), dockerDoerFunc(func(*http.Request) (*http.Response, error) {
		return dockerJSONResponse(fixture), nil
	}))
	if result.Status != shared.DockerStatusPartial || len(result.Containers) != 1 {
		t.Fatalf("unexpected partial result: %#v", result)
	}
	if result.Containers[0].State != "unknown" || result.Containers[0].Running != nil {
		t.Fatalf("unknown state was overclaimed: %#v", result.Containers[0])
	}
	if len(result.Containers[0].PublishedPorts) != 0 {
		t.Fatalf("invalid publication should be omitted: %#v", result.Containers[0].PublishedPorts)
	}
}

func TestDockerDiscoveryHandlerRejectsMutationMethod(t *testing.T) {
	rr := httptest.NewRecorder()
	handleDockerDiscovery(rr, httptest.NewRequest(http.MethodPost, "/docker/apps", strings.NewReader("{}")))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestCollectDockerDiscoveryBoundsContainerCount(t *testing.T) {
	raw := make([]dockerContainerJSON, shared.MaxDockerContainers+1)
	for i := range raw {
		raw[i] = dockerContainerJSON{
			ID:    fmt.Sprintf("%064x", i+1),
			Names: []string{fmt.Sprintf("/container-%d", i)},
			Image: "example@sha256:abc",
			State: "running",
		}
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	result := collectDockerDiscovery(context.Background(), dockerDoerFunc(func(*http.Request) (*http.Response, error) {
		return dockerJSONResponse(string(body)), nil
	}))
	if result.Status != shared.DockerStatusPartial || !result.Truncated || len(result.Containers) != shared.MaxDockerContainers {
		t.Fatalf("container bound not enforced: status=%s truncated=%v count=%d", result.Status, result.Truncated, len(result.Containers))
	}
}

func TestDockerDiscoveryHandlerBoundsConcurrency(t *testing.T) {
	dockerCollectionSlot <- struct{}{}
	defer func() { <-dockerCollectionSlot }()

	rr := httptest.NewRecorder()
	handleDockerDiscovery(rr, httptest.NewRequest(http.MethodGet, "/docker/apps", nil))

	var result shared.DockerDiscovery
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != shared.DockerStatusUnavailable || result.Reason != "busy" {
		t.Fatalf("unexpected busy result: %#v", result)
	}
}
