package elevation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"persistty/internal/files"
	"persistty/internal/storage"
)

type attempt struct {
	mu                    sync.Mutex
	cancel                context.CancelFunc
	cancelled, committing bool
}
type rateBucket struct {
	since time.Time
	n     int
}
type Service struct {
	store   *storage.Store
	backend Backend
	mu      sync.Mutex
	active  map[string]*attempt
	rates   map[string]rateBucket
	global  rateBucket
	slots   chan struct{}
}

func New(store *storage.Store, backend Backend) (*Service, error) {
	if err := store.RecoverElevation(context.Background()); err != nil {
		return nil, err
	}
	return &Service{store: store, backend: backend, active: make(map[string]*attempt), rates: make(map[string]rateBucket), slots: make(chan struct{}, 2)}, nil
}
func (s *Service) allow(hash string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, b := range s.rates {
		if now.Sub(b.since) >= time.Minute {
			delete(s.rates, key)
		}
	}
	if now.Sub(s.global.since) >= time.Minute {
		s.global = rateBucket{since: now}
	}
	b, ok := s.rates[hash]
	if !ok {
		if len(s.rates) >= 1024 {
			return false
		}
		b.since = now
	}
	if b.n >= 10 || s.global.n >= 60 {
		return false
	}
	b.n++
	s.global.n++
	s.rates[hash] = b
	return true
}
func (s *Service) Prepare(ctx context.Context, token string, root storage.RegisteredFolder, path string, expected files.Version, content []byte) (Prepared, error) {
	if !s.allow(storage.HashToken(token)) {
		return Prepared{}, ErrRateLimited
	}
	if !ValidContent(content) || !validRelativeTarget(path) {
		return Prepared{}, ErrInvalid
	}
	target, err := s.backend.Target(ctx, root, path)
	if err != nil {
		return Prepared{}, err
	}
	snapshot, err := files.ReadContent(root, path)
	if err != nil {
		return Prepared{}, err
	}
	if snapshot.Version != expected {
		return Prepared{}, files.ErrConflict
	}
	expires, err := s.store.Session(ctx, token)
	if err != nil {
		return Prepared{}, err
	}
	now := time.Now().UTC()
	if !expires.After(now) {
		return Prepared{}, storage.ErrSessionInvalid
	}
	if deadline := now.Add(time.Minute); expires.After(deadline) {
		expires = deadline
	}
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return Prepared{}, err
	}
	g := Grant{ID: hex.EncodeToString(raw), SessionHash: storage.HashToken(token), Root: root, Path: path, TargetID: target.ID, Expected: expected, ContentHash: Digest(content), IssuedAt: now, ExpiresAt: expires}
	if err = g.Validate(now); err != nil {
		return Prepared{}, err
	}
	data, err := json.Marshal(g)
	if err != nil {
		return Prepared{}, err
	}
	// Recheck current association when publishing request metadata, but never
	// retain this lock while a user types a password or sudo authenticates.
	err = s.store.WithElevationPublication(ctx, token, root, func() error {
		return s.store.CreateElevation(ctx, storage.ElevationRecord{ID: g.ID, SessionHash: g.SessionHash, PayloadJSON: string(data), ExpiresAt: g.ExpiresAt})
	})
	if err != nil {
		if errors.Is(err, storage.ErrQuota) {
			return Prepared{}, ErrUnavailable
		}
		return Prepared{}, err
	}
	return Prepared{g.ID, target.ID, target.Path, g.ContentHash, g.ExpiresAt, "prepared"}, nil
}
func grantFromRecord(record storage.ElevationRecord) (Grant, error) {
	var g Grant
	if strictJSON([]byte(record.PayloadJSON), &g) != nil || g.ID != record.ID || g.SessionHash != record.SessionHash {
		return g, ErrUnavailable
	}
	return g, nil
}
func recordResult(record storage.ElevationRecord) (Result, error) {
	result := Result{ID: record.ID, State: record.State, Code: record.Code}
	if record.VersionJSON != nil {
		var v files.Version
		if json.Unmarshal([]byte(*record.VersionJSON), &v) != nil {
			return result, ErrUnavailable
		}
		result.Version = &v
	}
	if !ResultValid(result, record.ID) {
		return result, ErrUnavailable
	}
	return result, nil
}
func (s *Service) persistResult(g Grant, result Result) error {
	if !ResultValid(result, g.ID) {
		return ErrUnavailable
	}
	var version *string
	if result.Version != nil {
		data, err := json.Marshal(result.Version)
		if err != nil {
			return err
		}
		text := string(data)
		version = &text
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.store.TransitionElevation(ctx, g.ID, g.SessionHash, "executing", result.State, result.Code, version)
}
func (s *Service) Execute(ctx context.Context, token, id string, password, content []byte) (Result, error) {
	if !noncePattern.MatchString(id) {
		return Result{}, ErrInvalid
	}
	hash := storage.HashToken(token)
	record, err := s.store.Elevation(ctx, id, hash)
	if err != nil {
		return Result{}, err
	}
	if record.State != "prepared" {
		return Result{}, storage.ErrRequestConsumed
	}
	if !s.allow(hash) {
		return Result{}, ErrRateLimited
	}
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		return Result{}, ErrRateLimited
	}
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	a := &attempt{cancel: cancel}
	s.mu.Lock()
	if _, exists := s.active[id]; exists {
		s.mu.Unlock()
		return Result{}, storage.ErrRequestConsumed
	}
	s.active[id] = a
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.active, id); s.mu.Unlock() }()
	// Consume before any authentication/validation failure, not after sudo.
	if err = s.store.TransitionElevation(ctx, id, hash, "prepared", "executing", nil, nil); err != nil {
		return Result{}, err
	}
	g, err := grantFromRecord(record)
	if err != nil {
		return Result{}, err
	}
	if validation := g.Validate(time.Now()); validation != nil {
		result := outcome(id, "rejected", "invalid_request")
		if errors.Is(validation, ErrExpired) {
			result = outcome(id, "expired", "expired")
		}
		if s.persistResult(g, result) != nil {
			return outcome(id, "indeterminate", "outcome_unknown"), nil
		}
		return result, nil
	}
	if !ValidPassword(password) || !ValidContent(content) || Digest(content) != g.ContentHash {
		result := outcome(id, "rejected", "invalid_request")
		if s.persistResult(g, result) != nil {
			return outcome(id, "indeterminate", "outcome_unknown"), nil
		}
		return result, nil
	}
	result, err := s.backend.Run(runCtx, g, password, content, func(commit func() (Result, error)) (Result, error) {
		var result Result
		err := s.store.WithElevationPublication(runCtx, token, g.Root, func() error {
			a.mu.Lock()
			defer a.mu.Unlock()
			if a.cancelled || runCtx.Err() != nil {
				return context.Canceled
			}
			if !g.ExpiresAt.After(time.Now()) {
				return ErrExpired
			}
			a.committing = true
			var err error
			result, err = commit()
			return err
		})
		return result, err
	})
	if err != nil {
		result = outcome(id, "indeterminate", "outcome_unknown")
		switch {
		case errors.Is(err, storage.ErrConflict), errors.Is(err, storage.ErrNotFound):
			result = outcome(id, "rejected", "conflict")
		case errors.Is(err, storage.ErrSessionInvalid):
			result = outcome(id, "rejected", "unauthenticated")
		case errors.Is(err, ErrExpired):
			result = outcome(id, "expired", "expired")
		}
	}
	a.mu.Lock()
	if a.cancelled && !a.committing {
		result = outcome(id, "cancelled", "cancelled")
	}
	a.mu.Unlock()
	if !ResultValid(result, id) {
		result = outcome(id, "indeterminate", "outcome_unknown")
	}
	if err = s.persistResult(g, result); err != nil {
		return outcome(id, "indeterminate", "outcome_unknown"), nil
	}
	return result, nil
}
func (s *Service) Status(ctx context.Context, token, id string) (Result, error) {
	if !noncePattern.MatchString(id) {
		return Result{}, ErrInvalid
	}
	hash := storage.HashToken(token)
	record, err := s.store.Elevation(ctx, id, hash)
	if err != nil {
		return Result{}, err
	}
	if record.State == "prepared" && !record.ExpiresAt.After(time.Now()) {
		code := "expired"
		if err = s.store.TransitionElevation(ctx, id, hash, "prepared", "expired", &code, nil); err != nil && !errors.Is(err, storage.ErrRequestConsumed) {
			return Result{}, err
		}
		record, err = s.store.Elevation(ctx, id, hash)
		if err != nil {
			return Result{}, err
		}
	}
	s.mu.Lock()
	live := s.active[id] != nil
	s.mu.Unlock()
	if !live && (record.State == "executing" || record.State == "indeterminate") {
		g, err := grantFromRecord(record)
		if err != nil {
			return Result{}, err
		}
		observed, probeErr := s.backend.Status(ctx, g)
		if probeErr == nil && ResultValid(observed, id) && observed.State != "indeterminate" && observed.State != "prepared" && observed.State != "executing" {
			var version *string
			if observed.Version != nil {
				data, err := json.Marshal(observed.Version)
				if err != nil {
					return Result{}, err
				}
				text := string(data)
				version = &text
			}
			if err = s.store.TransitionElevation(ctx, id, hash, record.State, observed.State, observed.Code, version); err != nil && !errors.Is(err, storage.ErrRequestConsumed) {
				return Result{}, err
			}
			record, err = s.store.Elevation(ctx, id, hash)
			if err != nil {
				return Result{}, err
			}
		}
	}
	return recordResult(record)
}
func (s *Service) Cancel(ctx context.Context, token, id string) (Result, error) {
	if !noncePattern.MatchString(id) {
		return Result{}, ErrInvalid
	}
	hash := storage.HashToken(token)
	record, err := s.store.Elevation(ctx, id, hash)
	if err != nil {
		return Result{}, err
	}
	if record.State == "prepared" {
		code := "cancelled"
		err = s.store.TransitionElevation(ctx, id, hash, "prepared", "cancelled", &code, nil)
		if errors.Is(err, storage.ErrRequestConsumed) {
			return s.Cancel(ctx, token, id)
		}
		if err != nil {
			return Result{}, err
		}
	} else if record.State == "executing" {
		s.mu.Lock()
		a := s.active[id]
		s.mu.Unlock()
		if a != nil {
			a.mu.Lock()
			if !a.committing {
				a.cancelled = true
				a.cancel()
			}
			a.mu.Unlock()
		}
	}
	return s.Status(ctx, token, id)
}
