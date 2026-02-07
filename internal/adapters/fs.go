package adapters

import (
	"io"
	"os"
	"path/filepath"

	"github.com/wwwyo/skillet/internal/service"
)

// RealFileSystem implements service.FileSystem using the real file system.
type RealFileSystem struct{}

// Compile-time interface check.
var _ service.FileSystem = (*RealFileSystem)(nil)

// NewFileSystem returns a new RealFileSystem.
func NewFileSystem() *RealFileSystem {
	return &RealFileSystem{}
}

func (r *RealFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (r *RealFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (r *RealFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (r *RealFileSystem) Lstat(path string) (os.FileInfo, error) {
	return os.Lstat(path)
}

func (r *RealFileSystem) Remove(path string) error {
	return os.Remove(path)
}

func (r *RealFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (r *RealFileSystem) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func (r *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (r *RealFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (r *RealFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r *RealFileSystem) IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (r *RealFileSystem) IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

func (r *RealFileSystem) Symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}

func (r *RealFileSystem) Readlink(path string) (string, error) {
	return os.Readlink(path)
}

func (r *RealFileSystem) CopyFile(src, dst string) error {
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

func (r *RealFileSystem) CopyDir(src, dst string) error {
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

func (r *RealFileSystem) Abs(path string) (string, error) {
	return filepath.Abs(path)
}

func (r *RealFileSystem) Rel(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}

func (r *RealFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (r *RealFileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

func (r *RealFileSystem) Base(path string) string {
	return filepath.Base(path)
}

func (r *RealFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}
