package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func (s *Store) Terminal(ctx context.Context, id string) (Terminal, error) {
	return s.terminalBy(ctx, "id", id)
}

func (s *Store) TerminalBySession(ctx context.Context, sessionName string) (Terminal, error) {
	return s.terminalBy(ctx, "tmux_session_name", sessionName)
}

func (s *Store) terminalBy(ctx context.Context, column, value string) (Terminal, error) {
	var terminal Terminal
	terminal.State = "unavailable"
	query := "SELECT id,tmux_session_name,display_name,project_id,working_directory FROM terminals WHERE " + column + "=?"
	err := s.db.QueryRowContext(ctx, query, value).Scan(
		&terminal.ID, &terminal.TmuxSessionName, &terminal.DisplayName,
		&terminal.ProjectID, &terminal.WorkingDirectory,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Terminal{}, ErrNotFound
	}
	return terminal, err
}

func (s *Store) InsertTerminal(ctx context.Context, terminal Terminal) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM terminals").Scan(&count); err != nil {
		return err
	}
	if count >= 200 {
		return ErrListLimit
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO terminals
		(id,tmux_session_name,display_name,project_id,working_directory,created_at)
		VALUES(?,?,?,?,?,?)`, terminal.ID, terminal.TmuxSessionName, terminal.DisplayName,
		terminal.ProjectID, terminal.WorkingDirectory, time.Now().UTC().Format(storedTimeFormat))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) RenameTerminal(ctx context.Context, id, expected, name string) (Terminal, error) {
	result, err := s.db.ExecContext(ctx, "UPDATE terminals SET display_name=? WHERE id=? AND display_name=?", name, id, expected)
	if err != nil {
		return Terminal{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return Terminal{}, err
	}
	item, err := s.Terminal(ctx, id)
	if err != nil {
		return Terminal{}, err
	}
	if count == 0 && item.DisplayName != name {
		return Terminal{}, ErrConflict
	}
	return item, nil
}
