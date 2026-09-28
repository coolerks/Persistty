//go:build linux

// This is a disposable Debian feasibility probe, not a product sandbox.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

const read = unix.LANDLOCK_ACCESS_FS_READ_FILE | unix.LANDLOCK_ACCESS_FS_READ_DIR

func fail(code int) {
	fmt.Fprintln(os.Stderr, "CLI confinement unavailable")
	os.Exit(code)
}

func addRule(ruleset int, path string, rights uint64) error {
	fd, err := unix.Open(path, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	rule := unix.LandlockPathBeneathAttr{Allowed_access: rights, Parent_fd: int32(fd)}
	_, _, errno := unix.RawSyscall6(unix.SYS_LANDLOCK_ADD_RULE, uintptr(ruleset),
		uintptr(unix.LANDLOCK_RULE_PATH_BENEATH), uintptr(unsafe.Pointer(&rule)), 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func confine(root string) int {
	abi, _, errno := unix.RawSyscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0,
		uintptr(unix.LANDLOCK_CREATE_RULESET_VERSION))
	if errno != 0 || abi < 3 {
		return 81
	}
	handled := uint64(unix.LANDLOCK_ACCESS_FS_EXECUTE | read |
		unix.LANDLOCK_ACCESS_FS_WRITE_FILE | unix.LANDLOCK_ACCESS_FS_REMOVE_DIR |
		unix.LANDLOCK_ACCESS_FS_REMOVE_FILE | unix.LANDLOCK_ACCESS_FS_MAKE_CHAR |
		unix.LANDLOCK_ACCESS_FS_MAKE_DIR | unix.LANDLOCK_ACCESS_FS_MAKE_REG |
		unix.LANDLOCK_ACCESS_FS_MAKE_SOCK | unix.LANDLOCK_ACCESS_FS_MAKE_FIFO |
		unix.LANDLOCK_ACCESS_FS_MAKE_BLOCK | unix.LANDLOCK_ACCESS_FS_MAKE_SYM |
		unix.LANDLOCK_ACCESS_FS_REFER | unix.LANDLOCK_ACCESS_FS_TRUNCATE)
	if abi >= 5 {
		handled |= unix.LANDLOCK_ACCESS_FS_IOCTL_DEV
	}
	attr := unix.LandlockRulesetAttr{Access_fs: handled}
	if abi >= 4 {
		attr.Access_net = unix.LANDLOCK_ACCESS_NET_BIND_TCP | unix.LANDLOCK_ACCESS_NET_CONNECT_TCP
	}
	ruleset, _, errno := unix.RawSyscall(unix.SYS_LANDLOCK_CREATE_RULESET,
		uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr), 0)
	if errno != 0 {
		return 82
	}
	defer unix.Close(int(ruleset))
	// Dynamic binaries need the loader and shared libraries. These system paths
	// remain readable; this probe only establishes protection for private data.
	for _, path := range []string{"/usr", "/lib", "/lib64", "/bin", "/etc", "/dev/null"} {
		if info, err := os.Stat(path); err == nil {
			rights := uint64(unix.LANDLOCK_ACCESS_FS_READ_FILE)
			if info.IsDir() {
				rights = read | unix.LANDLOCK_ACCESS_FS_EXECUTE
			} else if path == "/dev/null" {
				rights |= unix.LANDLOCK_ACCESS_FS_WRITE_FILE
			}
			if err := addRule(int(ruleset), path, rights); err != nil {
				if errno, ok := err.(unix.Errno); ok {
					return 100 + int(errno)
				}
				return 90
			}
		}
	}
	if err := addRule(int(ruleset), root, read); err != nil {
		return 84
	}
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return 85
	}
	_, _, errno = unix.RawSyscall(unix.SYS_LANDLOCK_RESTRICT_SELF, ruleset, 0, 0)
	if errno != 0 {
		return 86
	}
	return 0
}

func main() {
	if len(os.Args) != 4 {
		fail(80)
	}
	root, command, target := os.Args[1], os.Args[2], os.Args[3]
	if !filepath.IsAbs(root) || !filepath.IsAbs(target) || strings.Contains(root, "..") {
		fail(80)
	}
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		fail(80)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		fail(80)
	}
	var bin string
	var argv []string
	switch command {
	case "cat":
		bin, argv = "/bin/cat", []string{"cat", "--", target}
	case "rg":
		bin, argv = "/usr/bin/rg", []string{"rg", "--follow", "--no-ignore", "--fixed-strings", "SENTINEL", "--", target}
	case "git":
		bin, argv = "/usr/bin/git", []string{"git", "-C", target, "--no-pager", "log", "-1", "--format=%H"}
	default:
		fail(80)
	}
	runtime.LockOSThread()
	if code := confine(root); code != 0 {
		fail(code)
	}
	// No inherited probe descriptors; caller supplies only three standard FDs.
	if err := unix.Exec(bin, argv, []string{"PATH=/usr/bin:/bin", "HOME=/nonexistent",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_NO_REPLACE_OBJECTS=1"}); err != nil {
		fail(87)
	}
}
