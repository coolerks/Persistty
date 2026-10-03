package search

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"persistty/internal/config"
	"persistty/internal/files"
	"persistty/internal/storage"
	"persistty/internal/toolrunner"
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

// This setup intentionally needs neither rg nor git to verify missing-tool recovery.
func namesFixture(t *testing.T) (*Service, storage.Project, string) {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{}
	cfg.Storage.Path = filepath.Join(dir, "data/db.sqlite")
	store, err := storage.Open(context.Background(), cfg.Storage.Path, "test-only")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	p, err := store.CreateProject(context.Background(), "无工具", []string{root}, 0)
	if err != nil {
		t.Fatal(err)
	}
	return New(store, toolrunner.New("missing-git", filepath.Join(dir, "missing-rg"), cfg.ToolTimeout()), cfg), p, root
}

func TestNamesNativeFallbackAndIgnoreParity(t *testing.T) {
	s, p, root := namesFixture(t)
	put(t, root, ".git/info/exclude", "exclude-needle.txt\npriority-needle.txt\n")
	put(t, root, ".gitignore", "# comment\n*.tmp\n!keep-needle.tmp\nignored/\n!ignored/needle.txt\n/root-needle.txt\n**/deep-needle.txt\nfile[0-9]-needle.txt\nfile[!a]-needle.txt\n\\#needle.txt\n\\!needle.txt\nspace-needle.txt \\ \n!priority-needle.txt\n*.log\n")
	put(t, root, ".ignore", "priority-needle.txt\n!keep-needle.log\n")
	put(t, root, ".rgignore", "!priority-needle.txt\n{brace,other}-needle.txt\nposix[[:digit:]]-needle.txt\n")
	put(t, root, "nested/.gitignore", "!deep-needle.txt\nlocal-needle.txt\n!keep-needle.log\n")
	put(t, root, "nested/.ignore", "keep-needle.log\n")
	put(t, root, "nested/.rgignore", "!keep-needle.log\n")
	names := []string{"drop-needle.tmp", "keep-needle.tmp", "ignored/needle.txt", "root-needle.txt", "nested/root-needle.txt", "nested/deep-needle.txt", "deep-needle.txt", "file3-needle.txt", "fileb-needle.txt", "filea-needle.txt", "#needle.txt", "!needle.txt", "space-needle.txt  ", "priority-needle.txt", "exclude-needle.txt", "nested/local-needle.txt", "nested/keep-needle.log", "keep-needle.log", "normal-needle.bin", "brace-needle.txt", "other-needle.txt", "posix3-needle.txt", ".hidden/needle.txt", "node_modules/needle.txt", ".git/needle.txt"}
	for _, name := range names {
		put(t, root, name, "body must not be read")
	}
	outside := t.TempDir()
	put(t, outside, "needle.txt", "ROOT_OUTSIDE")
	if err := os.Symlink(outside, filepath.Join(root, "needle-link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "needle.txt"), filepath.Join(root, "needle-leaf.txt")); err != nil {
		t.Fatal(err)
	}
	expected := []string{".hidden/needle.txt", "filea-needle.txt", "keep-needle.log", "keep-needle.tmp", "nested/deep-needle.txt", "nested/keep-needle.log", "nested/root-needle.txt", "normal-needle.bin", "posix3-needle.txt", "priority-needle.txt"}
	check := func() FileNames {
		t.Helper()
		result, err := s.Names(context.Background(), p.ID, p.Version, "needle")
		if err != nil {
			t.Fatal(err)
		}
		paths := []string{}
		for _, item := range result.Items {
			paths = append(paths, item.Path)
		}
		if result.Truncated || !reflect.DeepEqual(paths, expected) {
			t.Fatalf("paths %q expected %q", paths, expected)
		}
		return result
	}
	native := check()
	// An installed binary that cannot understand the fixed options also falls back.
	script := filepath.Join(t.TempDir(), "unsupported-rg")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 2\n"), 0700); err != nil {
		t.Fatal(err)
	}
	s.Runner.Rg = script
	check()
	if rg, err := exec.LookPath("rg"); err == nil {
		s.Runner.Rg = rg
		primary := check()
		if !reflect.DeepEqual(native, primary) {
			t.Fatal("native and rg differ")
		}
		registered, err := s.Store.RegisteredFolder(context.Background(), p.ID, p.Folders[0].ID, p.Version)
		if err != nil {
			t.Fatal(err)
		}
		dir, clean, err := s.Runner.Stage(s.Config.ToolStagingPath())
		if err != nil {
			t.Fatal(err)
		}
		defer clean()
		reference, err := s.discoverTree(context.Background(), registered, filepath.Join(dir, "reference"), &discoveryBudget{}, func(string, string) {})
		if err != nil {
			t.Fatal(err)
		}
		fast, err := s.discoverCandidates(context.Background(), registered, &discoveryBudget{}, func(string, string) {})
		if err != nil || !reflect.DeepEqual(sortedKeys(reference), sortedKeys(fast)) {
			t.Fatalf("pruned discovery differs from real rg: %v", err)
		}
		clean()
	}
	s.Runner.Rg = filepath.Join(t.TempDir(), "missing")
	for i := 0; i < 105; i++ {
		put(t, root, fmt.Sprintf("limit-needle-%03d.txt", i), "")
	}
	result, err := s.Names(context.Background(), p.ID, p.Version, "limit-needle")
	if err != nil || len(result.Items) != 100 || !result.Truncated {
		t.Fatalf("limit %+v %v", result, err)
	}
	s.Config.Search.MaxEntries = 2
	result, err = s.Names(context.Background(), p.ID, p.Version, "needle")
	if err != nil || !result.Truncated {
		t.Fatalf("scan limit %+v %v", result, err)
	}
	s.Config.Search.MaxEntries = 0
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = s.Names(ctx, p.ID, p.Version, "needle"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel %v", err)
	}
	entries, err := os.ReadDir(s.Config.ToolStagingPath())
	if err != nil && !os.IsNotExist(err) || len(entries) != 0 {
		t.Fatalf("staging leak %v", err)
	}
}

