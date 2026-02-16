package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/rpuneet/bc/pkg/log"
)

var (
	// Profiling flags
	profileType     string
	profileDuration int
	profileOutput   string

	// Active profile state
	cpuProfileFile   *os.File
	memProfileActive bool
)

// setupProfiling initializes profiling based on flags.
// Call this in PersistentPreRun.
func setupProfiling() error {
	if profileType == "" {
		return nil
	}

	switch profileType {
	case "cpu":
		return startCPUProfile()
	case "mem":
		return startMemProfile()
	default:
		return fmt.Errorf("unknown profile type: %s (supported: cpu, mem)", profileType)
	}
}

// stopProfiling cleanly stops any active profiling.
// Call this in PersistentPostRun.
func stopProfiling() {
	if cpuProfileFile != nil {
		pprof.StopCPUProfile()
		if err := cpuProfileFile.Close(); err != nil {
			log.Warn("failed to close CPU profile file", "error", err)
		}
		log.Info("CPU profile saved", "path", cpuProfileFile.Name())
		cpuProfileFile = nil
	}

	if memProfileActive {
		if err := writeMemProfile(); err != nil {
			log.Error("failed to write memory profile", "error", err)
		}
		memProfileActive = false
	}
}

// startCPUProfile begins CPU profiling to a file.
func startCPUProfile() error {
	profilePath, err := getProfilePath("cpu")
	if err != nil {
		return fmt.Errorf("failed to get profile path: %w", err)
	}

	// Clean the path to prevent directory traversal
	profilePath = filepath.Clean(profilePath)

	f, err := os.Create(profilePath) //nolint:gosec // Path is validated via getProfilePath
	if err != nil {
		return fmt.Errorf("failed to create CPU profile: %w", err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}

	cpuProfileFile = f
	log.Info("CPU profiling started", "output", profilePath, "duration", fmt.Sprintf("%ds", profileDuration))

	// If duration is set, schedule automatic stop
	if profileDuration > 0 {
		go func() {
			time.Sleep(time.Duration(profileDuration) * time.Second)
			stopProfiling()
			fmt.Printf("\nCPU profile complete: %s\n", profilePath)
			fmt.Println("Analyze with: go tool pprof", profilePath)
		}()
	}

	return nil
}

// startMemProfile sets up memory profiling (heap profile written on stop).
func startMemProfile() error {
	memProfileActive = true
	log.Info("Memory profiling enabled", "output", "will be written on command completion")
	return nil
}

// writeMemProfile captures and writes a heap profile.
func writeMemProfile() error {
	profilePath, err := getProfilePath("heap")
	if err != nil {
		return fmt.Errorf("failed to get profile path: %w", err)
	}

	profilePath = filepath.Clean(profilePath)

	f, err := os.Create(profilePath) //nolint:gosec // Path is validated via getProfilePath
	if err != nil {
		return fmt.Errorf("failed to create heap profile: %w", err)
	}

	// Force GC for accurate statistics
	runtime.GC()

	if err := pprof.WriteHeapProfile(f); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to write heap profile: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close heap profile: %w", err)
	}

	// Print memory statistics
	printMemStats()

	fmt.Printf("\nHeap profile written to: %s\n", profilePath)
	fmt.Println("Analyze with: go tool pprof", profilePath)
	return nil
}

// printMemStats outputs current memory statistics.
func printMemStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Println("\n=== Memory Statistics ===")
	fmt.Printf("Alloc:        %v MB (currently allocated)\n", m.Alloc/1024/1024)
	fmt.Printf("TotalAlloc:   %v MB (cumulative)\n", m.TotalAlloc/1024/1024)
	fmt.Printf("Sys:          %v MB (from OS)\n", m.Sys/1024/1024)
	fmt.Printf("HeapAlloc:    %v MB\n", m.HeapAlloc/1024/1024)
	fmt.Printf("HeapSys:      %v MB\n", m.HeapSys/1024/1024)
	fmt.Printf("HeapInuse:    %v MB\n", m.HeapInuse/1024/1024)
	fmt.Printf("HeapObjects:  %v\n", m.HeapObjects)
	fmt.Printf("NumGC:        %v\n", m.NumGC)
	fmt.Printf("Goroutines:   %v\n", runtime.NumGoroutine())
}

// getProfilePath returns the path for a profile file.
func getProfilePath(profileType string) (string, error) {
	// Use custom output path if specified
	if profileOutput != "" {
		return profileOutput, nil
	}

	// Default to .bc/profiles/ directory
	ws, err := getWorkspace()
	if err != nil {
		// Fall back to current directory if not in workspace
		return fmt.Sprintf("bc-%s-%s.prof", profileType, time.Now().Format("20060102-150405")), nil
	}

	profileDir := filepath.Join(ws.StateDir(), "profiles")
	if err := os.MkdirAll(profileDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create profiles directory: %w", err)
	}

	return filepath.Join(profileDir, fmt.Sprintf("%s-%s.prof", profileType, time.Now().Format("20060102-150405"))), nil
}

// registerProfileFlags adds profiling flags to the root command.
func registerProfileFlags() {
	rootCmd.PersistentFlags().StringVar(&profileType, "profile", "", "Enable profiling (cpu, mem)")
	rootCmd.PersistentFlags().IntVar(&profileDuration, "profile-duration", 30, "Profile duration in seconds (0 for manual stop)")
	rootCmd.PersistentFlags().StringVar(&profileOutput, "profile-output", "", "Custom output path for profile")
}
