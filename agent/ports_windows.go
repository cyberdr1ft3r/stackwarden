//go:build windows

package main

import (
	"context"
	"os/exec"

	shared "github.com/m0b3u/stackwarden/pkg"
)

func collectPorts(ctx context.Context) PortsResponse {
	cmd := exec.CommandContext(ctx, "netstat", "-ano")
	out, err := cmd.CombinedOutput()
	raw := string(out)

	ports := parseWindowsNetstatOutput(raw)
	ok := err == nil
	errorMsg := ""

	if err != nil {
		errorMsg = err.Error()
	}

	// If we were able to parse ports, consider the call successful even if netstat returned an error.
	if len(ports) > 0 {
		ok = true
	}

	return PortsResponse{
		OperationResult: shared.OperationResult{
			OK:     ok,
			Output: raw,
			Error:  errorMsg,
		},
		Ports: ports,
	}
}
