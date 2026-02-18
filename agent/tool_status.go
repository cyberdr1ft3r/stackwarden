package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/m0b3u/stackwarden/pkg/tools"
)

type ToolStatus struct {
	ID          string            `json:"id"`
	InstallKind tools.InstallKind `json:"install_kind"`
	Staged      bool              `json:"staged"`
	Installed   bool              `json:"installed"`
	Running     bool              `json:"running"`
	Version     string            `json:"version,omitempty"`
	Details     *ToolStatusDetail `json:"details,omitempty"`
	CheckedAt   string            `json:"checked_at"`
	Errors      []string          `json:"errors,omitempty"`
}

type ToolStatusDetail struct {
	Compose *ComposeStatusDetail `json:"compose,omitempty"`
	CLI     *CLIStatusDetail     `json:"cli,omitempty"`
}

type ComposeStatusDetail struct {
	Containers []string `json:"containers,omitempty"`
	Project    string   `json:"project,omitempty"`
	PSRaw      string   `json:"ps_raw,omitempty"`
}

type CLIStatusDetail struct {
	Path       string `json:"path,omitempty"`
	VersionRaw string `json:"version_raw,omitempty"`
}

type uninstallResult struct {
	ID               string   `json:"id"`
	Uninstalled      bool     `json:"uninstalled"`
	Output           string   `json:"output,omitempty"`
	Warnings         []string `json:"warnings,omitempty"`
	RemovedStagedDir bool     `json:"removed_staged_dir"`
}

type composeCommand struct {
	name string
	args []string
}

func collectToolStatus(ctx context.Context, tool tools.Tool, runner Runner) ToolStatus {
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	status := ToolStatus{
		ID:          tool.ID,
		InstallKind: tool.InstallKind,
		CheckedAt:   checkedAt,
	}

	toolDir := filepath.Join(toolsBaseDir, tool.ID)
	status.Staged = dirExists(toolDir)

	switch tool.InstallKind {
	case tools.InstallKindCompose:
		status.Installed, status.Running = composeStatus(ctx, tool, toolDir, runner, &status)
	case tools.InstallKindLinuxCLI:
		installed, version := linuxCLIStatus(ctx, tool, runner, &status)
		status.Installed = installed
		status.Version = version
	case tools.InstallKindBundleOnly:
		status.Installed = status.Staged
		status.Running = false
	default:
		status.Errors = append(status.Errors, "unsupported install kind")
	}

	return status
}

func uninstallTool(ctx context.Context, tool tools.Tool, runner Runner) uninstallResult {
	result := uninstallResult{ID: tool.ID}
	var outputs []string
	var warnings []string

	toolDir := filepath.Join(toolsBaseDir, tool.ID)

	switch tool.InstallKind {
	case tools.InstallKindCompose:
		out, warn, ok := uninstallCompose(ctx, toolDir, runner)
		if out != "" {
			outputs = append(outputs, out)
		}
		warnings = append(warnings, warn...)
		result.Uninstalled = ok
	case tools.InstallKindLinuxCLI:
		out, warn, ok := uninstallLinuxCLI(ctx, tool, runner)
		if out != "" {
			outputs = append(outputs, out)
		}
		warnings = append(warnings, warn...)
		result.Uninstalled = ok
	case tools.InstallKindBundleOnly:
		result.Uninstalled = true
	default:
		warnings = append(warnings, "unsupported install kind")
		result.Uninstalled = false
	}

	if err := os.RemoveAll(toolDir); err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to remove staged dir: %v", err))
		result.RemovedStagedDir = false
		result.Uninstalled = false
	} else {
		result.RemovedStagedDir = true
	}

	result.Output = truncateOutput(strings.Join(outputs, "\n---\n"))
	if len(warnings) > 0 {
		result.Warnings = warnings
	}

	return result
}

func composeStatus(ctx context.Context, tool tools.Tool, toolDir string, runner Runner, status *ToolStatus) (bool, bool) {
	composeFile, ok := findComposeFile(toolDir)
	installed := status.Staged && ok
	if !installed {
		return false, false
	}

	detail := &ComposeStatusDetail{Project: tool.ID}
	if status.Details == nil {
		status.Details = &ToolStatusDetail{}
	}
	status.Details.Compose = detail

	commands := composeCommandCandidates(composeFile, "ps")
	if len(commands) == 0 {
		status.Errors = append(status.Errors, "docker compose not available")
		return true, false
	}

	var outputs []string
	for _, cmd := range commands {
		stdout, stderr, _, err := runner.Run(ctx, append([]string{cmd.name}, cmd.args...), toolDir)
		output := strings.TrimSpace(strings.TrimSpace(stdout + "\n" + stderr))
		if output != "" {
			outputs = append(outputs, fmt.Sprintf("%s %s\n%s", cmd.name, strings.Join(cmd.args, " "), output))
		}
		if err == nil {
			detail.PSRaw = truncateOutput(strings.Join(outputs, "\n---\n"))
			running, containers := parseComposePS(output)
			detail.Containers = containers
			return true, running
		}
		status.Errors = append(status.Errors, fmt.Sprintf("compose ps failed: %v", err))
	}

	detail.PSRaw = truncateOutput(strings.Join(outputs, "\n---\n"))
	return true, false
}

