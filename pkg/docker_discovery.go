package pkg

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

const (
	MaxDockerContainers        = 200
	MaxDockerPortsPerContainer = 64
	MaxDockerIDBytes           = 80
	MaxDockerNameBytes         = 256
	MaxDockerImageBytes        = 512
	MaxDockerStatusBytes       = 256
	MaxDockerHintBytes         = 128
	MaxDockerHostIPBytes       = 64
	MaxDockerReasonBytes       = 64
)

type DockerObservationStatus string

const (
	DockerStatusAvailable   DockerObservationStatus = "available"
	DockerStatusPartial     DockerObservationStatus = "partial"
	DockerStatusUnavailable DockerObservationStatus = "unavailable"
	DockerStatusUnknown     DockerObservationStatus = "unknown"
)

type DockerDiscovery struct {
	Status                DockerObservationStatus `json:"status"`
	Reason                string                  `json:"reason,omitempty"`
	Containers            []DockerContainer       `json:"containers"`
	Truncated             bool                    `json:"truncated,omitempty"`
	HostListenersEndpoint string                  `json:"host_listeners_endpoint,omitempty"`
}

type DockerContainer struct {
	ID             string                `json:"id"`
	Name           string                `json:"name,omitempty"`
	ImageReference string                `json:"image_reference,omitempty"`
	State          string                `json:"state"`
	Running        *bool                 `json:"running,omitempty"`
	StatusText     string                `json:"status_text,omitempty"`
	PublishedPorts []DockerPublishedPort `json:"published_ports,omitempty"`
	ComposeHint    *DockerComposeHint    `json:"compose_hint,omitempty"`
}

type DockerPublishedPort struct {
	HostIP        string `json:"host_ip,omitempty"`
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"`
}

type DockerComposeHint struct {
	Source  string `json:"source"`
	Project string `json:"project,omitempty"`
	Service string `json:"service,omitempty"`
}

func ValidateDockerDiscovery(value DockerDiscovery) error {
	switch value.Status {
	case DockerStatusAvailable, DockerStatusPartial, DockerStatusUnavailable, DockerStatusUnknown:
	default:
		return errors.New("invalid Docker observation status")
	}
	if err := validateBoundedString("reason", value.Reason, MaxDockerReasonBytes); err != nil {
		return err
	}
	if value.HostListenersEndpoint != "" && value.HostListenersEndpoint != "/v1/read/ports" {
		return errors.New("invalid host listeners endpoint")
	}
	if len(value.Containers) > MaxDockerContainers {
		return errors.New("too many Docker containers")
	}
	if value.Containers == nil {
		return errors.New("Docker containers must be an explicit array")
	}
	if (value.Status == DockerStatusUnavailable || value.Status == DockerStatusUnknown) && len(value.Containers) != 0 {
		return errors.New("unavailable or unknown Docker observation cannot contain containers")
	}
	if (value.Status == DockerStatusUnavailable || value.Status == DockerStatusUnknown) && value.Reason == "" {
		return errors.New("unavailable or unknown Docker observation requires a reason")
	}
	if value.Status == DockerStatusAvailable && value.Reason != "" {
		return errors.New("available Docker observation cannot contain an error reason")
	}
	if value.Truncated && value.Status != DockerStatusPartial {
		return errors.New("truncated Docker observation must be partial")
	}
	for i, container := range value.Containers {
		if err := validateBoundedString("container id", container.ID, MaxDockerIDBytes); err != nil {
			return fmt.Errorf("container %d: %w", i, err)
		}
		rawID := strings.TrimPrefix(container.ID, "docker:")
		if rawID == container.ID || len(rawID) < 12 || len(rawID) > 64 || !isLowerHexString(rawID) {
			return fmt.Errorf("container %d: invalid scoped id", i)
		}
		for name, item := range map[string]struct {
			value string
			limit int
		}{
			"name":            {container.Name, MaxDockerNameBytes},
			"image reference": {container.ImageReference, MaxDockerImageBytes},
			"status text":     {container.StatusText, MaxDockerStatusBytes},
		} {
			if err := validateBoundedString(name, item.value, item.limit); err != nil {
				return fmt.Errorf("container %d: %w", i, err)
			}
		}
		switch container.State {
		case "running":
			if container.Running == nil || !*container.Running {
				return fmt.Errorf("container %d: inconsistent running state", i)
			}
		case "exited", "paused", "restarting", "created", "removing", "dead":
			if container.Running == nil || *container.Running {
				return fmt.Errorf("container %d: inconsistent non-running state", i)
			}
		case "unknown":
			if container.Running != nil {
				return fmt.Errorf("container %d: unknown state cannot claim running status", i)
			}
		default:
			return fmt.Errorf("container %d: invalid state", i)
		}
		if len(container.PublishedPorts) > MaxDockerPortsPerContainer {
			return fmt.Errorf("container %d: too many published ports", i)
		}
		for _, port := range container.PublishedPorts {
			if port.HostPort < 1 || port.HostPort > 65535 || port.ContainerPort < 1 || port.ContainerPort > 65535 {
				return fmt.Errorf("container %d: invalid published port", i)
			}
			if port.Protocol != "tcp" && port.Protocol != "udp" && port.Protocol != "sctp" {
				return fmt.Errorf("container %d: invalid port protocol", i)
			}
			if err := validateBoundedString("host ip", port.HostIP, MaxDockerHostIPBytes); err != nil {
				return fmt.Errorf("container %d: %w", i, err)
			}
			if port.HostIP != "" && net.ParseIP(port.HostIP) == nil {
				return fmt.Errorf("container %d: invalid host ip", i)
			}
		}
		if container.ComposeHint != nil {
			if container.ComposeHint.Source != "compose_labels" {
				return fmt.Errorf("container %d: invalid Compose hint source", i)
			}
			if container.ComposeHint.Project == "" && container.ComposeHint.Service == "" {
				return fmt.Errorf("container %d: empty Compose hint", i)
			}
			if err := validateBoundedString("Compose project", container.ComposeHint.Project, MaxDockerHintBytes); err != nil {
				return fmt.Errorf("container %d: %w", i, err)
			}
			if err := validateBoundedString("Compose service", container.ComposeHint.Service, MaxDockerHintBytes); err != nil {
				return fmt.Errorf("container %d: %w", i, err)
			}
		}
	}
	return nil
}

func isLowerHexString(value string) bool {
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func validateBoundedString(name, value string, limit int) error {
	if len(value) > limit {
		return fmt.Errorf("%s exceeds limit", name)
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("%s contains control characters", name)
		}
	}
	return nil
}
