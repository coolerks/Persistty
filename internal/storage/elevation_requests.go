package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrRequestConsumed = errors.New("elevation request consumed")
var ErrSessionInvalid = errors.New("session revoked or expired")

type ElevationRecord struct {
	ID, SessionHash, PayloadJSON, State string
	Code, VersionJSON                   *string
	ExpiresAt                           time.Time
}

func (s *Store) CreateElevation(ctx context.Context, r ElevationRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(storedTimeFormat)
	// Retain consumed requests for reconciliation; no content or credentials.
	if _, err = tx.ExecContext(ctx, "DELETE FROM elevation_requests WHERE created_at < ? AND state NOT IN ('executing','indeterminate')", time.Now().UTC().Add(-24*time.Hour).Format(storedTimeFormat)); err != nil {
		return err
	}
	var total, perSession int
	if err = tx.QueryRowContext(ctx, "SELECT count(*),coalesce(sum(CASE WHEN session_hash=? THEN 1 ELSE 0 END),0) FROM elevation_requests", r.SessionHash).Scan(&total, &perSession); err != nil {
		return err
	}
	if total >= 512 || perSession >= 64 {
		return ErrQuota
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO elevation_requests(id,session_hash,payload_json,expires_at,state,created_at) VALUES(?,?,?,?, 'prepared',?)", r.ID, r.SessionHash, r.PayloadJSON, r.ExpiresAt.UTC().Format(storedTimeFormat), now)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) Elevation(ctx context.Context, id, sessionHash string) (ElevationRecord, error) {
	var r ElevationRecord
	var expires string
	err := s.db.QueryRowContext(ctx, "SELECT id,session_hash,payload_json,expires_at,state,code,version_json FROM elevation_requests WHERE id=? AND session_hash=?", id, sessionHash).Scan(&r.ID, &r.SessionHash, &r.PayloadJSON, &expires, &r.State, &r.Code, &r.VersionJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	r.ExpiresAt, err = time.Parse(storedTimeFormat, expires)
	return r, err
}
func (s *Store) TransitionElevation(ctx context.Context, id, sessionHash, from, to string, code, versionJSON *string) error {
	result, err := s.db.ExecContext(ctx, "UPDATE elevation_requests SET state=?,code=?,version_json=? WHERE id=? AND session_hash=? AND state=?", to, code, versionJSON, id, sessionHash, from)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n != 1 {
		return ErrRequestConsumed
	}
	return err
}
func (s *Store) RecoverElevation(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "UPDATE elevation_requests SET state='indeterminate',code='outcome_unknown' WHERE state='executing'")
	return err
}

// WithElevationPublication shares the existing configuration publication lock.
// Authentication must finish before entering; only the bounded commit runs here.
func (s *Store) WithElevationPublication(ctx context.Context, token string, root RegisteredFolder, run func() error) error {
	s.projectMu.Lock()
	defer s.projectMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	expires, err := s.Session(ctx, token)
	if errors.Is(err, ErrNotFound) || err == nil && !expires.After(time.Now()) {
		return ErrSessionInvalid
	}
	if err != nil {
		return err
	}
	current, err := s.RegisteredFolder(ctx, root.ProjectID, root.FolderID, root.ProjectVersion)
	if err != nil {
		return err
	}
	if current != root {
		return ErrConflict
	}
	return run()
}
