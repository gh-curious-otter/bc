package agent

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestOSFileSystem_Stat(t *testing.T) {
	fs := OSFileSystem{}

	// Test existing file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, err := fs.Stat(tmpFile)
	if err != nil {
		t.Errorf("Stat() error = %v, want nil", err)
	}
	if info.Name() != "test.txt" {
		t.Errorf("Stat() name = %v, want test.txt", info.Name())
	}

	// Test non-existing file
	_, err = fs.Stat(filepath.Join(tmpDir, "nonexistent.txt"))
	if err == nil {
		t.Error("Stat() error = nil for non-existing file, want error")
	}
}

func TestOSFileSystem_ReadFile(t *testing.T) {
	fs := OSFileSystem{}

	// Test existing file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(tmpFile, content, 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	got, err := fs.ReadFile(tmpFile)
	if err != nil {
		t.Errorf("ReadFile() error = %v, want nil", err)
	}
	if string(got) != string(content) {
		t.Errorf("ReadFile() = %v, want %v", string(got), string(content))
	}

	// Test non-existing file
	_, err = fs.ReadFile(filepath.Join(tmpDir, "nonexistent.txt"))
	if err == nil {
		t.Error("ReadFile() error = nil for non-existing file, want error")
	}
}

func TestOSFileSystem_WriteFile(t *testing.T) {
	osFS := OSFileSystem{}
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")

	err := osFS.WriteFile(tmpFile, content, 0600)
	if err != nil {
		t.Errorf("WriteFile() error = %v, want nil", err)
	}

	// Verify content
	got, err := os.ReadFile(tmpFile) //nolint:gosec // test file path is safe
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("WriteFile() wrote %v, want %v", string(got), string(content))
	}
}

func TestOSFileSystem_MkdirAll(t *testing.T) {
	osFS := OSFileSystem{}
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "a", "b", "c")

	err := osFS.MkdirAll(newDir, 0750)
	if err != nil {
		t.Errorf("MkdirAll() error = %v, want nil", err)
	}

	// Verify directory exists
	info, err := os.Stat(newDir)
	if err != nil {
		t.Errorf("MkdirAll() directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("MkdirAll() created a file, want directory")
	}
}

func TestOSFileSystem_RemoveAll(t *testing.T) {
	osFS := OSFileSystem{}
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "toremove")

	// Create directory with file
	if err := os.MkdirAll(newDir, 0750); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "file.txt"), []byte("test"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err := osFS.RemoveAll(newDir)
	if err != nil {
		t.Errorf("RemoveAll() error = %v, want nil", err)
	}

	// Verify directory is gone
	_, err = os.Stat(newDir)
	if err == nil {
		t.Error("RemoveAll() directory still exists, want removed")
	}
}

func TestDefaultFileSystem(t *testing.T) {
	fs := DefaultFileSystem()
	if fs == nil {
		t.Error("DefaultFileSystem() = nil, want non-nil")
	}
	if _, ok := fs.(OSFileSystem); !ok {
		t.Error("DefaultFileSystem() type is not OSFileSystem")
	}
}

// MockFileSystem is a test double for FileSystem.
type MockFileSystem struct {
	StatFunc      func(path string) (fs.FileInfo, error)
	ReadFileFunc  func(path string) ([]byte, error)
	WriteFileFunc func(path string, data []byte, perm fs.FileMode) error
	MkdirAllFunc  func(path string, perm fs.FileMode) error
	RemoveAllFunc func(path string) error
}

func (m *MockFileSystem) Stat(path string) (fs.FileInfo, error) {
	if m.StatFunc != nil {
		return m.StatFunc(path)
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(path)
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	if m.WriteFileFunc != nil {
		return m.WriteFileFunc(path, data, perm)
	}
	return nil
}

func (m *MockFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	if m.MkdirAllFunc != nil {
		return m.MkdirAllFunc(path, perm)
	}
	return nil
}

func (m *MockFileSystem) RemoveAll(path string) error {
	if m.RemoveAllFunc != nil {
		return m.RemoveAllFunc(path)
	}
	return nil
}

func TestMockFileSystem(t *testing.T) {
	// Verify MockFileSystem implements FileSystem
	var _ FileSystem = &MockFileSystem{}

	// Test with custom implementations
	mock := &MockFileSystem{
		ReadFileFunc: func(path string) ([]byte, error) {
			return []byte("mocked content"), nil
		},
	}

	content, err := mock.ReadFile("/any/path")
	if err != nil {
		t.Errorf("MockFileSystem.ReadFile() error = %v, want nil", err)
	}
	if string(content) != "mocked content" {
		t.Errorf("MockFileSystem.ReadFile() = %v, want mocked content", string(content))
	}
}
