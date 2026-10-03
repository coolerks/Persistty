package search

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"persistty/internal/toolrunner"
	"reflect"
	"strings"
	"testing"
)

func TestSearchWithoutToolsPreviewAndApply(t *testing.T) {
	s, p, root := namesFixture(t)
	original := "\ufeff中文😀a12\r\nlast a34\ra56\n"
	put(t, root, "src/文本.txt", original)
	put(t, root, ".gitignore", "ignored/\n")
	put(t, root, "ignored/skip.txt", "a99")
	put(t, root, "src/skip.txt", "a99")
	put(t, root, "src/file.go", "a99")
	put(t, root, "binary.txt", "a99\x00")
	outside := filepath.Join(t.TempDir(), "outside.txt")
	put(t, filepath.Dir(outside), filepath.Base(outside), "a99")
	if err := os.Symlink(outside, filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}
	q := Query{ProjectVersion: p.Version, Pattern: `a(?P<number>[0-9]+)`, Regex: true, CaseSensitive: true, Include: []string{"**/*.txt"}, Exclude: []string{"skip.txt"}}
	result, err := s.Search(context.Background(), "owner", p.ID, q)
	if err != nil || len(result.Files) != 1 || result.Files[0].Path != "src/文本.txt" {
		t.Fatalf("scope: %#v %v", result, err)
	}
	f := result.Files[0]
	if len(f.Matches) != 3 || f.Matches[0].Column != 5 || f.Matches[1].Line != 2 || f.Matches[2].Line != 3 {
		t.Fatalf("positions: %#v", f.Matches)
	}
	// A later tool installation must not change preview capture semantics.
	s.Runner.Rg = "/unused-after-search"
	preview, err := s.Preview(context.Background(), "owner", p.ID, PreviewInput{p.Version, result.ID, []string{f.Matches[0].ID, f.Matches[2].ID}, "<${number}>$$"})
	if err != nil {
		t.Fatal(err)
	}
	comparison, err := s.Comparison("owner", p.ID, preview.ID, f.ID)
	want := "\ufeff中文😀<12>$\r\nlast a34\r<56>$\n"
	if err != nil || comparison.Modified != want || comparison.Original != original {
		t.Fatalf("preview: %#v %v", comparison, err)
	}
	applied, err := s.Apply(context.Background(), "owner", p.ID, preview.ID, ApplyInput{p.Version, []string{f.ID}, nil})
	if err != nil || applied.Results[0].State != "applied" {
		t.Fatalf("apply: %#v %v", applied, err)
	}
	actual, err := os.ReadFile(filepath.Join(root, f.Path))
	if err != nil || string(actual) != want {
		t.Fatalf("bytes: %q %v", actual, err)
	}
}

