package main

import "testing"

func TestGreet(t *testing.T) {
	want := "Welcome"
	got := greet()
	if got != want {
		t.Errorf("greet() = %q, want %q", got, want)
	}
}
