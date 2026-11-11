package main

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	shared "github.com/m0b3u/stackwarden/pkg"
)

const maxAgentDockerResponseBytes = 256 << 10

func handleDockerDiscovery(w http.ResponseWriter, r *http.Request, audit *auditLog, agentClient *http.Client) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	ok := false
	defer func() { audit.recordAudit("docker.apps.read", ok) }()

	resp, err := callAgent(agentClient, http.MethodGet, "/docker/apps", nil, 5*time.Second)
	if err != nil {
		_ = json.NewEncoder(w).Encode(apiDockerUnavailable("agent_unavailable"))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_ = json.NewEncoder(w).Encode(apiDockerUnavailable("agent_error"))
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAgentDockerResponseBytes+1))
	if err != nil {
		_ = json.NewEncoder(w).Encode(apiDockerUnknown("agent_read_failed"))
		return
	}
	if len(body) > maxAgentDockerResponseBytes {
		_ = json.NewEncoder(w).Encode(apiDockerUnknown("agent_response_too_large"))
		return
	}

	var discovery shared.DockerDiscovery
	if err := json.Unmarshal(body, &discovery); err != nil {
		_ = json.NewEncoder(w).Encode(apiDockerUnknown("agent_invalid_response"))
		return
	}
	if err := shared.ValidateDockerDiscovery(discovery); err != nil {
		_ = json.NewEncoder(w).Encode(apiDockerUnknown("agent_validation_failed"))
		return
	}
	discovery.HostListenersEndpoint = "/v1/read/ports"
	ok = discovery.Status == shared.DockerStatusAvailable || discovery.Status == shared.DockerStatusPartial
	_ = json.NewEncoder(w).Encode(discovery)
}

func apiDockerUnavailable(reason string) shared.DockerDiscovery {
	return shared.DockerDiscovery{
		Status:                shared.DockerStatusUnavailable,
		Reason:                reason,
		Containers:            []shared.DockerContainer{},
		HostListenersEndpoint: "/v1/read/ports",
	}
}

func apiDockerUnknown(reason string) shared.DockerDiscovery {
	return shared.DockerDiscovery{
		Status:                shared.DockerStatusUnknown,
		Reason:                reason,
		Containers:            []shared.DockerContainer{},
		HostListenersEndpoint: "/v1/read/ports",
	}
}
