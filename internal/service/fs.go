package service

import "os"

// ModeSymlink is an alias for os.ModeSymlink.
const ModeSymlink = os.ModeSymlink

// FileSystem provides an abstraction over file system operations.
// This allows for easy mocking in tests.
type FileSystem interface {
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
