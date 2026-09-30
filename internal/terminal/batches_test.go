package terminal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"persistty/internal/storage"
)

func batchRuntime(t *testing.T) (*Runtime, []*Viewer, []*Viewer, []BatchTarget, *atomic.Int32) {
	t.Helper()
	r := NewRuntime(&Tmux{}, time.Hour, func(context.Context, string) error { return nil })
	kills := &atomic.Int32{}
	r.tmux.run = func(_ context.Context, _ int64, args ...string) ([]byte, error) {
		if args[0] == "kill-session" {
			kills.Add(1)
		}
		return nil, nil
	}
	var controllers, observers []*Viewer
	var targets []BatchTarget
	for i := 0; i < 5; i++ {
		h, first, second, _ := testHub(t, time.Hour)
		h.runtime = r
		h.terminal.ID = fmt.Sprintf("%032x", i+1)
		h.terminal.DisplayName = fmt.Sprintf("终端%d", i)
		h.terminal.TmuxSessionName = fmt.Sprintf("persistty_%032x", i+1)
		first.ID = fmt.Sprintf("%032x", i+11)
		second.ID = fmt.Sprintf("%032x", i+21)
		first.protocolVersion = 3
		h.controller = first.ID
		h.viewers = map[string]*Viewer{first.ID: first, second.ID: second}
		r.hubs[h.terminal.ID] = h
		controllers = append(controllers, first)
		observers = append(observers, second)
		targets = append(targets, BatchTarget{h.terminal.ID, first.ID, 5})
	}
	t.Cleanup(func() {
		r.controlMu.Lock()
		defer r.controlMu.Unlock()
		for _, p := range r.batches {
			r.cancelBatchLocked(p)
		}
	})
	return r, controllers, observers, targets, kills
}

func TestBatchCancellationIsScopedAndMixedProtocol(t *testing.T) {
	for _, reason := range []string{"observer-cancel", "takeover", "initiator-disconnected", "authentication"} {
		t.Run(reason, func(t *testing.T) {
			r, controllers, observers, targets, kills := batchRuntime(t)
			abc, err := r.TerminateBatch(context.Background(), "token", targets[:3])
			if err != nil {
				t.Fatal(err)
			}
			de, err := r.TerminateBatch(context.Background(), "token", targets[3:])
			if err != nil {
				t.Fatal(err)
			}
			var ready map[string]json.RawMessage
			if err := json.Unmarshal(observers[1].Ready(), &ready); err != nil {
				t.Fatal(err)
			}
			var legacy map[string]json.RawMessage
			if err := json.Unmarshal(ready["pending_termination"], &legacy); err != nil || len(legacy) != 2 {
				t.Fatal("v2 ready changed")
			}
			if err := json.Unmarshal(controllers[1].Ready(), &ready); err != nil {
				t.Fatal(err)
			}
			var modern struct {
				Members []BatchMember `json:"members"`
			}
			if err := json.Unmarshal(ready["pending_termination"], &modern); err != nil || len(modern.Members) != 3 {
				t.Fatal("v3 ready lost members")
			}
			switch reason {
			case "observer-cancel":
				if err := observers[1].CancelTermination(context.Background(), abc.RequestID); err != nil {
					t.Fatal(err)
				}
			case "takeover":
				if err := observers[1].Takeover(context.Background(), 5); err != nil {
					t.Fatal(err)
				}
			case "initiator-disconnected":
				controllers[1].cancel()
				r.executeBatch(r.batches[abc.RequestID])
			case "authentication":
				r.validate = func(context.Context, string) error { return errors.New("revoked") }
				r.executeBatch(r.batches[abc.RequestID])
			}
			cancelled, _ := r.Batch(abc.RequestID)
			independent, _ := r.Batch(de.RequestID)
			if cancelled.State != "cancelled" || independent.State != "pending" || kills.Load() != 0 {
				t.Fatalf("scope violated: %+v %+v", cancelled, independent)
			}
			for _, v := range controllers[:3] {
				v.hub.mu.Lock()
				pending := v.hub.pending
				v.hub.mu.Unlock()
				if pending != nil {
					t.Fatal("cancelled member retained countdown")
				}
			}
		})
	}
}

func TestBatchValidationStartsNothing(t *testing.T) {
	r, _, observers, targets, kills := batchRuntime(t)
	for _, kind := range []string{"stale", "observer", "wrong-token", "duplicate", "missing", "empty"} {
		input := append([]BatchTarget{}, targets[:3]...)
		token := "token"
		switch kind {
		case "stale":
			input[1].Generation--
		case "observer":
			input[1].ViewerID = observers[1].ID
		case "wrong-token":
			token = "other"
		case "duplicate":
			input[1] = input[0]
		case "missing":
			input[1].ViewerID = "missing"
		case "empty":
			input = nil
		}
		if _, err := r.TerminateBatch(context.Background(), token, input); err == nil {
			t.Fatalf("accepted %s", kind)
		}
		if len(r.batches) != 0 || kills.Load() != 0 {
			t.Fatalf("partially started %s", kind)
		}
	}
}

func TestBatchKillDoesNotHoldCoordinatorAndDoesNotRetry(t *testing.T) {
	r, controllers, observers, targets, _ := batchRuntime(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	var kills atomic.Int32
	r.tmux.run = func(_ context.Context, _ int64, args ...string) ([]byte, error) {
		if args[0] == "kill-session" {
			kills.Add(1)
			once.Do(func() { close(entered) })
			<-release
			return nil, errors.New("failed")
		}
		return nil, nil
	}
	abc, err := r.TerminateBatch(context.Background(), "token", targets[:3])
	if err != nil {
		t.Fatal(err)
	}
	de, err := r.TerminateBatch(context.Background(), "token", targets[3:])
	if err != nil {
		t.Fatal(err)
	}
	r.controlMu.Lock()
	p := r.batches[abc.RequestID]
	lockParticipants(p.Participants)
	p.Deadline = time.Now().Add(-time.Second)
	unlockParticipants(p.Participants)
	r.controlMu.Unlock()
	done := make(chan struct{})
	go func() { r.executeBatch(p); close(done) }()
	<-entered
	if err := observers[3].CancelTermination(context.Background(), de.RequestID); err != nil {
		t.Fatal(err)
	}
	close(release)
	<-done
	result, err := r.Batch(abc.RequestID)
	if err != nil || result.State != "completed" || len(result.Results) != 3 || kills.Load() != 3 {
		t.Fatalf("unexpected result: %+v %v kills=%d", result, err, kills.Load())
	}
	for _, item := range result.Results {
		if item.State != "running" {
			t.Fatalf("failed kill reported success: %+v", item)
		}
	}
	r.executeBatch(p)
	if kills.Load() != 3 {
		t.Fatal("replayed completed batch")
	}
	updated := controllers[0].hub.terminal
	updated.DisplayName = "构建"
	r.UpdateMetadata(updated)
	if controllers[0].hub.terminal.ID != updated.ID {
		t.Fatal("rename changed identity")
	}
	if _, err := r.Batch("missing"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal(err)
	}
}
