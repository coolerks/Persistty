package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"persistty/internal/files"
)

type fileEvent struct {
	ProjectID string `json:"project_id"`
	FolderID  string `json:"folder_id"`
	Revision  uint64 `json:"revision"`
	Rescan    bool   `json:"rescan"`
	Mode      string `json:"mode"`
}

func (a *api) events(c *gin.Context) {
	if !a.origin(c) {
		a.fail(c, 403, "forbidden", "请求来源无效。")
		return
	}
	if len(c.Query("project_id")) > 128 || len(c.Query("folder_id")) > 128 || len(c.Query("path")) > 4096 {
		a.fail(c, 400, "invalid_request", "监听目录无效。")
		return
	}
	version, err := strconv.ParseInt(c.Query("project_version"), 10, 64)
	if err != nil || version < 1 {
		a.fail(c, 428, "version_required", "需要项目配置版本。")
		return
	}
	root, err := a.store.RegisteredFolder(c.Request.Context(), c.Query("project_id"), c.Query("folder_id"), version)
	if err != nil {
		a.error(c, err)
		return
	}
	relative := c.Query("path")
	dir, err := files.OpenDirectory(root, relative)
	if err != nil {
		a.error(c, err)
		return
	}
	defer dir.Close()
	watcher, watchError := fsnotify.NewWatcher()
	mode := "polling"
	if watchError == nil {
		defer watcher.Close()
		watchPath := filepath.Join(root.Path, filepath.FromSlash(relative))
		current, statErr := os.Stat(watchPath)
		opened, openedErr := dir.Stat()
		if statErr == nil && openedErr == nil && os.SameFile(current, opened) {
			watchError = watcher.Add(watchPath)
		} else {
			watchError = files.ErrRootChanged
		}
		if watchError == nil {
			mode = "watching"
		}
	}
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{OriginPatterns: []string{a.cfg.Server.PublicOrigin}})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	ctx := conn.CloseRead(c.Request.Context())
	var events <-chan fsnotify.Event
	var watchErrors <-chan error
	if mode == "watching" {
		events = watcher.Events
		watchErrors = watcher.Errors
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	revision := uint64(0)
	send := func() error {
		revision++
		payload, err := json.Marshal(fileEvent{ProjectID: root.ProjectID, FolderID: root.FolderID, Revision: revision, Rescan: true, Mode: mode})
		if err != nil {
			return err
		}
		writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return conn.Write(writeCtx, websocket.MessageText, payload)
	}
	if err := send(); err != nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-events:
			if !ok {
				events = nil
				mode = "polling"
			}
			if err := send(); err != nil {
				return
			}
		case err, ok := <-watchErrors:
			if !ok || errors.Is(err, fsnotify.ErrEventOverflow) || err != nil {
				mode = "polling"
				watchErrors = nil
				events = nil
			}
			if err := send(); err != nil {
				return
			}
		case <-ticker.C:
			if _, err := a.auth.Lookup(ctx, c.MustGet("token").(string)); err != nil {
				return
			}
			if _, err := a.store.RegisteredFolder(ctx, root.ProjectID, root.FolderID, version); err != nil {
				return
			}
			if err := send(); err != nil {
				return
			}
		}
	}
}
