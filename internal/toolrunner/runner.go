package toolrunner

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var ErrUnavailable = errors.New("tool unavailable")
var ErrLimit = errors.New("tool output limit")
var ErrCapacity = errors.New("tool capacity")
var ErrInvalid = errors.New("invalid tool request")

type Runner struct {
	Git, Rg string
	Timeout time.Duration
	slots   chan struct{}
}

func New(git, rg string, timeout time.Duration) *Runner {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &Runner{Git: git, Rg: rg, Timeout: timeout, slots: make(chan struct{}, 2)}
}
func (r *Runner) Acquire(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case r.slots <- struct{}{}:
		return func() { <-r.slots }, nil
	default:
		return nil, ErrCapacity
	}
}
func (r *Runner) Stage(base string) (string, func(), error) {
	if err := os.MkdirAll(base, 0700); err != nil {
		return "", nil, err
	}
	info, err := os.Lstat(base)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return "", nil, ErrUnavailable
	}
	dir, err := os.MkdirTemp(base, "snapshot-")
	if err != nil {
		return "", nil, err
	}
	return dir, func() { os.RemoveAll(dir) }, nil
}

type capped struct {
	buf      bytes.Buffer
	max      int
	overflow bool
}

func (b *capped) Write(p []byte) (int, error) {
	if len(p) > b.max-b.buf.Len() {
		b.overflow = true
		return 0, ErrLimit
	}
	return b.buf.Write(p)
}
func (r *Runner) Run(ctx context.Context, tool, dir string, input []byte, args ...string) ([]byte, int, error) {
	binary := r.Rg
	if tool == "git" {
		binary = r.Git
	} else if tool != "rg" {
		return nil, -1, ErrInvalid
	}
	resolved, err := exec.LookPath(binary)
	if err != nil {
		return nil, -1, ErrUnavailable
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return nil, -1, ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, resolved, args...)
	cmd.Dir = dir
	// Nothing from the caller's Git/rg configuration or credential environment.
	cmd.Env = []string{"PATH=" + filepath.Dir(resolved), "LC_ALL=C", "GIT_CEILING_DIRECTORIES=" + dir, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "GIT_OPTIONAL_LOCKS=0", "GIT_NO_LAZY_FETCH=1", "GIT_TERMINAL_PROMPT=0", "GIT_PAGER=cat", "GIT_LITERAL_PATHSPECS=1"}
	cmd.Stdin = bytes.NewReader(input)
	out := &capped{max: 32 << 20}
	stderr := &capped{max: 64 << 10}
	cmd.Stdout = out
	cmd.Stderr = stderr
	prepare(cmd)
	cmd.WaitDelay = time.Second
	err = cmd.Run()
	if out.overflow || stderr.overflow {
		return nil, -1, ErrLimit
	}
	if ctx.Err() != nil {
		return nil, -1, ctx.Err()
	}
	if errors.Is(err, ErrLimit) {
		return nil, -1, ErrLimit
	}
	if out.buf.Len() >= out.max || stderr.buf.Len() >= stderr.max {
		return nil, -1, ErrLimit
	}
	if err == nil {
		return out.buf.Bytes(), 0, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return out.buf.Bytes(), exit.ExitCode(), nil
	}
	return nil, -1, ErrUnavailable
}
func (r *Runner) GitRun(ctx context.Context, dir string, args ...string) ([]byte, error) {
	fixed := []string{"--no-pager", "--literal-pathspecs", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null", "-c", "gc.auto=0", "-c", "maintenance.auto=false", "-c", "diff.external=", "-c", "core.attributesFile=/dev/null", "-c", "core.excludesFile=/dev/null"}
	out, code, err := r.Run(ctx, "git", dir, nil, append(fixed, args...)...)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, ErrUnavailable
	}
	return out, nil
}
