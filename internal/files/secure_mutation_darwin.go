//go:build darwin

package files

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
	"persistty/internal/storage"
)

// Walk parent components through directory descriptors. Repeated os.Root.Lstat
// calls each re-walk the complete prefix and made deep Git trees quadratic.
func openMutationParent(root storage.RegisteredFolder, relative string) (*os.File, error) {
	if !ValidRelative(relative, true) && relative != "." {
		return nil, ErrInvalidPath
	}
	flags := unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC | unix.O_NONBLOCK
	fd, err := unix.Open(root.Path, flags, 0)
	if err != nil {
		return nil, err
	}
	var stat unix.Stat_t
	if err = unix.Fstat(fd, &stat); err != nil {
		unix.Close(fd)
		return nil, err
	}
	if uint64(stat.Dev) != root.Device || stat.Ino != root.Inode {
		unix.Close(fd)
		return nil, ErrRootChanged
	}
	if relative != "." && relative != "" {
		for _, part := range strings.Split(relative, "/") {
			next, e := unix.Openat(fd, part, flags, 0)
			unix.Close(fd)
			if e != nil {
				return nil, e
			}
			fd = next
		}
	}
	// Check registration still names the root after opening the chain. Existing
	// operation/snapshot callers also re-open and compare parent/entry identities.
	if err = unix.Lstat(root.Path, &stat); err != nil {
		unix.Close(fd)
		return nil, err
	}
	if uint64(stat.Dev) != root.Device || stat.Ino != root.Inode {
		unix.Close(fd)
		return nil, ErrRootChanged
	}
	return os.NewFile(uintptr(fd), filepath.Join(root.Path, relative)), nil
}
