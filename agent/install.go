package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"net/http"

	"github.com/m0b3u/stackwarden/pkg/tools"
)

const (
	installCommandTimeout = 60 * time.Second
	maxCommandOutputBytes = 16000
)

var toolsBaseDir = "/var/lib/stackwarden/tools"

type installResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Output  string `json:"output,omitempty"`
	Path    string `json:"path,omitempty"`
}

func handleToolInstall(w http.ResponseWriter, r *http.Request, toolID string) {
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

	var res installResult
	switch tool.InstallKind {
	case tools.InstallKindCompose:
		res = installCompose(ctx, tool)
	case tools.InstallKindLinuxCLI:
		res = installLinuxCLI(ctx, tool)
	case tools.InstallKindBundleOnly:
		res = installBundleOnly(tool)
	default:
		res = installResult{OK: false, Message: "install kind not supported"}
	}

	status := http.StatusOK
	if !res.OK {
		status = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(res)
}

func installCompose(ctx context.Context, tool tools.Tool) installResult {
	toolDir, err := ensureToolDir(tool.ID)
	if err != nil {
		return installResult{OK: false, Message: "failed to prepare tool directory", Output: err.Error(), Path: toolDir}
	}

	if err := stageTemplateFiles(tool, toolDir); err != nil {
		return installResult{OK: false, Message: "failed to write compose files", Output: err.Error(), Path: toolDir}
	}

	commands := composeCommands()
	if len(commands) == 0 {
		return installResult{OK: false, Message: "Docker is required for compose install (docker or docker-compose not found)", Path: toolDir}
	}

	var outputs []string
	for _, cmd := range commands {
		out, runErr := runCommand(ctx, toolDir, cmd.name, cmd.args...)
		outputs = append(outputs, fmt.Sprintf("%s %s\n%s", cmd.name, strings.Join(cmd.args, " "), out))
		if runErr == nil {
			return installResult{
				OK:      true,
				Message: "compose stack deployed",
				Output:  strings.Join(outputs, "\n---\n"),
				Path:    toolDir,
			}
		}
	}

	return installResult{
		OK:      false,
		Message: "failed to deploy compose stack",
		Output:  strings.Join(outputs, "\n---\n"),
		Path:    toolDir,
	}
}

func installLinuxCLI(ctx context.Context, tool tools.Tool) installResult {
	toolDir, err := ensureToolDir(tool.ID)
	if err != nil {
		return installResult{OK: false, Message: "failed to prepare tool directory", Output: err.Error(), Path: toolDir}
	}

	if err := stageTemplateFiles(tool, toolDir); err != nil {
		return installResult{OK: false, Message: "failed to write tool files", Output: err.Error(), Path: toolDir}
	}

	if runtime.GOOS != "linux" {
		return installResult{OK: false, Message: "linux_cli installs are supported only on Linux hosts", Path: toolDir}
	}

	switch tool.ID {
	case "ddev":
		return installDDEV(ctx, toolDir)
	default:
		return installResult{OK: false, Message: "install steps not defined for this tool", Path: toolDir}
	}
}

func installBundleOnly(tool tools.Tool) installResult {
	toolDir, err := ensureToolDir(tool.ID)
	if err != nil {
		return installResult{OK: false, Message: "failed to prepare tool directory", Output: err.Error(), Path: toolDir}
	}

	if err := stageTemplateFiles(tool, toolDir); err != nil {
		return installResult{OK: false, Message: "failed to stage bundle", Output: err.Error(), Path: toolDir}
	}

	return installResult{
		OK:      true,
		Message: "staged on server (no automated install yet)",
		Path:    toolDir,
	}
}

func installDDEV(ctx context.Context, toolDir string) installResult {
	if !isDebianLike() {
		return installResult{OK: false, Message: "DDEV install is supported on Debian/Ubuntu hosts only", Path: toolDir}
	}

	if !hasBinary("curl") && !hasBinary("wget") {
		return installResult{OK: false, Message: "curl or wget is required to install DDEV", Path: toolDir}
	}

	if hasBinary("apt-get") {
		res := installDDEVWithAPT(ctx, toolDir)
		if res.OK {
			return res
		}
	}

	return installDDEVWithDeb(ctx, toolDir)
}

func installDDEVWithAPT(ctx context.Context, toolDir string) installResult {
	useSudo := os.Geteuid() != 0
	if useSudo && !hasBinary("sudo") {
		return installResult{OK: false, Message: "root privileges (sudo) are required for apt-based install", Path: toolDir}
	}

	outputs := []string{}
	run := func(name string, args ...string) error {
		cmdName := name
		cmdArgs := args
		if useSudo {
			cmdArgs = append([]string{name}, args...)
			cmdName = "sudo"
		}
		out, err := runCommand(ctx, "", cmdName, cmdArgs...)
		entry := fmt.Sprintf("%s %s\n%s", cmdName, strings.Join(cmdArgs, " "), out)
		outputs = append(outputs, entry)
		return err
	}

	if err := run("apt-get", "update"); err != nil {
		return installResult{OK: false, Message: "apt update failed", Output: strings.Join(outputs, "\n---\n"), Path: toolDir}
	}

	if err := run("apt-get", "install", "-y", "ca-certificates", "curl", "wget", "gnupg"); err != nil {
		return installResult{OK: false, Message: "failed to install prerequisites", Output: strings.Join(outputs, "\n---\n"), Path: toolDir}
	}

	if err := run("install", "-m", "0755", "-d", "/etc/apt/keyrings"); err != nil {
		return installResult{OK: false, Message: "failed to prepare apt keyring directory", Output: strings.Join(outputs, "\n---\n"), Path: toolDir}
	}

	keyDest := filepath.Join(toolDir, "ddev.gpg")
	if err := downloadFile(ctx, "https://ddev.com/install/ddev.gpg", keyDest); err != nil {
		return installResult{OK: false, Message: "failed to download ddev gpg key", Output: err.Error(), Path: toolDir}
	}

	if err := run("gpg", "--dearmor", "-o", "/etc/apt/keyrings/ddev.gpg", keyDest); err != nil {
		return installResult{OK: false, Message: "failed to install ddev gpg key", Output: strings.Join(outputs, "\n---\n"), Path: toolDir}
	}

	repoLine := "deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://ddev.com/apt/ stable main\n"
	repoSrc := filepath.Join(toolDir, "ddev.list")
	if err := os.WriteFile(repoSrc, []byte(repoLine), 0o644); err != nil {
		return installResult{OK: false, Message: "failed to stage ddev apt repository file", Output: err.Error(), Path: toolDir}
	}
	if err := run("install", "-m", "0644", repoSrc, "/etc/apt/sources.list.d/ddev.list"); err != nil {
		return installResult{OK: false, Message: "failed to add ddev apt repository", Output: strings.Join(outputs, "\n---\n"), Path: toolDir}
	}

	if err := run("apt-get", "update"); err != nil {
		return installResult{OK: false, Message: "apt update failed after adding repository", Output: strings.Join(outputs, "\n---\n"), Path: toolDir}
	}

	if err := run("apt-get", "install", "-y", "ddev", "mkcert"); err != nil {
		return installResult{OK: false, Message: "failed to install ddev via apt", Output: strings.Join(outputs, "\n---\n"), Path: toolDir}
	}

	_ = run("mkcert", "-install")

	if ok, output := verifyDDEV(ctx); ok {
		outputs = append(outputs, output)
		return installResult{OK: true, Message: "DDEV installed via apt", Output: strings.Join(outputs, "\n---\n"), Path: toolDir}
	}

	return installResult{OK: false, Message: "DDEV install verification failed", Output: strings.Join(outputs, "\n---\n"), Path: toolDir}
}

func installDDEVWithDeb(ctx context.Context, toolDir string) installResult {
	if runtime.GOARCH != "amd64" {
		return installResult{OK: false, Message: "pinned .deb install is only available for amd64", Path: toolDir}
	}

	debURL := "https://github.com/ddev/ddev/releases/download/v1.23.4/ddev_linux_amd64.deb"
	debPath := filepath.Join(toolDir, "ddev_linux_amd64.deb")

	if err := downloadFile(ctx, debURL, debPath); err != nil {
		return installResult{OK: false, Message: "failed to download DDEV installer", Output: err.Error(), Path: toolDir}
	}

	if !hasBinary("dpkg") {
		return installResult{OK: false, Message: "dpkg is required to install the .deb package", Path: toolDir}
	}

	out, err := runCommand(ctx, "", "dpkg", "-i", debPath)
	if err != nil {
		return installResult{OK: false, Message: "failed to install DDEV via dpkg", Output: out, Path: toolDir}
	}

	if ok, output := verifyDDEV(ctx); ok {
		return installResult{OK: true, Message: "DDEV installed via pinned .deb", Output: output, Path: toolDir}
	}

	return installResult{OK: false, Message: "DDEV install verification failed", Path: toolDir}
}

func verifyDDEV(ctx context.Context) (bool, string) {
	out, err := runCommand(ctx, "", "ddev", "version")
	return err == nil, out
}

func isDebianLike() bool {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return false
	}
	content := strings.ToLower(string(data))
	return strings.Contains(content, "debian") || strings.Contains(content, "ubuntu")
}

