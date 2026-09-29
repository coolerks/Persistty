package workspace

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unicode/utf8"
)

var ErrInvalidPath = errors.New("invalid directory path")
var ErrUnreachable = errors.New("directory unavailable")
var ErrTooMany = errors.New("too many directory entries")

type Directory struct {
	Path   string   `json:"path"`
	Parent string   `json:"parent"`
	Items  []string `json:"items"`
}

type Identity struct {
	Path string
	Dev  uint64
	Ino  uint64
}

func Resolve(path string) (Identity, error) {
	if !filepath.IsAbs(path) || !utf8.ValidString(path) || strings.ContainsRune(path, 0) || filepath.Clean(path) != path {
		return Identity{}, ErrInvalidPath
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return Identity{}, errors.Join(ErrUnreachable, err)
	}
	file, err := os.Open(canonical)
	if err != nil {
		return Identity{}, errors.Join(ErrUnreachable, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.IsDir() {
		return Identity{}, ErrUnreachable
	}
	if _, err = file.Readdirnames(1); err != nil && !errors.Is(err, io.EOF) {
		return Identity{}, errors.Join(ErrUnreachable, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return Identity{}, ErrUnreachable
	}
	return Identity{canonical, uint64(stat.Dev), stat.Ino}, nil
}

func Browse(path string) (Directory, error) {
	if path == "" {
		path = os.Getenv("HOME")
	}
	identity, err := Resolve(path)
	if err != nil {
		return Directory{}, err
	}
	file, err := os.Open(identity.Path)
	if err != nil {
		return Directory{}, errors.Join(ErrUnreachable, err)
	}
	defer file.Close()
	entries, err := file.ReadDir(10001)
	if err != nil && !errors.Is(err, io.EOF) {
		return Directory{}, errors.Join(ErrUnreachable, err)
	}
	if len(entries) > 10000 {
		return Directory{}, ErrTooMany
	}
	items := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !utf8.ValidString(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err == nil && info.IsDir() {
			items = append(items, entry.Name())
		}
	}
	sort.Strings(items)
	return Directory{identity.Path, filepath.Dir(identity.Path), items}, nil
}
