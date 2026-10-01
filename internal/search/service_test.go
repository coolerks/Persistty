package search

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"persistty/internal/config"
	"persistty/internal/files"
	"persistty/internal/storage"
	"persistty/internal/toolrunner"
	"strings"
	"testing"
)

func fixture(t *testing.T) (*Service, storage.Project, string) {
	t.Helper()
	rg, err := exec.LookPath("rg")
	if err != nil {
		t.Skip("rg unavailable")
	}
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	os.Mkdir(root, 0700)
	cfg := config.Config{}
	cfg.Storage.Path = filepath.Join(dir, "data/db.sqlite")
	store, err := storage.Open(context.Background(), cfg.Storage.Path, "test-only")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	p, err := store.CreateProject(context.Background(), "测试", []string{root}, 0)
	if err != nil {
		t.Fatal(err)
	}
	return New(store, toolrunner.New("git", rg, cfg.ToolTimeout()), cfg), p, root
}
func put(t *testing.T, root, path, text string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}
func TestSearchIgnoreUnicodeAndPreviewConflict(t *testing.T) {
	s, p, root := fixture(t)
	put(t, root, ".gitignore", "ignored/\n")
	os.Mkdir(filepath.Join(root, ".git"), 0700)
	put(t, root, "ignored/a.txt", "a12")
	put(t, root, "node_modules/a.txt", "a12")
	put(t, root, ".github/a.txt", "中文😀a12\r\nlast a34")
	put(t, root, "binary", "a12\x00")
	outside := filepath.Join(t.TempDir(), "sentinel")
	put(t, filepath.Dir(outside), filepath.Base(outside), "a12 ROOT_OUTSIDE")
	os.Symlink(outside, filepath.Join(root, "link"))
	q := Query{ProjectVersion: p.Version, Pattern: `a([0-9]+)`, Regex: true, CaseSensitive: true, Include: []string{"**/*.txt"}}
	result, err := s.Search(context.Background(), "owner", p.ID, q)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 || result.Files[0].Path != ".github/a.txt" {
		t.Fatalf("scope: %#v", result)
	}
	file := result.Files[0]
	if file.Matches[0].Column != 5 {
		t.Fatalf("UTF16=%d", file.Matches[0].Column)
	}
	preview, err := s.Preview(context.Background(), "owner", p.ID, PreviewInput{p.Version, result.ID, []string{file.Matches[0].ID}, "<$1>"})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := s.Comparison("owner", p.ID, preview.ID, file.ID)
	if err != nil || detail.Modified != "中文😀<12>\r\nlast a34" {
		t.Fatalf("replacement %q %v", detail.Modified, err)
	}
	put(t, root, file.Path, "external")
	out, err := s.Apply(context.Background(), "owner", p.ID, preview.ID, ApplyInput{p.Version, []string{file.ID}, nil})
	if err != nil || out.Results[0].State != "conflict" {
		t.Fatalf("conflict %#v %v", out, err)
	}
	bytes, _ := os.ReadFile(filepath.Join(root, file.Path))
	if string(bytes) != "external" {
		t.Fatal("overwrote external writer")
	}
	if _, err = s.Preview(context.Background(), "another", p.ID, PreviewInput{p.Version, result.ID, []string{file.Matches[0].ID}, "x"}); !errors.Is(err, ErrExpired) {
		t.Fatal("cross-session snapshot")
	}
}
func TestReplacePreservesBytesSelectionAndIdempotency(t *testing.T) {
	s, p, root := fixture(t)
	original := "\ufeffa12 b34\r\nx56\nlast78"
	put(t, root, "text", original)
	q := Query{ProjectVersion: p.Version, Pattern: `[a-z]([0-9]+)`, Regex: true}
	r, err := s.Search(context.Background(), "owner", p.ID, q)
	if err != nil {
		t.Fatal(err)
	}
	f := r.Files[0]
	ids := []string{}
	for _, m := range f.Matches {
		ids = append(ids, m.ID)
	}
	v, err := s.Preview(context.Background(), "owner", p.ID, PreviewInput{p.Version, r.ID, ids, "<$1>"})
	if err != nil {
		t.Fatal(err)
	}
	in := ApplyInput{p.Version, []string{f.ID}, nil}
	out, err := s.Apply(context.Background(), "owner", p.ID, v.ID, in)
	if err != nil || out.Results[0].State != "applied" {
		t.Fatalf("apply %v %#v", err, out)
	}
	actual, _ := os.ReadFile(filepath.Join(root, "text"))
	if string(actual) != "\ufeff<12> <34>\r\n<56>\nlas<78>" {
		t.Fatalf("bytes: %q", actual)
	}
	put(t, root, "text", "later")
	_, err = s.Apply(context.Background(), "owner", p.ID, v.ID, in)
	actual, _ = os.ReadFile(filepath.Join(root, "text"))
	if err != nil || string(actual) != "later" {
		t.Fatal("replayed completed apply")
	}
}
func TestSearchOverlappingRootsAndConfigurationChange(t *testing.T) {
	s, p, root := fixture(t)
	put(t, root, "nested/file", "match")
	next, err := s.Store.UpdateProject(context.Background(), p.ID, storage.ProjectChange{ExpectedVersion: p.Version, Name: p.Name, AddPaths: []string{filepath.Join(root, "nested")}})
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.Search(context.Background(), "owner", p.ID, Query{ProjectVersion: next.Version, Pattern: "match"})
	if err != nil || len(r.Files) != 1 {
		t.Fatalf("dedup %v %#v", err, r)
	}
	f := r.Files[0]
	v, err := s.Preview(context.Background(), "owner", p.ID, PreviewInput{next.Version, r.ID, []string{f.Matches[0].ID}, "new"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Store.UpdateProject(context.Background(), p.ID, storage.ProjectChange{ExpectedVersion: next.Version, Name: "renamed"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.Apply(context.Background(), "owner", p.ID, v.ID, ApplyInput{next.Version, []string{f.ID}, nil})
	if err != nil || out.Results[0].State != "conflict" {
		t.Fatalf("configuration %#v %v", out, err)
	}
}
func TestInvalidPatternProtectedFilesAndCancel(t *testing.T) {
	s, p, root := fixture(t)
	put(t, root, "text", "a")
	if _, err := s.Search(context.Background(), "owner", p.ID, Query{ProjectVersion: p.Version, Pattern: "(?=a)", Regex: true}); !errors.Is(err, ErrPattern) {
		t.Fatal(err)
	}
	r, err := s.Search(context.Background(), "owner", p.ID, Query{ProjectVersion: p.Version, Pattern: "^", Regex: true})
	if err != nil {
		t.Fatal(err)
	}
	f := r.Files[0]
	v, err := s.Preview(context.Background(), "owner", p.ID, PreviewInput{p.Version, r.ID, []string{f.Matches[0].ID}, "$0"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.Apply(context.Background(), "owner", p.ID, v.ID, ApplyInput{p.Version, []string{f.ID}, []string{f.ID}})
	if err != nil || out.Results[0].Reason != "unsaved_input" {
		t.Fatalf("protected %v %#v", err, out)
	}
	v, err = s.Preview(context.Background(), "owner", p.ID, PreviewInput{p.Version, r.ID, []string{f.Matches[0].ID}, "new"})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CancelPreview("owner", p.ID, v.ID); err != nil {
		t.Fatal(err)
	}
	out, err = s.Apply(context.Background(), "owner", p.ID, v.ID, ApplyInput{p.Version, []string{f.ID}, nil})
	if err != nil || out.State != "cancelled" {
		t.Fatal("cancel replay", err)
	}
	actual, _ := os.ReadFile(filepath.Join(root, "text"))
	if string(actual) != "a" {
		t.Fatal("cancel wrote")
	}
}
func TestIncludeDoesNotOverrideIgnoreAndLimits(t *testing.T) {
	s, p, root := fixture(t)
	put(t, root, ".gitignore", "ignored.txt\n")
	os.Mkdir(filepath.Join(root, ".git"), 0700)
	put(t, root, "ignored.txt", "needle")
	put(t, root, "other.txt", strings.Repeat("needle ", 10))
	s.Config.Search.MaxResults = 2
	r, err := s.Search(context.Background(), "owner", p.ID, Query{ProjectVersion: p.Version, Pattern: "needle", Include: []string{"*.txt"}})
	if err != nil || !r.Truncated || len(r.Files) != 1 || len(r.Files[0].Matches) != 2 {
		t.Fatalf("limits %#v %v", r, err)
	}
	folder, _ := s.Store.RegisteredFolder(context.Background(), p.ID, p.Folders[0].ID, p.Version)
	_, _, err = files.CopySnapshot(context.Background(), folder, "../outside", &strings.Builder{}, 100)
	if !errors.Is(err, files.ErrInvalidPath) {
		t.Fatal("traversal")
	}
}
func TestBOMBareCRCRLFAndTruncatedMatches(t *testing.T) {
	s, p, root := fixture(t)
	os.WriteFile(filepath.Join(root, "lines.txt"), []byte("\uFEFF😀hit\rhit\r\nhit"), 0600)
	q := Query{ProjectVersion: p.Version, Pattern: "hit", CaseSensitive: true}
	result, err := s.Search(context.Background(), "o", p.ID, q)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 || len(result.Files[0].Matches) != 3 {
		t.Fatalf("matches %#v", result)
	}
	matches := result.Files[0].Matches
	for i, m := range matches {
		want := 1
		if i == 0 {
			want = 3
		}
		if m.Line != i+1 || m.Column != want {
			t.Fatalf("location %#v", m)
		}
	}
	os.WriteFile(filepath.Join(root, "many.txt"), []byte(strings.Repeat("hit\n", 5002)), 0600)
	result, err = s.Search(context.Background(), "o", p.ID, q)
	if err != nil || !result.Truncated {
		t.Fatalf("limit %v %#v", err, result)
	}
}
func TestReleasedPreviewsFreeQuotaAndKeepReplayStatus(t *testing.T) {
	s, p, root := fixture(t)
	put(t, root, "text", "match")
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		r, err := s.Search(ctx, "o", p.ID, Query{ProjectVersion: p.Version, Pattern: "match"})
		if err != nil {
			t.Fatal(err)
		}
		v, err := s.Preview(ctx, "o", p.ID, PreviewInput{p.Version, r.ID, []string{r.Files[0].Matches[0].ID}, "new"})
		if err != nil {
			t.Fatal(err)
		}
		if err = s.CancelPreview("o", p.ID, v.ID); err != nil {
			t.Fatal(err)
		}
		out, err := s.Apply(ctx, "o", p.ID, v.ID, ApplyInput{p.Version, []string{v.Files[0].ID}, nil})
		if err != nil || out.State != "cancelled" {
			t.Fatal(out, err)
		}
		if _, err = s.Comparison("o", p.ID, v.ID, v.Files[0].ID); !errors.Is(err, ErrExpired) {
			t.Fatal("released body retained", err)
		}
		s.CancelSearch("o", p.ID, r.ID)
	}
	if len(s.previews) > 5 {
		t.Fatal("unbounded cancel tombstones", len(s.previews))
	}
}
