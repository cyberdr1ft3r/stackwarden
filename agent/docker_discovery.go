package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	shared "github.com/m0b3u/stackwarden/pkg"
)

const (
	dockerContainersPath   = "/v1.41/containers/json?all=1"
	dockerRequestTimeout   = 3 * time.Second
	maxDockerResponseBytes = 1 << 20
)

var dockerCollectionSlot = make(chan struct{}, 1)

type dockerDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type dockerContainerJSON struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	Image  string            `json:"Image"`
	State  string            `json:"State"`
	Status string            `json:"Status"`
	Ports  []dockerPortJSON  `json:"Ports"`
	Labels map[string]string `json:"Labels"`
}

type dockerPortJSON struct {
	IP          string `json:"IP"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort"`
	Type        string `json:"Type"`
}

func handleDockerDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	select {
	case dockerCollectionSlot <- struct{}{}:
		defer func() { <-dockerCollectionSlot }()
	default:
		_ = json.NewEncoder(w).Encode(unavailableDockerDiscovery("busy"))
		return
	}

	client, err := newDockerEngineClient()
	if err != nil {
		_ = json.NewEncoder(w).Encode(unavailableDockerDiscovery("unsupported"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), dockerRequestTimeout)
	defer cancel()
	_ = json.NewEncoder(w).Encode(collectDockerDiscovery(ctx, client))
}

func collectDockerDiscovery(ctx context.Context, client dockerDoer) shared.DockerDiscovery {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker"+dockerContainersPath, nil)
	if err != nil {
		return unknownDockerDiscovery("request_invalid")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded), errors.Is(ctx.Err(), context.DeadlineExceeded):
			return unavailableDockerDiscovery("timeout")
		case errors.Is(err, os.ErrPermission):
			return unavailableDockerDiscovery("permission_denied")
		case errors.Is(err, os.ErrNotExist):
			return unavailableDockerDiscovery("not_found")
		default:
			return unavailableDockerDiscovery("engine_unavailable")
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return unavailableDockerDiscovery("engine_error")
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDockerResponseBytes+1))
	if err != nil {
		return unknownDockerDiscovery("read_failed")
	}
	if len(body) > maxDockerResponseBytes {
		return unknownDockerDiscovery("response_too_large")
	}

	var raw []dockerContainerJSON
	if err := json.Unmarshal(body, &raw); err != nil {
		return unknownDockerDiscovery("invalid_response")
	}

	result := shared.DockerDiscovery{
		Status:     shared.DockerStatusAvailable,
		Containers: make([]shared.DockerContainer, 0, min(len(raw), shared.MaxDockerContainers)),
	}
	if len(raw) > shared.MaxDockerContainers {
		raw = raw[:shared.MaxDockerContainers]
		result.Status = shared.DockerStatusPartial
		result.Truncated = true
	}
	for _, item := range raw {
		container, partial, ok := normalizeDockerContainer(item)
		if !ok {
			result.Status = shared.DockerStatusPartial
			continue
		}
		if partial {
			result.Status = shared.DockerStatusPartial
		}
		result.Containers = append(result.Containers, container)
	}
	sort.Slice(result.Containers, func(i, j int) bool {
		return result.Containers[i].ID < result.Containers[j].ID
	})
	if err := shared.ValidateDockerDiscovery(result); err != nil {
		return unknownDockerDiscovery("validation_failed")
	}
	return result
}

func normalizeDockerContainer(raw dockerContainerJSON) (shared.DockerContainer, bool, bool) {
	rawID, idChanged := boundedDockerString(raw.ID, 64)
	if idChanged || len(rawID) < 12 || !isLowerHex(rawID) {
		return shared.DockerContainer{}, true, false
	}

	name := ""
	partial := false
	if len(raw.Names) > 0 {
		name, partial = boundedDockerString(strings.TrimPrefix(raw.Names[0], "/"), shared.MaxDockerNameBytes)
	}
	image, changed := boundedDockerString(raw.Image, shared.MaxDockerImageBytes)
	partial = partial || changed
	statusText, changed := boundedDockerString(raw.Status, shared.MaxDockerStatusBytes)
	partial = partial || changed

	state := strings.ToLower(strings.TrimSpace(raw.State))
	var running *bool
	switch state {
	case "running":
		value := true
		running = &value
	case "exited", "paused", "restarting", "created", "removing", "dead":
		value := false
		running = &value
	default:
		state = "unknown"
		partial = true
	}

	ports := make([]shared.DockerPublishedPort, 0, min(len(raw.Ports), shared.MaxDockerPortsPerContainer))
	for _, port := range raw.Ports {
		if port.PublicPort == 0 {
			continue
		}
		if port.PublicPort < 1 || port.PublicPort > 65535 || port.PrivatePort < 1 || port.PrivatePort > 65535 {
			partial = true
			continue
		}
		protocol := strings.ToLower(port.Type)
		if protocol != "tcp" && protocol != "udp" && protocol != "sctp" {
			partial = true
			continue
		}
		if len(ports) == shared.MaxDockerPortsPerContainer {
			partial = true
			break
		}
		hostIP, changed := boundedDockerString(port.IP, shared.MaxDockerHostIPBytes)
		partial = partial || changed
		if hostIP != "" && net.ParseIP(hostIP) == nil {
			partial = true
			continue
		}
		ports = append(ports, shared.DockerPublishedPort{
			HostIP:        hostIP,
			HostPort:      port.PublicPort,
			ContainerPort: port.PrivatePort,
			Protocol:      protocol,
		})
	}
	sort.Slice(ports, func(i, j int) bool {
		if ports[i].HostIP != ports[j].HostIP {
			return ports[i].HostIP < ports[j].HostIP
		}
		if ports[i].HostPort != ports[j].HostPort {
			return ports[i].HostPort < ports[j].HostPort
		}
		return ports[i].ContainerPort < ports[j].ContainerPort
	})

	var hint *shared.DockerComposeHint
	project, projectChanged := boundedDockerString(raw.Labels["com.docker.compose.project"], shared.MaxDockerHintBytes)
	service, serviceChanged := boundedDockerString(raw.Labels["com.docker.compose.service"], shared.MaxDockerHintBytes)
	partial = partial || projectChanged || serviceChanged
	if project != "" || service != "" {
		hint = &shared.DockerComposeHint{Source: "compose_labels", Project: project, Service: service}
	}

	return shared.DockerContainer{
		ID:             "docker:" + rawID,
		Name:           name,
		ImageReference: image,
		State:          state,
		Running:        running,
		StatusText:     statusText,
		PublishedPorts: ports,
		ComposeHint:    hint,
	}, partial, true
}

func boundedDockerString(value string, limit int) (string, bool) {
	original := value
	value = strings.TrimSpace(value)
	value = strings.ToValidUTF8(value, "")
	var builder strings.Builder
	changed := value != original
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			changed = true
			continue
		}
		if builder.Len()+utf8.RuneLen(r) > limit {
			changed = true
			break
		}
		builder.WriteRune(r)
	}
	return builder.String(), changed
}

func isLowerHex(value string) bool {
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func unavailableDockerDiscovery(reason string) shared.DockerDiscovery {
	return shared.DockerDiscovery{Status: shared.DockerStatusUnavailable, Reason: reason, Containers: []shared.DockerContainer{}}
}

func unknownDockerDiscovery(reason string) shared.DockerDiscovery {
	return shared.DockerDiscovery{Status: shared.DockerStatusUnknown, Reason: reason, Containers: []shared.DockerContainer{}}
}
