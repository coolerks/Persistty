package auth

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"persistty/internal/storage"
)

func TestPassword(t *testing.T) {
	password := []byte("test-only-password")
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseHash(hash)
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.Verify(password) || parsed.Verify([]byte("wrong")) {
		t.Fatal("verification")
	}
	for _, bad := range []string{"", strings.Replace(hash, "v=19", "v=18", 1), strings.Replace(hash, "m=65536", "m=99999999", 1), strings.Replace(hash, "t=3", "t=0", 1), strings.Replace(hash, "p=1", "p=0", 1), hash + "$", strings.Replace(hash, "argon2id", "argon2i", 1)} {
		if _, err := ParseHash(bad); err == nil {
			t.Fatal("accepted bad PHC")
		}
	}
	if _, err = HashPassword([]byte("short")); err == nil {
		t.Fatal("weak password accepted")
	}
}
func newService(t *testing.T) (*Service, *storage.Store) {
	t.Helper()
	hash, err := HashPassword([]byte("test-only-password"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "data", "db.sqlite"), hash)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	service, err := New(store, hash, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return service, store
}
func TestSessions(t *testing.T) {
	s, _ := newService(t)
	ctx := context.Background()
	first, a, err := s.Login(ctx, "a", []byte("test-only-password"))
	if err != nil {
		t.Fatal(err)
	}
	_, b, err := s.Login(ctx, "b", []byte("test-only-password"))
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("session fixation")
	}
	if !ValidCSRF(first.CSRFToken, csrf(a)) || ValidCSRF(first.CSRFToken, csrf(b)) {
		t.Fatal("csrf binding")
	}
	if err = s.Logout(ctx, a); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Lookup(ctx, a); !errors.Is(err, ErrUnauthenticated) {
		t.Fatal("not revoked")
	}
	if _, err = s.Lookup(ctx, b); err != nil {
		t.Fatal("other device revoked")
	}
	s.now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	if _, err = s.Lookup(ctx, b); !errors.Is(err, ErrUnauthenticated) {
		t.Fatal("expiry")
	}
	for _, bad := range []string{"", strings.Repeat("x", 44), "plaintext-secret"} {
		if _, err = s.Lookup(ctx, bad); !errors.Is(err, ErrUnauthenticated) {
			t.Fatal("bad token")
		}
	}
}
func TestLimits(t *testing.T) {
	s, _ := newService(t)
	for i := 0; i < 10; i++ {
		if !s.allow("same") {
			t.Fatal("early limit")
		}
	}
	if s.allow("same") {
		t.Fatal("source unbounded")
	}
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Go(func() { s.allow("parallel") })
	}
	wg.Wait()
	s.workers <- struct{}{}
	s.workers <- struct{}{}
	if _, _, err := s.Login(context.Background(), "other", []byte("test-only-password")); !errors.Is(err, ErrRateLimited) {
		t.Fatal("unbounded argon workers")
	}
}
