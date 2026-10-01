package gitview

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"persistty/internal/config"
	"persistty/internal/storage"
	"persistty/internal/toolrunner"
	"strings"
	"testing"
)

func command(t *testing.T, dir string, args ...string) []byte {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid", "GIT_COMMITTER_NAME=Fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid")
	b, e := c.CombinedOutput()
	if e != nil {
		t.Fatalf("fixture git %v: %s", e, b)
	}
	return b
}
func fixture(t *testing.T) (*Service, storage.Project, string) {
	t.Helper()
	git, e := exec.LookPath("git")
	if e != nil {
		t.Skip("Git unavailable")
	}
	dir := t.TempDir()
	root := filepath.Join(dir, "repo")
	os.Mkdir(root, 0700)
	command(t, root, "init", "--quiet")
	os.WriteFile(filepath.Join(root, "file.txt"), []byte("one\ntwo\n"), 0600)
	command(t, root, "add", "--", "file.txt")
	command(t, root, "commit", "--quiet", "-m", "initial")
	cfg := config.Config{}
	cfg.Storage.Path = filepath.Join(dir, "data/db.sqlite")
	store, e := storage.Open(context.Background(), cfg.Storage.Path, "test-only")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { store.Close() })
	p, e := store.CreateProject(context.Background(), "Git", []string{root}, 0)
	if e != nil {
		t.Fatal(e)
	}
	return New(store, toolrunner.New(git, "rg", cfg.ToolTimeout()), cfg), p, root
}
func repoID(t *testing.T, s *Service, p storage.Project) string {
	t.Helper()
	repos, e := s.Repositories(context.Background(), p.ID, p.Version)
	if e != nil || len(repos.Items) != 1 {
		t.Fatalf("repos %#v %v", repos, e)
	}
	return repos.Items[0].ID
}
func TestGitStatusComparisonsHistoryAreReadOnly(t *testing.T) {
	s, p, root := fixture(t)
	ctx := context.Background()
	id := repoID(t, s, p)
	originalIndex, _ := os.ReadFile(filepath.Join(root, ".git/index"))
	originalHead, _ := os.ReadFile(filepath.Join(root, ".git/HEAD"))
	os.WriteFile(filepath.Join(root, "file.txt"), []byte("one\nstaged\n"), 0600)
	command(t, root, "add", "--", "file.txt")
	originalIndex, _ = os.ReadFile(filepath.Join(root, ".git/index"))
	os.WriteFile(filepath.Join(root, "file.txt"), []byte("one\nworking\n"), 0600)
	os.WriteFile(filepath.Join(root, "-中文\nnew.txt"), []byte("new"), 0600)
	status, e := s.Status(ctx, p.ID, p.Version, id)
	if e != nil {
		t.Fatal(e)
	}
	if len(status.Changes) != 2 || status.Head == "" {
		t.Fatalf("status %#v", status)
	}
	for _, c := range status.Changes {
		if c.Path == "file.txt" && (c.Index != "M" || c.Worktree != "M") {
			t.Fatalf("staged %v", c)
		}
	}
	for _, kind := range []string{"head", "staged", "unstaged"} {
		comp, e := s.Compare(ctx, p.ID, id, CompareInput{ProjectVersion: p.Version, Path: "file.txt", Kind: kind})
		if e != nil {
			t.Fatal(kind, e)
		}
		switch kind {
		case "head":
			if comp.Original != "one\ntwo\n" || comp.Modified != "one\nworking\n" {
				t.Fatal(comp)
			}
		case "staged":
			if comp.Modified != "one\nstaged\n" {
				t.Fatal(comp)
			}
		case "unstaged":
			if comp.Original != "one\nstaged\n" {
				t.Fatal(comp)
			}
		}
	}
	refs, e := s.Refs(ctx, p.ID, p.Version, id)
	if e != nil || len(refs) != 1 {
		t.Fatal(refs, e)
	}
	log, e := s.Log(ctx, p.ID, p.Version, id, 0)
	if e != nil || len(log.Items) != 1 {
		t.Fatal(log, e)
	}
	detail, e := s.Detail(ctx, p.ID, p.Version, id, log.Items[0].ID, "")
	if e != nil || detail.ParentID != "" || len(detail.Files) != 1 {
		t.Fatal(detail, e)
	}
	baseline, e := s.Baseline(ctx, p.ID, p.Version, p.Folders[0].ID, "file.txt")
	if e != nil || baseline.State != "tracked" || baseline.Content != "one\ntwo\n" {
		t.Fatal(baseline, e)
	}
	currentIndex, _ := os.ReadFile(filepath.Join(root, ".git/index"))
	currentHead, _ := os.ReadFile(filepath.Join(root, ".git/HEAD"))
	if string(currentIndex) != string(originalIndex) || string(currentHead) != string(originalHead) {
		t.Fatal("source metadata mutated")
	}
}
func TestGitMaliciousConfigAndObjectIndirection(t *testing.T) {
	s, p, root := fixture(t)
	id := repoID(t, s, p)
	marker := filepath.Join(t.TempDir(), "ran")
	command(t, root, "config", "diff.evil.command", "touch "+marker)
	command(t, root, "config", "core.fsmonitor", "touch "+marker)
	command(t, root, "config", "include.path", filepath.Join(t.TempDir(), "outside"))
	os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("*.txt diff=evil\n"), 0600)
	os.WriteFile(filepath.Join(root, "file.txt"), []byte("changed\n"), 0600)
	_, e := s.Status(context.Background(), p.ID, p.Version, id)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Compare(context.Background(), p.ID, id, CompareInput{ProjectVersion: p.Version, Path: "file.txt", Kind: "head"}); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(marker); !os.IsNotExist(e) {
		t.Fatal("helper executed")
	}
	os.WriteFile(filepath.Join(root, ".git/objects/info/alternates"), []byte("/outside\n"), 0600)
	if _, e = s.Status(context.Background(), p.ID, p.Version, id); !errors.Is(e, ErrUnavailable) {
		t.Fatal("alternates accepted", e)
	}
}
func TestGitSymlinkSnapshotAndNestedOwnership(t *testing.T) {
	s, p, root := fixture(t)
	outside := filepath.Join(t.TempDir(), "secret")
	os.WriteFile(outside, []byte("SENTINEL"), 0600)
	os.Symlink(outside, filepath.Join(root, "link"))
	command(t, root, "add", "--", "link")
	command(t, root, "commit", "--quiet", "-m", "link")
	id := repoID(t, s, p)
	status, e := s.Status(context.Background(), p.ID, p.Version, id)
	if e != nil {
		t.Fatal(e)
	}
	for _, c := range status.Changes {
		if c.Path == "link" {
			t.Fatal("symlink snapshot changed type", c)
		}
	}
	comp, e := s.Compare(context.Background(), p.ID, id, CompareInput{ProjectVersion: p.Version, Path: "link", Kind: "head"})
	if e != nil || comp.Modified != outside || strings.Contains(comp.Modified, "SENTINEL") {
		t.Fatal(comp, e)
	}
	nested := filepath.Join(root, "nested")
	os.Mkdir(nested, 0700)
	command(t, nested, "init", "--quiet")
	os.WriteFile(filepath.Join(nested, "file"), []byte("inner"), 0600)
	command(t, nested, "add", "--", "file")
	command(t, nested, "commit", "--quiet", "-m", "inner")
	repos, e := s.Repositories(context.Background(), p.ID, p.Version)
	if e != nil || len(repos.Items) != 2 {
		t.Fatal(repos, e)
	}
	baseline, e := s.Baseline(context.Background(), p.ID, p.Version, p.Folders[0].ID, "nested/file")
	if e != nil || baseline.Content != "inner" || baseline.RepoID == id {
		t.Fatal(baseline, e)
	}
}
func TestTotalHEADDiffMissingObjectMergeAndDeletedPath(t *testing.T) {
	s, p, root := fixture(t)
	ctx := context.Background()
	id := repoID(t, s, p)
	os.WriteFile(filepath.Join(root, "file.txt"), []byte("staged\n"), 0600)
	command(t, root, "add", "--", "file.txt")
	os.WriteFile(filepath.Join(root, "file.txt"), []byte("one\ntwo\n"), 0600)
	status, err := s.Status(ctx, p.ID, p.Version, id)
	if err != nil || len(status.Changes) != 1 || len(status.TotalPaths) != 0 {
		t.Fatalf("HEAD vs index %#v %v", status, err)
	}
	command(t, root, "reset", "--hard", "--quiet")
	main := strings.TrimSpace(string(command(t, root, "symbolic-ref", "--short", "HEAD")))
	command(t, root, "checkout", "--quiet", "-b", "side")
	os.WriteFile(filepath.Join(root, "side.txt"), []byte("side\n"), 0600)
	command(t, root, "add", "--", "side.txt")
	command(t, root, "commit", "--quiet", "-m", "side")
	command(t, root, "checkout", "--quiet", main)
	os.WriteFile(filepath.Join(root, "main.txt"), []byte("main\n"), 0600)
	command(t, root, "add", "--", "main.txt")
	command(t, root, "commit", "--quiet", "-m", "main")
	command(t, root, "merge", "--quiet", "--no-ff", "side", "-m", "merge")
	head := strings.TrimSpace(string(command(t, root, "rev-parse", "HEAD")))
	detail, err := s.Detail(ctx, p.ID, p.Version, id, head, "")
	if err != nil || len(detail.Commit.Parents) != 2 || detail.ParentID != detail.Commit.Parents[0] {
		t.Fatalf("merge %#v %v", detail, err)
	}
	second, err := s.Detail(ctx, p.ID, p.Version, id, head, detail.Commit.Parents[1])
	if err != nil || second.ParentID != detail.Commit.Parents[1] {
		t.Fatal(second, err)
	}
	os.Remove(filepath.Join(root, "file.txt"))
	comparison, err := s.Compare(ctx, p.ID, id, CompareInput{ProjectVersion: p.Version, Path: "file.txt", Kind: "head"})
	if err != nil || comparison.Original != "one\ntwo\n" || comparison.Modified != "" {
		t.Fatal(comparison, err)
	}
	object := strings.TrimSpace(string(command(t, root, "rev-parse", "HEAD:file.txt")))
	os.Remove(filepath.Join(root, ".git/objects", object[:2], object[2:]))
	if _, err = s.Compare(ctx, p.ID, id, CompareInput{ProjectVersion: p.Version, Path: "file.txt", Kind: "head"}); err == nil {
		t.Fatal("missing blob reported empty")
	}
	if _, err = s.Baseline(ctx, p.ID, p.Version, p.MainFolderID, "file.txt"); err == nil {
		t.Fatal("missing object became untracked")
	}
}
func TestOverlappingFolderBaselineAndExternalAttributes(t *testing.T) {
	s, p, root := fixture(t)
	os.Mkdir(filepath.Join(root, "src"), 0700)
	os.WriteFile(filepath.Join(root, "src/file.txt"), []byte("nested\n"), 0600)
	command(t, root, "add", "--", "src/file.txt")
	command(t, root, "commit", "--quiet", "-m", "nested")
	updated, err := s.Store.UpdateProject(context.Background(), p.ID, storage.ProjectChange{ExpectedVersion: p.Version, Name: p.Name, AddPaths: []string{filepath.Join(root, "src")}, MainFolderID: p.MainFolderID})
	if err != nil {
		t.Fatal(err)
	}
	nestedFolder := ""
	for _, folder := range updated.Folders {
		if folder.ID != p.MainFolderID {
			nestedFolder = folder.ID
		}
	}
	baseline, err := s.Baseline(context.Background(), p.ID, updated.Version, nestedFolder, "file.txt")
	if err != nil || baseline.State != "tracked" || baseline.Content != "nested\n" {
		t.Fatal(baseline, err)
	}
	os.MkdirAll(filepath.Join(root, ".git/info"), 0700)
	os.WriteFile(filepath.Join(root, ".git/info/attributes"), []byte("*.txt filter=external\n"), 0600)
	id := repoID(t, s, updated)
	if _, err := s.Status(context.Background(), p.ID, updated.Version, id); err == nil {
		t.Fatal("external content filter accepted")
	}
}
func TestHistoryPaginationPinsHeadAcrossNewCommits(t *testing.T) {
	s, p, root := fixture(t)
	ctx := context.Background()
	id := repoID(t, s, p)
	for i := 0; i < 51; i++ {
		os.WriteFile(filepath.Join(root, "file.txt"), []byte(fmt.Sprintf("revision %d\n", i)), 0600)
		command(t, root, "add", "--", "file.txt")
		command(t, root, "commit", "--quiet", "-m", fmt.Sprintf("commit %d", i))
	}
	first, err := s.Log(ctx, p.ID, p.Version, id, 0)
	if err != nil || len(first.Items) != 50 || first.NextOffset != 50 || first.Head != first.Items[0].ID {
		t.Fatal(first, err)
	}
	os.WriteFile(filepath.Join(root, "new.txt"), []byte("new\n"), 0600)
	command(t, root, "add", "--", "new.txt")
	command(t, root, "commit", "--quiet", "-m", "new HEAD")
	second, err := s.LogAt(ctx, p.ID, p.Version, id, first.NextOffset, first.Head)
	if err != nil || second.Head != first.Head || len(second.Items) != 2 || second.NextOffset != -1 {
		t.Fatal(second, err)
	}
	seen := map[string]bool{}
	for _, commit := range append(first.Items, second.Items...) {
		if seen[commit.ID] {
			t.Fatal("duplicate history item")
		}
		seen[commit.ID] = true
	}
}
func TestStagedRenameComparesOriginalPath(t *testing.T) {
	s, p, root := fixture(t)
	command(t, root, "mv", "--", "file.txt", "renamed.txt")
	id := repoID(t, s, p)
	comparison, err := s.Compare(context.Background(), p.ID, id, CompareInput{ProjectVersion: p.Version, Path: "renamed.txt", Kind: "staged"})
	if err != nil || comparison.OldPath != "file.txt" || comparison.Original != "one\ntwo\n" || comparison.Modified != comparison.Original {
		t.Fatal(comparison, err)
	}
}