func TestNamesNativeHonorsIgnoreWithoutRepositoryAndFailsControlLimits(t *testing.T) {
	s, p, root := namesFixture(t)
	put(t, root, ".gitignore", "needle-secret.txt\n")
	put(t, root, "needle-secret.txt", "secret")
	put(t, root, "needle-public.txt", "")
	result, err := s.Names(context.Background(), p.ID, p.Version, "needle")
	if err != nil || len(result.Items) != 1 || result.Items[0].Path != "needle-public.txt" {
		t.Fatalf("nonrepo %+v %v", result, err)
	}
	put(t, root, ".ignore", "*.txt\n")
	s.Config.Search.MaxBytes = 1
	if _, err := s.Names(context.Background(), p.ID, p.Version, "needle"); !errors.Is(err, toolrunner.ErrLimit) {
		t.Fatalf("must not bypass control limit %v", err)
	}
	s.Config.Search.MaxBytes = 0
	put(t, root, ".ignore", "")
	if err := os.Truncate(filepath.Join(root, ".ignore"), (1<<20)+1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Names(context.Background(), p.ID, p.Version, "needle"); !errors.Is(err, files.ErrTooLarge) {
		t.Fatalf("must not bypass control read %v", err)
	}
}

func TestNamesNativeMalformedIgnoreDoesNotPanic(t *testing.T) {
	s, p, root := namesFixture(t)
	put(t, root, "needle.txt", "")
	for _, pattern := range []string{"[", "[abc\\", "trailing\\"} {
		put(t, root, ".rgignore", pattern+"\n")
		result, err := s.Names(context.Background(), p.ID, p.Version, "needle")
		if err != nil || len(result.Items) != 1 {
			t.Fatalf("malformed pattern %q: %+v %v", pattern, result, err)
		}
	}
}

func TestNamesNativeIgnoreNamedDirectoriesRemainSearchable(t *testing.T) {
	s, p, root := namesFixture(t)
	for _, name := range []string{".gitignore", ".ignore", ".rgignore"} {
		put(t, root, name+"/needle.txt", "")
	}
	result, err := s.Names(context.Background(), p.ID, p.Version, "needle")
	if err != nil || len(result.Items) != 3 {
		t.Fatalf("ignore-named directories %+v %v", result, err)
	}
}
