package cmd

import (
	"testing"
)

func TestHomeCmd_Usage(t *testing.T) {
	if homeCmd.Use != "home" {
		t.Errorf("unexpected usage: %s", homeCmd.Use)
	}
	if homeCmd.Short == "" {
		t.Error("home command should have short description")
	}
}

func TestHomeCmd_LongDescription(t *testing.T) {
	if homeCmd.Long == "" {
		t.Error("home command should have long description")
	}
	// Should document navigation keys
	long := homeCmd.Long
	if !containsAll(long, []string{"j/k", "Enter", "Tab", "Esc", "q"}) {
		t.Errorf("long description should document navigation keys, got: %s", long)
	}
}

func containsAll(s string, substrs []string) bool {
	for _, sub := range substrs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
