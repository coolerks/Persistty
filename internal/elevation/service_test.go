package elevation

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"persistty/internal/files"
	"persistty/internal/storage"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type testBackend struct {
	run    func(context.Context, Grant, []byte, []byte, Publish) (Result, error)
	status func(Grant) (Result, error)
}

func (b testBackend) Target(_ context.Context, r storage.RegisteredFolder, path string) (Target, error) {
	return Target{"example", filepath.Join(r.Path, path)}, nil
}
func (b testBackend) Run(ctx context.Context, g Grant, p, c []byte, publish Publish) (Result, error) {
	if b.run != nil {
		return b.run(ctx, g, p, c, publish)
	}
	return publish(func() (Result, error) {
		v, err := files.Save(ctx, g.Root, g.Path, g.Expected, string(c))
		if err != nil {
			return Result{}, err
		}
		return Result{ID: g.ID, State: "applied", Version: &v}, nil
	})
}
func (b testBackend) Status(_ context.Context, g Grant) (Result, error) {
	if b.status != nil {
		return b.status(g)
	}
	return Result{}, ErrUnavailable
}

type serviceFixture struct {
	s       *Service
	store   *storage.Store
	root    storage.RegisteredFolder
	version files.Version
	token   string
}

func fixture(t *testing.T, b Backend) serviceFixture {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root")
	if err := os.Mkdir(rootPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootPath, "file.txt"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	store, err := storage.Open(ctx, filepath.Join(dir, "data", "db.sqlite"), "test-hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	p, err := store.CreateProject(ctx, "测试", []string{rootPath}, 0)
	if err != nil {
		t.Fatal(err)
	}
	r, err := store.RegisteredFolder(ctx, p.ID, p.MainFolderID, p.Version)
	if err != nil {
		t.Fatal(err)
	}
	v, err := files.ReadContent(r, "file.txt")
	if err != nil {
		t.Fatal(err)
	}
	token := "synthetic-session"
	if err = store.CreateSession(ctx, token, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	s, err := New(store, b)
	if err != nil {
		t.Fatal(err)
	}
	return serviceFixture{s, store, r, v.Version, token}
}
func (f serviceFixture) prepare(t *testing.T) Prepared {
	t.Helper()
	p, err := f.s.Prepare(context.Background(), f.token, f.root, "file.txt", f.version, []byte("new"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestExecuteSingleUseAndSessionBinding(t *testing.T) {
	f := fixture(t, testBackend{})
	p := f.prepare(t)
	ctx := context.Background()
	if _, err := f.s.Status(ctx, "another-session", p.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal(err)
	}
	result, err := f.s.Execute(ctx, f.token, p.ID, []byte("synthetic-system-password"), []byte("new"))
	if err != nil || result.State != "applied" {
		t.Fatalf("%+v %v", result, err)
	}
	if _, err = f.s.Execute(ctx, f.token, p.ID, []byte("synthetic-system-password"), []byte("new")); !errors.Is(err, storage.ErrRequestConsumed) {
		t.Fatal(err)
	}
	record, err := f.store.Elevation(ctx, p.ID, storage.HashToken(f.token))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(record.PayloadJSON, "synthetic-system-password") || strings.Contains(record.PayloadJSON, `"content"`) {
		t.Fatal("secret persisted")
	}
}
func TestInvalidContentConsumesAndCancelExpires(t *testing.T) {
	f := fixture(t, testBackend{})
	p := f.prepare(t)
	ctx := context.Background()
	r, err := f.s.Execute(ctx, f.token, p.ID, []byte("password"), []byte("other"))
	if err != nil || r.State != "rejected" {
		t.Fatal(r, err)
	}
	if _, err = f.s.Execute(ctx, f.token, p.ID, []byte("password"), []byte("new")); !errors.Is(err, storage.ErrRequestConsumed) {
		t.Fatal(err)
	}
	p = f.prepare(t)
	if r, err = f.s.Cancel(ctx, f.token, p.ID); err != nil || r.State != "cancelled" {
		t.Fatal(r, err)
	}
	if _, err = f.s.Execute(ctx, f.token, p.ID, []byte("password"), []byte("new")); !errors.Is(err, storage.ErrRequestConsumed) {
		t.Fatal(err)
	}
	if err = f.store.DeleteSession(ctx, f.token); err != nil {
		t.Fatal(err)
	}
	if err = f.store.CreateSession(ctx, f.token, time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	p = f.prepare(t)
	<-time.After(time.Until(p.ExpiresAt) + time.Millisecond)
	if r, err = f.s.Status(ctx, f.token, p.ID); err != nil || r.State != "expired" {
		t.Fatal(r, err)
	}
}
func TestWaitingAuthenticationDoesNotBlockRevocation(t *testing.T) {
	for _, action := range []string{"logout", "project", "cancel", "duplicate"} {
		t.Run(action, func(t *testing.T) {
			ready, release := make(chan struct{}), make(chan struct{})
			var commits atomic.Int32
			b := testBackend{run: func(ctx context.Context, g Grant, _, _ []byte, publish Publish) (Result, error) {
				close(ready)
				select {
				case <-release:
				case <-ctx.Done():
					return Result{}, ctx.Err()
				}
				return publish(func() (Result, error) { commits.Add(1); return outcome(g.ID, "rejected", "synthetic"), nil })
			}}
			f := fixture(t, b)
			p := f.prepare(t)
			ctx := context.Background()
			done := make(chan Result, 1)
			go func() { r, _ := f.s.Execute(ctx, f.token, p.ID, []byte("password"), []byte("new")); done <- r }()
			<-ready
			switch action {
			case "logout":
				if err := f.store.DeleteSession(ctx, f.token); err != nil {
					t.Fatal(err)
				}
			case "project":
				if err := f.store.DeleteProject(ctx, f.root.ProjectID, f.root.ProjectVersion); err != nil {
					t.Fatal(err)
				}
			case "cancel":
				if _, err := f.s.Cancel(ctx, f.token, p.ID); err != nil {
					t.Fatal(err)
				}
			case "duplicate":
				if _, err := f.s.Execute(ctx, f.token, p.ID, []byte("password"), []byte("new")); !errors.Is(err, storage.ErrRequestConsumed) {
					t.Fatal(err)
				}
				if _, err := f.s.Cancel(ctx, f.token, p.ID); err != nil {
					t.Fatal(err)
				}
			}
			close(release)
			select {
			case r := <-done:
				if r.State == "applied" || commits.Load() != 0 {
					t.Fatal(r, commits.Load())
				}
			case <-time.After(time.Second):
				t.Fatal("execution did not stop")
			}
		})
	}
}
func TestRestartNeverReplaysAndStatusResolves(t *testing.T) {
	f := fixture(t, testBackend{status: func(g Grant) (Result, error) { return outcome(g.ID, "rejected", "authorization_failed"), nil }})
	p := f.prepare(t)
	ctx := context.Background()
	if err := f.store.TransitionElevation(ctx, p.ID, storage.HashToken(f.token), "prepared", "executing", nil, nil); err != nil {
		t.Fatal(err)
	}
	restarted, err := New(f.store, f.s.backend)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = restarted.Execute(ctx, f.token, p.ID, []byte("password"), []byte("new")); !errors.Is(err, storage.ErrRequestConsumed) {
		t.Fatal(err)
	}
	r, err := restarted.Status(ctx, f.token, p.ID)
	if err != nil || r.State != "rejected" {
		t.Fatal(r, err)
	}
}
func TestIndeterminateStatusNeverReturnsPrepared(t *testing.T) {
	f := fixture(t, testBackend{run: func(context.Context, Grant, []byte, []byte, Publish) (Result, error) { return Result{}, ErrUnavailable }, status: func(g Grant) (Result, error) { return Result{ID: g.ID, State: "prepared"}, nil }})
	p := f.prepare(t)
	ctx := context.Background()
	_, _ = f.s.Execute(ctx, f.token, p.ID, []byte("pwd"), []byte("new"))
	r, err := f.s.Status(ctx, f.token, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.State == "prepared" {
		t.Fatal("status restored replayable request")
	}
}
func TestGrantBounds(t *testing.T) {
	f := fixture(t, testBackend{})
	p := f.prepare(t)
	r, _ := f.store.Elevation(context.Background(), p.ID, storage.HashToken(f.token))
	g, err := grantFromRecord(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Grant){func(g *Grant) { g.ExpiresAt = g.IssuedAt.Add(61 * time.Second) }, func(g *Grant) { g.Path = "../escape" }, func(g *Grant) { g.ContentHash = "sha256:" + strings.Repeat("z", 64) }, func(g *Grant) { g.Expected.Mtime = "invalid" }} {
		bad := g
		change(&bad)
		if bad.Validate(time.Now()) == nil {
			t.Fatal("invalid grant accepted")
		}
	}
	data, _ := json.Marshal(g)
	if strictJSON(data, &Grant{}) != nil {
		t.Fatal("valid grant rejected")
	}
}

func TestDisabledBackendUnavailable(t *testing.T) {
	f := fixture(t, Client{})
	if _, err := f.s.Prepare(context.Background(), f.token, f.root, "file.txt", f.version, []byte("new")); !errors.Is(err, ErrUnavailable) {
		t.Fatal("default exposed elevation", err)
	}
}

func TestServiceCapacityRejectsBeforeConsume(t *testing.T) {
	started := make(chan struct{}, 2)
	b := testBackend{run: func(ctx context.Context, g Grant, _, _ []byte, _ Publish) (Result, error) {
		started <- struct{}{}
		<-ctx.Done()
		return Result{}, ctx.Err()
	}}
	f := fixture(t, b)
	first, second, third := f.prepare(t), f.prepare(t), f.prepare(t)
	ctx := context.Background()
	done := make(chan Result, 2)
	for _, id := range []string{first.ID, second.ID} {
		go func(id string) { r, _ := f.s.Execute(ctx, f.token, id, []byte("synthetic"), []byte("new")); done <- r }(id)
		<-started
	}
	if _, err := f.s.Execute(ctx, f.token, third.ID, []byte("synthetic"), []byte("new")); !errors.Is(err, ErrRateLimited) {
		t.Fatal(err)
	}
	if r, err := f.s.Status(ctx, f.token, third.ID); err != nil || r.State != "prepared" {
		t.Fatal("capacity denial consumed a request", r, err)
	}
	for _, id := range []string{first.ID, second.ID} {
		if _, err := f.s.Cancel(ctx, f.token, id); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		select {
		case r := <-done:
			if r.State != "cancelled" {
				t.Fatal(r)
			}
		case <-time.After(time.Second):
			t.Fatal("cancellation failed")
		}
	}
}
