package cmd

import "testing"

func TestUpCmd_DefaultPort(t *testing.T) {
	f := upCmd.Flags().Lookup("port")
	if f == nil {
		t.Fatal("port flag not found")
	}
	if f.DefValue != "9374" {
		t.Errorf("got %q, want %q", f.DefValue, "9374")
	}
}
