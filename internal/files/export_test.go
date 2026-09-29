package files

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifiedDownloadAndZIP(t *testing.T) {
	root, _ := rootFixture(t)
	if err := os.Mkdir(filepath.Join(root.Path, "folder"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root.Path, "folder", "empty"), 0700); err != nil {
		t.Fatal(err)
	}
	data := []byte{0, 1, 2, 3, 255}
	if err := os.WriteFile(filepath.Join(root.Path, "folder", "binary"), data, 0600); err != nil {
		t.Fatal(err)
	}
	metadata, err := ReadMetadata(root, "folder/binary")
	if err != nil || metadata.Kind != "file" || metadata.Version.Size != int64(len(data)) || metadata.Version.ETag == "" {
		t.Fatalf("binary metadata=%#v %v", metadata, err)
	}
	var downloaded bytes.Buffer
	version, err := CopyVerified(context.Background(), root, "folder/binary", &downloaded, 1024)
	if err != nil || version.Size != int64(len(data)) || !bytes.Equal(downloaded.Bytes(), data) {
		t.Fatalf("download=%#v %v", version, err)
	}
	if _, err := CopyVerified(context.Background(), root, "folder/binary", io.Discard, 2); err != ErrTooLarge {
		t.Fatal("download limit ignored")
	}
	var archive bytes.Buffer
	count, size, err := WriteZIP(context.Background(), root, "folder", &archive, 1024)
	if err != nil || count != 3 || size != int64(len(data)) {
		t.Fatalf("zip count=%d size=%d err=%v", count, size, err)
	}
	reader, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]byte{"folder/": nil, "folder/empty/": nil, "folder/binary": data}
	for _, file := range reader.File {
		expected, ok := want[file.Name]
		if !ok {
			t.Fatalf("unexpected ZIP entry %q", file.Name)
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(stream)
		stream.Close()
		if err != nil || !bytes.Equal(body, expected) {
			t.Fatalf("ZIP entry %s changed", file.Name)
		}
		delete(want, file.Name)
	}
	if len(want) != 0 {
		t.Fatal("ZIP omitted entries")
	}
	if err := os.Symlink("binary", filepath.Join(root.Path, "folder", "link")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := WriteZIP(context.Background(), root, "folder", io.Discard, 1024); err == nil {
		t.Fatal("ZIP followed symlink")
	}
}

func TestDirectorySnapshotRejectsNewEntry(t *testing.T) {
	root, _ := rootFixture(t)
	dir, err := openMutationParent(root, ".")
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	before, err := dir.ReadDir(10001)
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}
	snapshot, err := snapshotEntries(dir, before)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root.Path, "new"), []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyDirectorySnapshot(root, ".", dir, snapshot); err != ErrConflict {
		t.Fatalf("changed listing accepted: %v", err)
	}
}
