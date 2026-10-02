package elevation

import (
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type broker struct {
	policy    Policy
	checkPeer func(*net.UnixConn, int) error
	run       func(context.Context, Grant, []byte, []byte, Publish) (Result, error)
	status    func(Policy, Grant) (Result, error)
	slots     chan struct{}
}

func ServeBroker(ctx context.Context, listener *net.UnixListener) error {
	if os.Geteuid() == 0 {
		return ErrForbidden
	}
	if err := disableCore(); err != nil {
		return err
	}
	p, err := LoadPolicy()
	if err != nil {
		return err
	}
	if os.Getuid() != p.CallerUID || os.Getgid() != p.CallerGID {
		return ErrForbidden
	}
	for _, path := range []string{HelperPath, BrokerPath, "/usr/bin/sudo"} {
		f, err := openTrusted(path, false)
		if err != nil {
			return ErrUnavailable
		}
		info, err := f.Stat()
		f.Close()
		if err != nil || info.Mode().Perm()&0100 == 0 {
			return ErrUnavailable
		}
	}
	b := broker{policy: p, checkPeer: peerAllowed, run: runSudo, status: ledgerStatus, slots: make(chan struct{}, 2)}
	return b.serve(ctx, listener)
}
func (b *broker) serve(ctx context.Context, l *net.UnixListener) error {
	ctx, cancel := context.WithCancel(ctx)
	var handlers sync.WaitGroup
	defer handlers.Wait()
	defer cancel()
	stop := context.AfterFunc(ctx, func() { l.Close() })
	defer stop()
	for {
		conn, err := l.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		select {
		case b.slots <- struct{}{}:
			handlers.Add(1)
			go func() { defer handlers.Done(); defer func() { <-b.slots }(); b.handle(ctx, conn) }()
		default:
			_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
			_ = WriteFrame(conn, reply{Code: "rate_limited"})
			conn.Close()
		}
	}
}
func (b *broker) handle(ctx context.Context, conn *net.UnixConn) {
	defer conn.Close()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	if b.checkPeer(conn, b.policy.CallerUID) != nil {
		return
	}
	var request brokerRequest
	if ReadFrame(conn, &request) != nil {
		_ = WriteFrame(conn, reply{Code: "invalid_request"})
		return
	}
	switch request.Action {
	case "target":
		target, err := b.policy.target(request.Root, request.Path)
		if err != nil {
			_ = WriteFrame(conn, reply{Code: "forbidden"})
			return
		}
		parent, err := openTrusted(filepath.Dir(target.Path), true)
		if err != nil {
			_ = WriteFrame(conn, reply{Code: "elevation_unavailable"})
			return
		}
		parent.Close()
		_ = WriteFrame(conn, reply{Phase: "target", Target: &target})
	case "status":
		if !noncePattern.MatchString(request.Grant.ID) {
			return
		}
		target, err := b.policy.target(request.Grant.Root, request.Grant.Path)
		if err != nil || target.ID != request.Grant.TargetID {
			return
		}
		result, err := b.status(b.policy, request.Grant)
		if err != nil {
			_ = WriteFrame(conn, reply{Code: "elevation_unavailable"})
			return
		}
		_ = WriteFrame(conn, reply{Phase: "result", Result: &result})
	case "execute":
		if b.policy.validateGrant(request.Grant) != nil {
			_ = WriteFrame(conn, reply{Code: "invalid_request"})
			return
		}
		if WriteFrame(conn, reply{Phase: "accepted"}) != nil {
			return
		}
		password, err := ReadBytes(conn, 1024)
		if err != nil {
			return
		}
		defer clear(password)
		content, err := ReadBytes(conn, MaxContent)
		if err != nil {
			return
		}
		defer clear(content)
		if !ValidPassword(password) || !ValidContent(content) || Digest(content) != request.Grant.ContentHash {
			result := outcome(request.Grant.ID, "rejected", "invalid_request")
			_ = WriteFrame(conn, reply{Phase: "result", Result: &result})
			return
		}
		result, err := b.run(ctx, request.Grant, password, content, func(commit func() (Result, error)) (Result, error) {
			if WriteFrame(conn, reply{Phase: "ready"}) != nil {
				return Result{}, ErrUnavailable
			}
			deadline := time.Now().Add(5 * time.Second)
			if request.Grant.ExpiresAt.Before(deadline) {
				deadline = request.Grant.ExpiresAt
			}
			_ = conn.SetDeadline(deadline)
			var decision struct {
				Phase string `json:"phase"`
			}
			if ReadFrame(conn, &decision) != nil || decision.Phase != "commit" || !deadline.After(time.Now()) {
				return outcome(request.Grant.ID, "cancelled", "cancelled"), nil
			}
			return commit()
		})
		if err != nil {
			result = outcome(request.Grant.ID, "indeterminate", "outcome_unknown")
		}
		_ = WriteFrame(conn, reply{Phase: "result", Result: &result})
	}
}

// discardedOutput bounds output without ever retaining sudo stderr in memory.
type discardedOutput struct{ n int }

func (w *discardedOutput) Write(p []byte) (int, error) {
	w.n += len(p)
	if w.n > 64<<10 {
		return len(p), ErrUnavailable
	}
	return len(p), nil
}

func runSudo(ctx context.Context, g Grant, password, content []byte, publish Publish) (Result, error) {
	return runCommand(ctx, g, password, content, publish, exec.CommandContext(ctx, "/usr/bin/sudo", "-k", "-S", "-p", "", "-u", "root", "-C", "5", "--", HelperPath))
}
func runCommand(ctx context.Context, g Grant, password, content []byte, publish Publish, cmd *exec.Cmd) (Result, error) {
	controlR, controlW, err := os.Pipe()
	if err != nil {
		return Result{}, err
	}
	defer controlR.Close()
	defer controlW.Close()
	contentR, contentW, err := os.Pipe()
	if err != nil {
		return Result{}, err
	}
	defer contentR.Close()
	defer contentW.Close()
	authR, authW, err := os.Pipe()
	if err != nil {
		return Result{}, err
	}
	defer authR.Close()
	defer authW.Close()
	cmd.ExtraFiles = []*os.File{controlR, contentR}
	cmd.Stdin = authR
	cmd.Stderr = &discardedOutput{}
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C", "LC_ALL=C"}
	cmd.Dir = "/"
	// sudo relays catchable signals; root helper also has its own deadline.
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = 32 * time.Second
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, err
	}
	defer stdout.Close()
	if err = cmd.Start(); err != nil {
		return Result{}, ErrUnavailable
	}
	controlR.Close()
	contentR.Close()
	authR.Close()
	waited := false
	defer func() {
		if !waited {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			controlW.Close()
			contentW.Close()
			authW.Close()
			stdout.Close()
			_ = cmd.Wait()
		}
	}()
	writers := make(chan error, 1)
	go func() {
		_, e := io.Copy(authW, io.MultiReader(bytes.NewReader(password), strings.NewReader("\n")))
		authW.Close()
		if e == nil {
			e = WriteFrame(controlW, g)
		}
		if e == nil {
			_, e = io.Copy(contentW, bytes.NewReader(content))
		}
		contentW.Close()
		writers <- e
	}()
	// Join before the caller clears credential/content slices. Close all writer
	// endpoints first so failed authentication or early exit cannot strand it.
	defer func() { controlW.Close(); contentW.Close(); authW.Close(); <-writers }()
	stop := context.AfterFunc(ctx, func() { controlW.Close(); contentW.Close(); authW.Close(); stdout.Close() })
	defer stop()
	var response reply
	if ReadFrame(stdout, &response) != nil {
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		return outcome(g.ID, "rejected", "authorization_failed"), nil
	}
	if response.Phase == "ready" {
		result, e := publish(func() (Result, error) {
			if ctx.Err() != nil {
				return Result{}, ctx.Err()
			}
			if WriteFrame(controlW, struct {
				Phase string `json:"phase"`
			}{"commit"}) != nil {
				return Result{}, ErrUnavailable
			}
			if ReadFrame(stdout, &response) != nil || response.Phase != "result" || response.Result == nil || !ResultValid(*response.Result, g.ID) {
				return Result{}, ErrUnavailable
			}
			return *response.Result, nil
		})
		controlW.Close()
		contentW.Close()
		authW.Close()
		// Closing the control channel releases a helper awaiting COMMIT.
		waitErr := cmd.Wait()
		waited = true
		if e != nil {
			return Result{}, e
		}
		if waitErr != nil && result.State == "applied" {
			return Result{}, ErrUnavailable
		}
		return result, nil
	}
	if response.Phase != "result" || response.Result == nil || !ResultValid(*response.Result, g.ID) {
		return Result{}, ErrUnavailable
	}
	controlW.Close()
	contentW.Close()
	authW.Close()
	err = cmd.Wait()
	waited = true
	if err != nil && response.Result.State == "applied" {
		return Result{}, ErrUnavailable
	}
	return *response.Result, nil
}
