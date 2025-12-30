package main

import shared "github.com/m0b3u/stackwarden/pkg"

// DiskUsage represents usage information for a mounted filesystem.
type DiskUsage struct {
	Mount        string  `json:"mount"`
	UsagePercent float64 `json:"usage_percent"`
}

// MetricsResponse is returned by the /metrics handler.
type MetricsResponse struct {
	shared.OperationResult
	CPUUsagePercent float64     `json:"cpu_usage_percent,omitempty"`
	MemUsagePercent float64     `json:"mem_usage_percent,omitempty"`
	DiskUsage       []DiskUsage `json:"disk_usage,omitempty"`
	UptimeSeconds   float64     `json:"uptime_seconds,omitempty"`
	Errors          []string    `json:"errors,omitempty"`
}

func (m MetricsResponse) hasAnyMetric() bool {
	return m.CPUUsagePercent > 0 || m.MemUsagePercent > 0 || len(m.DiskUsage) > 0 || m.UptimeSeconds > 0
}
