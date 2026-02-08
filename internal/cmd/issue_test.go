package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestIssueSubcommandsExist(t *testing.T) {
	var issueCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "issue" {
			issueCmd = c
			break
		}
	}
	if issueCmd == nil {
		t.Fatal("root command has no issue subcommand")
	}
	subs := issueCmd.Commands()
	names := make([]string, 0, len(subs))
	for _, s := range subs {
		names = append(names, s.Name())
	}
	for _, want := range []string{"create", "view", "comment", "react"} {
		if !sliceContains(names, want) {
			t.Errorf("issue subcommands = %v, missing %q", names, want)
		}
	}
}

func sliceContains(s []string, x string) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}

func TestIssueViewRequiresWorkspace(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	_, err := executeCmd("issue", "view", "1")
	if err == nil {
		t.Fatal("expected error when not in workspace")
	}
	if !strings.Contains(err.Error(), "workspace") {
		t.Errorf("error should mention workspace, got: %v", err)
	}
}

func TestIssueCommentRequiresWorkspace(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	_, err := executeCmd("issue", "comment", "1", "--body", "hello")
	if err == nil {
		t.Fatal("expected error when not in workspace")
	}
	if !strings.Contains(err.Error(), "workspace") {
		t.Errorf("error should mention workspace, got: %v", err)
	}
}

func TestIssueReactRequiresWorkspace(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	_, err := executeCmd("issue", "react", "1", "+1")
	if err == nil {
		t.Fatal("expected error when not in workspace")
	}
	if !strings.Contains(err.Error(), "workspace") {
		t.Errorf("error should mention workspace, got: %v", err)
	}
}

func TestIssueCreateRequiresTitle(t *testing.T) {
	// Without --title, cobra should report required flag
	_, err := executeCmd("issue", "create")
	if err == nil {
		t.Fatal("expected error when --title missing")
	}
}