func TestNativeEngineMatchesRGCommonPatterns(t *testing.T) {
	rg, err := exec.LookPath("rg")
	if err != nil {
		t.Skip("rg comparison unavailable; missing-tool tests still run")
	}
	runner := toolrunner.New("git", rg, 0)
	capabilityErr := rgReplacementCapability(context.Background(), runner, t.TempDir())
	if capabilityErr != nil && !errors.Is(capabilityErr, toolrunner.ErrUnavailable) {
		t.Fatal(capabilityErr)
	}
	content := "\ufeff中文😀a12 A34\r\ncat scatter cat\ra56\n中文 中文字符 _中文\n\n"
	for _, q := range []Query{
		{Pattern: "a", CaseSensitive: false}, {Pattern: "cat", WholeWord: true},
		{Pattern: "中文", WholeWord: true}, {Pattern: `a([0-9]+)`, Regex: true},
		{Pattern: `^`, Regex: true}, {Pattern: `$`, Regex: true},
	} {
		t.Run(q.Pattern, func(t *testing.T) {
			actual, err := nativeEngine(context.Background(), q, content, nil)
			if err != nil {
				t.Fatal(err)
			}
			want, err := rgEngine(context.Background(), runner, t.TempDir(), q, content, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(actual, want) {
				t.Fatalf("native %#v\nrg %#v", actual, want)
			}
			replacement := "<$1>$$"
			actual, err = nativeEngine(context.Background(), q, content, &replacement)
			if err != nil {
				t.Fatal(err)
			}
			for _, match := range actual {
				expected := replacement
				if q.Regex {
					expected = "<>$"
					if q.Pattern == `a([0-9]+)` {
						expected = "<" + content[match.Match.Start+1:match.Match.End] + ">$"
					}
				}
				if match.Replacement != expected {
					t.Fatalf("replacement %q, want %q", match.Replacement, expected)
				}
			}
			if capabilityErr == nil {
				want, err = rgEngine(context.Background(), runner, t.TempDir(), q, content, &replacement)
				if err != nil || !reflect.DeepEqual(actual, want) {
					t.Fatalf("replacement native %#v\nrg %#v: %v", actual, want, err)
				}
			}
		})
	}
}

func TestSearchWithLegacyRGReplacementCapability(t *testing.T) {
	s, p, root := fixture(t)
	// Retain real rg validation/matching but reproduce pre-15 replacement JSON.
	quoted := "'" + strings.ReplaceAll(s.Runner.Rg, "'", "'\"'\"'") + "'"
	script := filepath.Join(t.TempDir(), "legacy-rg")
	legacy := `{"type":"match","data":{"lines":{"text":"a\n"},"line_number":1,"absolute_offset":0,"submatches":[{"match":{"text":"a"},"start":0,"end":1}]}}`
	body := "#!/bin/sh\nfor arg do\nif [ \"$arg\" = --replace ]; then\ncat >/dev/null\nprintf '%s\\n' '" + legacy + "'\nexit 0\nfi\ndone\nexec " + quoted + " \"$@\"\n"
	if err := os.WriteFile(script, []byte(body), 0700); err != nil {
		t.Fatal(err)
	}
	s.Runner.Rg = script
	original := "\ufeff中文😀a12\r\na34\ra56\n"
	put(t, root, "text.txt", original)
	result, err := s.Search(context.Background(), "owner", p.ID, Query{ProjectVersion: p.Version, Pattern: `a(?P<number>[0-9]+)`, Regex: true})
	if err != nil || len(result.Files) != 1 || !result.Files[0].Native || len(result.Files[0].Matches) != 3 {
		t.Fatalf("legacy capability: %#v %v", result, err)
	}
	f := result.Files[0]
	// Installation/removal after search must not change the selected engine.
	s.Runner.Rg = "/unused-after-legacy-probe"
	preview, err := s.Preview(context.Background(), "owner", p.ID, PreviewInput{p.Version, result.ID, []string{f.Matches[0].ID, f.Matches[2].ID}, "<${number}>$$"})
	if err != nil {
		t.Fatal(err)
	}
	comparison, err := s.Comparison("owner", p.ID, preview.ID, f.ID)
	want := "\ufeff中文😀<12>$\r\na34\r<56>$\n"
	if err != nil || comparison.Original != original || comparison.Modified != want {
		t.Fatalf("legacy preview: %#v %v", comparison, err)
	}
	applied, err := s.Apply(context.Background(), "owner", p.ID, preview.ID, ApplyInput{p.Version, []string{f.ID}, nil})
	if err != nil || applied.Results[0].State != "applied" {
		t.Fatalf("legacy apply: %#v %v", applied, err)
	}
	actual, err := os.ReadFile(filepath.Join(root, f.Path))
	if err != nil || string(actual) != want {
		t.Fatalf("legacy bytes: %q %v", actual, err)
	}
}

func TestMissingToolsErrorsAndLimits(t *testing.T) {
	s, p, root := namesFixture(t)
	put(t, root, "text", strings.Repeat("x ", 6000))
	result, err := s.Search(context.Background(), "owner", p.ID, Query{ProjectVersion: p.Version, Pattern: "x"})
	if err != nil || !result.Truncated || len(result.Files[0].Matches) != 5000 {
		t.Fatalf("limit: %#v %v", result, err)
	}
	if _, err := s.Search(context.Background(), "owner", p.ID, Query{ProjectVersion: p.Version, Pattern: "(?=x)", Regex: true}); !errors.Is(err, ErrPattern) {
		t.Fatalf("invalid regex: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Search(ctx, "owner", p.ID, Query{ProjectVersion: p.Version, Pattern: "x"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	s.Config.Search.MaxBytes = 4
	result, err = s.Search(context.Background(), "owner", p.ID, Query{ProjectVersion: p.Version, Pattern: "x"})
	if err != nil || !result.Truncated || len(result.Files) != 0 {
		t.Fatalf("byte limit: %#v %v", result, err)
	}
}

func TestNativeSearchHandlesUnavailableCapabilitiesAndWordBudget(t *testing.T) {
	s, p, root := namesFixture(t)
	put(t, root, "file.txt", "cat CAT scatter")
	script := filepath.Join(t.TempDir(), "unsupported-rg")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 2\n"), 0700); err != nil {
		t.Fatal(err)
	}
	s.Runner.Rg = script
	result, err := s.Search(context.Background(), "owner", p.ID, Query{ProjectVersion: p.Version, Pattern: "cat", WholeWord: true, Include: []string{"*.txt"}})
	if err != nil || len(result.Files) != 1 || len(result.Files[0].Matches) != 2 || !result.Files[0].Native {
		t.Fatalf("capability: %#v %v", result, err)
	}
	_, err = nativeEngine(context.Background(), Query{Pattern: "x", WholeWord: true}, strings.Repeat("xx ", 6000)+"x", nil)
	if !errors.Is(err, toolrunner.ErrLimit) {
		t.Fatalf("must not report empty after bounded candidates: %v", err)
	}
}

func TestRGUnicodeEmptyMatchCapabilityFallsBackPerFile(t *testing.T) {
	s, p, root := fixture(t)
	q := Query{ProjectVersion: p.Version, Pattern: "a*", Regex: true}
	content := "中文😀a\n"
	put(t, root, "text", content)
	result, err := s.Search(context.Background(), "owner", p.ID, q)
	if err != nil || len(result.Files) != 1 {
		t.Fatalf("search: %#v %v", result, err)
	}
	f := result.Files[0]
	// Native coordinates always land on UTF-8 boundaries; source engines that
	// already produce valid offsets remain pinned to rg instead.
	for _, match := range f.Matches {
		if match.Column < 1 || match.EndColumn < match.Column {
			t.Fatalf("invalid position: %#v", match)
		}
	}
	preview, err := s.Preview(context.Background(), "owner", p.ID, PreviewInput{p.Version, result.ID, []string{f.Matches[0].ID}, "x"})
	if err != nil || len(preview.Files) != 1 {
		t.Fatalf("preview: %#v %v", preview, err)
	}
	actual, _ := os.ReadFile(filepath.Join(root, "text"))
	if string(actual) != content {
		t.Fatal("preview wrote source")
	}
}
