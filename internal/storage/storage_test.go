package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func openTest(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "data", "db.sqlite"), "testhash")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func TestMetadataAndIndependentTerminal(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO projects VALUES('p','项目',1,'f'); INSERT INTO folders VALUES('f','p','/tmp/project'); INSERT INTO terminals VALUES('t','tmux-test','终端','p','/tmp/project','2026-09-27T00:00:00Z')"); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	projects, err := s.Projects(ctx)
	if err != nil || len(projects) != 1 || len(projects[0].Folders) != 1 {
		t.Fatalf("projects=%v err=%v", projects, err)
	}
	if _, err = s.Project(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing")
	}
	if _, err = s.db.ExecContext(ctx, "DELETE FROM projects WHERE id='p'"); err != nil {
		t.Fatal(err)
	}
	terminals, err := s.Terminals(ctx)
	if err != nil || len(terminals) != 1 || terminals[0].ProjectID != nil || terminals[0].State != "unavailable" {
		t.Fatalf("terminals=%v err=%v", terminals, err)
	}
	if _, err = s.db.ExecContext(ctx, "INSERT INTO projects VALUES('invalid','bad',1,'missing')"); err == nil {
		t.Fatal("missing main folder allowed")
	}
}

func TestMetadataRejectsNullIDs(t *testing.T) {
	s := openTest(t)
	for _, query := range []string{
		"INSERT INTO projects VALUES(NULL,'项目',1,'missing')",
		"INSERT INTO terminals VALUES(NULL,'null-id-session','终端',NULL,'/tmp','2026-09-27T00:00:00Z')",
	} {
		if _, err := s.db.Exec(query); err == nil {
			t.Errorf("NULL resource identity accepted: %s", query)
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec("INSERT INTO projects VALUES('p','项目',1,'f'); INSERT INTO folders VALUES('f','p','/tmp/project')"); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec("INSERT INTO folders VALUES(NULL,'p','/tmp/other')"); err == nil {
		t.Error("NULL folder identity accepted")
	}
}

func legacyDatabase(t *testing.T, invalid bool) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "data")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "legacy.sqlite")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	body, err := migrationFiles.ReadFile("migrations/0001_metadata.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, checksum TEXT NOT NULL, applied_at TEXT NOT NULL);" + string(body)); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("INSERT INTO schema_migrations VALUES(1,?,?)", HashToken(string(body)), "2026-09-27T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("INSERT INTO auth_config VALUES(1,?)", HashToken("testhash")); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("INSERT INTO sessions VALUES(?,?)", HashToken("legacy-token"), time.Now().Add(time.Hour).UTC().Format(storedTimeFormat)); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("INSERT INTO projects VALUES('p','原项目',7,'f'); INSERT INTO folders VALUES('f','p','/tmp/original'); INSERT INTO terminals VALUES('t','original-session','原终端','p','/tmp/original','2026-09-27T00:00:00Z')"); err != nil {
		t.Fatal(err)
	}
	if invalid {
		if _, err = db.Exec("INSERT INTO projects VALUES(NULL,'异常项目',1,'missing')"); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestUpgradePreservesMetadataAndRejectsInvalidLegacy(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		path := legacyDatabase(t, invalid)
		s, err := Open(context.Background(), path, "testhash")
		if invalid {
			if err == nil {
				s.Close()
				t.Fatal("invalid legacy identities were accepted")
			}
			db, e := sql.Open("sqlite", path)
			if e != nil {
				t.Fatal(e)
			}
			var version, count int
			if e = db.QueryRow("SELECT max(version) FROM schema_migrations").Scan(&version); e != nil || version != 1 {
				t.Fatalf("failed upgrade advanced version: %d %v", version, e)
			}
			if e = db.QueryRow("SELECT count(*) FROM projects WHERE id IS NULL").Scan(&count); e != nil || count != 1 {
				t.Fatal("failed upgrade changed legacy data")
			}
			if e = db.QueryRow("SELECT count(*) FROM terminals WHERE id='t' AND project_id='p'").Scan(&count); e != nil || count != 1 {
				t.Fatal("failed upgrade changed terminal association")
			}
			if e = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name LIKE '%_new'").Scan(&count); e != nil || count != 0 {
				t.Fatal("failed upgrade left partial tables")
			}
			if e = db.QueryRow("SELECT count(*) FROM sessions WHERE token_hash=?", HashToken("legacy-token")).Scan(&count); e != nil || count != 1 {
				t.Fatal("failed upgrade lost authentication session")
			}
			db.Close()
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.Session(context.Background(), "legacy-token"); err != nil {
			t.Fatal("upgrade lost authentication session")
		}
		var enabled int
		if err = s.db.QueryRow("PRAGMA foreign_keys").Scan(&enabled); err != nil || enabled != 1 {
			t.Fatal("successful upgrade did not restore foreign keys")
		}
		project, err := s.Project(context.Background(), "p")
		if err != nil || project.Name != "原项目" || project.Version != 7 || project.MainFolderID != "f" || len(project.Folders) != 1 || project.Folders[0].Path != "/tmp/original" {
			t.Fatalf("upgrade changed project: %#v %v", project, err)
		}
		terminals, err := s.Terminals(context.Background())
		if err != nil || len(terminals) != 1 || terminals[0].ProjectID == nil || *terminals[0].ProjectID != "p" || terminals[0].DisplayName != "原终端" {
			t.Fatalf("upgrade changed terminal: %#v %v", terminals, err)
		}
		var sessionName string
		if err = s.db.QueryRow("SELECT tmux_session_name FROM terminals WHERE id='t'").Scan(&sessionName); err != nil || sessionName != "original-session" {
			t.Fatal("upgrade changed real session identity")
		}
		if _, err = s.db.Exec("DELETE FROM projects WHERE id='p'"); err != nil {
			t.Fatal(err)
		}
		terminals, err = s.Terminals(context.Background())
		if err != nil || len(terminals) != 1 || terminals[0].ProjectID != nil {
			t.Fatal("post-upgrade foreign keys changed terminal deletion semantics")
		}
		s.Close()
	}
}
func TestMigrationsAndPermissions(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "data", "db.sqlite")
	s, err := Open(ctx, path, "a")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{path, path + "-wal", path + "-shm"} {
		info, e := os.Stat(file)
		if e != nil || info.Mode().Perm()&0077 != 0 {
			t.Fatalf("permissions %s %v", file, e)
		}
	}
	if err = s.CreateSession(ctx, "test-token", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = Open(ctx, path, "a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Session(ctx, "test-token"); err != nil {
		t.Fatal("restart lost session")
	}
	s.Close()
	s, err = Open(ctx, path, "b")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Session(ctx, "test-token"); !errors.Is(err, ErrNotFound) {
		t.Fatal("password change retained session")
	}
	if _, err = s.db.ExecContext(ctx, "UPDATE schema_migrations SET checksum='tampered'"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if _, err = Open(ctx, path, "b"); err == nil {
		t.Fatal("changed migration accepted")
	}
	if err = os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = Open(ctx, path, "b"); err == nil {
		t.Fatal("insecure DB accepted")
	}
}
func TestHighVersionAndRollback(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO schema_migrations VALUES(?,'x','2026-09-27T00:00:00Z')", len(entries)+1); err != nil {
		t.Fatal(err)
	}
	if err := s.migrate(ctx); err == nil {
		t.Fatal("high version accepted")
	}
	var enabled int
	if err := s.db.QueryRow("PRAGMA foreign_keys").Scan(&enabled); err != nil || enabled != 1 {
		t.Fatal("failed migration did not restore foreign keys")
	}
	path := filepath.Join(t.TempDir(), "rollback.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE sessions(incompatible TEXT)"); err != nil {
		t.Fatal(err)
	}
	s2 := &Store{db: db}
	if err = s2.migrate(ctx); err == nil {
		t.Fatal("bad schema accepted")
	}
	var n int
	if err = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='auth_config' OR name='schema_migrations'").Scan(&n); err != nil || n != 0 {
		t.Fatalf("partial migration=%d err=%v", n, err)
	}
	db.Close()
}
func TestListLimitsAndSessionHash(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	if err := s.CreateSession(ctx, "private-token", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	var saved string
	if err := s.db.QueryRow("SELECT token_hash FROM sessions").Scan(&saved); err != nil || saved == "private-token" || saved != HashToken("private-token") {
		t.Fatal("token not hashed")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 201; i++ {
		if _, err = tx.ExecContext(ctx, "INSERT INTO terminals VALUES(?,?,?,NULL,'/tmp','2026-09-27T00:00:00Z')", i, i, "test"); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Terminals(ctx); !errors.Is(err, ErrListLimit) {
		t.Fatal("list silently truncated")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err = s.Projects(canceled); !errors.Is(err, context.Canceled) {
		t.Fatal("context not honored")
	}
}

func TestSessionCleanupFractionalExpiry(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	expires := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	if err := s.CreateSession(ctx, "future-whole-second", expires); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO sessions VALUES(?,?)", HashToken("expired-fractional"), time.Now().UTC().Add(-time.Hour).Format(storedTimeFormat)); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSession(ctx, "trigger-cleanup", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Session(ctx, "future-whole-second"); err != nil {
		t.Fatal("valid expiry was deleted")
	}
	if _, err := s.Session(ctx, "expired-fractional"); !errors.Is(err, ErrNotFound) {
		t.Fatal("expired expiry retained")
	}
	base := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		expiry time.Time
		want   int
	}{{base, 1}, {base.Add(200 * time.Millisecond), 0}} {
		var expired int
		if err := s.db.QueryRowContext(ctx, "SELECT ?<=?", tt.expiry.Format(storedTimeFormat), base.Add(100*time.Millisecond).Format(storedTimeFormat)).Scan(&expired); err != nil || expired != tt.want {
			t.Fatalf("fractional order=%d err=%v", expired, err)
		}
	}
}

func TestConnectionForeignKeys(t *testing.T) {
	s := openTest(t)
	s.db.SetMaxIdleConns(0)
	for i := 0; i < 3; i++ {
		var enabled int
		if err := s.db.QueryRow("PRAGMA foreign_keys").Scan(&enabled); err != nil || enabled != 1 {
			t.Fatalf("connection FK=%d err=%v", enabled, err)
		}
	}
}

func TestSymlinkAndInsecureDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), filepath.Join(dir, "db.sqlite"), "x"); err == nil {
		t.Fatal("insecure directory accepted")
	}
	private := filepath.Join(dir, "private")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(private, "target.sqlite")
	if err := os.WriteFile(target, nil, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(private, "link.sqlite")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), link, "x"); err == nil {
		t.Fatal("database symlink accepted")
	}
}
