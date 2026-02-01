package fs

import (
	"io"
	"os"
	"path/filepath"
)

// ModeSymlink is an alias for os.ModeSymlink.
const ModeSymlink = os.ModeSymlink

// System provides an abstraction over file system operations.
// This allows for easy mocking in tests.
type System interface {
	// File operations
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	Stat(path string) (os.FileInfo, error)
	Lstat(path string) (os.FileInfo, error)
	Remove(path string) error
	RemoveAll(path string) error
	Rename(oldpath, newpath string) error

	// Directory operations
	MkdirAll(path string, perm os.FileMode) error
	ReadDir(path string) ([]os.DirEntry, error)

	// Path operations
	Exists(path string) bool
	IsDir(path string) bool
	IsSymlink(path string) bool

	// Symlink operations
	Symlink(oldname, newname string) error
	Readlink(path string) (string, error)

	// Copy operations
	CopyFile(src, dst string) error
	CopyDir(src, dst string) error

	// Path utilities
	Abs(path string) (string, error)
	Rel(basepath, targpath string) (string, error)
	Join(elem ...string) string
	Dir(path string) string
	Base(path string) string

	// Home directory
	UserHomeDir() (string, error)
}

// RealSystem implements System using the real file system.
type RealSystem struct{}

// New returns a new RealSystem.
func New() *RealSystem {
	return &RealSystem{}
}

func (r *RealSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (r *RealSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (r *RealSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (r *RealSystem) Lstat(path string) (os.FileInfo, error) {
	return os.Lstat(path)
}

func (r *RealSystem) Remove(path string) error {
	return os.Remove(path)
}

func (r *RealSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (r *RealSystem) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func (r *RealSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (r *RealSystem) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (r *RealSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r *RealSystem) IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (r *RealSystem) IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&ModeSymlink != 0
}

func (r *RealSystem) Symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}

func (r *RealSystem) Readlink(path string) (string, error) {
	return os.Readlink(path)
}

func (r *RealSystem) CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	info, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (r *RealSystem) CopyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := r.CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := r.CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *RealSystem) Abs(path string) (string, error) {
	return filepath.Abs(path)
}

func (r *RealSystem) Rel(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}

func (r *RealSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (r *RealSystem) Dir(path string) string {
	return filepath.Dir(path)
}

func (r *RealSystem) Base(path string) string {
	return filepath.Base(path)
}

func (r *RealSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}
