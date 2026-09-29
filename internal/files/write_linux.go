//go:build linux

package files

import (
	"os"

	"golang.org/x/sys/unix"
)

func openLeaf(dir *os.File, name string) (*os.File, error) {
	fd, err := unix.Openat(int(dir.Fd()), name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}
func createTemp(dir *os.File, name string) (*os.File, error) {
	fd, err := unix.Openat(int(dir.Fd()), name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}
func removeLeaf(dir *os.File, name string) error { return unix.Unlinkat(int(dir.Fd()), name, 0) }
func replaceLeaf(dir *os.File, from, to string) error {
	return unix.Renameat(int(dir.Fd()), from, int(dir.Fd()), to)
}
func renameNoReplace(source *os.File, from string, target *os.File, to string) error {
	return unix.Renameat2(int(source.Fd()), from, int(target.Fd()), to, unix.RENAME_NOREPLACE)
}
func mkdirLeaf(dir *os.File, name string) error { return unix.Mkdirat(int(dir.Fd()), name, 0700) }
func removeDirLeaf(dir *os.File, name string) error {
	return unix.Unlinkat(int(dir.Fd()), name, unix.AT_REMOVEDIR)
}
