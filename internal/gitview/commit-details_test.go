package gitview

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGitHubOriginCanonicalURL(t *testing.T) {
	for _, origin := range []string{"https://github.com/owner/repo.git", "git@github.com:owner/repo.git", "ssh://git@github.com/owner/repo.git", "https://fake-token@github.com/owner/repo.git", "https://github.com:443/owner/repo/"} {
		if got := githubRepository(origin); got != "https://github.com/owner/repo" {
			t.Errorf("canonical link mismatch: %q", got)
		}
	}
	for _, origin := range []string{"https://github.com.evil.invalid/owner/repo", "javascript:alert(1)", "https://github.com/owner/repo?token=fake", "https://github.com/owner/repo#x", "https://github.com:8443/owner/repo", "file:///github.com/owner/repo", "https://github.com/../repo", "https://github.com/owner/%2e%2e", "https://gitlab.com/owner/repo", "https://github.com/owner/repo/extra"} {
		if githubRepository(origin) != "" {
			t.Error("unsafe/non-GitHub origin accepted")
		}
	}
}

func TestCommitFullMessageStatsAndOriginAreReadOnly(t *testing.T) {
	s, p, root := fixture(t)
	ctx := context.Background()
	command(t, root, "remote", "add", "origin", "git@github.com:fixture/repository.git")
	id := repoID(t, s, p)
	head := strings.TrimSpace(string(command(t, root, "rev-parse", "HEAD")))
	initial, err := s.Detail(ctx, p.ID, p.Version, id, head, "")
	if err != nil || len(initial.Stats) != 1 || initial.Stats[0].Status != "A" || *initial.Stats[0].Additions != 2 || *initial.Stats[0].Deletions != 0 || initial.ParentID != "" {
		t.Fatalf("root stats: %#v %v", initial, err)
	}
	for name, body := range map[string]string{"delete.txt": "delete\n", "modify.txt": "before\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	command(t, root, "add", "--all")
	command(t, root, "commit", "--quiet", "-m", "stats base")
	command(t, root, "mv", "--", "file.txt", "-改\t名\n文件.txt")
	if err := os.Remove(filepath.Join(root, "delete.txt")); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"modify.txt": "after\nextra\n", "new\t文件.txt": "一\n二\n", "binary.bin": "\x00\x01\x02"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	message := "完整标题\n\n" + strings.Repeat("多行说明 😀\n", 800)
	command(t, root, "add", "--all")
	command(t, root, "commit", "--quiet", "-m", message)
	head = strings.TrimSpace(string(command(t, root, "rev-parse", "HEAD")))
	before := map[string]string{}
	for _, name := range []string{"HEAD", "index", "config"} {
		b, err := os.ReadFile(filepath.Join(root, ".git", name))
		if err != nil {
			t.Fatal(err)
		}
		before[name] = string(b)
	}
	detail, err := s.Detail(ctx, p.ID, p.Version, id, head, "")
	if err != nil {
		t.Fatal(err)
	}
	wantMessage := strings.TrimSuffix(string(command(t, root, "show", "--no-patch", "--format=%B", head, "--")), "\n")
	if detail.Message != wantMessage || detail.GitHubURL != "https://github.com/fixture/repository/commit/"+head {
		t.Fatal("full message or sanitized origin mismatch")
	}
	stats := map[string]FileStat{}
	for _, stat := range detail.Stats {
		stats[stat.Path] = stat
	}
	if len(stats) != 5 || len(detail.Files) != 5 {
		t.Fatalf("stats/files count %d/%d", len(stats), len(detail.Files))
	}
	for name, counts := range map[string][2]int64{"modify.txt": {2, 1}, "delete.txt": {0, 1}, "new\t文件.txt": {2, 0}, "-改\t名\n文件.txt": {0, 0}} {
		stat := stats[name]
		if stat.Additions == nil || stat.Deletions == nil || *stat.Additions != counts[0] || *stat.Deletions != counts[1] {
			t.Fatalf("counts %q: %#v", name, stat)
		}
	}
	if stats["binary.bin"].Additions != nil || stats["binary.bin"].Deletions != nil || stats["binary.bin"].Status != "A" || stats["delete.txt"].Status != "D" || stats["modify.txt"].Status != "M" || stats["-改\t名\n文件.txt"].Status != "R" || stats["-改\t名\n文件.txt"].OldPath != "file.txt" {
		t.Fatal("binary/status/rename mismatch")
	}
	comparison, err := s.Compare(ctx, p.ID, id, CompareInput{ProjectVersion: p.Version, Path: "-改\t名\n文件.txt", Kind: "commit", CommitID: head})
	if err != nil || comparison.OldPath != "file.txt" || comparison.Original != "one\ntwo\n" || comparison.Original != comparison.Modified {
		t.Fatalf("rename comparison %#v %v", comparison, err)
	}
	for name, original := range before {
		b, err := os.ReadFile(filepath.Join(root, ".git", name))
		if err != nil || string(b) != original {
			t.Fatal("source metadata changed")
		}
	}
}

func TestStatsParserRejectsMalformedRecords(t *testing.T) {
	for _, data := range []string{"1\t2\tfile", "1\t2\tfile\x00", ":broken\x00file\x00", ":100644 100644 " + strings.Repeat("a", 40) + " " + strings.Repeat("b", 40) + " M\x00../escape\x001\t1\t../escape\x00"} {
		if _, err := parseFileStats([]byte(data)); err == nil {
			t.Error("malformed stat accepted")
		}
	}
	stats, err := parseFileStats(nil)
	if err != nil || !reflect.DeepEqual(stats, []FileStat{}) {
		t.Fatal("empty stats must be a list")
	}
}
