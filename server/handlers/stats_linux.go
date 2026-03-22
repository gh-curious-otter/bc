//go:build linux

package handlers

import (
	"context"
	"syscall"
)

// getSystemMetrics returns system resource metrics using Linux-specific syscalls.
func getSystemMetrics(_ context.Context, rootDir string) systemMetrics {
	var m systemMetrics

	// System memory via Sysinfo
	var sysInfo syscall.Sysinfo_t
	if err := syscall.Sysinfo(&sysInfo); err == nil {
		m.MemoryTotalBytes = sysInfo.Totalram * uint64(sysInfo.Unit)
		freeRAM := sysInfo.Freeram * uint64(sysInfo.Unit)
		m.MemoryUsedBytes = m.MemoryTotalBytes - freeRAM
		if m.MemoryTotalBytes > 0 {
			m.MemoryPercent = roundTo(float64(m.MemoryUsedBytes)/float64(m.MemoryTotalBytes)*100, 1)
		}
	}

	// Disk usage via Statfs
	var statfs syscall.Statfs_t
	if err := syscall.Statfs(rootDir, &statfs); err == nil && statfs.Bsize > 0 {
		bsize := uint64(statfs.Bsize) //nolint:gosec // Bsize is always positive from the kernel
		m.DiskTotalBytes = statfs.Blocks * bsize
		diskFree := statfs.Bavail * bsize
		m.DiskUsedBytes = m.DiskTotalBytes - diskFree
		if m.DiskTotalBytes > 0 {
			m.DiskPercent = roundTo(float64(m.DiskUsedBytes)/float64(m.DiskTotalBytes)*100, 1)
		}
	}

	return m
}
