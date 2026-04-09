package cmd

import "testing"

func TestUpCmd_DefaultAddr(t *testing.T) {
	f := upCmd.Flags().Lookup("addr")
	if f == nil {
		t.Fatal("addr flag not found")
	}
	if f.DefValue != "127.0.0.1:9374" {
		t.Errorf("got %q, want %q", f.DefValue, "127.0.0.1:9374")
	}
}
