package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"persistty/internal/workspace"
)

var ErrConflict = errors.New("project version conflict")
var ErrInvalidProject = errors.New("invalid project")

type ProjectChange struct {
	ExpectedVersion int64    `json:"expected_version"`
	Name            string   `json:"name"`
	AddPaths        []string `json:"add_paths"`
	RemoveFolderIDs []string `json:"remove_folder_ids"`
	MainFolderID    string   `json:"main_folder_id"`
	MainAddedIndex  *int     `json:"main_added_index"`
}

func resourceID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func validProjectName(name string) bool {
	return len(name) > 0 && len(name) <= 200 && utf8.ValidString(name) && strings.TrimSpace(name) == name && !strings.ContainsRune(name, 0)
}

func resolveRoots(paths []string) ([]workspace.Identity, error) {
	if len(paths) == 0 || len(paths) > 200 {
		return nil, ErrInvalidProject
	}
	roots := make([]workspace.Identity, 0, len(paths))
	seen := map[string]bool{}
	for _, path := range paths {
		root, err := workspace.Resolve(path)
		if err != nil {
			return nil, err
		}
		if seen[root.Path] {
			return nil, ErrInvalidProject
		}
		seen[root.Path] = true
		roots = append(roots, root)
	}
	return roots, nil
}

func (s *Store) CreateProject(ctx context.Context, name string, paths []string, mainIndex int) (Project, error) {
	if !validProjectName(name) || mainIndex < 0 || mainIndex >= len(paths) {
		return Project{}, ErrInvalidProject
	}
	roots, err := resolveRoots(paths)
	if err != nil {
		return Project{}, err
	}
	id, err := resourceID()
	if err != nil {
		return Project{}, err
	}
	p := Project{ID: id, Name: name, Version: 1, Folders: make([]Folder, 0, len(roots))}
	for _, root := range roots {
		folderID, e := resourceID()
		if e != nil {
			return Project{}, e
		}
		p.Folders = append(p.Folders, Folder{ID: folderID, Path: root.Path})
	}
	p.MainFolderID = p.Folders[mainIndex].ID
	s.projectMu.Lock()
	defer s.projectMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Project{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "INSERT INTO projects(id,name,version,main_folder_id) VALUES(?,?,?,?)", p.ID, p.Name, p.Version, p.MainFolderID); err != nil {
		return Project{}, err
	}
	for i, folder := range p.Folders {
		if _, err = tx.ExecContext(ctx, "INSERT INTO folders(id,project_id,path) VALUES(?,?,?)", folder.ID, p.ID, folder.Path); err != nil {
			return Project{}, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO folder_roots(folder_id,device,inode) VALUES(?,?,?)", folder.ID, uintString(roots[i].Dev), uintString(roots[i].Ino)); err != nil {
			return Project{}, err
		}
	}
	return p, tx.Commit()
}

func uintString(value uint64) string { return strconv.FormatUint(value, 10) }

func (s *Store) UpdateProject(ctx context.Context, id string, change ProjectChange) (Project, error) {
	if !validProjectName(change.Name) || change.ExpectedVersion < 1 || len(change.AddPaths) > 200 || len(change.RemoveFolderIDs) > 200 {
		return Project{}, ErrInvalidProject
	}
	roots := []workspace.Identity{}
	if len(change.AddPaths) > 0 {
		var err error
		roots, err = resolveRoots(change.AddPaths)
		if err != nil {
			return Project{}, err
		}
	}
	s.projectMu.Lock()
	defer s.projectMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Project{}, err
	}
	defer tx.Rollback()
	var current Project
	err = tx.QueryRowContext(ctx, "SELECT id,name,version,main_folder_id FROM projects WHERE id=?", id).Scan(&current.ID, &current.Name, &current.Version, &current.MainFolderID)
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	if err != nil {
		return Project{}, err
	}
	if current.Version != change.ExpectedVersion {
		return Project{}, ErrConflict
	}
	current.Folders, err = projectFolders(ctx, tx, id)
	if err != nil {
		return Project{}, err
	}
	remove := map[string]bool{}
	for _, folderID := range change.RemoveFolderIDs {
		if remove[folderID] {
			return Project{}, ErrInvalidProject
		}
		remove[folderID] = true
	}
	paths := map[string]bool{}
	next := make([]Folder, 0, len(current.Folders)+len(roots))
	for _, folder := range current.Folders {
		if remove[folder.ID] {
			delete(remove, folder.ID)
			continue
		}
		paths[folder.Path] = true
		next = append(next, folder)
	}
	if len(remove) != 0 {
		return Project{}, ErrInvalidProject
	}
	for _, root := range roots {
		if paths[root.Path] {
			return Project{}, ErrInvalidProject
		}
		paths[root.Path] = true
		folderID, e := resourceID()
		if e != nil {
			return Project{}, e
		}
		next = append(next, Folder{ID: folderID, Path: root.Path})
	}
	if len(next) == 0 || len(next) > 200 {
		return Project{}, ErrInvalidProject
	}
	main := change.MainFolderID
	if change.MainAddedIndex != nil {
		if main != "" || *change.MainAddedIndex < 0 || *change.MainAddedIndex >= len(roots) {
			return Project{}, ErrInvalidProject
		}
		main = next[len(next)-len(roots)+*change.MainAddedIndex].ID
	}
	if main == "" {
		main = current.MainFolderID
	}
	mainExists := false
	for _, folder := range next {
		mainExists = mainExists || folder.ID == main
	}
	if !mainExists {
		return Project{}, ErrInvalidProject
	}
	if _, err = tx.ExecContext(ctx, "UPDATE projects SET name=?,version=version+1,main_folder_id=? WHERE id=?", change.Name, main, id); err != nil {
		return Project{}, err
	}
	for _, folder := range current.Folders {
		keep := false
		for _, candidate := range next {
			keep = keep || candidate.ID == folder.ID
		}
		if !keep {
			if _, err = tx.ExecContext(ctx, "DELETE FROM folders WHERE id=? AND project_id=?", folder.ID, id); err != nil {
				return Project{}, err
			}
		}
	}
	for i, root := range roots {
		folder := next[len(next)-len(roots)+i]
		if _, err = tx.ExecContext(ctx, "INSERT INTO folders(id,project_id,path) VALUES(?,?,?)", folder.ID, id, folder.Path); err != nil {
			return Project{}, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO folder_roots(folder_id,device,inode) VALUES(?,?,?)", folder.ID, uintString(root.Dev), uintString(root.Ino)); err != nil {
			return Project{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return Project{}, err
	}
	return s.Project(ctx, id)
}

func (s *Store) DeleteProject(ctx context.Context, id string, expectedVersion int64) error {
	if expectedVersion < 1 {
		return ErrInvalidProject
	}
	s.projectMu.Lock()
	defer s.projectMu.Unlock()
	result, err := s.db.ExecContext(ctx, "DELETE FROM projects WHERE id=? AND version=?", id, expectedVersion)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 0 {
		return nil
	}
	var version int64
	err = s.db.QueryRowContext(ctx, "SELECT version FROM projects WHERE id=?", id).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return ErrConflict
}
