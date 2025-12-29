package main

import (
	"fmt"
	"strconv"
	"strings"

	shared "github.com/m0b3u/stackwarden/pkg"
)

// Port represents a listening socket.
type Port struct {
	Protocol     string `json:"protocol"`
	LocalAddress string `json:"local_address"`
	LocalPort    int    `json:"local_port"`
	PID          int    `json:"pid"`
	State        string `json:"state,omitempty"`
}

// PortsResponse is returned by the /ports handler.
type PortsResponse struct {
	shared.OperationResult
	Ports []Port `json:"ports,omitempty"`
}

func splitAddressPort(value string) (string, int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0, fmt.Errorf("empty address")
	}

	// Handle bracketed IPv6: [::]:443
	if strings.HasPrefix(value, "[") && strings.Contains(value, "]") {
		end := strings.LastIndex(value, "]")
		if end == -1 || end+1 >= len(value) {
			return "", 0, fmt.Errorf("invalid bracketed address")
		}
		host := strings.TrimPrefix(value[:end], "[")
		portPart := value[end+1:]
		portPart = strings.TrimPrefix(portPart, ":")
		port, err := strconv.Atoi(portPart)
		if err != nil {
			return "", 0, err
		}
		return host, port, nil
	}

	idx := strings.LastIndex(value, ":")
	if idx == -1 || idx == len(value)-1 {
		return "", 0, fmt.Errorf("missing port")
	}

	host := value[:idx]
	portStr := value[idx+1:]

	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = strings.Trim(host, "[]")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, err
	}

	if host == "" {
		host = "*"
	}

	return host, port, nil
}

func parsePIDField(value string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("empty pid")
	}

	if idx := strings.Index(value, "/"); idx != -1 {
		value = value[:idx]
	}

	return strconv.Atoi(value)
}

func extractPIDFromProcessField(value string) int {
	lower := strings.ToLower(value)
	idx := strings.Index(lower, "pid=")
	if idx == -1 {
		return 0
	}

	start := idx + len("pid=")
	end := start
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}

	if start == end {
		return 0
	}

	pid, err := strconv.Atoi(value[start:end])
	if err != nil {
		return 0
	}

	return pid
}

func parseWindowsNetstatOutput(output string) []Port {
	var ports []Port

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "proto") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		proto := strings.ToLower(fields[0])
		if proto != "tcp" && proto != "udp" {
			continue
		}

		localField := fields[1]
		state := ""
		pidField := fields[len(fields)-1]

		if proto == "tcp" {
			if len(fields) < 5 {
				continue
			}

			state = strings.ToUpper(fields[len(fields)-2])
			if state != "LISTENING" {
				continue
			}
		}

		addr, port, err := splitAddressPort(localField)
		if err != nil {
			continue
		}

		pid, err := parsePIDField(pidField)
		if err != nil {
			continue
		}

		ports = append(ports, Port{
			Protocol:     proto,
			LocalAddress: addr,
			LocalPort:    port,
			PID:          pid,
			State:        state,
		})
	}

	return ports
}
