package config

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Read the service UID's account, not the shell environment inherited by tmux.
func defaultShell() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	uid := strconv.Itoa(os.Geteuid())
	var candidate string
	switch runtime.GOOS {
	case "linux":
		output, err := shellAccountOutput(ctx, "/usr/bin/getent", "passwd", uid)
		if err == nil {
			candidate = passwdShell(output, uid)
		}
	case "darwin":
		account, err := user.LookupId(uid)
		if err == nil && account.Uid == uid && account.Username != "" && !strings.ContainsAny(account.Username, "/\r\n\x00") {
			output, err := shellAccountOutput(ctx, "/usr/bin/dscl", ".", "-read", "/Users/"+account.Username, "UserShell")
			if err == nil {
				candidate = strings.TrimSpace(strings.TrimPrefix(output, "UserShell:"))
			}
		}
	}
	return usableShell(candidate)
}

func passwdShell(output, uid string) string {
	fields := strings.Split(strings.TrimSpace(output), ":")
	if len(fields) != 7 || fields[2] != uid {
		return ""
	}
	return fields[6]
}

func usableShell(candidate string) string {
	if candidate != "" && filepath.IsAbs(candidate) && filepath.Clean(candidate) == candidate && !strings.ContainsAny(candidate, "\r\n\x00") {
		info, err := os.Stat(candidate)
		if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
			return candidate
		}
	}
	return "/bin/sh"
}

type shellOutput struct{ bytes.Buffer }

func (b *shellOutput) Write(p []byte) (int, error) {
	if len(p) > 8192-b.Len() {
		return 0, errors.New("account output too large")
	}
	return b.Buffer.Write(p)
}
func shellAccountOutput(ctx context.Context, binary string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LC_ALL=C"}
	output := &shellOutput{}
	cmd.Stdout = output
	cmd.Stderr = io.Discard
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	return output.String(), err
}
