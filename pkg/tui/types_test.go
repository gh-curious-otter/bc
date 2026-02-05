package tui

import "testing"

func TestCol(t *testing.T) {
	c := Col("Name", 20)
	if c.Name != "Name" {
		t.Errorf("expected Name 'Name', got '%s'", c.Name)
	}
	if c.Width != 20 {
		t.Errorf("expected Width 20, got %d", c.Width)
	}
	if c.Alignment != AlignLeft {
		t.Errorf("expected AlignLeft, got %d", c.Alignment)
	}
}

func TestColRight(t *testing.T) {
	c := ColRight("Count", 10)
	if c.Alignment != AlignRight {
		t.Errorf("expected AlignRight, got %d", c.Alignment)
	}
	if c.Name != "Count" {
		t.Errorf("expected Name 'Count', got '%s'", c.Name)
	}
}

func TestColCenter(t *testing.T) {
	c := ColCenter("Status", 12)
	if c.Alignment != AlignCenter {
		t.Errorf("expected AlignCenter, got %d", c.Alignment)
	}
}

func TestBind(t *testing.T) {
	called := false
	b := Bind("p", "peek", func() Cmd {
		called = true
		return nil
	})

	if b.Key != "p" {
		t.Errorf("expected Key 'p', got '%s'", b.Key)
	}
	if b.Label != "peek" {
		t.Errorf("expected Label 'peek', got '%s'", b.Label)
	}
	if b.Hidden {
		t.Error("expected Hidden to be false")
	}

	b.Handler()
	if !called {
		t.Error("expected handler to be called")
	}
}
