//go:build linux

package w04

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"persistty/internal/config"
	"persistty/internal/files"
	"persistty/internal/storage"
	"persistty/internal/transfer"
)

func TestLinuxW04ProductFlow(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	first, second := filepath.Join(base, "first"), filepath.Join(base, "second")
	for _, directory := range []string{first, second} {
		if err := os.Mkdir(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	var cfg config.Config
	cfg.Storage.Path = filepath.Join(base, "data", "db.sqlite")
	cfg.Transfer.ChunkBytes = 64 << 10
	cfg.Transfer.MaxFileBytes = 1 << 20
	cfg.Transfer.MaxBatchBytes = 2 << 20
	cfg.Transfer.MaxStagingBytes = 3 << 20
	store, err := storage.Open(ctx, cfg.Storage.Path, "testhash")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	project, err := store.CreateProject(ctx, "W04隔离测试", []string{first, second}, 0)
	if err != nil {
		t.Fatal(err)
	}
	source, err := store.RegisteredFolder(ctx, project.ID, project.Folders[0].ID, project.Version)
	if err != nil {
		t.Fatal(err)
	}
	target, err := store.RegisteredFolder(ctx, project.ID, project.Folders[1].ID, project.Version)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(base, "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(first, "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := files.ReadContent(source, "escape"); err == nil {
		t.Fatal("openat2 escaped root")
	}
	if err := os.WriteFile(filepath.Join(first, "-文本"), []byte("hello\r\n"), 0640); err != nil {
		t.Fatal(err)
	}
	content, err := files.ReadContent(source, "-文本")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := files.Save(ctx, source, "-文本", content.Version, "updated\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := files.Save(ctx, source, "-文本", content.Version, "stale"); !errors.Is(err, files.ErrConflict) {
		t.Fatalf("stale save accepted: %v", err)
	}
	current, err := files.ReadContent(source, "-文本")
	if err != nil {
		t.Fatal(err)
	}
	op := files.Operation{Kind: "move", ProjectVersion: project.Version, SourceFolderID: source.FolderID, SourcePath: "-文本", TargetFolderID: target.FolderID, TargetPath: "moved.txt", ExpectedIdentity: current.Version.Identity, ExpectedVersion: &current.Version}
	if result, err := files.Execute(ctx, source, target, op); err != nil || !result.SourceRemoved {
		t.Fatalf("cross-root move=%#v %v", result, err)
	}
	if body, err := os.ReadFile(filepath.Join(second, "moved.txt")); err != nil || string(body) != "updated\n" {
		t.Fatal("moved bytes differ")
	}
	service, err := transfer.New(ctx, cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	data := bytes.Repeat([]byte("Z"), 100000)
	hash := sha256.Sum256(data)
	input := transfer.UploadInput{ProjectID: project.ID, FolderID: target.FolderID, ProjectVersion: project.Version, Path: "blob.bin", Size: int64(len(data)), SHA256: hex.EncodeToString(hash[:])}
	upload, err := service.Start(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	firstChunk := data[:cfg.ChunkBytes()]
	chunkHash := sha256.Sum256(firstChunk)
	if _, err := service.Chunk(ctx, upload.ID, 0, firstChunk, hex.EncodeToString(chunkHash[:])); err != nil {
		t.Fatal(err)
	}
	service, err = transfer.New(ctx, cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	status, err := service.Status(ctx, upload.ID)
	if err != nil || len(status.Received) != 1 {
		t.Fatalf("upload resume=%#v %v", status, err)
	}
	lastChunk := data[cfg.ChunkBytes():]
	lastHash := sha256.Sum256(lastChunk)
	if _, err := service.Chunk(ctx, upload.ID, 1, lastChunk, hex.EncodeToString(lastHash[:])); err != nil {
		t.Fatal(err)
	}
	if result, err := service.Complete(ctx, upload.ID); err != nil || result.State != "uploaded" {
		t.Fatalf("upload complete=%#v %v", result, err)
	}
	if err := os.Mkdir(filepath.Join(second, "empty"), 0700); err != nil {
		t.Fatal(err)
	}
	listing, err := files.List(target, "", "", 100)
	if err != nil || len(listing.Items) != 3 {
		t.Fatalf("Linux root listing=%#v %v", listing, err)
	}
	var directZIP bytes.Buffer
	if _, _, err := files.WriteZIP(ctx, target, "", &directZIP, cfg.MaxBatchBytes()); err != nil {
		t.Fatalf("direct ZIP: %v", err)
	}
	archive, err := service.CreateArchive(ctx, transfer.ArchiveInput{ProjectID: project.ID, FolderID: target.FolderID, ProjectVersion: project.Version, Path: ""})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		status, err := service.ArchiveStatus(ctx, archive.ID)
		if err != nil {
			t.Fatal(err)
		}
		if status.Status == "ready" {
			break
		}
		if status.Status == "failed" {
			t.Fatalf("ZIP failed: %s", status.ErrorCode)
		}
		time.Sleep(10 * time.Millisecond)
	}
	download, err := service.ArchiveDownload(ctx, archive.ID)
	if err != nil {
		t.Fatal(err)
	}
	archiveBytes, err := io.ReadAll(download.File)
	download.Close()
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	if err != nil {
		t.Fatal(err)
	}
	foundBlob, foundEmpty := false, false
	for _, entry := range reader.File {
		if entry.Name == "second/blob.bin" {
			foundBlob = true
		}
		if entry.Name == "second/empty/" {
			foundEmpty = true
		}
	}
	if !foundBlob || !foundEmpty {
		t.Fatal("ZIP structure incomplete")
	}
	if err := os.Rename(first, first+"-moved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(first, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := files.List(source, "", "", 100); !errors.Is(err, files.ErrRootChanged) {
		t.Fatalf("root replacement accepted: %v", err)
	}
}
