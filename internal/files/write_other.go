//go:build !linux

package files

import (
	"os"
	"path/filepath"
	"syscall"
)

func openLeaf(dir *os.File, name string) (*os.File, error) {
	return os.OpenFile(filepath.Join(dir.Name(), name), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
}
func createTemp(dir *os.File, name string) (*os.File, error) {
	return os.OpenFile(filepath.Join(dir.Name(), name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
}
func removeLeaf(dir *os.File, name string) error { return os.Remove(filepath.Join(dir.Name(), name)) }
func replaceLeaf(dir *os.File, from, to string) error {
	return os.Rename(filepath.Join(dir.Name(), from), filepath.Join(dir.Name(), to))
}
func renameNoReplace(source *os.File, from string, target *os.File, to string) error {
	toPath := filepath.Join(target.Name(), to)
	if _, err := os.Lstat(toPath); err == nil {
		return os.ErrExist
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Rename(filepath.Join(source.Name(), from), toPath)
}
func mkdirLeaf(dir *os.File, name string) error {
	return os.Mkdir(filepath.Join(dir.Name(), name), 0700)
}
func removeDirLeaf(dir *os.File, name string) error {
	return os.Remove(filepath.Join(dir.Name(), name))
}
