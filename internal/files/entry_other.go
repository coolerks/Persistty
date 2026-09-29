//go:build !linux

package files

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func statEntry(dir *os.File, name string) (Entry, error) {
	info, err := os.Lstat(filepath.Join(dir.Name(), name))
	if err != nil {
		return Entry{}, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return Entry{}, ErrUnsupported
	}
	kind := "unsupported"
	if info.IsDir() {
		kind = "directory"
	} else if info.Mode().IsRegular() {
		kind = "file"
	}
	return Entry{Name: name, Kind: kind, Size: info.Size(), Mtime: info.ModTime().UTC().Format("2006-01-02T15:04:05.000000000Z"), Identity: fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)}, nil
}
