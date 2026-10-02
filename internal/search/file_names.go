package search

import (
	"context"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"persistty/internal/storage"
)

type FileName struct {
	FolderID string `json:"folder_id"`
	Path     string `json:"path"`
}

type FileNames struct {
	ProjectVersion int64      `json:"project_version"`
	Items          []FileName `json:"items"`
	Truncated      bool       `json:"truncated"`
}

// Names has no content snapshot or cross-request cache. Each request revalidates its roots.
func (s *Service) Names(ctx context.Context, project string, version int64, query string) (FileNames, error) {
	if version < 1 || strings.TrimSpace(query) == "" || len(query) > 256 || !utf8.ValidString(query) || strings.ContainsAny(query, "\x00\r\n") {
		return FileNames{}, ErrInvalid
	}
	query = strings.TrimSpace(query)
	release, err := s.Runner.Acquire(ctx)
	if err != nil {
		return FileNames{}, err
	}
	defer release()
	ctx, cancel := context.WithTimeout(ctx, s.Config.ToolTimeout())
	defer cancel()
	p, err := s.Store.Project(ctx, project)
	if err != nil {
		return FileNames{}, err
	}
	if p.Version != version {
		return FileNames{}, storage.ErrConflict
	}
	dir, clean, err := s.Runner.Stage(s.Config.ToolStagingPath())
	if err != nil {
		return FileNames{}, err
	}
	defer clean()
	result := FileNames{ProjectVersion: version, Items: []FileName{}}
	budget := discoveryBudget{}
	seen := map[string]bool{}
	options := s.Config.SearchOptions()
	query = strings.ToLower(query)
	for index, folder := range p.Folders {
		if budget.entries >= options.MaxEntries {
			result.Truncated = true
			break
		}
		root, err := s.Store.RegisteredFolder(ctx, project, folder.ID, version)
		if err != nil {
			return FileNames{}, err
		}
		allowed, err := s.discoverTree(ctx, root, filepath.Join(dir, "names-"+strconv.Itoa(index)), &budget, func(string, string) {})
		if err != nil {
			return FileNames{}, err
		}
		for _, relative := range sortedKeys(allowed) {
			if !strings.Contains(strings.ToLower(path.Base(relative)), query) {
				continue
			}
			absolute := filepath.Join(root.Path, filepath.FromSlash(relative))
			if seen[absolute] {
				continue
			}
			seen[absolute] = true
			if len(result.Items) == 100 {
				result.Truncated = true
				break
			}
			result.Items = append(result.Items, FileName{FolderID: folder.ID, Path: relative})
		}
		if result.Truncated || budget.truncated {
			result.Truncated = true
			break
		}
	}
	current, err := s.Store.Project(ctx, project)
	if err != nil {
		return FileNames{}, err
	}
	if current.Version != version {
		return FileNames{}, storage.ErrConflict
	}
	return result, nil
}
