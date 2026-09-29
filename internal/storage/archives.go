package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type ArchiveRecord struct {
	ID             string `json:"id"`
	ProjectID      string `json:"project_id"`
	FolderID       string `json:"folder_id"`
	ProjectVersion int64  `json:"project_version"`
	RelativePath   string `json:"relative_path"`
	CreatedAt      string `json:"created_at"`
	ExpiresAt      string `json:"expires_at"`
	Status         string `json:"status"`
	Size           int64  `json:"size"`
	ErrorCode      string `json:"error_code"`
}

func (s *Store) CreateArchive(ctx context.Context, item ArchiveRecord) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO archives(id,project_id,folder_id,project_version,relative_path,created_at,expires_at,status,size,error_code) VALUES(?,?,?,?,?,?,?,'pending',0,'')`, item.ID, item.ProjectID, item.FolderID, item.ProjectVersion, item.RelativePath, item.CreatedAt, item.ExpiresAt)
	return err
}
func (s *Store) Archive(ctx context.Context, id string) (ArchiveRecord, error) {
	var item ArchiveRecord
	err := s.db.QueryRowContext(ctx, "SELECT id,project_id,folder_id,project_version,relative_path,created_at,expires_at,status,size,error_code FROM archives WHERE id=?", id).
		Scan(&item.ID, &item.ProjectID, &item.FolderID, &item.ProjectVersion, &item.RelativePath, &item.CreatedAt, &item.ExpiresAt, &item.Status, &item.Size, &item.ErrorCode)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	return item, err
}
func (s *Store) FinishArchive(ctx context.Context, id, status string, size int64, errorCode string) error {
	if status != "ready" && status != "failed" {
		return ErrInvalidProject
	}
	result, err := s.db.ExecContext(ctx, "UPDATE archives SET status=?,size=?,error_code=? WHERE id=? AND status='pending'", status, size, errorCode, id)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return ErrConflict
	}
	return nil
}
func (s *Store) CancelArchive(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "UPDATE archives SET status='cancelled' WHERE id=? AND status IN ('pending','ready')", id)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return ErrConflict
	}
	return nil
}
func (s *Store) FailPendingArchives(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "UPDATE archives SET status='failed',error_code='restarted' WHERE status='pending'")
	return err
}
func (s *Store) ExpiredArchives(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM archives WHERE expires_at<=? LIMIT 1000", time.Now().UTC().Format(storedTimeFormat))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}
func (s *Store) DeleteArchive(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM archives WHERE id=?", id)
	return err
}