func linuxCLIStatus(ctx context.Context, tool tools.Tool, runner Runner, status *ToolStatus) (bool, string) {
	spec := tool.Status
	var installed bool
	var path string

	if len(spec.CheckCmd) > 0 {
		stdout, stderr, code, err := runner.Run(ctx, spec.CheckCmd, "")
		if err == nil && code == 0 {
			installed = true
			path = strings.TrimSpace(stdout)
			if path == "" {
				path = strings.TrimSpace(stderr)
			}
		}
	} else if spec.Binary != "" {
		cmd := []string{"which", spec.Binary}
		stdout, _, code, err := runner.Run(ctx, cmd, "")
		if err == nil && code == 0 {
			installed = true
			path = strings.TrimSpace(stdout)
		}
	}

	var version string
	if installed && len(spec.VersionCmd) > 0 {
		stdout, stderr, _, err := runner.Run(ctx, spec.VersionCmd, "")
		raw := strings.TrimSpace(stdout)
		if raw == "" {
			raw = strings.TrimSpace(stderr)
		}
		if raw != "" {
			if status.Details == nil {
				status.Details = &ToolStatusDetail{}
			}
			status.Details.CLI = &CLIStatusDetail{Path: path, VersionRaw: raw}
		}
		if err == nil {
			version = extractVersion(raw, spec.VersionRegex, status)
		} else {
			status.Errors = append(status.Errors, fmt.Sprintf("version command failed: %v", err))
		}
	}

	if installed && status.Details == nil {
		status.Details = &ToolStatusDetail{CLI: &CLIStatusDetail{Path: path}}
	} else if installed && status.Details != nil && status.Details.CLI != nil && status.Details.CLI.Path == "" {
		status.Details.CLI.Path = path
	}

	return installed, version
}

func uninstallCompose(ctx context.Context, toolDir string, runner Runner) (string, []string, bool) {
	composeFile, ok := findComposeFile(toolDir)
	if !ok {
		return "", nil, true
	}

	commands := composeCommandCandidates(composeFile, "down")
	if len(commands) == 0 {
		return "", []string{"docker compose not available"}, false
	}

	var outputs []string
	var warnings []string
	for _, cmd := range commands {
		stdout, stderr, _, err := runner.Run(ctx, append([]string{cmd.name}, cmd.args...), toolDir)
		output := strings.TrimSpace(strings.TrimSpace(stdout + "\n" + stderr))
		if output != "" {
			outputs = append(outputs, fmt.Sprintf("%s %s\n%s", cmd.name, strings.Join(cmd.args, " "), output))
		}
		if err == nil {
			return truncateOutput(strings.Join(outputs, "\n---\n")), warnings, true
		}
		warnings = append(warnings, fmt.Sprintf("compose down failed: %v", err))
	}

	return truncateOutput(strings.Join(outputs, "\n---\n")), warnings, false
}

func uninstallLinuxCLI(ctx context.Context, tool tools.Tool, runner Runner) (string, []string, bool) {
	spec := tool.Uninstall
	if len(spec.UninstallCmds) == 0 {
		return "", []string{"no uninstall commands defined"}, true
	}

	var outputs []string
	var warnings []string
	ok := true

	for _, cmd := range spec.UninstallCmds {
		if len(cmd) == 0 {
			continue
		}
		stdout, stderr, _, err := runner.Run(ctx, cmd, "")
		output := strings.TrimSpace(strings.TrimSpace(stdout + "\n" + stderr))
		if output != "" {
			outputs = append(outputs, fmt.Sprintf("%s\n%s", strings.Join(cmd, " "), output))
		}
		if err != nil {
			ok = false
			warnings = append(warnings, fmt.Sprintf("uninstall command failed: %v", err))
		}
	}

	return truncateOutput(strings.Join(outputs, "\n---\n")), warnings, ok
}

func findComposeFile(toolDir string) (string, bool) {
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	for _, name := range candidates {
		path := filepath.Join(toolDir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}

	return "", false
}

func composeCommandCandidates(composeFile string, args ...string) []composeCommand {
	commands := []composeCommand{}

	if hasBinary("docker") {
		commands = append(commands, composeCommand{
			name: "docker",
			args: append([]string{"compose", "-f", composeFile}, args...),
		})
	}

	if hasBinary("docker-compose") {
		commands = append(commands, composeCommand{
			name: "docker-compose",
			args: append([]string{"-f", composeFile}, args...),
		})
	}

	return commands
}

func parseComposePS(output string) (bool, []string) {
	lines := strings.Split(output, "\n")
	var containers []string
	running := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "name") || strings.HasPrefix(lower, "service") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		containers = append(containers, fields[0])
		if strings.Contains(line, "Up") || strings.Contains(lower, "running") {
			running = true
		}
	}

	return running, containers
}

func extractVersion(raw, regex string, status *ToolStatus) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if regex == "" {
		return strings.Fields(raw)[0]
	}

	re, err := regexp.Compile(regex)
	if err != nil {
		status.Errors = append(status.Errors, fmt.Sprintf("invalid version regex: %v", err))
		return strings.Fields(raw)[0]
	}

	match := re.FindStringSubmatch(raw)
	if len(match) > 1 {
		return match[1]
	}
	if len(match) == 1 {
		return match[0]
	}

	return strings.Fields(raw)[0]
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
