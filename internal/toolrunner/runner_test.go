package toolrunner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPrivateStageCapacityEnvironmentAndCancel(t *testing.T) {
	base := filepath.Join(t.TempDir(), "staging")
	r := New("git", "rg", time.Second)
	dir, clean, err := r.Stage(base)
	if err != nil {
		t.Fatal(err)
	}
	defer clean()
	info, _ := os.Stat(dir)
	if info.Mode().Perm() != 0700 {
		t.Fatal("mode")
	}
	a, _ := r.Acquire(context.Background())
	b, _ := r.Acquire(context.Background())
	if _, err := r.Acquire(context.Background()); !errors.Is(err, ErrCapacity) {
		t.Fatal(err)
	}
	a()
	b()
	tool := filepath.Join(t.TempDir(), "tool")
	os.WriteFile(tool, []byte("#!/bin/sh\n/usr/bin/env\n"), 0700)
	r.Rg = tool
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("RIPGREP_CONFIG_PATH", "/outside")
	t.Setenv("SECRET", "private")
	out, code, err := r.Run(context.Background(), "rg", dir, nil)
	if err != nil || code != 0 || strings.Contains(string(out), "SECRET") || strings.Contains(string(out), "RIPGREP_CONFIG") || strings.Contains(string(out), "GIT_CONFIG_COUNT") {
		t.Fatalf("environment %d %v", code, err)
	}
	os.WriteFile(tool, []byte("#!/bin/sh\n/bin/sleep 30 &\nwait\n"), 0700)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, _, err = r.Run(ctx, "rg", dir, nil)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 2*time.Second {
		t.Fatal("cancel failed", err)
	}
	clean()
	entries, _ := os.ReadDir(base)
	if len(entries) != 0 {
		t.Fatal("staging leaked")
	}
}
func TestOutputAndUnsafeStageFailClosed(t *testing.T) {
	r := New("git", "rg", time.Second)
	tmp := t.TempDir()
	link := filepath.Join(tmp, "link")
	os.Symlink(t.TempDir(), link)
	if _, _, err := r.Stage(link); err == nil {
		t.Fatal("followed staging link")
	}
	b := &capped{max: 4}
	b.Write([]byte("1234"))
	if _, err := b.Write([]byte("5")); !errors.Is(err, ErrLimit) || !b.overflow {
		t.Fatal("unbounded output")
	}
	r.Rg = filepath.Join(tmp, "missing")
	if _, _, err := r.Run(context.Background(), "rg", tmp, nil); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}
