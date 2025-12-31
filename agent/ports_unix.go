//go:build !windows

package main

import (
	"context"
	"os/exec"
	"sort"
	"strings"

	shared "github.com/m0b3u/stackwarden/pkg"
)

func collectPorts(ctx context.Context) PortsResponse {
	var attempts []string
	var lastOutput string

	if ports, output, err := listPortsWithSS(ctx); err == nil {
		return PortsResponse{
			OperationResult: shared.OperationResult{
				OK:     true,
				Output: output,
			},
			Ports: ports,
		}
	} else {
		if output != "" {
			lastOutput = output
		}
		attempts = append(attempts, "ss: "+err.Error())
	}

	commands := []struct {
		args   []string
		parser func(string) []Port
	}{
		{[]string{"ss", "-lntup"}, parseSSOutput},
		{[]string{"netstat", "-tulpn"}, parseNetstatOutput},
		{[]string{"netstat", "-lntup"}, parseNetstatOutput},
	}

	for _, cmdCfg := range commands {
		cmd := exec.CommandContext(ctx, cmdCfg.args[0], cmdCfg.args[1:]...)
		out, err := cmd.CombinedOutput()
		if len(out) > 0 {
			lastOutput = string(out)
		}

		if err == nil {
			ports := cmdCfg.parser(lastOutput)
			return PortsResponse{
				OperationResult: shared.OperationResult{
					OK:     true,
					Output: lastOutput,
				},
				Ports: ports,
			}
		}

		attempts = append(attempts, cmdCfg.args[0]+": "+err.Error())
		if ctx.Err() != nil {
			break
		}
	}

	errorMsg := "failed to list ports"
	if len(attempts) > 0 {
		errorMsg += ": " + strings.Join(attempts, "; ")
	}

	return PortsResponse{
		OperationResult: shared.OperationResult{
			OK:     false,
			Output: lastOutput,
			Error:  errorMsg,
		},
	}
}

func listPortsWithSS(ctx context.Context) ([]Port, string, error) {
	cmd := exec.CommandContext(ctx, "ss", "-H", "-ltn")
	out, err := cmd.CombinedOutput()
	output := string(out)
	if err != nil {
		if len(output) == 0 {
			output = err.Error()
		}
		return nil, output, err
	}

	ports := parseSSLtnOutput(output)
	return ports, output, nil
}

func parseSSLtnOutput(output string) []Port {
	var ports []Port
	seenPorts := make(map[int]Port)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "State") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		localField := fields[3]
		addr, port, err := splitAddressPort(localField)
		if err != nil {
			continue
		}

		if _, exists := seenPorts[port]; exists {
			continue
		}

		state := strings.ToUpper(fields[0])
		seenPorts[port] = Port{
			Protocol:     "tcp",
			LocalAddress: addr,
			LocalPort:    port,
			State:        state,
		}
	}

	for _, p := range seenPorts {
		ports = append(ports, p)
	}

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].LocalPort < ports[j].LocalPort
	})

	return ports
}

func parseSSOutput(output string) []Port {
	var ports []Port
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Netid") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		proto := strings.ToLower(fields[0])
		if proto != "tcp" && !strings.HasPrefix(proto, "udp") {
			continue
		}

		state := strings.ToUpper(fields[1])
		if strings.HasPrefix(proto, "tcp") && state != "LISTEN" {
			continue
		}

		localField := fields[4]
		addr, port, err := splitAddressPort(localField)
		if err != nil {
			continue
		}

		processField := strings.Join(fields[5:], " ")
		pid := extractPIDFromProcessField(processField)
		if pid == 0 {
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

func parseNetstatOutput(output string) []Port {
	var ports []Port
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(strings.ToLower(line), "proto") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		proto := strings.ToLower(fields[0])
		if proto != "tcp" && proto != "udp" {
			continue
		}

		localField := fields[3]
		state := ""

		if proto == "tcp" {
			state = strings.ToUpper(fields[len(fields)-2])
			if state != "LISTEN" && state != "LISTENING" {
				continue
			}
		}

		addr, port, err := splitAddressPort(localField)
		if err != nil {
			continue
		}

		pidField := fields[len(fields)-1]
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
