package gitview

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"persistty/internal/files"
	"persistty/internal/storage"
	"strings"
	"testing"
)

func TestObjectAndSingleFileReadsDoNotSnapshotUnrelatedWorktree(t *testing.T) {
	s, p, root := fixture(t)
	id := repoID(t, s, p)
	s.Config.Git.MaxBytes = 1 << 20
	// A full snapshot would exceed its budget. History, staged/commit comparison,
	// baseline and comparisons of an existing path must remain usable.
	if err := os.WriteFile(filepath.Join(root, "unrelated.bin"), []byte(strings.Repeat("x", 2<<20)), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	log, err := s.Log(ctx, p.ID, p.Version, id, 0)
	if err != nil || len(log.Items) != 1 {
		t.Fatal(log, err)
	}
	if _, err = s.Refs(ctx, p.ID, p.Version, id); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Detail(ctx, p.ID, p.Version, id, log.Items[0].ID, ""); err != nil {
		t.Fatal(err)
	}
	if b, err := s.Baseline(ctx, p.ID, p.Version, p.Folders[0].ID, "file.txt"); err != nil || b.Content != "one\ntwo\n" {
		t.Fatal(b, err)
	}
	for _, kind := range []string{"head", "unstaged", "staged", "commit"} {
		b, err := s.Compare(ctx, p.ID, id, CompareInput{ProjectVersion: p.Version, Path: "file.txt", Kind: kind, CommitID: log.Items[0].ID})
		if err != nil || b.Modified != "one\ntwo\n" {
			t.Fatal(kind, b, err)
		}
	}
	if _, err = s.Status(ctx, p.ID, p.Version, id); !errors.Is(err, files.ErrTooLarge) {
		t.Fatal("status must not silently omit worktree paths", err)
	}
}

func TestKnownRepositoryStillValidatesMetadataIdentityAndProjectVersion(t *testing.T) {
	s, p, root := fixture(t)
	id := repoID(t, s, p)
	if _, err := s.Log(context.Background(), p.ID, p.Version+1, id, 0); !errors.Is(err, storage.ErrConflict) {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, ".git"), filepath.Join(root, "old-git")); err != nil {
		t.Fatal(err)
	}
	command(t, root, "init", "--quiet")
	if _, err := s.Log(context.Background(), p.ID, p.Version, id, 0); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("stale opaque ID accepted replacement metadata", err)
	}
	newID := repoID(t, s, p)
	if newID == id {
		t.Fatal("metadata identity did not change")
	}
	log, err := s.Log(context.Background(), p.ID, p.Version, newID, 0)
	if err != nil || len(log.Items) != 0 {
		t.Fatal(log, err)
	}
}

func TestGitWorktreePathsCannotOverwritePrivateMetadata(t *testing.T) {
	s, p, _ := fixture(t)
	id := repoID(t, s, p)
	for _, path := range []string{".git/config", ".GIT/index", "nested/.GiT/config"} {
		if _, err := s.Baseline(context.Background(), p.ID, p.Version, p.Folders[0].ID, path); !errors.Is(err, ErrInvalid) {
			t.Fatal(path, err)
		}
		if _, err := s.Compare(context.Background(), p.ID, id, CompareInput{ProjectVersion: p.Version, Path: path, Kind: "head"}); !errors.Is(err, ErrInvalid) {
			t.Fatal(path, err)
		}
	}
}

func TestKnownUnsupportedRepositoryRetainsDomainError(t *testing.T) {
	s, p, root := fixture(t)
	if err := os.RemoveAll(filepath.Join(root, ".git")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: /outside\n"), 0600); err != nil {
		t.Fatal(err)
	}
	id := repoID(t, s, p)
	if _, err := s.Log(context.Background(), p.ID, p.Version, id, 0); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}
