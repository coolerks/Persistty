//go:build linux

package files

import (
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"persistty/internal/storage"
	"persistty/internal/workspace"
	"testing"
)

func privilegedFixture(t *testing.T) (storage.RegisteredFolder, Version, string) {
	t.Helper()
	dir := t.TempDir()
	root, err := workspace.Resolve(dir)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "file.txt")
	if err = os.WriteFile(path, []byte("old"), 0640); err != nil {
		t.Fatal(err)
	}
	r := storage.RegisteredFolder{Path: root.Path, Device: root.Dev, Inode: root.Ino}
	snapshot, err := ReadContent(r, "file.txt")
	if err != nil {
		t.Fatal(err)
	}
	return r, snapshot.Version, path
}
func TestPrivilegedStagePreservesBytesModeAndUserXattr(t *testing.T) {
	r, v, path := privilegedFixture(t)
	if err := unix.Setxattr(path, "user.persistty-test", []byte("metadata"), 0); err != nil {
		t.Fatal("required xattr fixture unavailable", err)
	}
	content := []byte("\uFEFFa\r\nb\nc")
	p, err := PreparePrivileged(context.Background(), r, "file.txt", v, content)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	before, _ := os.ReadFile(path)
	if string(before) != "old" {
		t.Fatal("prepare changed original")
	}
	version, err := p.Commit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(content) || version.Size != int64(len(content)) {
		t.Fatal("changed bytes")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0640 {
		t.Fatal("mode changed")
	}
	value := make([]byte, 32)
	n, err := unix.Getxattr(path, "user.persistty-test", value)
	if err != nil || string(value[:n]) != "metadata" {
		t.Fatal("lost xattr", err)
	}
	if _, err = p.Commit(context.Background()); !errors.Is(err, ErrConflict) {
		t.Fatal("double commit", err)
	}
}
func TestPrivilegedStageAbortConflictAndTempTampering(t *testing.T) {
	for _, mode := range []string{"close", "cancel", "conflict", "metadata", "temp"} {
		t.Run(mode, func(t *testing.T) {
			r, v, path := privilegedFixture(t)
			p, err := PreparePrivileged(context.Background(), r, "file.txt", v, []byte("new"))
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "close":
				_ = p.Close()
			case "cancel":
				cancel()
			case "conflict":
				_ = os.WriteFile(path, []byte("external"), 0640)
			case "metadata":
				_ = os.Chmod(path, 0644)
			case "temp":
				_ = os.WriteFile(filepath.Join(r.Path, p.tempName), []byte("bad"), 0640)
			}
			if _, err = p.Commit(ctx); err == nil {
				t.Fatal("unsafe commit accepted")
			}
			got, _ := os.ReadFile(path)
			want := "old"
			if mode == "conflict" {
				want = "external"
			}
			if string(got) != want {
				t.Fatal("original modified")
			}
			_ = p.Close()
			entries, _ := os.ReadDir(r.Path)
			if len(entries) != 1 {
				t.Fatal("temp retained")
			}
		})
	}
}
func TestPrivilegedRejectsHardlinkSymlinkAndSpecial(t *testing.T) {
	for _, mode := range []string{"hardlink", "symlink", "fifo"} {
		t.Run(mode, func(t *testing.T) {
			r, v, path := privilegedFixture(t)
			switch mode {
			case "hardlink":
				if err := os.Link(path, filepath.Join(r.Path, "alias")); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				_ = os.Rename(path, filepath.Join(r.Path, "original"))
				_ = os.Symlink("original", path)
			case "fifo":
				_ = os.Remove(path)
				if err := unix.Mkfifo(path, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if p, err := PreparePrivileged(context.Background(), r, "file.txt", v, []byte("new")); err == nil {
				p.Close()
				t.Fatal("unsafe leaf accepted")
			}
		})
	}
}
