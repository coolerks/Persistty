//go:build linux

package elevation

import (
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
	"persistty/internal/files"
)

func timeNow() time.Time { return time.Now() }
func validRelativeTarget(path string) bool {
	return len(path) <= 4096 && files.ValidRelative(path, false)
}

// Every component is root-owned and not group/other-writable; all actual opens
// are relative no-follow opens. A lexical path check is never the read boundary.
func openTrusted(path string, directory bool) (*os.File, error) {
	if !validAbsolute(path) {
		return nil, ErrInvalid
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	pieces := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i, part := range pieces {
		if part == "" {
			continue
		}
		flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK
		if i < len(pieces)-1 || directory {
			flags |= unix.O_DIRECTORY
		}
		next, e := unix.Openat(fd, part, flags, 0)
		unix.Close(fd)
		if e != nil {
			return nil, e
		}
		fd = next
		var st unix.Stat_t
		if e = unix.Fstat(fd, &st); e != nil || st.Uid != 0 || st.Mode&0022 != 0 {
			unix.Close(fd)
			return nil, ErrForbidden
		}
	}
	var st unix.Stat_t
	if err = unix.Fstat(fd, &st); err != nil {
		unix.Close(fd)
		return nil, err
	}
	kind := uint32(unix.S_IFREG)
	if directory {
		kind = unix.S_IFDIR
	}
	if st.Uid != 0 || st.Mode&0022 != 0 || st.Mode&unix.S_IFMT != kind || !directory && st.Nlink != 1 {
		unix.Close(fd)
		return nil, ErrForbidden
	}
	return os.NewFile(uintptr(fd), path), nil
}
func peerAllowed(conn *net.UnixConn, uid int) error {
	raw, err := conn.SyscallConn()
	if err != nil {
		return err
	}
	var cred *unix.Ucred
	err = raw.Control(func(fd uintptr) { cred, _ = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED) })
	if err != nil || cred == nil || int(cred.Uid) != uid {
		return ErrForbidden
	}
	return nil
}
func disableCore() error { return unix.Setrlimit(unix.RLIMIT_CORE, &unix.Rlimit{}) }
func helperFD(fd int) (*os.File, error) {
	if err := unix.SetNonblock(fd, true); err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "helper-pipe"), nil
}
