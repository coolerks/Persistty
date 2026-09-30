package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"persistty/internal/auth"
	"persistty/internal/storage"
	"persistty/internal/terminal"
)

var stage = "arguments"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		if err := serveBrowser(); err != nil {
			os.Exit(1)
		}
		return
	}
	if err := run(); err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"failed_stage": stage, "checks": map[string]bool{}})
	}
}

func run() error {
	if len(os.Args) != 4 {
		return errors.New("invalid arguments")
	}
	root, binary, socket := os.Args[1], os.Args[2], os.Args[3]
	if filepath.Dir(root) != "/tmp" || !strings.HasPrefix(filepath.Base(root), "persistty-recovery-") ||
		!filepath.IsAbs(binary) || socket != filepath.Join(root, "tmux.sock") {
		return errors.New("invalid probe paths")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	tmux := &terminal.Tmux{Binary: binary, Socket: socket, Shell: "/bin/sh",
		HistoryLines: 120, RestoreLines: 120, HistoryBytes: 1 << 20}
	checks := map[string]bool{}
	stage = "probe"
	if err := tmux.Probe(ctx); err != nil {
		return err
	}
	checks["private_server_probe"] = true
	stage = "storage"
	hash, err := auth.HashPassword([]byte("isolated-test-password"))
	if err != nil {
		return err
	}
	store, err := storage.Open(ctx, filepath.Join(root, "runtime-data", "db.sqlite"), hash)
	if err != nil {
		return err
	}
	defer store.Close()
	stage = "project"
	project, err := store.CreateProject(ctx, "隔离终端项目", []string{root}, 0)
	if err != nil {
		return err
	}
	stage = "create"
	service := &terminal.Service{Store: store, Tmux: tmux}
	created, err := service.Create(ctx, terminal.CreateRequest{
		ProjectID: project.ID, ProjectVersion: project.Version, Cols: 80, Rows: 24,
	})
	if err != nil {
		return err
	}
	defer tmux.Kill(context.Background(), created.TmuxSessionName)
	checks["created_in_registered_folder"] = created.WorkingDirectory == root && created.State == "running"
	stage = "metadata"
	cwd, title, projectID, err := tmux.Metadata(ctx, created.TmuxSessionName)
	if err != nil {
		return err
	}
	checks["atomic_session_metadata"] = cwd == root && title == "终端" && projectID == project.ID
	limit, err := tmux.Run(ctx, 1024, "display-message", "-p", "-t", "="+created.TmuxSessionName+":0.0", "#{history_limit}")
	if err != nil {
		return err
	}
	checks["effective_pane_history_limit"] = strings.TrimSpace(string(limit)) == "120"
	stage = "attach"
	before, err := tmux.Run(ctx, 1024, "list-panes", "-t", "="+created.TmuxSessionName, "-F", "#{pane_pid}")
	if err != nil {
		return err
	}
	owner, err := tmux.Attach(created.TmuxSessionName, false, 80, 24)
	if err != nil {
		return err
	}
	viewer, err := tmux.Attach(created.TmuxSessionName, true, 40, 10)
	if err != nil {
		owner.Close()
		return err
	}
	viewer.Close()
	owner.Close()
	after, err := tmux.Run(ctx, 1024, "list-panes", "-t", "="+created.TmuxSessionName, "-F", "#{pane_pid}")
	if err != nil {
		return err
	}
	checks["attach_cleanup_preserves_pane"] = len(before) > 0 && string(before) == string(after)
	stage = "history"
	if _, err := tmux.Run(ctx, 1024, "send-keys", "-t", "="+created.TmuxSessionName+":0.0", "seq 1 80", "Enter"); err != nil {
		return err
	}
	var history terminal.History
	for range 30 {
		history, err = service.History(ctx, created.ID)
		if err == nil && history.HistorySize > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	checks["history_snapshot_bounded"] = err == nil && history.HistorySize > 0 && history.HistorySize <= 120 &&
		history.Cols == 80 && history.Rows == 24 && len(history.Content) <= 1<<20
	stage = "list"
	items, err := service.List(ctx)
	if err != nil {
		return err
	}
	checks["real_running_list"] = len(items) == 1 && items[0].State == "running"
	stage = "http_ws"
	webChecks, err := runHTTPWebSocketSmoke(ctx, root, binary, socket, hash, store, created)
	if err != nil {
		return err
	}
	for key, passed := range webChecks {
		checks[key] = passed
	}
	stage = "unlink_project"
	if err := store.DeleteProject(ctx, project.ID, project.Version); err != nil {
		return err
	}
	items, err = service.List(ctx)
	if err != nil {
		return err
	}
	checks["project_removal_preserves_session"] = len(items) == 1 && items[0].ProjectID == nil && items[0].State == "running"
	stage = "kill"
	if err := tmux.Kill(ctx, created.TmuxSessionName); err != nil {
		return err
	}
	exists, err := tmux.SessionExists(ctx, created.TmuxSessionName)
	checks["precise_termination"] = err == nil && !exists
	stage = "missing_server"
	missingSocket := filepath.Join(root, "never-started.sock")
	missing := &terminal.Tmux{Binary: binary, Socket: missingSocket}
	checks["missing_server_not_started"] = errors.Is(missing.Probe(ctx), terminal.ErrUnavailable)
	if _, err := os.Lstat(missingSocket); !errors.Is(err, os.ErrNotExist) {
		checks["missing_server_not_started"] = false
	}
	stage = "complete"
	return json.NewEncoder(os.Stdout).Encode(struct {
		Checks map[string]bool `json:"checks"`
	}{checks})
}
