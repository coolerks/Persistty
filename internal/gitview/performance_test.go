package gitview

import (
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"persistty/internal/config"
	"persistty/internal/storage"
	"persistty/internal/toolrunner"
	"testing"
	"time"
)

// Explicitly opt in to read-only checks of a local repository. All Git commands
// run in private snapshots; the source is never passed to Git or changed.
func TestLocalRepositoryPerformance(t *testing.T) {
	root := os.Getenv("PERSISTTY_GIT_PERF_ROOT")
	if root == "" {
		t.Skip("set PERSISTTY_GIT_PERF_ROOT for a read-only local performance check")
	}
	for _, name := range []string{"HEAD", "index", "config"} {
		file := filepath.Join(root, ".git", name)
		before, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			after, err := os.ReadFile(file)
			if err != nil || sha256.Sum256(before) != sha256.Sum256(after) {
				t.Errorf("source metadata changed: %s", name)
			}
		})
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{}
	cfg.Storage.Path = filepath.Join(t.TempDir(), "data/db.sqlite")
	store, err := storage.Open(context.Background(), cfg.Storage.Path, "performance-only")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	p, err := store.CreateProject(context.Background(), "Performance", []string{root}, 0)
	if err != nil {
		t.Fatal(err)
	}
	s := New(store, toolrunner.New(git, "rg", cfg.ToolTimeout()), cfg)
	start := time.Now()
	id := repoID(t, s, p)
	t.Logf("discovery=%s", time.Since(start))
	start = time.Now()
	status, err := s.Status(context.Background(), p.ID, p.Version, id)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("status=%s changes=%d", time.Since(start), len(status.Changes))
	start = time.Now()
	baseline, err := s.Baseline(context.Background(), p.ID, p.Version, p.Folders[0].ID, "AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("baseline=%s state=%s", time.Since(start), baseline.State)
	start = time.Now()
	if _, err = s.Compare(context.Background(), p.ID, id, CompareInput{ProjectVersion: p.Version, Path: "AGENTS.md", Kind: "head"}); err != nil {
		t.Fatal(err)
	}
	t.Logf("HEAD comparison=%s", time.Since(start))
	start = time.Now()
	log, err := s.Log(context.Background(), p.ID, p.Version, id, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("log=%s commits=%d", time.Since(start), len(log.Items))
	if len(log.Items) > 0 {
		start = time.Now()
		detail, err := s.Detail(context.Background(), p.ID, p.Version, id, log.Items[0].ID, "")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("detail=%s files=%d", time.Since(start), len(detail.Files))
		if len(detail.Files) > 0 {
			start = time.Now()
			_, err = s.Compare(context.Background(), p.ID, id, CompareInput{ProjectVersion: p.Version, Path: detail.Files[0], Kind: "commit", CommitID: detail.Commit.ID})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("commit comparison=%s", time.Since(start))
		}
	}
}
