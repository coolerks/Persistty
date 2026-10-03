package search

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSearchPrunesIgnoredLargeDirectoryBeforeEnumeration(t *testing.T) {
	s, p, root := namesFixture(t)
	put(t, root, ".gitignore", "ignored/\n")
	put(t, root, "src/project.txt", "Backend\r\n")
	// Entering this directory would exceed both the per-directory and global limits.
	for i := 0; i < 10001; i++ {
		put(t, root, fmt.Sprintf("ignored/project-%05d.txt", i), "Backend")
	}
	s.Config.Search.MaxEntries = 20
	names, err := s.Names(context.Background(), p.ID, p.Version, "project")
	if err != nil || names.Truncated || len(names.Items) != 1 || names.Items[0].Path != "src/project.txt" {
		t.Fatalf("names: %+v %v", names, err)
	}
	result, err := s.Search(context.Background(), "owner", p.ID, Query{ProjectVersion: p.Version, Pattern: "Backend"})
	if err != nil || result.Truncated || len(result.Files) != 1 || result.Files[0].Path != "src/project.txt" {
		t.Fatalf("search: %+v %v", result, err)
	}
}

func TestRegexSearchBatchesToolStartupWithinBudget(t *testing.T) {
	s, p, root := fixture(t)
	for i := 0; i < 140; i++ {
		put(t, root, fmt.Sprintf("src/deep/project-%03d.txt", i), "中文😀Backend42\r\n")
	}
	script := filepath.Join(t.TempDir(), "slow-start-rg")
	quoted := "'" + strings.ReplaceAll(s.Runner.Rg, "'", "'\"'\"'") + "'"
	if err := os.WriteFile(script, []byte("#!/bin/sh\n/bin/sleep 0.03\nexec "+quoted+" \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	s.Runner.Rg = script
	s.Config.Search.TimeoutSeconds = 2
	start := time.Now()
	result, err := s.Search(context.Background(), "owner", p.ID, Query{ProjectVersion: p.Version, Pattern: `Backend([0-9]+)`, Regex: true})
	if err != nil || result.Truncated || len(result.Files) != 140 {
		t.Fatalf("elapsed=%s files=%d truncated=%v error=%v", time.Since(start), len(result.Files), result.Truncated, err)
	}
	for _, file := range result.Files {
		if len(file.Matches) != 1 || file.Matches[0].Column != 5 {
			t.Fatalf("positions: %+v", file.Matches)
		}
	}
	entries, err := os.ReadDir(s.Config.ToolStagingPath())
	if err != nil || len(entries) != 0 {
		t.Fatalf("staging cleanup: %v", err)
	}
}

func TestRGBatchRetainsPerFileCoordinatesAndReplacementEngine(t *testing.T) {
	s, p, root := fixture(t)
	texts := []string{"\ufeff中文😀a12\r\na34\ra56\n", "a78\n", "no matches"}
	batch := make([]File, len(texts))
	for i, text := range texts {
		batch[i].Content = text
	}
	q := Query{ProjectVersion: p.Version, Pattern: `a([0-9]+)`, Regex: true}
	dir, clean, err := s.Runner.Stage(s.Config.ToolStagingPath())
	if err != nil {
		t.Fatal(err)
	}
	defer clean()
	values, failures, err := rgBatch(context.Background(), s.Runner, dir, q, batch)
	if err != nil {
		t.Fatal(err)
	}
	for i, text := range texts {
		want, err := rgEngine(context.Background(), s.Runner, dir, q, text, nil)
		if err != nil || failures[i] != nil || !reflect.DeepEqual(values[i], want) {
			t.Fatalf("file %d error=%v failure=%v", i, err, failures[i])
		}
		put(t, root, fmt.Sprintf("source-%d\n.txt", i), text)
	}
	result, err := s.Search(context.Background(), "owner", p.ID, q)
	if err != nil || len(result.Files) != 2 {
		t.Fatalf("search: %+v %v", result, err)
	}
	file := result.Files[0]
	preview, err := s.Preview(context.Background(), "owner", p.ID, PreviewInput{p.Version, result.ID, []string{file.Matches[0].ID}, "<$1>"})
	if err != nil {
		t.Fatal(err)
	}
	comparison, err := s.Comparison("owner", p.ID, preview.ID, file.ID)
	if err != nil || comparison.Modified != "\ufeff中文😀<12>\r\na34\ra56\n" {
		t.Fatalf("preview: %+v %v", comparison, err)
	}
}
