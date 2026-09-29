package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectMutationsPreserveFilesystem(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	root := t.TempDir()
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	for _, path := range []string{a, b} {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	p, err := s.CreateProject(ctx, "项目", []string{a, b}, 1)
	if err != nil || len(p.Folders) != 2 || p.MainFolderID != p.Folders[1].ID {
		t.Fatalf("create=%#v %v", p, err)
	}
	if _, err := s.CreateProject(ctx, "重复", []string{a, a}, 0); !errors.Is(err, ErrInvalidProject) {
		t.Fatal("duplicate root accepted")
	}
	if _, err := s.UpdateProject(ctx, p.ID, ProjectChange{ExpectedVersion: 2, Name: "新名"}); !errors.Is(err, ErrConflict) {
		t.Fatal("stale update accepted")
	}
	if _, err := s.UpdateProject(ctx, p.ID, ProjectChange{ExpectedVersion: 1, Name: "新名", RemoveFolderIDs: []string{p.MainFolderID}}); !errors.Is(err, ErrInvalidProject) {
		t.Fatal("removed main without replacement")
	}
	next, err := s.UpdateProject(ctx, p.ID, ProjectChange{ExpectedVersion: 1, Name: "新名", RemoveFolderIDs: []string{p.MainFolderID}, MainFolderID: p.Folders[0].ID})
	if err != nil || next.Version != 2 || len(next.Folders) != 1 || next.MainFolderID != p.Folders[0].ID {
		t.Fatalf("update=%#v %v", next, err)
	}
	if err := s.DeleteProject(ctx, p.ID, 1); !errors.Is(err, ErrConflict) {
		t.Fatal("stale delete accepted")
	}
	if err := s.DeleteProject(ctx, p.ID, 2); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{a, b} {
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Fatal("project removal touched filesystem")
		}
	}
	if _, err := s.Project(ctx, p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("removed project still present")
	}
}
