package terminal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"persistty/internal/storage"
	"persistty/internal/workspace"
)

var ErrInvalidRequest = errors.New("invalid terminal request")

type CreateRequest struct {
	ProjectID      string `json:"project_id"`
	ProjectVersion int64  `json:"project_version"`
	FolderID       string `json:"folder_id"`
	DisplayName    string `json:"display_name"`
	Cols           int    `json:"cols"`
	Rows           int    `json:"rows"`
}

type Service struct {
	Store       *storage.Store
	Tmux        *Tmux
	reconcileMu sync.Mutex
}

func (s *Service) Create(ctx context.Context, request CreateRequest) (storage.Terminal, error) {
	if request.ProjectID == "" || request.ProjectVersion < 1 ||
		!validSize(request.Cols, request.Rows) {
		return storage.Terminal{}, ErrInvalidRequest
	}
	name := request.DisplayName
	if name == "" {
		name = "终端"
	}
	if !utf8.ValidString(name) || len(name) > 200 || strings.TrimSpace(name) != name || strings.ContainsAny(name, "\x00\r\n") {
		return storage.Terminal{}, ErrInvalidRequest
	}
	project, err := s.Store.Project(ctx, request.ProjectID)
	if err != nil {
		return storage.Terminal{}, err
	}
	if project.Version != request.ProjectVersion {
		return storage.Terminal{}, storage.ErrConflict
	}
	folderID := request.FolderID
	if folderID == "" {
		folderID = project.MainFolderID
	}
	idBytes := make([]byte, 16)
	if _, err = rand.Read(idBytes); err != nil {
		return storage.Terminal{}, err
	}
	id := hex.EncodeToString(idBytes)
	sessionName := "persistty_" + id
	var created storage.Terminal
	err = s.Store.WithRegisteredFolder(ctx, project.ID, folderID, project.Version, func(root storage.RegisteredFolder) error {
		identity, resolveErr := workspace.Resolve(root.Path)
		if resolveErr != nil {
			return resolveErr
		}
		if identity.Path != root.Path || identity.Dev != root.Device || identity.Ino != root.Inode {
			return storage.ErrConflict
		}
		if err := s.Tmux.Create(ctx, sessionName, identity.Path, name, project.ID, request.Cols, request.Rows); err != nil {
			return err
		}
		created = storage.Terminal{
			ID: id, TmuxSessionName: sessionName, DisplayName: name,
			ProjectID: &project.ID, WorkingDirectory: identity.Path, State: "running",
		}
		return s.Store.InsertTerminal(ctx, created)
	})
	if err != nil {
		return storage.Terminal{}, err
	}
	return created, nil
}

func (s *Service) List(ctx context.Context) ([]storage.Terminal, error) {
	names, err := s.Tmux.SessionNames(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.reconcileNames(ctx, names); err != nil {
		return nil, err
	}
	items, err := s.Store.Terminals(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].State = "terminated"
		if names[items[i].TmuxSessionName] {
			items[i].State = "running"
		}
	}
	return items, nil
}

func (s *Service) Get(ctx context.Context, id string) (storage.Terminal, error) {
	item, err := s.Store.Terminal(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		if reconcileErr := s.Reconcile(ctx); reconcileErr != nil {
			return storage.Terminal{}, reconcileErr
		}
		item, err = s.Store.Terminal(ctx, id)
	}
	if err != nil {
		return item, err
	}
	exists, err := s.Tmux.SessionExists(ctx, item.TmuxSessionName)
	if err != nil {
		return storage.Terminal{}, err
	}
	item.State = "terminated"
	if exists {
		item.State = "running"
	}
	return item, nil
}

func (s *Service) History(ctx context.Context, id string) (History, error) {
	item, err := s.Get(ctx, id)
	if err != nil {
		return History{}, err
	}
	if item.State != "running" {
		return History{}, storage.ErrNotFound
	}
	return s.Tmux.CaptureHistory(ctx, item.TmuxSessionName)
}

func (s *Service) Reconcile(ctx context.Context) error {
	names, err := s.Tmux.SessionNames(ctx)
	if err != nil {
		return err
	}
	return s.reconcileNames(ctx, names)
}

func (s *Service) reconcileNames(ctx context.Context, names map[string]bool) error {
	s.reconcileMu.Lock()
	defer s.reconcileMu.Unlock()
	for name := range names {
		id := strings.TrimPrefix(name, "persistty_")
		if len(id) != 32 {
			continue
		}
		if _, err := hex.DecodeString(id); err != nil {
			continue
		}
		if _, err := s.Store.TerminalBySession(ctx, name); err == nil {
			continue
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		cwd, displayName, projectID, err := s.Tmux.Metadata(ctx, name)
		if err != nil {
			return err
		}
		if cwd == "" || !filepath.IsAbs(cwd) || filepath.Clean(cwd) != cwd ||
			displayName == "" || !utf8.ValidString(displayName) || len(displayName) > 200 {
			continue
		}
		var project *string
		if projectID != "" {
			if _, err := s.Store.Project(ctx, projectID); err == nil {
				project = &projectID
			} else if !errors.Is(err, storage.ErrNotFound) {
				return err
			}
		}
		item := storage.Terminal{ID: id, TmuxSessionName: name, DisplayName: displayName,
			ProjectID: project, WorkingDirectory: cwd}
		if err := s.Store.InsertTerminal(ctx, item); err != nil {
			return err
		}
	}
	return nil
}
