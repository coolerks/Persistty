package transfer

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func waitArchive(t *testing.T, service *Service, id string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		item, err := service.ArchiveStatus(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if item.Status != "pending" {
			return item.Status
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("archive did not finish")
	return ""
}

func TestArchiveReadyAndSpecialFileFailure(t *testing.T) {
	service, _, _, project, root := testTransfer(t)
	ctx := context.Background()
	if err := os.MkdirAll(filepath.Join(root, "folder", "empty"), 0700); err != nil {
		t.Fatal(err)
	}
	content := []byte{0, 1, 255}
	if err := os.WriteFile(filepath.Join(root, "folder", "blob"), content, 0600); err != nil {
		t.Fatal(err)
	}
	input := ArchiveInput{ProjectID: project.ID, FolderID: project.MainFolderID, ProjectVersion: project.Version, Path: "folder"}
	item, err := service.CreateArchive(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if status := waitArchive(t, service, item.ID); status != "ready" {
		t.Fatalf("archive status = %s", status)
	}
	download, err := service.ArchiveDownload(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer download.Close()
	archive, err := io.ReadAll(download.File)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]byte{"folder/": nil, "folder/empty/": nil, "folder/blob": content}
	for _, entry := range reader.File {
		expected, ok := want[entry.Name]
		if !ok {
			t.Fatalf("unexpected entry %q", entry.Name)
		}
		stream, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(stream)
		stream.Close()
		if err != nil || !bytes.Equal(body, expected) {
			t.Fatalf("bad entry %q: %v", entry.Name, err)
		}
		delete(want, entry.Name)
	}
	if len(want) != 0 {
		t.Fatalf("missing entries: %v", want)
	}
	if err := os.Symlink("blob", filepath.Join(root, "folder", "link")); err != nil {
		t.Fatal(err)
	}
	bad, err := service.CreateArchive(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if status := waitArchive(t, service, bad.ID); status != "failed" {
		t.Fatalf("special file archive status = %s", status)
	}
	if _, err := service.ArchiveDownload(ctx, bad.ID); !errors.Is(err, ErrNotReady) {
		t.Fatalf("failed archive downloadable: %v", err)
	}
	if err := service.CancelArchive(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ArchiveDownload(ctx, item.ID); !errors.Is(err, ErrNotReady) {
		t.Fatalf("cancelled archive downloadable: %v", err)
	}
}
