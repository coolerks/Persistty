//go:build linux || darwin

package files

import (
	"golang.org/x/sys/unix"
	"os"
)

func snapshotReadlink(dir *os.File, name string) (string, error) {
	b := make([]byte, 4097)
	n, err := unix.Readlinkat(int(dir.Fd()), name, b)
	if err != nil {
		return "", err
	}
	if n >= 4097 {
		return "", ErrTooLarge
	}
	return string(b[:n]), nil
}
