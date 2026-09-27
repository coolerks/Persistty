package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"persistty/internal/storage"
)

var ErrUnauthenticated = errors.New("unauthenticated")
var ErrRateLimited = errors.New("rate limited")

type Session struct {
	Authenticated bool      `json:"authenticated"`
	ExpiresAt     time.Time `json:"expires_at"`
	CSRFToken     string    `json:"csrf_token"`
}
type bucket struct {
	start time.Time
	count int
}
type Service struct {
	store    *storage.Store
	password PasswordHash
	ttl      time.Duration
	workers  chan struct{}
	mu       sync.Mutex
	global   bucket
	sources  map[string]bucket
	now      func() time.Time
}

func New(store *storage.Store, phc string, ttl time.Duration) (*Service, error) {
	p, err := ParseHash(phc)
	if err != nil {
		return nil, err
	}
	return &Service{store: store, password: p, ttl: ttl, workers: make(chan struct{}, 2), sources: make(map[string]bucket), now: time.Now}, nil
}
func (s *Service) allow(source string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if now.Sub(s.global.start) >= time.Minute {
		s.global = bucket{start: now}
	}
	if s.global.count >= 60 {
		return false
	}
	s.global.count++
	for key, b := range s.sources {
		if now.Sub(b.start) >= time.Minute {
			delete(s.sources, key)
		}
	}
	b, exists := s.sources[source]
	if !exists {
		if len(s.sources) >= 1024 {
			return false
		}
		b.start = now
	}
	if b.count >= 10 {
		return false
	}
	b.count++
	s.sources[source] = b
	return true
}
func (s *Service) Login(ctx context.Context, source string, password []byte) (Session, string, error) {
	if !s.allow(source) {
		return Session{}, "", ErrRateLimited
	}
	select {
	case s.workers <- struct{}{}:
		defer func() { <-s.workers }()
	default:
		return Session{}, "", ErrRateLimited
	}
	if err := ctx.Err(); err != nil {
		return Session{}, "", err
	}
	if !s.password.Verify(password) {
		return Session{}, "", ErrUnauthenticated
	}
	if err := ctx.Err(); err != nil {
		return Session{}, "", err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Session{}, "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	clear(raw)
	expires := s.now().UTC().Add(s.ttl)
	if err := s.store.CreateSession(ctx, token, expires); err != nil {
		return Session{}, "", err
	}
	return Session{true, expires, csrf(token)}, token, nil
}
func (s *Service) Lookup(ctx context.Context, token string) (Session, error) {
	raw, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(raw) != 32 {
		return Session{}, ErrUnauthenticated
	}
	clear(raw)
	expires, err := s.store.Session(ctx, token)
	if errors.Is(err, storage.ErrNotFound) {
		return Session{}, ErrUnauthenticated
	}
	if err != nil {
		return Session{}, err
	}
	if !expires.After(s.now()) {
		return Session{}, ErrUnauthenticated
	}
	return Session{true, expires, csrf(token)}, nil
}
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, token)
}
func csrf(token string) string {
	hash := sha256.Sum256([]byte("persistty-csrf-v1:" + token))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
func ValidCSRF(actual, expected string) bool {
	return len(actual) == len(expected) && subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}
