package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetupProfiling_NoProfile(t *testing.T) {
	// Reset state
	profileType = ""
	profileDuration = 30
	profileOutput = ""
	cpuProfileFile = nil

	err := setupProfiling()
	if err != nil {
		t.Errorf("setupProfiling() with no profile type should not error: %v", err)
	}
	if cpuProfileFile != nil {
		t.Error("cpuProfileFile should be nil when no profile type specified")
	}
}

func TestSetupProfiling_InvalidType(t *testing.T) {
	profileType = "invalid"
	profileDuration = 30
	profileOutput = ""
	cpuProfileFile = nil

	err := setupProfiling()
	if err == nil {
		t.Error("setupProfiling() with invalid profile type should error")
	}

	// Reset
	profileType = ""
}

func TestSetupProfiling_CPUProfile(t *testing.T) {
	// Create temp directory for profile output
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test-cpu.prof")

	profileType = "cpu"
	profileDuration = 0 // Don't auto-stop
	profileOutput = outputPath
	cpuProfileFile = nil

	err := setupProfiling()
	if err != nil {
		t.Fatalf("setupProfiling() failed: %v", err)
	}

	// Verify profile file was created
	if cpuProfileFile == nil {
		t.Fatal("cpuProfileFile should not be nil after starting CPU profile")
	}

	// Verify file exists
	if _, statErr := os.Stat(outputPath); os.IsNotExist(statErr) {
		t.Error("profile file should exist")
	}

	// Stop profiling
	stopProfiling()

	// Verify cpuProfileFile is nil after stop
	if cpuProfileFile != nil {
		t.Error("cpuProfileFile should be nil after stopProfiling()")
	}

	// Verify file is readable (valid profile)
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("failed to stat profile file: %v", err)
	}
	if info.Size() == 0 {
		t.Error("profile file should not be empty")
	}

	// Reset
	profileType = ""
	profileOutput = ""
}

func TestStopProfiling_NoActiveProfile(t *testing.T) {
	cpuProfileFile = nil
	// Should not panic
	stopProfiling()
}

func TestGetProfilePath_CustomOutput(t *testing.T) {
	profileOutput = "/custom/path/profile.prof"
	defer func() { profileOutput = "" }()

	path, err := getProfilePath("cpu")
	if err != nil {
		t.Errorf("getProfilePath() should not error: %v", err)
	}
	if path != "/custom/path/profile.prof" {
		t.Errorf("getProfilePath() = %q, want %q", path, "/custom/path/profile.prof")
	}
}

func TestGetProfilePath_DefaultNaming(t *testing.T) {
	profileOutput = ""

	path, err := getProfilePath("cpu")
	if err != nil {
		t.Errorf("getProfilePath() should not error: %v", err)
	}

	// Path should contain "cpu" and ".prof"
	if filepath.Ext(path) != ".prof" {
		t.Errorf("profile path should have .prof extension: %s", path)
	}
}

func TestRegisterProfileFlags(t *testing.T) {
	// Flags should already be registered from init()
	// Just verify they exist
	profileFlag := rootCmd.PersistentFlags().Lookup("profile")
	if profileFlag == nil {
		t.Fatal("--profile flag should be registered")
	}

	durationFlag := rootCmd.PersistentFlags().Lookup("profile-duration")
	if durationFlag == nil {
		t.Fatal("--profile-duration flag should be registered")
	}
	if durationFlag.DefValue != "30" {
		t.Errorf("--profile-duration default should be 30, got %s", durationFlag.DefValue)
	}

	outputFlag := rootCmd.PersistentFlags().Lookup("profile-output")
	if outputFlag == nil {
		t.Fatal("--profile-output flag should be registered")
	}
}
