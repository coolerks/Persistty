package transfer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"persistty/internal/config"
	"persistty/internal/files"
	"persistty/internal/storage"
)

func testTransfer(t *testing.T) (*Service, *storage.Store, config.Config, storage.Project, string) {
	t.Helper()
	base := t.TempDir()
	projectDir := filepath.Join(base, "project")
	if err := os.Mkdir(projectDir, 0700); err != nil {
		t.Fatal(err)
	}
	var cfg config.Config
	cfg.Storage.Path = filepath.Join(base, "data", "db.sqlite")
	cfg.Transfer.ChunkBytes = 64 << 10
	cfg.Transfer.MaxFileBytes = 1 << 20
	cfg.Transfer.MaxBatchBytes = 2 << 20
	cfg.Transfer.MaxStagingBytes = 3 << 20
	store, err := storage.Open(context.Background(), cfg.Storage.Path, "testhash")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	project, err := store.CreateProject(context.Background(), "项目", []string{projectDir}, 0)
	if err != nil {
		t.Fatal(err)
	}
	service, err := New(context.Background(), cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	return service, store, cfg, project, projectDir
}
func sum(data []byte) string { hash := sha256.Sum256(data); return hex.EncodeToString(hash[:]) }

func TestUploadResumeAndConflict(t *testing.T) {
	service, store, cfg, project, projectDir := testTransfer(t)
	ctx := context.Background()
	data := bytes.Repeat([]byte("a"), 100000)
	input := UploadInput{ProjectID: project.ID, FolderID: project.MainFolderID, ProjectVersion: 1, Path: "data.bin", Size: int64(len(data)), SHA256: sum(data)}
	state, err := service.Start(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	first := data[:cfg.ChunkBytes()]
	if _, err := service.Chunk(ctx, state.ID, 0, first, sum([]byte("wrong"))); !errors.Is(err, ErrHash) {
		t.Fatal("bad chunk hash accepted")
	}
	if _, err := service.Chunk(ctx, state.ID, 0, first, sum(first)); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Chunk(ctx, state.ID, 0, first, sum(first)); err != nil {
		t.Fatal("identical duplicate not idempotent")
	}
	if _, err := service.Complete(ctx, state.ID); !errors.Is(err, ErrInvalid) {
		t.Fatal("incomplete upload completed")
	}
	restarted, err := New(ctx, cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	status, err := restarted.Status(ctx, state.ID)
	if err != nil || len(status.Received) != 1 || status.Received[0] != 0 {
		t.Fatalf("restore=%#v %v", status, err)
	}
	second := data[cfg.ChunkBytes():]
	if _, err := restarted.Chunk(ctx, state.ID, 1, second, sum(second)); err != nil {
		t.Fatal(err)
	}
	result, err := restarted.Complete(ctx, state.ID)
	if err != nil || result.State != "uploaded" {
		t.Fatalf("complete=%#v %v", result, err)
	}
	if repeated, err := restarted.Complete(ctx, state.ID); err != nil || repeated.State != "uploaded" {
		t.Fatal("completed upload not idempotent")
	}
	read, err := os.ReadFile(filepath.Join(projectDir, "data.bin"))
	if err != nil || !bytes.Equal(read, data) {
		t.Fatal("published bytes differ")
	}
	skip, err := restarted.Start(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Chunk(ctx, skip.ID, 0, first, sum(first)); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Chunk(ctx, skip.ID, 1, second, sum(second)); err != nil {
		t.Fatal(err)
	}
	if result, err := restarted.Complete(ctx, skip.ID); err != nil || result.State != "skipped" {
		t.Fatalf("same content not skipped: %#v %v", result, err)
	}
	different := []byte("changed")
	input.Size = int64(len(different))
	input.SHA256 = sum(different)
	conflict, err := restarted.Start(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Chunk(ctx, conflict.ID, 0, different, sum(different)); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Complete(ctx, conflict.ID); !errors.Is(err, files.ErrConflict) {
		t.Fatalf("different content replaced silently: %v", err)
	}
	root, err := store.RegisteredFolder(ctx, project.ID, project.MainFolderID, 1)
	if err != nil {
		t.Fatal(err)
	}
	current, err := files.ReadContent(root, "data.bin")
	if err != nil {
		t.Fatal(err)
	}
	input.ExpectedVersion = &current.Version
	confirmed, err := restarted.Start(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Chunk(ctx, confirmed.ID, 0, different, sum(different)); err != nil {
		t.Fatal(err)
	}
	if result, err := restarted.Complete(ctx, confirmed.ID); err != nil || result.State != "uploaded" {
		t.Fatalf("confirmed replace=%#v %v", result, err)
	}
	read, err = os.ReadFile(filepath.Join(projectDir, "data.bin"))
	if err != nil || !bytes.Equal(read, different) {
		t.Fatal("confirmed replacement incorrect")
	}
}

func TestUploadReplaceRejectsChangedTargetAfterConfirmation(t *testing.T) {
	service, store, _, project, projectDir := testTransfer(t)
	ctx := context.Background()
	target := filepath.Join(projectDir, "target.txt")
	if err := os.WriteFile(target, []byte("before"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := store.RegisteredFolder(ctx, project.ID, project.MainFolderID, project.Version)
	if err != nil {
		t.Fatal(err)
	}
	content, err := files.ReadContent(root, "target.txt")
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("replacement")
	state, err := service.Start(ctx, UploadInput{
		ProjectID: project.ID, FolderID: project.MainFolderID, ProjectVersion: project.Version,
		Path: "target.txt", Size: int64(len(data)), SHA256: sum(data), ExpectedVersion: &content.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Chunk(ctx, state.ID, 0, data, sum(data)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("changed after confirmation"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Complete(ctx, state.ID); !errors.Is(err, files.ErrConflict) {
		t.Fatalf("changed target was overwritten: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "changed after confirmation" {
		t.Fatalf("target changed on conflict: %q, %v", got, err)
	}
}

func TestUploadQuotaAndExpiration(t *testing.T) {
	service, _, _, project, _ := testTransfer(t)
	ctx := context.Background()
	service.cfg.Transfer.MaxFileBytes = 5
	service.cfg.Transfer.MaxBatchBytes = 6
	service.cfg.Transfer.MaxStagingBytes = 7
	input := UploadInput{
		ProjectID: project.ID, FolderID: project.MainFolderID, ProjectVersion: project.Version,
		Path: "first", BatchID: "quota-batch", Size: 4, SHA256: sum([]byte("four")),
	}
	if _, err := service.Start(ctx, UploadInput{ProjectID: input.ProjectID, FolderID: input.FolderID, ProjectVersion: input.ProjectVersion, Path: "oversized", Size: 6, SHA256: input.SHA256}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("single-file limit ignored: %v", err)
	}
	first, err := service.Start(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	input.Path = "second"
	if _, err := service.Start(ctx, input); !errors.Is(err, storage.ErrQuota) {
		t.Fatalf("batch quota ignored: %v", err)
	}
	input.BatchID = "other-batch"
	if _, err := service.Start(ctx, input); !errors.Is(err, storage.ErrQuota) {
		t.Fatalf("staging quota ignored: %v", err)
	}
	if err := service.Cancel(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	service.cfg.Transfer.UploadTTL = "1ns"
	expired, err := service.Start(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Status(ctx, expired.ID); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired transfer remained usable: %v", err)
	}
	if err := service.CleanExpired(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Status(ctx, expired.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expired transfer metadata remained: %v", err)
	}
	if _, err := os.Stat(service.partPath(expired.ID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expired staging file remained: %v", err)
	}
}
