package terminal

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"persistty/internal/storage"
)

type RenameRequest struct {
	ExpectedDisplayName string `json:"expected_display_name"`
	DisplayName         string `json:"display_name"`
}

func validDisplayName(name string) bool {
	return name != "" && utf8.ValidString(name) && len(name) <= 200 &&
		strings.TrimSpace(name) == name && !strings.ContainsAny(name, "\x00\r\n")
}

// Called under metadataMu so concurrent creates and renames share one allocation boundary.
func (s *Service) availableName(ctx context.Context, projectID string) (string, error) {
	names, err := s.Tmux.SessionNames(ctx)
	if err != nil {
		return "", err
	}
	if err = s.reconcileNames(ctx, names); err != nil {
		return "", err
	}
	items, err := s.Store.Terminals(ctx)
	if err != nil {
		return "", err
	}
	occupied := make(map[string]bool)
	for _, item := range items {
		if item.ProjectID != nil && *item.ProjectID == projectID && names[item.TmuxSessionName] {
			occupied[item.DisplayName] = true
		}
	}
	for i := 0; ; i++ {
		name := "终端"
		if i > 0 {
			name += strconv.Itoa(i)
		}
		if !occupied[name] {
			return name, nil
		}
	}
}

func (s *Service) Rename(ctx context.Context, id string, request RenameRequest) (storage.Terminal, error) {
	if !validDisplayName(request.DisplayName) || !validDisplayName(request.ExpectedDisplayName) {
		return storage.Terminal{}, ErrInvalidRequest
	}
	s.metadataMu.Lock()
	defer s.metadataMu.Unlock()
	item, err := s.Store.RenameTerminal(ctx, id, request.ExpectedDisplayName, request.DisplayName)
	if err != nil {
		return storage.Terminal{}, err
	}
	exists, observeErr := s.Tmux.SessionExists(ctx, item.TmuxSessionName)
	if observeErr == nil {
		item.State = "terminated"
		if exists {
			item.State = "running"
			_ = s.Tmux.SetDisplayName(ctx, item.TmuxSessionName, item.DisplayName)
		}
	}
	return item, nil
}
