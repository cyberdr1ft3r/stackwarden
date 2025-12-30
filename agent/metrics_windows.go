//go:build windows

package main

import (
	"context"
	"encoding/json"
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
		resp.MemUsagePercent = mem.UsagePercent
		resp.MemTotalBytes = mem.TotalBytes
		resp.MemUsedBytes = mem.UsedBytes
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
	sample, err := readCPUUsageWithTypeperf(ctx)
	if err == nil {
		return sample, nil
	}

	if sample, errPS := readCPUUsageWithPowershell(ctx); errPS == nil {
		return sample, nil
	}

	return 0, errors.New("load percentage not found")
}

type windowsMem struct {
	UsagePercent float64
	TotalBytes   uint64
	UsedBytes    uint64
}

func readMemUsageWindows(ctx context.Context) (windowsMem, error) {
	cmd := `(Get-CimInstance Win32_OperatingSystem | Select-Object -Property TotalVisibleMemorySize,FreePhysicalMemory | ConvertTo-Json -Compress)`
	out, err := runCommand(ctx, "powershell", "-Command", cmd)
	if err != nil {
		return windowsMem{}, err
	}

	type memInfo struct {
		TotalVisibleMemorySize uint64 `json:"TotalVisibleMemorySize"`
		FreePhysicalMemory     uint64 `json:"FreePhysicalMemory"`
	}

	var info memInfo
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		return windowsMem{}, err
	}

	if info.TotalVisibleMemorySize == 0 {
		return windowsMem{}, errors.New("missing memory totals")
	}

	usedKB := info.TotalVisibleMemorySize - info.FreePhysicalMemory
	if info.FreePhysicalMemory > info.TotalVisibleMemorySize {
		usedKB = 0
	}
	totalBytes := info.TotalVisibleMemorySize * 1024
	usedBytes := usedKB * 1024

	usage := (float64(usedKB) / float64(info.TotalVisibleMemorySize)) * 100
	if usage < 0 {
		usage = 0
	}

	return windowsMem{
		UsagePercent: usage,
		TotalBytes:   totalBytes,
		UsedBytes:    usedBytes,
	}, nil
}

func readDiskUsageWindows(ctx context.Context) ([]DiskUsage, error) {
	cmd := `(Get-CimInstance Win32_LogicalDisk -Filter "DriveType=3" | Select-Object DeviceID,Size,FreeSpace | ConvertTo-Json -Compress)`
	out, err := runCommand(ctx, "powershell", "-Command", cmd)
	if err != nil {
		return nil, err
	}

	var raw json.RawMessage
	dec := json.NewDecoder(strings.NewReader(out))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}

	// Output may be object or array depending on single/multiple disks.
	if len(raw) == 0 {
		return nil, errors.New("no disk data")
	}

	var disks []map[string]interface{}
	if raw[0] == '[' {
		if err := json.Unmarshal(raw, &disks); err != nil {
			return nil, err
		}
	} else {
		var single map[string]interface{}
		if err := json.Unmarshal(raw, &single); err != nil {
			return nil, err
		}
		disks = append(disks, single)
	}

	var results []DiskUsage
	for _, d := range disks {
		mount, _ := d["DeviceID"].(string)
		if mount == "" {
			continue
		}

		sizeNum, _ := d["Size"].(json.Number)
		freeNum, _ := d["FreeSpace"].(json.Number)

		sizeFloat, _ := sizeNum.Float64()
		freeFloat, _ := freeNum.Float64()
		if sizeFloat <= 0 {
			continue
		}

		used := sizeFloat - freeFloat
		if used < 0 {
			used = 0
		}

		usage := (used / sizeFloat) * 100
		results = append(results, DiskUsage{Mount: mount, UsagePercent: usage})
	}

	if len(results) == 0 {
		return nil, errors.New("no disk data")
	}

	return results, nil
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

func readCPUUsageWithTypeperf(ctx context.Context) (float64, error) {
	out, err := runCommand(ctx, "typeperf", `\Processor(_Total)\% Processor Time`, "-sc", "1")
	if err != nil {
		return 0, err
	}

	return parseTypeperfValue(out)
}

func readCPUUsageWithPowershell(ctx context.Context) (float64, error) {
	cmd := `Get-Counter '\Processor(_Total)\% Processor Time' -SampleInterval 1 -MaxSamples 1 | Select -ExpandProperty CounterSamples | Select -First 1 | Select -ExpandProperty CookedValue`
	out, err := runCommand(ctx, "powershell", "-Command", cmd)
	if err != nil {
		return 0, err
	}

	valueStr := strings.TrimSpace(out)
	valueStr = strings.Trim(valueStr, "\"")
	valueStr = strings.ReplaceAll(valueStr, ",", "")
	return strconv.ParseFloat(valueStr, 64)
}

func parseTypeperfValue(output string) (float64, error) {
	lines := strings.Split(output, "\n")
	var lastVal float64
	found := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(strings.ToLower(line), "pdh-csv") {
			continue
		}

		parts := strings.Split(line, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			part = strings.Trim(part, "\"")
			part = strings.ReplaceAll(part, ",", "")
			if v, err := strconv.ParseFloat(part, 64); err == nil {
				lastVal = v
				found = true
			}
		}
	}

	if !found {
		return 0, errors.New("no cpu samples")
	}

	return lastVal, nil
}
