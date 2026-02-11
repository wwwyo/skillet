package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MockFileSystem implements FileSystem for testing purposes.
type MockFileSystem struct {
	Files    map[string][]byte
	Dirs     map[string]bool
	Symlinks map[string]string
	HomeDir  string
}

// NewMockFileSystem returns a new MockFileSystem.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		Files:    make(map[string][]byte),
		Dirs:     make(map[string]bool),
		Symlinks: make(map[string]string),
		HomeDir:  "/home/test",
	}
}

func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	path = m.normalizePath(path)
	if data, ok := m.Files[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) WriteFile(path string, data []byte, _ os.FileMode) error {
	path = m.normalizePath(path)
	m.Files[path] = data
	return nil
}

func (m *MockFileSystem) Stat(path string) (os.FileInfo, error) {
	path = m.normalizePath(path)

	// Follow symlinks
	if target, ok := m.Symlinks[path]; ok {
		return m.Stat(target)
	}

	if _, ok := m.Files[path]; ok {
		return &mockFileInfo{name: filepath.Base(path), isDir: false}, nil
	}
	if m.Dirs[path] {
		return &mockFileInfo{name: filepath.Base(path), isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) Lstat(path string) (os.FileInfo, error) {
	path = m.normalizePath(path)

	if _, ok := m.Symlinks[path]; ok {
		return &mockFileInfo{name: filepath.Base(path), isDir: false, mode: os.ModeSymlink}, nil
	}
	if _, ok := m.Files[path]; ok {
		return &mockFileInfo{name: filepath.Base(path), isDir: false}, nil
	}
	if m.Dirs[path] {
		return &mockFileInfo{name: filepath.Base(path), isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) Remove(path string) error {
	path = m.normalizePath(path)
	delete(m.Files, path)
	delete(m.Dirs, path)
	delete(m.Symlinks, path)
	return nil
}

func (m *MockFileSystem) RemoveAll(path string) error {
	path = m.normalizePath(path)

	// Remove exact match
	delete(m.Files, path)
	delete(m.Dirs, path)
	delete(m.Symlinks, path)

	// Remove all children
	prefix := path + "/"
	for k := range m.Files {
		if strings.HasPrefix(k, prefix) {
			delete(m.Files, k)
		}
	}
	for k := range m.Dirs {
		if strings.HasPrefix(k, prefix) {
			delete(m.Dirs, k)
		}
	}
	for k := range m.Symlinks {
		if strings.HasPrefix(k, prefix) {
			delete(m.Symlinks, k)
		}
	}
	return nil
}

func (m *MockFileSystem) Rename(oldpath, newpath string) error {
	oldpath = m.normalizePath(oldpath)
	newpath = m.normalizePath(newpath)

	if data, ok := m.Files[oldpath]; ok {
		m.Files[newpath] = data
		delete(m.Files, oldpath)
		return nil
	}
	if m.Dirs[oldpath] {
		m.Dirs[newpath] = true
		delete(m.Dirs, oldpath)
		return nil
	}
	return os.ErrNotExist
}

func (m *MockFileSystem) MkdirAll(path string, _ os.FileMode) error {
	path = m.normalizePath(path)
	m.Dirs[path] = true

	// Also create parent directories
	parts := strings.Split(path, "/")
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[:i+1], "/")
		if parent != "" {
			m.Dirs[parent] = true
		}
	}
	return nil
}

func (m *MockFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	path = m.normalizePath(path)

	if !m.Dirs[path] {
		return nil, os.ErrNotExist
	}

	var entries []os.DirEntry
	seen := make(map[string]bool)

	prefix := path + "/"

	// Find files in this directory
	for p := range m.Files {
		if strings.HasPrefix(p, prefix) {
			rel := strings.TrimPrefix(p, prefix)
			if !strings.Contains(rel, "/") {
				if !seen[rel] {
					entries = append(entries, &mockDirEntry{name: rel, isDir: false})
					seen[rel] = true
				}
			}
		}
	}

	// Find subdirectories
	for p := range m.Dirs {
		if strings.HasPrefix(p, prefix) && p != path {
			rel := strings.TrimPrefix(p, prefix)
			parts := strings.Split(rel, "/")
			name := parts[0]
			if !seen[name] {
				entries = append(entries, &mockDirEntry{name: name, isDir: true})
				seen[name] = true
			}
		}
	}

	// Find symlinks
	for p := range m.Symlinks {
		if strings.HasPrefix(p, prefix) {
			rel := strings.TrimPrefix(p, prefix)
			if !strings.Contains(rel, "/") && !seen[rel] {
				entries = append(entries, &mockDirEntry{name: rel, isDir: false, isSymlink: true})
				seen[rel] = true
			}
		}
	}

	return entries, nil
}

func (m *MockFileSystem) Exists(path string) bool {
	path = m.normalizePath(path)
	if _, ok := m.Files[path]; ok {
		return true
	}
	if m.Dirs[path] {
		return true
	}
	if _, ok := m.Symlinks[path]; ok {
		return true
	}
	return false
}

func (m *MockFileSystem) IsDir(path string) bool {
	path = m.normalizePath(path)

	// Follow symlinks
	if target, ok := m.Symlinks[path]; ok {
		return m.IsDir(target)
	}

	return m.Dirs[path]
}

func (m *MockFileSystem) IsSymlink(path string) bool {
	path = m.normalizePath(path)
	_, ok := m.Symlinks[path]
	return ok
}

func (m *MockFileSystem) Symlink(oldname, newname string) error {
	newname = m.normalizePath(newname)
	m.Symlinks[newname] = oldname
	return nil
}

func (m *MockFileSystem) Readlink(path string) (string, error) {
	path = m.normalizePath(path)
	if target, ok := m.Symlinks[path]; ok {
		return target, nil
	}
	return "", fmt.Errorf("not a symlink: %s", path)
}

func (m *MockFileSystem) CopyFile(src, dst string) error {
	src = m.normalizePath(src)
	dst = m.normalizePath(dst)

	data, ok := m.Files[src]
	if !ok {
		return os.ErrNotExist
	}
	m.Files[dst] = make([]byte, len(data))
	copy(m.Files[dst], data)
	return nil
}

func (m *MockFileSystem) CopyDir(src, dst string) error {
	src = m.normalizePath(src)
	dst = m.normalizePath(dst)

	if !m.Dirs[src] {
		return os.ErrNotExist
	}

	m.Dirs[dst] = true

	prefix := src + "/"
	for p, data := range m.Files {
		if strings.HasPrefix(p, prefix) {
			rel := strings.TrimPrefix(p, prefix)
			newPath := dst + "/" + rel
			m.Files[newPath] = make([]byte, len(data))
			copy(m.Files[newPath], data)
		}
	}

	for p := range m.Dirs {
		if strings.HasPrefix(p, prefix) {
			rel := strings.TrimPrefix(p, prefix)
			m.Dirs[dst+"/"+rel] = true
		}
	}

	return nil
}

func (m *MockFileSystem) Abs(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	return "/" + path, nil
}

func (m *MockFileSystem) Rel(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}

func (m *MockFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (m *MockFileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

func (m *MockFileSystem) Base(path string) string {
	return filepath.Base(path)
}

func (m *MockFileSystem) UserHomeDir() (string, error) {
	return m.HomeDir, nil
}

func (m *MockFileSystem) normalizePath(path string) string {
	// Replace ~ with home directory
	if strings.HasPrefix(path, "~") {
		path = m.HomeDir + path[1:]
	}
	return filepath.Clean(path)
}

// mockFileInfo implements os.FileInfo for testing
type mockFileInfo struct {
	name  string
	isDir bool
	mode  os.FileMode
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() any           { return nil }

// mockDirEntry implements os.DirEntry for testing
type mockDirEntry struct {
	name      string
	isDir     bool
	isSymlink bool
}

func (m *mockDirEntry) Name() string { return m.name }
func (m *mockDirEntry) IsDir() bool  { return m.isDir }
func (m *mockDirEntry) Type() os.FileMode {
	if m.isSymlink {
		return os.ModeSymlink
	}
	if m.isDir {
		return os.ModeDir
	}
	return 0
}
func (m *mockDirEntry) Info() (os.FileInfo, error) {
	mode := os.FileMode(0)
	if m.isSymlink {
		mode = os.ModeSymlink
	}
	return &mockFileInfo{name: m.name, isDir: m.isDir, mode: mode}, nil
}
