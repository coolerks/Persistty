//go:build linux

package files

import (
	"os"

	"golang.org/x/sys/unix"
	"persistty/internal/storage"
)

func openConstrained(root storage.RegisteredFolder, relative string, directory bool) (*os.File, error) {
	return openWithPolicy(root, relative, directory, false)
}

func openMutationParent(root storage.RegisteredFolder, relative string) (*os.File, error) {
	return openWithPolicy(root, relative, true, true)
}

func openWithPolicy(root storage.RegisteredFolder, relative string, directory, noSymlink bool) (*os.File, error) {
	rootFD, err := unix.Open(root.Path, unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer unix.Close(rootFD)
	var stat unix.Stat_t
	if err = unix.Fstat(rootFD, &stat); err != nil {
		return nil, err
	}
	if uint64(stat.Dev) != root.Device || stat.Ino != root.Inode {
		return nil, ErrRootChanged
	}
	if err = unix.Stat(root.Path, &stat); err != nil {
		return nil, err
	}
	if uint64(stat.Dev) != root.Device || stat.Ino != root.Inode {
		return nil, ErrRootChanged
	}
	flags := uint64(unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOCTTY | unix.O_NONBLOCK)
	if directory {
		flags |= unix.O_DIRECTORY
	}
	resolve := uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV)
	if noSymlink {
		resolve |= unix.RESOLVE_NO_SYMLINKS
	}
	fd, err := unix.Openat2(rootFD, relative, &unix.OpenHow{Flags: flags, Resolve: resolve})
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), relative), nil
}
