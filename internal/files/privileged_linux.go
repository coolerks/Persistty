//go:build linux

package files

import (
	"bytes"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

func readPrivilegedMetadata(file *os.File) (privilegedMetadata, error) {
	var st unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &st); err != nil {
		return privilegedMetadata{}, err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Nlink != 1 || st.Mode&07000 != 0 {
		return privilegedMetadata{}, ErrUnsupported
	}
	result := privilegedMetadata{UID: int(st.Uid), GID: int(st.Gid), Mode: os.FileMode(st.Mode & 0777), Attrs: make(map[string][]byte)}
	size, err := unix.Flistxattr(int(file.Fd()), nil)
	if err != nil {
		return result, err
	}
	if size > 64<<10 {
		return result, ErrTooLarge
	}
	names := make([]byte, size)
	n, err := unix.Flistxattr(int(file.Fd()), names)
	if err != nil {
		return result, err
	}
	if n > len(names) {
		return result, ErrConflict
	}
	total := n
	for _, raw := range bytes.Split(names[:n], []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		name := string(raw)
		if !strings.HasPrefix(name, "user.") && name != "system.posix_acl_access" && name != "security.selinux" {
			return result, ErrUnsupported
		}
		size, err = unix.Fgetxattr(int(file.Fd()), name, nil)
		if err != nil {
			return result, err
		}
		total += size
		if total > 64<<10 {
			return result, ErrTooLarge
		}
		value := make([]byte, size)
		n, err = unix.Fgetxattr(int(file.Fd()), name, value)
		if err != nil {
			return result, err
		}
		if n > len(value) {
			return result, ErrConflict
		}
		result.Attrs[name] = value[:n]
	}
	return result, nil
}
func restorePrivilegedMetadata(file *os.File, m privilegedMetadata) error {
	if err := file.Chown(m.UID, m.GID); err != nil {
		return err
	}
	if err := file.Chmod(m.Mode); err != nil {
		return err
	}
	// Remove inherited ACLs/labels absent on the original before copying.
	actual, err := readPrivilegedMetadata(file)
	if err != nil {
		return err
	}
	for name := range actual.Attrs {
		if _, ok := m.Attrs[name]; !ok {
			if err = unix.Fremovexattr(int(file.Fd()), name); err != nil {
				return err
			}
		}
	}
	for name, value := range m.Attrs {
		if err = unix.Fsetxattr(int(file.Fd()), name, value, 0); err != nil {
			return err
		}
	}
	return nil
}