func composeCommands() []struct {
	name string
	args []string
} {
	commands := []struct {
		name string
		args []string
	}{}

	if hasBinary("docker") {
		commands = append(commands, struct {
			name string
			args []string
		}{name: "docker", args: []string{"compose", "up", "-d"}})
	}

	if hasBinary("docker-compose") {
		commands = append(commands, struct {
			name string
			args []string
		}{name: "docker-compose", args: []string{"up", "-d"}})
	}

	return commands
}

func stageTemplateFiles(tool tools.Tool, destDir string) error {
	base, err := tools.TemplateBasePath(tool.ID)
	if err != nil {
		return err
	}

	return fs.WalkDir(tools.TemplateFS(), base, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel := strings.TrimPrefix(p, base+"/")
		if rel == "" {
			return nil
		}
		if strings.Contains(rel, "..") {
			return errors.New("invalid template path")
		}

		target := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, err := fs.ReadFile(tools.TemplateFS(), p)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		mode := fs.FileMode(0o644)
		if info, err := fs.Stat(tools.TemplateFS(), p); err == nil {
			mode = info.Mode() & 0o777
		}

		return os.WriteFile(target, data, mode)
	})
}

func ensureToolDir(toolID string) (string, error) {
	toolDir, err := managedToolDir(toolID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return toolDir, err
	}
	return toolDir, nil
}

