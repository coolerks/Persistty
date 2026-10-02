package gitview

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStatusPrunesIgnoredTreesButKeepsTrackedAndNestedExceptions(t *testing.T) {
	s, p, root := fixture(t)
	write := func(p, text string) {
		t.Helper()
		target := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("ignored/tracked.txt", "tracked\n")
	command(t, root, "add", "--", "ignored/tracked.txt")
	command(t, root, "commit", "--quiet", "-m", "track before ignoring")
	write(".gitignore", "bulk/\nignored/\n*.tmp\n!keep.tmp\n")
	write("ignored/tracked.txt", "modified\n")
	write("ignored/untracked.bin", strings.Repeat("x", 2<<20))
	write("bulk/huge.bin", strings.Repeat("x", 2<<20))
	write("drop.tmp", strings.Repeat("x", 2<<20))
	write("keep.tmp", "exception")
	write("src/.gitignore", "*.bin\n!keep.bin\n")
	write("src/drop.bin", strings.Repeat("x", 2<<20))
	write("src/keep.bin", "nested exception")
	// A pruned directory can contain more entries than an allowed enumeration.
	for i := 0; i < 10002; i++ {
		if err := os.WriteFile(filepath.Join(root, "bulk", fmt.Sprintf("ignored-%05d", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	s.Config.Git.MaxBytes = 1 << 20
	id := repoID(t, s, p)
	got, err := s.Status(context.Background(), p.ID, p.Version, id)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, c := range got.Changes {
		names[c.Path] = true
	}
	for _, name := range []string{"ignored/tracked.txt", "keep.tmp", "src/keep.bin", ".gitignore", "src/.gitignore"} {
		if !names[name] {
			t.Errorf("missing change %q: %#v", name, got)
		}
	}
	for _, name := range []string{"bulk/huge.bin", "ignored/untracked.bin", "drop.tmp", "src/drop.bin"} {
		if names[name] {
			t.Errorf("ignored change %q", name)
		}
	}
	// Compare the entire serialized path/status result against source Git in this
	// synthetic fixture only; product commands still receive private snapshots.
	raw := command(t, root, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if len(strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00")) != len(got.Changes) {
		t.Fatalf("CLI change count differs: %#v", got)
	}
	for _, c := range got.Changes {
		if !strings.Contains(string(raw), c.Index+c.Worktree+" "+c.Path+"\x00") {
			t.Errorf("CLI mismatch: %#v", c)
		}
	}
}

func TestIgnoredAttributesStillRejectExternalFilters(t *testing.T) {
	s, p, root := fixture(t)
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".gitattributes\n"), 0600)
	os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("* filter=external\n"), 0600)
	if _, err := s.Status(context.Background(), p.ID, p.Version, repoID(t, s, p)); err != ErrUnavailable {
		t.Fatal(err)
	}
}

func TestSnapshotOperationReceivesBudgetedContext(t *testing.T) {
	s, p, _ := fixture(t)
	if err := s.with(context.Background(), p.ID, p.Version, repoID(t, s, p), snapshotScope{}, func(ctx context.Context, _ *snapshot) error {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > s.Config.ToolTimeout() {
			t.Error("operation did not inherit the snapshot deadline")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
