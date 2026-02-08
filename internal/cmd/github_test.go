package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestGitHubCommandHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"github", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "GitHub") {
		t.Errorf("Expected help to mention GitHub, got: %s", output)
	}
	if !strings.Contains(output, "auth") {
		t.Errorf("Expected help to mention auth, got: %s", output)
	}
}

func TestGitHubAuthStatusHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"github", "auth", "status", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "status") {
		t.Errorf("Expected status help, got: %s", output)
	}
}