func managedToolDir(toolID string) (string, error) {
	if !isValidToolID(toolID) {
		return "", errors.New("invalid tool id")
	}
	toolDir := filepath.Join(toolsBaseDir, toolID)
	cleanBase := filepath.Clean(toolsBaseDir)
	cleanTarget := filepath.Clean(toolDir)
	rel, err := filepath.Rel(cleanBase, cleanTarget)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("invalid tool path")
	}
	if info, err := os.Lstat(toolDir); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("managed tool directory cannot be a symlink")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return toolDir, nil
}

func runCommand(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	return truncateOutput(string(output)), err
}

func downloadFile(ctx context.Context, url, dest string) error {
	var cmdName string
	var args []string

	switch {
	case hasBinary("curl"):
		cmdName = "curl"
		args = []string{"-fsSL", url, "-o", dest}
	case hasBinary("wget"):
		cmdName = "wget"
		args = []string{"-q", "-O", dest, url}
	default:
		return errors.New("curl or wget is required")
	}

	out, err := runCommand(ctx, "", cmdName, args...)
	if err != nil {
		return fmt.Errorf("%s download failed: %s", cmdName, out)
	}
	return nil
}

func truncateOutput(out string) string {
	if len(out) <= maxCommandOutputBytes {
		return out
	}
	return out[:maxCommandOutputBytes] + "\n...[truncated]..."
}

func hasBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
