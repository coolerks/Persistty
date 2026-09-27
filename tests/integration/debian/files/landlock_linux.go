package main

import "golang.org/x/sys/unix"

// 仅查询 ABI，不创建规则或改变当前进程权限，不能视为 CLI 隔离验收。
func landlockABI() (int, string) {
	abi, _, errno := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0, unix.LANDLOCK_CREATE_RULESET_VERSION)
	if errno != 0 {
		return 0, errno.Error()
	}
	return int(abi), ""
}
