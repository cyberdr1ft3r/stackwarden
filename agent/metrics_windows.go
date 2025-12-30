//go:build windows

package main

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func collectMetrics(ctx context.Context) MetricsResponse {
	var resp MetricsResponse
	var errorsList []string
	hasData := false

	if usage, err := readCPUUsageWindows(ctx); err == nil {
		resp.CPUUsagePercent = usage
		hasData = true
	} else {
		errorsList = append(errorsList, "cpu: "+err.Error())
	}

	if mem, err := readMemUsageWindows(ctx); err == nil {
		resp.MemUsagePercent = mem
		hasData = true
	} else {
		errorsList = append(errorsList, "memory: "+err.Error())
	}

	if disks, err := readDiskUsageWindows(ctx); err == nil {
		resp.DiskUsage = disks
		hasData = true
	} else {
		errorsList = append(errorsList, "disk: "+err.Error())
	}

	if uptime, err := readUptimeWindows(ctx); err == nil {
		resp.UptimeSeconds = uptime
		hasData = true
	} else {
		errorsList = append(errorsList, "uptime: "+err.Error())
	}

	resp.Errors = errorsList
	resp.OK = len(errorsList) == 0 || hasData
	return resp
}

func readCPUUsageWindows(ctx context.Context) (float64, error) {
	out, err := runCommand(ctx, "wmic", "cpu", "get", "loadpercentage", "/value")
	if err != nil {
		return 0, err
	}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "loadpercentage=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			v, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err != nil {
				continue
			}
			return v, nil
		}
	}

	return 0, errors.New("load percentage not found")
}

func readMemUsageWindows(ctx context.Context) (float64, error) {
	out, err := runCommand(ctx, "wmic", "OS", "get", "FreePhysicalMemory,TotalVisibleMemorySize", "/value")
	if err != nil {
		return 0, err
	}

	var free, total float64
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "freephysicalmemory=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				free, _ = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			}
		} else if strings.HasPrefix(strings.ToLower(line), "totalvisiblememorysize=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				total, _ = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			}
		}
	}

	if total <= 0 {
		return 0, errors.New("missing memory totals")
	}

	used := total - free
	if used < 0 {
		used = 0
	}

	return (used / total) * 100, nil
}

func readDiskUsageWindows(ctx context.Context) ([]DiskUsage, error) {
	out, err := runCommand(ctx, "wmic", "logicaldisk", "get", "name,freespace,size", "/format:csv")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out, "\n")
	var disks []DiskUsage
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(strings.ToLower(line), "node") {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) != 4 {
			continue
		}

		freeStr := strings.TrimSpace(parts[1])
		name := strings.TrimSpace(parts[2])
		sizeStr := strings.TrimSpace(parts[3])

		if name == "" {
			continue
		}

		free, _ := strconv.ParseFloat(freeStr, 64)
		size, _ := strconv.ParseFloat(sizeStr, 64)
		if size <= 0 {
			continue
		}

		used := size - free
		if used < 0 {
			used = 0
		}

		usage := (used / size) * 100
		disks = append(disks, DiskUsage{Mount: name, UsagePercent: usage})
	}

	if len(disks) == 0 {
		return nil, errors.New("no disk data")
	}

	return disks, nil
}

func readUptimeWindows(ctx context.Context) (float64, error) {
	out, err := runCommand(ctx, "wmic", "os", "get", "lastbootuptime", "/value")
	if err != nil {
		return 0, err
	}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "lastbootuptime=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			value := strings.TrimSpace(parts[1])
			if len(value) < 14 {
				continue
			}
			parsed, err := time.Parse("20060102150405", value[:14])
			if err != nil {
				continue
			}
			return time.Since(parsed).Seconds(), nil
		}
	}

	return 0, errors.New("boot time not found")
}

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
