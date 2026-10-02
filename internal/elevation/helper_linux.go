//go:build linux

package elevation

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"persistty/internal/files"
)

// RunHelper accepts no command-line operands. The policy path and FDs are fixed.
func RunHelper(ctx context.Context) error {
	if os.Geteuid() != 0 || os.Getuid() != 0 {
		return ErrForbidden
	}
	if err := disableCore(); err != nil {
		return ErrUnavailable
	}
	p, err := LoadPolicy()
	if err != nil {
		return err
	}
	uid, err := strconv.Atoi(os.Getenv("SUDO_UID"))
	if err != nil || uid != p.CallerUID {
		return ErrForbidden
	}
	gid, err := strconv.Atoi(os.Getenv("SUDO_GID"))
	if err != nil || gid != p.CallerGID {
		return ErrForbidden
	}
	control, err := helperFD(3)
	if err != nil {
		return err
	}
	defer control.Close()
	contentFD, err := helperFD(4)
	if err != nil {
		return err
	}
	defer contentFD.Close()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	stop := context.AfterFunc(ctx, func() { control.Close(); contentFD.Close() })
	defer stop()
	var g Grant
	if err = ReadFrame(control, &g); err != nil {
		return err
	}
	if err = p.validateGrant(g); err != nil {
		return helperReject(os.Stdout, g, err)
	}
	target, err := p.target(g.Root, g.Path)
	if err != nil {
		return helperReject(os.Stdout, g, err)
	}
	// The root-owned parent chain prevents caller-writable path redirection.
	parent, err := openTrusted(filepath.Dir(target.Path), true)
	if err != nil {
		return helperReject(os.Stdout, g, err)
	}
	parent.Close()
	content, err := io.ReadAll(io.LimitReader(contentFD, MaxContent+1))
	if err != nil {
		return helperReject(os.Stdout, g, err)
	}
	defer clear(content)
	if !ValidContent(content) || Digest(content) != g.ContentHash {
		return helperReject(os.Stdout, g, ErrInvalid)
	}
	l, err := openLedger(p)
	if err != nil {
		return helperReject(os.Stdout, g, err)
	}
	defer l.close()
	if err = l.consume(ctx, g); err != nil {
		return helperReject(os.Stdout, g, err)
	}
	lock, err := l.lock(ctx, ".target-"+g.TargetID)
	if err != nil {
		return helperReject(os.Stdout, g, err)
	}
	defer lock.Close()
	prepared, err := files.PreparePrivileged(ctx, g.Root, g.Path, g.Expected, content)
	if err != nil {
		result := outcome(g.ID, "rejected", classify(err))
		_ = l.finish(g, result)
		return WriteFrame(os.Stdout, reply{Phase: "result", Result: &result})
	}
	defer prepared.Close()
	if err = WriteFrame(os.Stdout, reply{Phase: "ready"}); err != nil {
		return err
	}
	// A missing/late/cancelled commit cannot publish the prepared file.
	deadline := time.Now().Add(5 * time.Second)
	if g.ExpiresAt.Before(deadline) {
		deadline = g.ExpiresAt
	}
	if err = control.SetReadDeadline(deadline); err != nil {
		return err
	}
	var decision struct {
		Phase string `json:"phase"`
	}
	err = ReadFrame(control, &decision)
	if err != nil || decision.Phase != "commit" || !deadline.After(time.Now()) || ctx.Err() != nil {
		result := outcome(g.ID, "cancelled", "cancelled")
		_ = l.finish(g, result)
		_ = WriteFrame(os.Stdout, reply{Phase: "result", Result: &result})
		return nil
	}
	commitCtx, commitCancel := context.WithDeadline(ctx, deadline)
	defer commitCancel()
	// Revocation or newly writable/symlinked ancestors while authentication
	// was pending must also stop publication, beyond the web configuration lock.
	latest, policyErr := LoadPolicy()
	if policyErr == nil && !reflect.DeepEqual(latest, p) {
		policyErr = ErrForbidden
	}
	if policyErr == nil {
		trusted, trustErr := openTrusted(filepath.Dir(target.Path), true)
		policyErr = trustErr
		if trusted != nil {
			trusted.Close()
		}
	}
	if policyErr != nil {
		result := outcome(g.ID, "rejected", classify(policyErr))
		_ = l.finish(g, result)
		return WriteFrame(os.Stdout, reply{Phase: "result", Result: &result})
	}
	version, err := prepared.Commit(commitCtx)
	result := outcome(g.ID, "rejected", classify(err))
	if err == nil {
		result = Result{ID: g.ID, State: "applied", Version: &version}
	}
	if err != nil && prepared.Published() {
		result = outcome(g.ID, "indeterminate", "outcome_unknown")
	}
	if ledgerErr := l.finish(g, result); ledgerErr != nil {
		result = outcome(g.ID, "indeterminate", "outcome_unknown")
	}
	return WriteFrame(os.Stdout, reply{Phase: "result", Result: &result})
}
func helperReject(w io.Writer, g Grant, err error) error {
	result := outcome(g.ID, "rejected", classify(err))
	return WriteFrame(w, reply{Phase: "result", Result: &result})
}
func classify(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, files.ErrConflict), errors.Is(err, files.ErrRootChanged):
		return "conflict"
	case errors.Is(err, ErrExpired):
		return "expired"
	case errors.Is(err, ErrInvalid):
		return "invalid_request"
	case errors.Is(err, ErrForbidden):
		return "forbidden"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "cancelled"
	default:
		return "elevation_unavailable"
	}
}
