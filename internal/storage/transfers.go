package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrQuota = errors.New("transfer quota exceeded")

type UploadRecord struct {
	ID                  string  `json:"id"`
	ProjectID           string  `json:"project_id"`
	FolderID            string  `json:"folder_id"`
	ProjectVersion      int64   `json:"project_version"`
	RelativePath        string  `json:"relative_path"`
	BatchID             string  `json:"batch_id"`
	Size                int64   `json:"size"`
	SHA256              string  `json:"sha256"`
	ChunkBytes          int64   `json:"chunk_bytes"`
	ExpectedVersionJSON *string `json:"-"`
	CreatedAt           string  `json:"created_at"`
	ExpiresAt           string  `json:"expires_at"`
	Status              string  `json:"status"`
	ResultJSON          *string `json:"-"`
}

type UploadChunk struct {
	Index  int64  `json:"index"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

func (s *Store) ReserveUpload(ctx context.Context, item UploadRecord, maxBatch, maxStaging int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(storedTimeFormat)
	var batch, staging int64
	if err = tx.QueryRowContext(ctx, "SELECT COALESCE(SUM(size),0) FROM uploads WHERE batch_id=? AND expires_at>?", item.BatchID, now).Scan(&batch); err != nil {
		return err
	}
	if err = tx.QueryRowContext(ctx, "SELECT COALESCE(SUM(size),0) FROM uploads WHERE status='pending' AND expires_at>?", now).Scan(&staging); err != nil {
		return err
	}
	if item.Size > maxBatch-batch || item.Size > maxStaging-staging {
		return ErrQuota
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO uploads(id,project_id,folder_id,project_version,relative_path,batch_id,size,sha256,chunk_bytes,expected_version_json,created_at,expires_at,status,result_json)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,'pending',NULL)`, item.ID, item.ProjectID, item.FolderID, item.ProjectVersion, item.RelativePath, item.BatchID, item.Size, item.SHA256, item.ChunkBytes, item.ExpectedVersionJSON, item.CreatedAt, item.ExpiresAt)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Upload(ctx context.Context, id string) (UploadRecord, error) {
	var item UploadRecord
	err := s.db.QueryRowContext(ctx, `SELECT id,project_id,folder_id,project_version,relative_path,batch_id,size,sha256,chunk_bytes,expected_version_json,created_at,expires_at,status,result_json FROM uploads WHERE id=?`, id).
		Scan(&item.ID, &item.ProjectID, &item.FolderID, &item.ProjectVersion, &item.RelativePath, &item.BatchID, &item.Size, &item.SHA256, &item.ChunkBytes, &item.ExpectedVersionJSON, &item.CreatedAt, &item.ExpiresAt, &item.Status, &item.ResultJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	return item, err
}

func (s *Store) UploadChunks(ctx context.Context, id string) ([]UploadChunk, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT chunk_index,sha256,size FROM upload_chunks WHERE upload_id=? ORDER BY chunk_index LIMIT 8193", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]UploadChunk, 0)
	for rows.Next() {
		var item UploadChunk
		if err = rows.Scan(&item.Index, &item.SHA256, &item.Size); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(items) > 8192 {
		return nil, ErrListLimit
	}
	return items, nil
}

func (s *Store) RecordUploadChunk(ctx context.Context, id string, chunk UploadChunk) error {
	var status string
	err := s.db.QueryRowContext(ctx, "SELECT status FROM uploads WHERE id=?", id).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != "pending" {
		return ErrConflict
	}
	var hash string
	var size int64
	err = s.db.QueryRowContext(ctx, "SELECT sha256,size FROM upload_chunks WHERE upload_id=? AND chunk_index=?", id, chunk.Index).Scan(&hash, &size)
	if err == nil {
		if hash == chunk.SHA256 && size == chunk.Size {
			return nil
		}
		return ErrConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO upload_chunks(upload_id,chunk_index,sha256,size) VALUES(?,?,?,?)", id, chunk.Index, chunk.SHA256, chunk.Size)
	return err
}

func (s *Store) FinishUpload(ctx context.Context, id, status, resultJSON string) error {
	if status != "completed" && status != "skipped" && status != "failed" {
		return ErrInvalidProject
	}
	result, err := s.db.ExecContext(ctx, "UPDATE uploads SET status=?,result_json=? WHERE id=? AND status='pending'", status, resultJSON, id)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return ErrConflict
	}
	return nil
}

func (s *Store) ExpiredUploads(ctx context.Context) ([]string, error) {
	now := time.Now().UTC().Format(storedTimeFormat)
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM uploads WHERE expires_at<=? LIMIT 1000", now)
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
func (s *Store) DeleteUpload(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM uploads WHERE id=?", id)
	return err
}
