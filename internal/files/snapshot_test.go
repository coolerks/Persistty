package files

import (
	"bytes"
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type changingWriter struct {
	buffer bytes.Buffer
	change func()
}

func TestBatchSnapshotRevalidatesParentAndRejectsLeafLinks(t *testing.T) {
	root, base := rootFixture(t)
	out := filepath.Join(base, "outside")
	os.Mkdir(out, 0700)
	os.WriteFile(filepath.Join(out, "file"), []byte("outside sentinel"), 0600)
	os.Mkdir(filepath.Join(root.Path, "dir"), 0700)
	os.WriteFile(filepath.Join(root.Path, "dir/file"), []byte("inside"), 0600)
	w := &changingWriter{change: func() {
		os.Rename(filepath.Join(root.Path, "dir"), filepath.Join(root.Path, "old"))
		os.Symlink(out, filepath.Join(root.Path, "dir"))
	}}
	err := WalkSnapshotWithCopy(context.Background(), root, "dir", 10, func(_ string, _ Entry, copy SnapshotCopy) (bool, error) { _, _, err := copy(w, 100); return false, err })
	if err == nil || bytes.Contains(w.buffer.Bytes(), []byte("outside sentinel")) {
		t.Fatal("accepted replacement or read outside", err)
	}
	os.Symlink(filepath.Join(out, "file"), filepath.Join(root.Path, "leaf"))
	err = WalkSnapshotWithCopy(context.Background(), root, "", 10, func(p string, e Entry, copy SnapshotCopy) (bool, error) {
		if p != "leaf" {
			return false, nil
		}
		_, _, err := copy(io.Discard, 100)
		return false, err
	})
	if err == nil {
		t.Fatal("followed leaf link")
	}
}

func (w *changingWriter) Write(b []byte) (int, error) {
	if w.change != nil {
		change := w.change
		w.change = nil
		change()
	}
	return w.buffer.Write(b)
}
func TestSnapshotRejectsLinksSpecialFilesTraversalAndParentSwap(t *testing.T) {
	root, base := rootFixture(t)
	outside := filepath.Join(base, "outside")
	os.Mkdir(outside, 0700)
	os.WriteFile(filepath.Join(outside, "secret"), []byte("outside sentinel"), 0600)
	os.Symlink(filepath.Join(outside, "secret"), filepath.Join(root.Path, "leaf"))
	os.Symlink(outside, filepath.Join(root.Path, "parent"))
	unix.Mkfifo(filepath.Join(root.Path, "fifo"), 0600)
	for _, path := range []string{"leaf", "parent/secret", "../outside/secret", "fifo"} {
		var b bytes.Buffer
		if _, _, err := CopySnapshot(context.Background(), root, path, &b, 100); err == nil || bytes.Contains(b.Bytes(), []byte("outside sentinel")) {
			t.Fatalf("unsafe snapshot %s %v", path, err)
		}
	}
	if _, err := SnapshotEntries(context.Background(), root, "parent"); err == nil {
		t.Fatal("followed directory link")
	}
	os.Mkdir(filepath.Join(root.Path, "dir"), 0700)
	os.WriteFile(filepath.Join(root.Path, "dir/file"), []byte("inside"), 0600)
	w := &changingWriter{change: func() {
		os.Rename(filepath.Join(root.Path, "dir"), filepath.Join(root.Path, "old"))
		os.Symlink(outside, filepath.Join(root.Path, "dir"))
	}}
	if _, _, err := CopySnapshot(context.Background(), root, "dir/file", w, 100); err == nil || bytes.Contains(w.buffer.Bytes(), []byte("outside sentinel")) {
		t.Fatal("accepted parent replacement", err)
	}
}
func TestSnapshotBudgetCancellationAndMode(t *testing.T) {
	root, _ := rootFixture(t)
	os.WriteFile(filepath.Join(root.Path, "file"), []byte("content"), 0750)
	var b bytes.Buffer
	version, mode, err := CopySnapshot(context.Background(), root, "file", &b, 100)
	if err != nil || mode != 0750 || version.Size != 7 || b.String() != "content" {
		t.Fatal(version, mode, err)
	}
	b.Reset()
	if _, _, err := CopySnapshot(context.Background(), root, "file", &b, 3); !errors.Is(err, ErrTooLarge) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := WalkSnapshot(ctx, root, "", 5, func(string, Entry) (bool, error) { t.Fatal("visited cancelled input"); return false, nil }); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestEntryMetadataUsesPinnedDirectoryHandle(t *testing.T) {
	root, base := rootFixture(t)
	source := filepath.Join(root.Path, "dir")
	os.Mkdir(source, 0700)
	os.WriteFile(filepath.Join(source, "file"), []byte("inside"), 0600)
	dir, err := openMutationParent(root, "dir")
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	outside := filepath.Join(base, "outside")
	os.Mkdir(outside, 0700)
	os.WriteFile(filepath.Join(outside, "file"), []byte("outside sentinel"), 0600)
	os.Rename(source, filepath.Join(root.Path, "old"))
	os.Symlink(outside, source)
	entry, err := statEntry(dir, "file")
	if err != nil || entry.Size != 6 {
		t.Fatal("metadata escaped opened directory", entry, err)
	}
}
