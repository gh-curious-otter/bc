package attachment

import (
	"bytes"
	"strings"
	"testing"
)

func TestStore_SaveAndGet(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	// PNG header (minimal valid PNG-like content for MIME detection)
	data := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 100))

	meta, err := store.Save(data, "screenshot.png", "general", "swift-hawk")
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if meta.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if meta.Filename != "screenshot.png" {
		t.Errorf("filename = %q, want screenshot.png", meta.Filename)
	}
	if meta.MIMEType != "image/png" {
		t.Errorf("mime = %q, want image/png", meta.MIMEType)
	}
	if meta.Size != int64(len(data)) {
		t.Errorf("size = %d, want %d", meta.Size, len(data))
	}
	if meta.Channel != "general" {
		t.Errorf("channel = %q, want general", meta.Channel)
	}

	// Get it back
	got, gotMeta, err := store.Get(meta.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Error("data mismatch")
	}
	if gotMeta.Filename != "screenshot.png" {
		t.Errorf("get filename = %q, want screenshot.png", gotMeta.Filename)
	}
}

func TestStore_SaveRejectsUnsupportedType(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	// HTML content (not in allowed list)
	data := []byte("<html><body>hello</body></html>")

	_, err := store.Save(data, "page.html", "general", "test")
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error = %q, want unsupported mention", err.Error())
	}
}

func TestStore_SaveRejectsTooLarge(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	data := make([]byte, MaxFileSize+1)
	_, err := store.Save(data, "big.png", "general", "test")
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
}

func TestStore_GetInvalidID(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	_, _, err := store.Get("../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal ID")
	}
}

func TestStore_GetNonexistent(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	_, _, err := store.Get("deadbeef")
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

func TestStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	data := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 100))
	meta, err := store.Save(data, "test.png", "general", "test")
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := store.Delete(meta.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, _, err = store.Get(meta.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"screenshot.png", "screenshot.png"},
		{"../../../etc/passwd", "passwd"},
		{"/absolute/path/file.txt", "file.txt"},
		{"", "."},
		{"a" + strings.Repeat("b", 300), "a" + strings.Repeat("b", 254)},
	}
	for _, tt := range tests {
		got := sanitizeFilename(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsValidID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"deadbeef1234", true},
		{"abcdef012345", true},
		{"", false},
		{"../etc/passwd", false},
		{"DEADBEEF", false},        // uppercase not allowed
		{"hello world", false},     // spaces
		{strings.Repeat("a", 65), false}, // too long
	}
	for _, tt := range tests {
		got := isValidID(tt.id)
		if got != tt.want {
			t.Errorf("isValidID(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}
