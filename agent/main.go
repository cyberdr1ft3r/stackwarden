package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	shared "github.com/m0b3u/stackwarden/pkg"
)

type Health struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type PortsResponse struct {
	shared.OperationResult
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Health{Status: "ok", Service: "agent"})
	})

	mux.HandleFunc("/ports", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		resp := collectPorts(ctx)
		_ = json.NewEncoder(w).Encode(resp)
	})

	addr := "127.0.0.1:9091"
	log.Printf("agent listening on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func collectPorts(ctx context.Context) PortsResponse {
	if runtime.GOOS == "windows" {
		return PortsResponse{OperationResult: shared.OperationResult{OK: false, Error: "ports listing is not supported on Windows"}}
	}

	commands := [][]string{
		{"ss", "-lntup"},
		{"netstat", "-lntup"},
		{"netstat", "-ano"},
	}

	var attempts []string
	var lastOutput string

	for _, cmdArgs := range commands {
		cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
		out, err := cmd.CombinedOutput()
		if len(out) > 0 {
			lastOutput = string(out)
		}
		if err == nil {
			return PortsResponse{OperationResult: shared.OperationResult{OK: true, Output: string(out)}}
		}

		attempts = append(attempts, cmdArgs[0]+": "+err.Error())
		if ctx.Err() != nil {
			break
		}
	}

	errorMsg := "failed to list ports"
	if len(attempts) > 0 {
		errorMsg += ": " + strings.Join(attempts, "; ")
	}

	return PortsResponse{OperationResult: shared.OperationResult{OK: false, Output: lastOutput, Error: errorMsg}}
}
