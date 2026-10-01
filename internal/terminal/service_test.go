package terminal

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"persistty/internal/storage"
)

func TestCreateAndProjectRemovalPreservesTerminal(t *testing.T) {
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
	var session string
	var calls [][]string
	tmux := &Tmux{Shell: "/bin/sh", HistoryLines: 5000, RestoreLines: 5000, HistoryBytes: 8 << 20}
	tmux.run = func(_ context.Context, _ int64, args ...string) ([]byte, error) {
		calls = append(calls, append([]string(nil), args...))
		switch args[0] {
		case "new-session":
			session = args[3]
		case "list-sessions":
			return []byte(session + "\n"), nil
		}
		return nil, nil
	}
	service := &Service{Store: store, Tmux: tmux}
	request := CreateRequest{ProjectID: project.ID, ProjectVersion: project.Version, Cols: 80, Rows: 24}
	created, err := service.Create(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if created.State != "running" || created.WorkingDirectory != project.Folders[0].Path || session != created.TmuxSessionName {
		t.Fatalf("created = %+v, session = %q", created, session)
	}
	var createCall []string
	for _, call := range calls {
		if call[0] == "new-session" {
			createCall = call
		}
	}
	if !reflect.DeepEqual(createCall, []string{"new-session", "-d", "-s", session,
		"-c", created.WorkingDirectory, "-x", "80", "-y", "24",
		"-e", "PERSISTTY_INITIAL_CWD=" + created.WorkingDirectory,
		"-e", "PERSISTTY_DISPLAY_NAME=终端", "-e", "PERSISTTY_PROJECT_ID=" + project.ID,
		"-e", "SHELL=/bin/sh", "/bin/sh", "-i"}) {
		t.Fatalf("tmux new-session args = %v", createCall)
	}
	if _, err := service.Create(ctx, CreateRequest{ProjectID: project.ID, ProjectVersion: project.Version - 1, Cols: 80, Rows: 24}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("invalid version: %v", err)
	}
	if err := store.DeleteProject(ctx, project.ID, project.Version); err != nil {
		t.Fatal(err)
	}
	items, err := service.List(ctx)
	if err != nil || len(items) != 1 || items[0].ProjectID != nil || items[0].State != "running" {
		t.Fatalf("terminals after project removal = %+v, %v", items, err)
	}
	for _, call := range calls {
		if call[0] == "kill-session" {
			t.Fatal("project removal killed terminal")
		}
	}
}

func TestTmuxHistoryIsBoundedAndSeparate(t *testing.T) {
	ctx := context.Background()
	tmux := &Tmux{HistoryLines: 5000, RestoreLines: 120, HistoryBytes: 1024}
	var capture []string
	var display []string
	tmux.run = func(_ context.Context, limit int64, args ...string) ([]byte, error) {
		switch args[0] {
		case "display-message":
			display = append([]string(nil), args...)
			return []byte("300:1:80:24\n"), nil
		case "capture-pane":
			capture = append([]string(nil), args...)
			if limit != 1024 {
				t.Fatalf("history limit = %d", limit)
			}
			return []byte("one\ntwo\n"), nil
		}
		return nil, nil
	}
	history, err := tmux.CaptureHistory(ctx, "persistty_abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if history.HistorySize != 300 || history.ReturnedLines != 2 || !history.AlternateOn || !history.Truncated || history.Cols != 80 || history.Rows != 24 {
		t.Fatalf("history = %+v", history)
	}
	if !reflect.DeepEqual(display, []string{"display-message", "-p", "-t", "=persistty_abcdef:0.0", "#{history_size}:#{alternate_on}:#{window_width}:#{window_height}"}) {
		t.Fatalf("display args = %v", display)
	}
	if !reflect.DeepEqual(capture, []string{"capture-pane", "-p", "-e", "-t", "=persistty_abcdef:0.0", "-S", "-120", "-E", "-1"}) {
		t.Fatalf("capture args = %v", capture)
	}
	var output limitedBuffer
	output.max = 3
	if _, err := output.Write([]byte("four")); !errors.Is(err, ErrHistoryTooLarge) || !output.exceeded {
		t.Fatalf("unbounded history write: %v", err)
	}
	if _, err := tmux.CaptureHistory(ctx, "other-session"); err == nil || !strings.Contains(err.Error(), "invalid terminal name") {
		t.Fatalf("invalid target: %v", err)
	}
}

func TestSelectedFolderAndChangedMainDoNotRebindOldTerminal(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, filepath.Join(t.TempDir(), "private", "db.sqlite"), "testhash")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	project, err := store.CreateProject(ctx, "项目", []string{t.TempDir(), t.TempDir()}, 0)
	if err != nil {
		t.Fatal(err)
	}
	creates := 0
	tmux := &Tmux{Shell: "/bin/sh", HistoryLines: 120, run: func(_ context.Context, _ int64, args ...string) ([]byte, error) {
		if args[0] == "new-session" {
			creates++
		}
		return nil, nil
	}}
	service := &Service{Store: store, Tmux: tmux}
	request := CreateRequest{ProjectID: project.ID, ProjectVersion: project.Version, Cols: 80, Rows: 24}
	old, err := service.Create(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	request.FolderID = project.Folders[1].ID
	selected, err := service.Create(ctx, request)
	if err != nil || selected.WorkingDirectory != project.Folders[1].Path {
		t.Fatalf("selected folder: %+v %v", selected, err)
	}
	updated, err := store.UpdateProject(ctx, project.ID, storage.ProjectChange{ExpectedVersion: project.Version, Name: "改名", MainFolderID: project.Folders[1].ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(ctx, request); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("stale project created a terminal: %v", err)
	}
	request.ProjectVersion, request.FolderID = updated.Version, ""
	current, err := service.Create(ctx, request)
	if err != nil || current.WorkingDirectory != selected.WorkingDirectory {
		t.Fatalf("new main folder: %+v %v", current, err)
	}
	preserved, err := store.Terminal(ctx, old.ID)
	if err != nil || preserved.WorkingDirectory != project.Folders[0].Path || creates != 3 {
		t.Fatalf("old terminal rebound: %+v %v creates=%d", preserved, err, creates)
	}
	request.FolderID = "not-associated"
	if _, err := service.Create(ctx, request); err == nil || creates != 3 {
		t.Fatalf("unregistered folder accepted: %v", err)
	}
}

func TestReconcileSessionAfterMetadataWriteFailure(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "private", "db.sqlite")
	store, err := storage.Open(ctx, path, "testhash")
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(ctx, "项目", []string{t.TempDir()}, 0)
	if err != nil {
		t.Fatal(err)
	}
	var name string
	values := map[string]string{}
	closed := false
	tmux := &Tmux{Shell: "/bin/sh", HistoryLines: 5000}
	tmux.run = func(_ context.Context, _ int64, args ...string) ([]byte, error) {
		switch args[0] {
		case "new-session":
			name = args[3]
			for i := 0; i < len(args)-1; i++ {
				if args[i] == "-e" {
					parts := strings.SplitN(args[i+1], "=", 2)
					values[parts[0]] = parts[1]
				}
			}
		case "set-option":
			if !closed {
				closed = true
				_ = store.Close()
			}
		case "list-sessions":
			return []byte(name + "\n"), nil
		case "show-environment":
			key := args[len(args)-1]
			return []byte(key + "=" + values[key] + "\n"), nil
		}
		return nil, nil
	}
	service := &Service{Store: store, Tmux: tmux}
	_, err = service.Create(ctx, CreateRequest{
		ProjectID: project.ID, ProjectVersion: project.Version, Cols: 80, Rows: 24,
	})
	if err == nil || name == "" {
		t.Fatalf("expected metadata failure after live tmux creation: %v %q", err, name)
	}
	reopened, err := storage.Open(ctx, path, "testhash")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { reopened.Close() })
	service.Store = reopened
	if err := service.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	items, err := service.List(ctx)
	if err != nil || len(items) != 1 || items[0].TmuxSessionName != name || items[0].State != "running" ||
		items[0].ProjectID == nil || *items[0].ProjectID != project.ID ||
		items[0].WorkingDirectory != project.Folders[0].Path {
		t.Fatalf("recovered terminal = %+v, %v", items, err)
	}
	if err := service.Reconcile(ctx); err != nil {
		t.Fatalf("reconcile must be idempotent: %v", err)
	}
}
