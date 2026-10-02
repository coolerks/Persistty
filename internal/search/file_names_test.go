package search

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"persistty/internal/storage"
	"reflect"
	"testing"
)

func TestNamesIgnoreUnicodeBinaryAndBoundaries(t *testing.T) {
	s, p, root := fixture(t)
	put(t, root, ".gitignore", "ignored/\n")
	put(t, root, ".ignore", "ignore-资料.txt\n")
	put(t, root, ".rgignore", "rg-资料.txt\n")
	put(t, root, ".git/info/exclude", "exclude-资料.txt\n")
	for _, name := range []string{"ignored/资料.txt", "ignore-资料.txt", "rg-资料.txt", "exclude-资料.txt", "node_modules/资料.txt", ".git/资料.txt"} {
		put(t, root, name, "secret")
	}
	put(t, root, ".github/资料😀.BIN", "\x00binary")
	put(t, root, "资料-large.bin", "")
	if err := os.Truncate(filepath.Join(root, "资料-large.bin"), 80<<20); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	put(t, outside, "资料-outside.txt", "ROOT_OUTSIDE")
	if err := os.Symlink(outside, filepath.Join(root, "资料-link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "资料-outside.txt"), filepath.Join(root, "资料-leaf.txt")); err != nil {
		t.Fatal(err)
	}
	result, err := s.Names(context.Background(), p.ID, p.Version, "  资料  ")
	if err != nil {
		t.Fatal(err)
	}
	want := []FileName{{p.Folders[0].ID, ".github/资料😀.BIN"}, {p.Folders[0].ID, "资料-large.bin"}}
	if result.Truncated || !reflect.DeepEqual(result.Items, want) {
		t.Fatalf("names %+v", result)
	}
	result, err = s.Names(context.Background(), p.ID, p.Version, "bin")
	if err != nil || len(result.Items) != 2 {
		t.Fatalf("case folding %+v %v", result, err)
	}
	for _, query := range []string{"", "  ", "資料\n", "x\x00", string(make([]byte, 257)), "\xff"} {
		if _, err := s.Names(context.Background(), p.ID, p.Version, query); !errors.Is(err, ErrInvalid) {
			t.Fatalf("query should fail: %v", err)
		}
	}
	if _, err := s.Names(context.Background(), p.ID, p.Version+1, "资料"); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("version %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Names(ctx, p.ID, p.Version, "资料"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel %v", err)
	}
	entries, err := os.ReadDir(s.Config.ToolStagingPath())
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("private staging leaked")
	}
}

func TestNamesOverlappingRootsAndTruncation(t *testing.T) {
	s, _, root := fixture(t)
	nested := filepath.Join(root, "nested")
	os.Mkdir(nested, 0700)
	p, err := s.Store.CreateProject(context.Background(), "重叠", []string{root, nested}, 0)
	if err != nil {
		t.Fatal(err)
	}
	put(t, root, "nested/needle.txt", "content")
	result, err := s.Names(context.Background(), p.ID, p.Version, "needle")
	if err != nil || len(result.Items) != 1 {
		t.Fatalf("overlap %+v %v", result, err)
	}
	for i := 0; i < 105; i++ {
		put(t, root, fmt.Sprintf("needle-%03d.txt", i), "")
	}
	result, err = s.Names(context.Background(), p.ID, p.Version, "needle")
	if err != nil || len(result.Items) != 100 || !result.Truncated {
		t.Fatalf("limit %+v %v", result, err)
	}
	s.Config.Search.MaxEntries = 2
	result, err = s.Names(context.Background(), p.ID, p.Version, "needle")
	if err != nil || !result.Truncated {
		t.Fatalf("entries %+v %v", result, err)
	}
}
