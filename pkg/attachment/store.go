// Package attachment provides file storage for channel attachments.
// Files are stored on the local filesystem under {stateDir}/attachments/{id}/{filename}.
// This is a stop-gap implementation; the storage backend can be swapped later.
package attachment

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MaxFileSize is the default maximum file size (50MB).
const MaxFileSize = 50 * 1024 * 1024

// MaxFilesPerMessage is the default maximum number of attachments per message.
const MaxFilesPerMessage = 5

// allowedMIME is the whitelist of permitted MIME types.
var allowedMIME = map[string]bool{
	"image/jpeg":         true,
	"image/png":          true,
	"image/gif":          true,
	"image/webp":         true,
	"application/pdf":    true,
	"text/plain":         true,
	"application/json":   true,
	"application/zip":    true,
	"application/gzip":   true,
	"video/mp4":          true,
	"audio/mpeg":         true,
}

// Metadata holds information about a stored attachment.
type Metadata struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	MIMEType  string    `json:"mime_type"`
	Channel   string    `json:"channel"`
	Sender    string    `json:"sender"`
	Size      int64     `json:"size"`
}

// Store manages attachment file storage on the local filesystem.
type Store struct {
	dir string // base directory for attachments
}

// NewStore creates an attachment store rooted at the given directory.
func NewStore(stateDir string) *Store {
	dir := filepath.Join(stateDir, "attachments")
	return &Store{dir: dir}
}

// Save stores file data and returns metadata. The filename is sanitized.
func (s *Store) Save(data []byte, filename, channel, sender string) (*Metadata, error) {
	if len(data) > MaxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", len(data), MaxFileSize)
	}

	// Detect and validate MIME type
	mimeType := detectMIME(data, filename)
	if !allowedMIME[mimeType] {
		return nil, fmt.Errorf("unsupported file type: %s", mimeType)
	}

	// Generate unique ID
	id := generateID()

	// Sanitize filename — strip path components, keep only base name
	safeName := sanitizeFilename(filename)
	if safeName == "" {
		safeName = "attachment"
	}

	// Create directory
	attachDir := filepath.Join(s.dir, id)
	if err := os.MkdirAll(attachDir, 0o755); err != nil {
		return nil, fmt.Errorf("create attachment dir: %w", err)
	}

	// Write file
	filePath := filepath.Join(attachDir, safeName)
	if err := os.WriteFile(filePath, data, 0o644); err != nil { //nolint:gosec // stored in controlled directory
		return nil, fmt.Errorf("write attachment: %w", err)
	}

	return &Metadata{
		ID:        id,
		Filename:  safeName,
		MIMEType:  mimeType,
		Size:      int64(len(data)),
		Channel:   channel,
		Sender:    sender,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// Get returns the file data and metadata for the given attachment ID.
func (s *Store) Get(id string) ([]byte, *Metadata, error) {
	if !isValidID(id) {
		return nil, nil, fmt.Errorf("invalid attachment ID")
	}

	attachDir := filepath.Join(s.dir, id)
	entries, err := os.ReadDir(attachDir)
	if err != nil {
		return nil, nil, fmt.Errorf("attachment not found: %s", id)
	}

	if len(entries) == 0 {
		return nil, nil, fmt.Errorf("attachment empty: %s", id)
	}

	// First file in the directory is the attachment
	filename := entries[0].Name()
	filePath := filepath.Join(attachDir, filename)

	data, err := os.ReadFile(filePath) //nolint:gosec // path constructed from validated ID
	if err != nil {
		return nil, nil, fmt.Errorf("read attachment: %w", err)
	}

	info, _ := entries[0].Info()
	var modTime time.Time
	if info != nil {
		modTime = info.ModTime()
	}

	mimeType := detectMIME(data, filename)

	return data, &Metadata{
		ID:        id,
		Filename:  filename,
		MIMEType:  mimeType,
		Size:      int64(len(data)),
		CreatedAt: modTime,
	}, nil
}

// Delete removes an attachment by ID.
func (s *Store) Delete(id string) error {
	if !isValidID(id) {
		return fmt.Errorf("invalid attachment ID")
	}
	return os.RemoveAll(filepath.Join(s.dir, id))
}

// generateID creates a random hex ID.
func generateID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// isValidID checks that an ID is a hex string (no path traversal).
func isValidID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// sanitizeFilename strips directory components and dangerous characters.
func sanitizeFilename(name string) string {
	// Take only the base name
	name = filepath.Base(name)
	// Remove null bytes and path separators
	name = strings.ReplaceAll(name, "\x00", "")
	// Limit length
	if len(name) > 255 {
		name = name[:255]
	}
	return name
}

// detectMIME detects MIME type from content and filename.
func detectMIME(data []byte, filename string) string {
	// Try content-based detection first
	mimeType := "application/octet-stream"
	if len(data) >= 512 {
		mimeType = http.DetectContentType(data[:512])
	} else if len(data) > 0 {
		mimeType = http.DetectContentType(data)
	}

	// Override with extension for known types (DetectContentType can be imprecise)
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png":
		mimeType = "image/png"
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	case ".pdf":
		mimeType = "application/pdf"
	case ".json":
		mimeType = "application/json"
	case ".txt":
		mimeType = "text/plain"
	case ".mp4":
		mimeType = "video/mp4"
	case ".mp3":
		mimeType = "audio/mpeg"
	case ".zip":
		mimeType = "application/zip"
	case ".gz", ".gzip":
		mimeType = "application/gzip"
	}

	return mimeType
}
