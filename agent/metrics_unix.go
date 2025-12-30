//go:build !windows

package main

import (
	"bufio"
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func collectMetrics(ctx context.Context) MetricsResponse {
	var resp MetricsResponse
	var errorsList []string
	hasData := false

	if usage, err := readCPUUsage(ctx); err == nil {
		resp.CPUUsagePercent = usage
		hasData = true
	} else {
		errorsList = append(errorsList, "cpu: "+err.Error())
	}

	if memUsage, err := readMemUsage(); err == nil {
		resp.MemUsagePercent = memUsage
		hasData = true
	} else {
		errorsList = append(errorsList, "memory: "+err.Error())
	}

	if disks, err := readDiskUsage(); err == nil {
		resp.DiskUsage = disks
		hasData = true
	} else {
		errorsList = append(errorsList, "disk: "+err.Error())
	}

	if uptime, err := readUptime(); err == nil {
		resp.UptimeSeconds = uptime
		hasData = true
	} else {
		errorsList = append(errorsList, "uptime: "+err.Error())
	}

	resp.Errors = errorsList
	resp.OK = len(errorsList) == 0 || hasData
	return resp
}

func readCPUUsage(ctx context.Context) (float64, error) {
	firstIdle, firstTotal, err := readCPUSample()
	if err != nil {
		return 0, err
	}

	wait := time.After(120 * time.Millisecond)
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-wait:
	}

	secondIdle, secondTotal, err := readCPUSample()
	if err != nil {
		return 0, err
	}

	idleTicks := float64(secondIdle - firstIdle)
	totalTicks := float64(secondTotal - firstTotal)
	if totalTicks <= 0 {
		return 0, errors.New("invalid cpu totals")
	}

	usage := (1 - idleTicks/totalTicks) * 100
	if usage < 0 {
		usage = 0
	}
	return usage, nil
}

func readCPUSample() (uint64, uint64, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			return 0, 0, errors.New("unexpected cpu line")
		}

		var values []uint64
		for _, f := range fields[1:] {
			v, err := strconv.ParseUint(f, 10, 64)
			if err != nil {
				return 0, 0, err
			}
			values = append(values, v)
		}

		var total uint64
		for _, v := range values {
			total += v
		}

		idle := values[3]
		if len(values) > 4 {
			idle += values[4] // iowait
		}

		return idle, total, nil
	}

	return 0, 0, errors.New("cpu line not found")
}

func readMemUsage() (float64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	var memTotal, memAvailable uint64
	var memFree, buffers, cached uint64

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			memTotal, _ = parseMeminfoValue(line)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			memAvailable, _ = parseMeminfoValue(line)
		} else if strings.HasPrefix(line, "MemFree:") {
			memFree, _ = parseMeminfoValue(line)
		} else if strings.HasPrefix(line, "Buffers:") {
			buffers, _ = parseMeminfoValue(line)
		} else if strings.HasPrefix(line, "Cached:") {
			cached, _ = parseMeminfoValue(line)
		}

		if memTotal > 0 && memAvailable > 0 {
			break
		}
	}

	if memAvailable == 0 && memFree > 0 {
		memAvailable = memFree + buffers + cached
	}

	if memTotal == 0 || memAvailable == 0 {
		return 0, errors.New("meminfo missing fields")
	}

	used := memTotal - memAvailable
	usage := (float64(used) / float64(memTotal)) * 100
	if usage < 0 {
		usage = 0
	}
	return usage, nil
}

func parseMeminfoValue(line string) (uint64, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, errors.New("invalid meminfo line")
	}

	value, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, err
	}

	return value, nil
}

func readDiskUsage() ([]DiskUsage, error) {
	mountFile, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer mountFile.Close()

	scanner := bufio.NewScanner(mountFile)
	seen := make(map[string]bool)
	var disks []DiskUsage
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		mountPoint := parts[1]
		fsType := parts[2]
		if skipFilesystem(fsType) || seen[mountPoint] {
			continue
		}
		seen[mountPoint] = true

		var st syscall.Statfs_t
		if err := syscall.Statfs(mountPoint, &st); err != nil {
			continue
		}

		if st.Blocks == 0 {
			continue
		}

		total := float64(st.Blocks) * float64(st.Bsize)
		available := float64(st.Bavail) * float64(st.Bsize)
		used := total - available
		if total <= 0 {
			continue
		}

		usage := (used / total) * 100
		disks = append(disks, DiskUsage{
			Mount:        mountPoint,
			UsagePercent: usage,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(disks) == 0 {
		return nil, errors.New("no disk stats available")
	}

	return disks, nil
}

func skipFilesystem(fsType string) bool {
	switch fsType {
	case "proc", "sysfs", "tmpfs", "devtmpfs", "devpts", "cgroup", "cgroup2":
		return true
	default:
		return false
	}
}

func readUptime() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, errors.New("invalid uptime content")
	}

	val, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}

	if val < 0 {
		return 0, errors.New("negative uptime")
	}

	return val, nil
}
