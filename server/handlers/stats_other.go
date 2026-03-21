//go:build !linux

package handlers

import "runtime"

// getSystemMetrics returns system resource metrics using portable Go APIs.
// On non-Linux platforms, disk and system memory stats are unavailable, so we
// fall back to Go runtime memory stats and report 0 for disk.
func getSystemMetrics(_ string) systemMetrics {
	var m systemMetrics

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.MemoryTotalBytes = memStats.Sys
	m.MemoryUsedBytes = memStats.Alloc
	if m.MemoryTotalBytes > 0 {
		m.MemoryPercent = roundTo(float64(m.MemoryUsedBytes)/float64(m.MemoryTotalBytes)*100, 1)
	}

	// Disk stats are not available without platform-specific syscalls.

	return m
}
