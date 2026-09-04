package main

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/m0b3u/stackwarden/pkg/tools"
)

const statusCommandTimeout = 30 * time.Second

func toolActionHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/tools/"), "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}

	toolID := parts[0]
	if !isValidToolID(toolID) {
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}
	action := parts[1]

	switch action {
	case "install":
		handleToolInstall(w, r, toolID)
	case "status":
		handleToolStatus(w, r, toolID)
	case "uninstall":
		handleToolUninstall(w, r, toolID)
	default:
		http.NotFound(w, r)
	}
}

func isValidToolID(toolID string) bool {
	if toolID == "" || strings.Contains(toolID, "..") || filepath.IsAbs(toolID) {
		return false
	}
	for _, c := range toolID {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			continue
		}
		return false
	}
	return true
}

func handleToolStatus(w http.ResponseWriter, r *http.Request, toolID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tool, err := tools.Find(toolID)
	if err != nil {
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), statusCommandTimeout)
	defer cancel()

	status := collectToolStatus(ctx, tool, execRunner{})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func handleToolUninstall(w http.ResponseWriter, r *http.Request, toolID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tool, err := tools.Find(toolID)
	if err != nil {
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), installCommandTimeout)
	defer cancel()

	res := uninstallTool(ctx, tool, execRunner{})

	status := http.StatusOK
	if !res.Uninstalled {
		status = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(res)
}
