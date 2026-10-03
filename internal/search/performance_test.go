package search

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"persistty/internal/config"
	"persistty/internal/storage"
	"persistty/internal/toolrunner"
	"testing"
	"time"
)

// Opt in to read-only local roots. Only the temporary database and private
// snapshots are written; commands never receive source roots or file bodies.
func TestLocalSearchPerformance(t *testing.T) {
	input := os.Getenv("PERSISTTY_SEARCH_PERF_ROOTS")
	if input == "" {
		t.Skip("set PERSISTTY_SEARCH_PERF_ROOTS to a JSON array for a read-only performance check")
	}
	var roots []string
	if err := json.Unmarshal([]byte(input), &roots); err != nil || len(roots) == 0 {
		t.Fatal("invalid performance roots")
	}
	cfg := config.Config{}
	cfg.Storage.Path = filepath.Join(t.TempDir(), "data/db.sqlite")
	store, err := storage.Open(context.Background(), cfg.Storage.Path, "performance-only")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	p, err := store.CreateProject(context.Background(), "只读搜索性能检查", roots, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, binary := range []string{"rg", "/nonexistent/rg"} {
		t.Run(filepath.Base(binary), func(t *testing.T) {
			svc := New(store, toolrunner.New("git", binary, cfg.ToolTimeout()), cfg)
			start := time.Now()
			names, err := svc.Names(context.Background(), p.ID, p.Version, "project")
			if err != nil {
				t.Fatalf("names elapsed=%s error=%v", time.Since(start), err)
			}
			t.Logf("names elapsed=%s items=%d truncated=%v", time.Since(start), len(names.Items), names.Truncated)
			for _, q := range []Query{{ProjectVersion: p.Version, Pattern: "Backend"}, {ProjectVersion: p.Version, Pattern: `Back[a-z]+`, Regex: true}} {
				start := time.Now()
				result, err := svc.Search(context.Background(), "performance", p.ID, q)
				if err != nil {
					t.Fatalf("regex=%v elapsed=%s error=%v", q.Regex, time.Since(start), err)
				}
				matches := 0
				for _, f := range result.Files {
					matches += len(f.Matches)
				}
				t.Logf("regex=%v elapsed=%s files=%d matches=%d truncated=%v skipped=%d", q.Regex, time.Since(start), len(result.Files), matches, result.Truncated, len(result.Skipped))
				if err := svc.CancelSearch("performance", p.ID, result.ID); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
