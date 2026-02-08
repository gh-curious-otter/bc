package cmd

import (
	"testing"
)

func TestUICmd_Flags(t *testing.T) {
	demoFlag := uiCmd.Flags().Lookup("demo")
	if demoFlag == nil {
		t.Fatal("expected --demo flag")
	}
	if demoFlag.DefValue != "false" {
		t.Errorf("demo default should be 'false', got: %s", demoFlag.DefValue)
	}
}

func TestUICmd_Usage(t *testing.T) {
	if uiCmd.Use != "ui" {
		t.Errorf("unexpected usage: %s", uiCmd.Use)
	}
	if uiCmd.Short == "" {
		t.Error("ui command should have short description")
	}
}
