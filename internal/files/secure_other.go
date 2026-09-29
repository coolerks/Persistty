//go:build !linux

package files

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"persistty/internal/storage"
)

func openConstrained(folder storage.RegisteredFolder, relative string, _ bool) (*os.File, error) {
	root, err := os.OpenRoot(folder.Path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	info, err := root.Stat(".")
	if err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || uint64(stat.Dev) != folder.Device || stat.Ino != folder.Inode {
		return nil, ErrRootChanged
	}
	info, err = os.Stat(folder.Path)
	if err != nil {
		return nil, err
	}
	stat, ok = info.Sys().(*syscall.Stat_t)
	if !ok || uint64(stat.Dev) != folder.Device || stat.Ino != folder.Inode {
		return nil, ErrRootChanged
	}
	file, err := root.Open(relative)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	actual := filepath.Join(folder.Path, relative)
	opened, err := os.Open(actual)
	if err != nil {
		return nil, err
	}
	checked, err := file.Stat()
	if err != nil {
		opened.Close()
		return nil, err
	}
	current, err := opened.Stat()
	if err != nil || !os.SameFile(checked, current) {
		opened.Close()
		return nil, ErrRootChanged
	}
	return opened, nil
}

func openMutationParent(folder storage.RegisteredFolder, relative string) (*os.File, error) {
	root, err := os.OpenRoot(folder.Path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	if relative != "." {
		current := ""
		for _, part := range strings.Split(relative, "/") {
			if current == "" {
				current = part
			} else {
				current += "/" + part
			}
			info, e := root.Lstat(current)
			if e != nil {
				return nil, e
			}
			if !info.IsDir() {
				return nil, ErrUnsupported
			}
		}
	}
	return openConstrained(folder, relative, true)
}
