package terminal

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"persistty/internal/storage"
)

func TestDefaultNamesConcurrentReuseAndRename(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, filepath.Join(t.TempDir(), "private", "db.sqlite"), "testhash")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	project, err := store.CreateProject(ctx, "项目", []string{t.TempDir()}, 0)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	sessions := make(map[string]map[string]string)
	failMirror := false
	kills := 0
	tmux := &Tmux{Shell: "/bin/sh", run: func(_ context.Context, _ int64, args ...string) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		switch args[0] {
		case "list-sessions":
			var names []string
			for name := range sessions {
				names = append(names, name)
			}
			return []byte(strings.Join(names, "\n")), nil
		case "new-session":
			values := make(map[string]string)
			for i := 0; i < len(args)-1; i++ {
				if args[i] == "-e" {
					pair := strings.SplitN(args[i+1], "=", 2)
					values[pair[0]] = pair[1]
				}
			}
			sessions[args[3]] = values
		case "show-environment":
			return []byte(args[3] + "=" + sessions[strings.TrimPrefix(args[2], "=")][args[3]] + "\n"), nil
		case "set-environment":
			if failMirror {
				return nil, ErrUnavailable
			}
			sessions[strings.TrimPrefix(args[2], "=")][args[3]] = args[4]
		case "kill-session":
			kills++
		}
		return nil, nil
	}}
	service := &Service{Store: store, Tmux: tmux}
	request := CreateRequest{ProjectID: project.ID, ProjectVersion: project.Version, Cols: 80, Rows: 24}
	results := make(chan storage.Terminal, 8)
	errorsCh := make(chan error, 8)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			item, err := service.Create(ctx, request)
			if err != nil {
				errorsCh <- err
			} else {
				results <- item
			}
		})
	}
	wg.Wait()
	close(results)
	close(errorsCh)
	for err := range errorsCh {
		t.Fatal(err)
	}
	var names []string
	var first storage.Terminal
	for item := range results {
		names = append(names, item.DisplayName)
		if item.DisplayName == "终端1" {
			first = item
		}
	}
	sort.Strings(names)
	if strings.Join(names, ",") != "终端,终端1,终端2,终端3,终端4,终端5,终端6,终端7" {
		t.Fatalf("concurrent names: %v", names)
	}
	mu.Lock()
	failMirror = true
	mu.Unlock()
	renamed, err := service.Rename(ctx, first.ID, RenameRequest{"终端1", "构建"})
	if err != nil || renamed.ID != first.ID || renamed.TmuxSessionName != first.TmuxSessionName || renamed.DisplayName != "构建" {
		t.Fatalf("rename failed: %+v %v", renamed, err)
	}
	if _, err := service.Rename(ctx, first.ID, RenameRequest{"终端1", "其他"}); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("stale rename: %v", err)
	}
	if _, err := service.Rename(ctx, first.ID, RenameRequest{"终端1", "构建"}); err != nil {
		t.Fatalf("idempotent rename: %v", err)
	}
	mu.Lock()
	failMirror = false
	mu.Unlock()
	if err := service.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	mirror := sessions[first.TmuxSessionName]["PERSISTTY_DISPLAY_NAME"]
	mu.Unlock()
	if mirror != "构建" {
		t.Fatal("metadata mirror was not repaired")
	}
	next, err := service.Create(ctx, request)
	if err != nil || next.DisplayName != "终端1" || next.ID == first.ID {
		t.Fatalf("rename did not free smallest name: %+v %v", next, err)
	}
	mu.Lock()
	delete(sessions, next.TmuxSessionName)
	mu.Unlock()
	reused, err := service.Create(ctx, request)
	if err != nil || reused.DisplayName != "终端1" || reused.ID == next.ID {
		t.Fatalf("ended name not reused: %+v %v", reused, err)
	}
	other, err := store.CreateProject(ctx, "另一个项目", []string{t.TempDir()}, 0)
	if err != nil {
		t.Fatal(err)
	}
	request.ProjectID, request.ProjectVersion = other.ID, other.Version
	independent, err := service.Create(ctx, request)
	if err != nil || independent.DisplayName != "终端" {
		t.Fatalf("project namespace not independent: %+v %v", independent, err)
	}
	if kills != 0 {
		t.Fatal("metadata operation killed a session")
	}
}

func TestDisplayNameValidationAndUnavailableAllocation(t *testing.T) {
	for _, name := range []string{"", " trailing ", "a\n", "a\r", "a\x00", string([]byte{255}), strings.Repeat("中", 67)} {
		if validDisplayName(name) {
			t.Fatalf("invalid name accepted: %q", name)
		}
	}
	ctx := context.Background()
	store, err := storage.Open(ctx, filepath.Join(t.TempDir(), "private", "db.sqlite"), "testhash")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	project, err := store.CreateProject(ctx, "项目", []string{t.TempDir()}, 0)
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{Store: store, Tmux: &Tmux{run: func(context.Context, int64, ...string) ([]byte, error) {
		return nil, ErrUnavailable
	}}}
	if _, err := service.Create(ctx, CreateRequest{ProjectID: project.ID, ProjectVersion: project.Version, Cols: 80, Rows: 24}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unavailable server allocated a name: %v", err)
	}
	items, err := store.Terminals(ctx)
	if err != nil || len(items) != 0 {
		t.Fatal("failed observation inserted metadata")
	}
}
