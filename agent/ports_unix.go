//go:build !windows

package main

import (
	"context"
	"os/exec"
	"strings"

	shared "github.com/m0b3u/stackwarden/pkg"
)

func collectPorts(ctx context.Context) PortsResponse {
	commands := []struct {
		args   []string
		parser func(string) []Port
	}{
		{[]string{"ss", "-lntup"}, parseSSOutput},
		{[]string{"netstat", "-lntup"}, parseNetstatOutput},
	}

	var attempts []string
	var lastOutput string

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
