package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

var ErrNotFound = errors.New("not found")
var ErrListLimit = errors.New("list limit exceeded")

// Fixed fractional precision keeps UTC expiry strings lexically sortable.
const storedTimeFormat = "2006-01-02T15:04:05.000000000Z"

type Store struct {
	db        *sql.DB
	projectMu sync.Mutex
}
type Folder struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}
type Project struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      int64    `json:"version"`
	MainFolderID string   `json:"main_folder_id"`
	Folders      []Folder `json:"folders"`
}
type Terminal struct {
	ID               string  `json:"id"`
	TmuxSessionName  string  `json:"-"`
	DisplayName      string  `json:"display_name"`
	ProjectID        *string `json:"project_id"`
	WorkingDirectory string  `json:"working_directory"`
	State            string  `json:"state"`
}

func Open(ctx context.Context, path string, passwordHash string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	if err := checkPrivate(dir, true); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err == nil {
		err = f.Close()
	} else if errors.Is(err, os.ErrExist) {
		err = nil
	}
	if err != nil {
		return nil, err
	}
	if err = checkPrivate(path, false); err != nil {
		return nil, err
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, e := os.Lstat(path + suffix); e == nil {
			if e = checkPrivate(path+suffix, false); e != nil {
				return nil, e
			}
		} else if !errors.Is(e, os.ErrNotExist) {
			return nil, e
		}
	}
	u := url.URL{Scheme: "file", Path: path}
	q := u.Query()
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "journal_mode(WAL)")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	s := &Store{db: db}
	if err = s.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		db.Close()
		return nil, err
	}
	defer tx.Rollback()
	fingerprint := HashToken(passwordHash)
	var previous string
	err = tx.QueryRowContext(ctx, "SELECT password_fingerprint FROM auth_config WHERE id=1").Scan(&previous)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.ExecContext(ctx, "INSERT INTO auth_config VALUES(1,?)", fingerprint)
	} else if err == nil && previous != fingerprint {
		if _, err = tx.ExecContext(ctx, "DELETE FROM sessions"); err == nil {
			_, err = tx.ExecContext(ctx, "UPDATE auth_config SET password_fingerprint=? WHERE id=1", fingerprint)
		}
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func checkPrivate(path string, dir bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0077 != 0 || (dir && !info.IsDir()) || (!dir && !info.Mode().IsRegular()) {
		return errors.New("数据库文件和目录必须为服务用户私有普通文件/目录")
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && int(stat.Uid) != os.Geteuid() {
		return errors.New("数据库必须属于服务用户")
	}
	return nil
}
func (s *Store) migrate(ctx context.Context) (resultErr error) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return err
	}
	type migration struct {
		body     []byte
		checksum string
	}
	migrations := make([]migration, 0, len(entries))
	for i, entry := range entries {
		version, e := strconv.Atoi(strings.SplitN(entry.Name(), "_", 2)[0])
		if e != nil || entry.IsDir() || version != i+1 || !strings.HasSuffix(entry.Name(), ".sql") {
			return errors.New("迁移文件必须连续有序")
		}
		body, e := migrationFiles.ReadFile("migrations/" + entry.Name())
		if e != nil {
			return e
		}
		migrations = append(migrations, migration{body, HashToken(string(body))})
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	// 重建循环外键表时暂关自动级联，提交前显式验证全库，始终恢复连接约束。
	if _, err = conn.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
		return err
	}
	defer func() {
		restoreCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, restoreErr := conn.ExecContext(restoreCtx, "PRAGMA foreign_keys=ON")
		if restoreErr == nil {
			var enabled int
			restoreErr = conn.QueryRowContext(restoreCtx, "PRAGMA foreign_keys").Scan(&enabled)
			if restoreErr == nil && enabled != 1 {
				restoreErr = errors.New("数据库外键未恢复")
			}
		}
		if restoreErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("恢复数据库外键约束: %w", restoreErr))
		}
	}()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY, checksum TEXT NOT NULL, applied_at TEXT NOT NULL)"); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, "SELECT version,checksum FROM schema_migrations ORDER BY version")
	if err != nil {
		return err
	}
	applied := 0
	for rows.Next() {
		var version int
		var saved string
		if err = rows.Scan(&version, &saved); err != nil {
			rows.Close()
			return err
		}
		if version != applied+1 || version > len(migrations) || saved != migrations[version-1].checksum {
			rows.Close()
			return errors.New("数据库版本过高或迁移checksum不一致")
		}
		applied = version
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for i := applied; i < len(migrations); i++ {
		if _, err = tx.ExecContext(ctx, string(migrations[i].body)); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations VALUES(?,?,?)", i+1, migrations[i].checksum, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	violations, err := tx.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	invalid := violations.Next()
	err = violations.Err()
	violations.Close()
	if err != nil {
		return err
	}
	if invalid {
		return errors.New("数据库外键约束无效，迁移已回滚")
	}
	return tx.Commit()
}
func HashToken(secret string) string {
	hash := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(hash[:])
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) CreateSession(ctx context.Context, token string, expires time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at<=?", time.Now().UTC().Format(storedTimeFormat)); err != nil {
		return err
	}
	var count int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM sessions").Scan(&count); err != nil {
		return err
	}
	if count >= 200 {
		return ErrListLimit
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO sessions VALUES(?,?)", HashToken(token), expires.UTC().Format(storedTimeFormat)); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) Session(ctx context.Context, token string) (time.Time, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, "SELECT expires_at FROM sessions WHERE token_hash=?", HashToken(token)).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, ErrNotFound
	}
	if err != nil {
		return time.Time{}, err
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid stored session expiry: %w", err)
	}
	return t, nil
}
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash=?", HashToken(token))
	return err
}
func (s *Store) Projects(ctx context.Context) ([]Project, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT id,name,version,main_folder_id FROM projects ORDER BY name,id LIMIT 201")
	if err != nil {
		return nil, err
	}
	items := make([]Project, 0)
	for rows.Next() {
		var p Project
		if err = rows.Scan(&p.ID, &p.Name, &p.Version, &p.MainFolderID); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if len(items) > 200 {
		return nil, ErrListLimit
	}
	for i := range items {
		items[i].Folders, err = projectFolders(ctx, tx, items[i].ID)
		if err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return items, nil
}
func (s *Store) Project(ctx context.Context, id string) (Project, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Project{}, err
	}
	defer tx.Rollback()
	var p Project
	err = tx.QueryRowContext(ctx, "SELECT id,name,version,main_folder_id FROM projects WHERE id=?", id).Scan(&p.ID, &p.Name, &p.Version, &p.MainFolderID)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	p.Folders, err = projectFolders(ctx, tx, id)
	if err != nil {
		return p, err
	}
	return p, tx.Commit()
}
func projectFolders(ctx context.Context, tx *sql.Tx, id string) ([]Folder, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id,path FROM folders WHERE project_id=? ORDER BY id LIMIT 201", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	folders := make([]Folder, 0)
	for rows.Next() {
		var f Folder
		if err = rows.Scan(&f.ID, &f.Path); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	if len(folders) > 200 {
		return nil, ErrListLimit
	}
	return folders, rows.Err()
}
func (s *Store) Terminals(ctx context.Context) ([]Terminal, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id,tmux_session_name,display_name,project_id,working_directory FROM terminals ORDER BY created_at,id LIMIT 201")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Terminal, 0)
	for rows.Next() {
		var t Terminal
		t.State = "unavailable"
		if err = rows.Scan(&t.ID, &t.TmuxSessionName, &t.DisplayName, &t.ProjectID, &t.WorkingDirectory); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	if len(items) > 200 {
		return nil, ErrListLimit
	}
	return items, rows.Err()
}
