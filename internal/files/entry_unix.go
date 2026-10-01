//go:build linux || darwin

package files

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func statEntry(dir *os.File, name string) (Entry, error) {
	var stat unix.Stat_t
	if err := unix.Fstatat(int(dir.Fd()), name, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return Entry{}, err
	}
	kind := "unsupported"
	switch stat.Mode & unix.S_IFMT {
	case unix.S_IFDIR:
		kind = "directory"
	case unix.S_IFREG:
		kind = "file"
	}
	return Entry{Name: name, Kind: kind, Size: stat.Size, Mtime: time.Unix(stat.Mtim.Sec, stat.Mtim.Nsec).UTC().Format("2006-01-02T15:04:05.000000000Z"), Identity: fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)}, nil
}
