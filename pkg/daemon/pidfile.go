// Package daemon provides daemon management utilities for bcd.
package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Info holds runtime information about a running daemon.
type Info struct {
	StartedAt time.Time `json:"started_at"`
	Addr      string    `json:"addr"`
	PID       int       `json:"pid"`
}

const (
	pidFileName  = "bcd.pid"
	infoFileName = "bcd.json"
)

// PIDPath returns the full path to the PID file.
func PIDPath(stateDir string) string {
	return filepath.Join(stateDir, pidFileName)
}

// WritePID writes the given PID to .bc/bcd.pid.
func WritePID(stateDir string, pid int) error {
	path := PIDPath(stateDir)
	data := []byte(strconv.Itoa(pid))
	return os.WriteFile(path, data, 0600)
}

// ReadPID reads the PID from .bc/bcd.pid.
func ReadPID(stateDir string) (int, error) {
	path := PIDPath(stateDir)
	// #nosec G304 - path is constructed from workspace state directory
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read pid file: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse pid: %w", err)
	}
	return pid, nil
}

// RemovePID removes the PID file.
func RemovePID(stateDir string) error {
	return os.Remove(PIDPath(stateDir))
}

// IsRunning reads the PID file and checks if the process is alive.
func IsRunning(stateDir string) bool {
	pid, err := ReadPID(stateDir)
	if err != nil {
		return false
	}
	// Signal 0 checks if process exists without sending a signal.
	return syscall.Kill(pid, 0) == nil
}

// infoPath returns the full path to the daemon info file.
func infoPath(stateDir string) string {
	return filepath.Join(stateDir, infoFileName)
}

// WriteInfo writes daemon info (PID, addr, start time) to .bc/bcd.json.
func WriteInfo(stateDir, addr string) error {
	info := Info{
		PID:       os.Getpid(),
		Addr:      addr,
		StartedAt: time.Now(),
	}
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("marshal daemon info: %w", err)
	}
	return os.WriteFile(infoPath(stateDir), data, 0600)
}

// ReadInfo reads daemon info from .bc/bcd.json.
func ReadInfo(stateDir string) (*Info, error) {
	path := infoPath(stateDir)
	// #nosec G304 - path is constructed from workspace state directory
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read daemon info: %w", err)
	}
	var info Info
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parse daemon info: %w", err)
	}
	return &info, nil
}

// RemoveInfo removes the daemon info file.
func RemoveInfo(stateDir string) error {
	return os.Remove(infoPath(stateDir))
}
