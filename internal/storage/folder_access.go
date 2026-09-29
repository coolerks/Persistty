package storage

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
)

type RegisteredFolder struct {
	ProjectID      string
	FolderID       string
	Path           string
	Device         uint64
	Inode          uint64
	ProjectVersion int64
}

func (s *Store) RegisteredFolder(ctx context.Context, projectID, folderID string, version int64) (RegisteredFolder, error) {
	var root RegisteredFolder
	var device, inode string
	err := s.db.QueryRowContext(ctx, `SELECT p.id,f.id,f.path,p.version,r.device,r.inode
FROM projects p JOIN folders f ON f.project_id=p.id JOIN folder_roots r ON r.folder_id=f.id
WHERE p.id=? AND f.id=?`, projectID, folderID).Scan(&root.ProjectID, &root.FolderID, &root.Path, &root.ProjectVersion, &device, &inode)
	if errors.Is(err, sql.ErrNoRows) {
		return root, ErrNotFound
	}
	if err != nil {
		return root, err
	}
	if root.ProjectVersion != version {
		return root, ErrConflict
	}
	root.Device, err = strconv.ParseUint(device, 10, 64)
	if err != nil {
		return root, err
	}
	root.Inode, err = strconv.ParseUint(inode, 10, 64)
	return root, err
}

func (s *Store) WithRegisteredFolder(ctx context.Context, projectID, folderID string, version int64, run func(RegisteredFolder) error) error {
	s.projectMu.Lock()
	defer s.projectMu.Unlock()
	root, err := s.RegisteredFolder(ctx, projectID, folderID, version)
	if err != nil {
		return err
	}
	return run(root)
}

func (s *Store) WithRegisteredFolders(ctx context.Context, projectID, sourceFolderID, targetFolderID string, version int64, run func(RegisteredFolder, RegisteredFolder) error) error {
	s.projectMu.Lock()
	defer s.projectMu.Unlock()
	target, err := s.RegisteredFolder(ctx, projectID, targetFolderID, version)
	if err != nil {
		return err
	}
	var source RegisteredFolder
	if sourceFolderID != "" {
		source, err = s.RegisteredFolder(ctx, projectID, sourceFolderID, version)
		if err != nil {
			return err
		}
	}
	return run(source, target)
}
