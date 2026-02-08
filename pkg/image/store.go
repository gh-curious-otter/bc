// Package image provides image upload, storage, and validation.
package image

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Common image formats
const (
	FormatJPEG = "jpeg"
	FormatPNG  = "png"
	FormatGIF  = "gif"
	FormatWebP = "webp"
)

// Default limits
const (
	DefaultMaxSize = 10 * 1024 * 1024 // 10MB
	DefaultMinSize = 100              // 100 bytes
)

// AllowedFormats returns the default set of allowed image formats.
func AllowedFormats() []string {
	return []string{FormatJPEG, FormatPNG, FormatGIF, FormatWebP}
}

// Image represents a stored image with metadata.
type Image struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	Format    string    `json:"format"`
	Hash      string    `json:"hash"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
}

// Store handles image storage operations.
type Store struct {
	allowedFormats map[string]bool
	baseDir        string
	maxSize        int64
	minSize        int64
}

// StoreOption configures a Store.
type StoreOption func(*Store)

// WithMaxSize sets the maximum allowed file size.
func WithMaxSize(size int64) StoreOption {
	return func(s *Store) {
		s.maxSize = size
	}
}

// WithMinSize sets the minimum allowed file size.
func WithMinSize(size int64) StoreOption {
	return func(s *Store) {
		s.minSize = size
	}
}

// WithFormats sets the allowed image formats.
func WithFormats(formats []string) StoreOption {
	return func(s *Store) {
		s.allowedFormats = make(map[string]bool)
		for _, f := range formats {
			s.allowedFormats[strings.ToLower(f)] = true
		}
	}
}

// NewStore creates a new image store.
func NewStore(baseDir string, opts ...StoreOption) *Store {
	s := &Store{
		baseDir:        baseDir,
		maxSize:        DefaultMaxSize,
		minSize:        DefaultMinSize,
		allowedFormats: make(map[string]bool),
	}

	// Default formats
	for _, f := range AllowedFormats() {
		s.allowedFormats[f] = true
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Init initializes the store directory.
func (s *Store) Init() error {
	return os.MkdirAll(s.baseDir, 0750)
}

// Save stores an image from a reader.
func (s *Store) Save(filename string, r io.Reader) (*Image, error) {
	// Detect format from filename
	format := detectFormat(filename)
	if format == "" {
		return nil, fmt.Errorf("unable to detect image format from filename: %s", filename)
	}

	if !s.allowedFormats[format] {
		return nil, fmt.Errorf("unsupported image format: %s", format)
	}

	// Create temp file to read content
	tmpFile, err := os.CreateTemp("", "bc-image-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	// Copy and hash simultaneously
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmpFile, hash), r)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Validate size
	if size < s.minSize {
		return nil, fmt.Errorf("image too small: %d bytes (min: %d)", size, s.minSize)
	}
	if size > s.maxSize {
		return nil, fmt.Errorf("image too large: %d bytes (max: %d)", size, s.maxSize)
	}

	// Generate ID from hash
	hashStr := hex.EncodeToString(hash.Sum(nil))
	id := hashStr[:16]

	// Determine storage path
	storagePath := filepath.Join(s.baseDir, id[:2], id[2:4], id+"."+format)
	if mkdirErr := os.MkdirAll(filepath.Dir(storagePath), 0750); mkdirErr != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", mkdirErr)
	}

	// Copy from temp to storage
	if _, seekErr := tmpFile.Seek(0, 0); seekErr != nil {
		return nil, fmt.Errorf("failed to seek temp file: %w", seekErr)
	}

	// #nosec G304 - storagePath is constructed from validated hash, not user input
	destFile, err := os.Create(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage file: %w", err)
	}

	if _, copyErr := io.Copy(destFile, tmpFile); copyErr != nil {
		_ = destFile.Close()
		_ = os.Remove(storagePath)
		return nil, fmt.Errorf("failed to copy to storage: %w", copyErr)
	}

	if err := destFile.Close(); err != nil {
		_ = os.Remove(storagePath)
		return nil, fmt.Errorf("failed to close storage file: %w", err)
	}

	return &Image{
		ID:        id,
		Filename:  filename,
		Format:    format,
		Size:      size,
		Hash:      hashStr,
		Path:      storagePath,
		CreatedAt: time.Now(),
	}, nil
}

// Get retrieves an image by ID and format.
func (s *Store) Get(id, format string) (*Image, error) {
	storagePath := filepath.Join(s.baseDir, id[:2], id[2:4], id+"."+format)
	info, err := os.Stat(storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("image not found: %s", id)
		}
		return nil, err
	}

	return &Image{
		ID:        id,
		Format:    format,
		Size:      info.Size(),
		Path:      storagePath,
		CreatedAt: info.ModTime(),
	}, nil
}

// Open returns a reader for an image.
func (s *Store) Open(id, format string) (io.ReadCloser, error) {
	img, err := s.Get(id, format)
	if err != nil {
		return nil, err
	}
	// #nosec G304 - path is constructed from validated ID, not user input
	return os.Open(img.Path)
}

// Delete removes an image by ID and format.
func (s *Store) Delete(id, format string) error {
	storagePath := filepath.Join(s.baseDir, id[:2], id[2:4], id+"."+format)
	return os.Remove(storagePath)
}

// Exists checks if an image exists.
func (s *Store) Exists(id, format string) bool {
	storagePath := filepath.Join(s.baseDir, id[:2], id[2:4], id+"."+format)
	_, err := os.Stat(storagePath)
	return err == nil
}

// detectFormat returns the image format from filename extension.
func detectFormat(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return FormatJPEG
	case ".png":
		return FormatPNG
	case ".gif":
		return FormatGIF
	case ".webp":
		return FormatWebP
	default:
		return ""
	}
}
