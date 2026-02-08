package image

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewStore(t *testing.T) {
	s := NewStore("/tmp/images")
	if s.baseDir != "/tmp/images" {
		t.Errorf("baseDir = %s, want /tmp/images", s.baseDir)
	}
	if s.maxSize != DefaultMaxSize {
		t.Errorf("maxSize = %d, want %d", s.maxSize, DefaultMaxSize)
	}
}

func TestNewStore_WithOptions(t *testing.T) {
	s := NewStore("/tmp/images",
		WithMaxSize(1024),
		WithMinSize(10),
		WithFormats([]string{"png", "jpeg"}),
	)
	if s.maxSize != 1024 {
		t.Errorf("maxSize = %d, want 1024", s.maxSize)
	}
	if s.minSize != 10 {
		t.Errorf("minSize = %d, want 10", s.minSize)
	}
	if !s.allowedFormats["png"] {
		t.Error("png should be allowed")
	}
	if s.allowedFormats["gif"] {
		t.Error("gif should not be allowed")
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"image.jpg", FormatJPEG},
		{"image.jpeg", FormatJPEG},
		{"image.JPG", FormatJPEG},
		{"image.png", FormatPNG},
		{"image.PNG", FormatPNG},
		{"image.gif", FormatGIF},
		{"image.webp", FormatWebP},
		{"image.txt", ""},
		{"noextension", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := detectFormat(tt.filename)
			if got != tt.want {
				t.Errorf("detectFormat(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestStore_SaveAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStore(tmpDir, WithMinSize(1))
	if err := s.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create test image data
	data := bytes.Repeat([]byte{0xFF, 0xD8, 0xFF}, 100) // Fake JPEG-ish data
	reader := bytes.NewReader(data)

	// Save image
	img, err := s.Save("test.jpg", reader)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if img.ID == "" {
		t.Error("ID should not be empty")
	}
	if img.Format != FormatJPEG {
		t.Errorf("Format = %s, want %s", img.Format, FormatJPEG)
	}
	if img.Size != int64(len(data)) {
		t.Errorf("Size = %d, want %d", img.Size, len(data))
	}

	// Get image
	retrieved, err := s.Get(img.ID, img.Format)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.ID != img.ID {
		t.Errorf("retrieved ID = %s, want %s", retrieved.ID, img.ID)
	}

	// Verify file exists
	if !s.Exists(img.ID, img.Format) {
		t.Error("Exists should return true")
	}
}

func TestStore_Open(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStore(tmpDir, WithMinSize(1))
	_ = s.Init()

	data := []byte("test image data for reading")
	reader := bytes.NewReader(data)

	img, err := s.Save("test.png", reader)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Open and read
	rc, err := s.Open(img.ID, img.Format)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = rc.Close() }()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(rc)
	if !bytes.Equal(buf.Bytes(), data) {
		t.Error("Read data does not match original")
	}
}

func TestStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStore(tmpDir, WithMinSize(1))
	_ = s.Init()

	data := []byte("test image to delete")
	img, _ := s.Save("test.gif", bytes.NewReader(data))

	// Delete
	if err := s.Delete(img.ID, img.Format); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	if s.Exists(img.ID, img.Format) {
		t.Error("Image should not exist after delete")
	}
}

func TestStore_Save_TooSmall(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStore(tmpDir, WithMinSize(100))
	_ = s.Init()

	data := []byte("tiny")
	_, err := s.Save("test.png", bytes.NewReader(data))
	if err == nil {
		t.Error("expected error for image too small")
	}
	if !strings.Contains(err.Error(), "too small") {
		t.Errorf("error should mention 'too small': %v", err)
	}
}

func TestStore_Save_TooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStore(tmpDir, WithMaxSize(100), WithMinSize(1))
	_ = s.Init()

	data := bytes.Repeat([]byte{0}, 200)
	_, err := s.Save("test.png", bytes.NewReader(data))
	if err == nil {
		t.Error("expected error for image too large")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("error should mention 'too large': %v", err)
	}
}

func TestStore_Save_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStore(tmpDir, WithFormats([]string{"png"}))
	_ = s.Init()

	_, err := s.Save("test.jpg", bytes.NewReader([]byte("data")))
	if err == nil {
		t.Error("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error should mention 'unsupported': %v", err)
	}
}

func TestStore_Save_UnknownFormat(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStore(tmpDir)
	_ = s.Init()

	_, err := s.Save("test.txt", bytes.NewReader([]byte("data")))
	if err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStore(tmpDir)
	_ = s.Init()

	_, err := s.Get("nonexistent123456", "png")
	if err == nil {
		t.Error("expected error for not found")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestStore_Init(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "nested", "dir")
	s := NewStore(tmpDir)

	if err := s.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("should be a directory")
	}
}

func TestAllowedFormats(t *testing.T) {
	formats := AllowedFormats()
	if len(formats) != 4 {
		t.Errorf("AllowedFormats() returned %d formats, want 4", len(formats))
	}
}
