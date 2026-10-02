//go:build linux

package elevation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

type ledgerEntry struct {
	Binding   string    `json:"binding"`
	ExpiresAt time.Time `json:"expires_at"`
	Result    Result    `json:"result"`
}
type ledger struct {
	dir *os.File
	gid int
}

func openLedger(p Policy) (*ledger, error) {
	dir, err := openTrusted(p.LedgerPath, true)
	if err != nil {
		return nil, err
	}
	var st unix.Stat_t
	if err = unix.Fstat(int(dir.Fd()), &st); err != nil || int(st.Gid) != p.CallerGID || st.Mode&0777 != 0750 {
		dir.Close()
		return nil, ErrForbidden
	}
	return &ledger{dir: dir, gid: p.CallerGID}, nil
}
func (l *ledger) close() { l.dir.Close() }
func (l *ledger) open(name string, flags int) (*os.File, error) {
	fd, err := unix.Openat(int(l.dir.Fd()), name, flags|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0640)
	if err != nil {
		return nil, err
	}
	var st unix.Stat_t
	if err = unix.Fstat(fd, &st); err != nil || st.Uid != 0 || st.Mode&0022 != 0 || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Nlink != 1 {
		unix.Close(fd)
		return nil, ErrForbidden
	}
	return os.NewFile(uintptr(fd), name), nil
}
func (l *ledger) lock(ctx context.Context, name string) (*os.File, error) {
	f, err := l.open(name, unix.O_RDWR|unix.O_CREAT)
	if err != nil {
		return nil, err
	}
	timer := time.NewTicker(10 * time.Millisecond)
	defer timer.Stop()
	for {
		err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) {
			f.Close()
			return nil, err
		}
		select {
		case <-ctx.Done():
			f.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
func (l *ledger) read(name string, value any) error {
	f, err := l.open(name, unix.O_RDONLY)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.Size() > 64<<10 {
		return ErrInvalid
	}
	data, err := io.ReadAll(io.LimitReader(f, 64<<10+1))
	if err != nil {
		return err
	}
	return strictJSON(data, value)
}
func (l *ledger) write(name string, value any, exclusive bool) error {
	data, err := json.Marshal(value)
	if err != nil || len(data) > 64<<10 {
		return ErrInvalid
	}
	random := make([]byte, 16)
	if _, err = rand.Read(random); err != nil {
		return err
	}
	temp := ".tmp-" + hex.EncodeToString(random)
	f, err := l.open(temp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL)
	if err != nil {
		return err
	}
	defer unix.Unlinkat(int(l.dir.Fd()), temp, 0)
	if err = f.Chown(0, l.gid); err == nil {
		err = f.Chmod(0640)
	}
	if err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if exclusive {
		err = unix.Renameat2(int(l.dir.Fd()), temp, int(l.dir.Fd()), name, unix.RENAME_NOREPLACE)
	} else {
		err = unix.Renameat(int(l.dir.Fd()), temp, int(l.dir.Fd()), name)
	}
	if err != nil {
		return err
	}
	return l.dir.Sync()
}
func (l *ledger) consume(ctx context.Context, g Grant) error {
	lock, err := l.lock(ctx, ".lock")
	if err != nil {
		return err
	}
	defer lock.Close()
	now := time.Now().UTC()
	var clock time.Time
	err = l.read(".clock", &clock)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if !clock.IsZero() && now.Before(clock) {
		return ErrUnavailable
	}
	if err = g.Validate(now); err != nil {
		return err
	}
	if err = l.write(".clock", now, false); err != nil {
		return err
	}
	// Bounded enumeration, never removing an unexpired entry. Clock high-water
	// is retained even when old nonces are pruned.
	scanFD, err := unix.Openat(int(l.dir.Fd()), ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	scan := os.NewFile(uintptr(scanFD), "ledger-scan")
	names, scanErr := scan.Readdirnames(10200)
	scan.Close()
	if scanErr != nil && scanErr != io.EOF {
		return scanErr
	}
	if len(names) >= 10200 {
		return ErrUnavailable
	}
	count := 0
	for _, name := range names {
		if !noncePattern.MatchString(name) {
			continue
		}
		count++
		var e ledgerEntry
		if err = l.read(name, &e); err != nil {
			return err
		}
		if e.ExpiresAt.Add(time.Minute).Before(now) {
			if err = unix.Unlinkat(int(l.dir.Fd()), name, 0); err != nil {
				return err
			}
			count--
		}
	}
	if count >= 10000 {
		return ErrUnavailable
	}
	return l.write(g.ID, ledgerEntry{Binding: g.Binding(), ExpiresAt: g.ExpiresAt, Result: outcome(g.ID, "indeterminate", "outcome_unknown")}, true)
}
func (l *ledger) finish(g Grant, result Result) error {
	return l.write(g.ID, ledgerEntry{g.Binding(), g.ExpiresAt, result}, false)
}
func ledgerStatus(p Policy, g Grant) (Result, error) {
	// Broker is not root: the same trusted open validates a root-owned,
	// service-group-readable record and its full grant binding.
	f, err := openTrusted(filepath.Join(p.LedgerPath, g.ID), false)
	if err != nil {
		return Result{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 64<<10+1))
	if err != nil || len(data) > 64<<10 {
		return Result{}, ErrUnavailable
	}
	var entry ledgerEntry
	if strictJSON(data, &entry) != nil || entry.Binding != g.Binding() || !ResultValid(entry.Result, g.ID) {
		return Result{}, ErrForbidden
	}
	return entry.Result, nil
}
