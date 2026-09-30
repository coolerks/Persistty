package terminal

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"persistty/internal/storage"
)

func testHub(t *testing.T, deadline time.Duration) (*hub, *Viewer, *Viewer, *os.File) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { reader.Close(); writer.Close() })
	runtime := NewRuntime(&Tmux{}, deadline, func(context.Context, string) error { return nil })
	h := &hub{runtime: runtime, terminal: storage.Terminal{ID: "terminal", TmuxSessionName: "persistty_abcdef"},
		owner: &attachClient{file: writer}, viewers: map[string]*Viewer{}, controller: "first", generation: 5}
	firstCtx, firstCancel := context.WithCancel(context.Background())
	secondCtx, secondCancel := context.WithCancel(context.Background())
	t.Cleanup(firstCancel)
	t.Cleanup(secondCancel)
	first := &Viewer{ID: "first", Token: "token", hub: h, queue: make(chan Frame, 32), ctx: firstCtx, cancel: firstCancel}
	second := &Viewer{ID: "second", Token: "token", hub: h, queue: make(chan Frame, 32), ctx: secondCtx, cancel: secondCancel}
	h.viewers[first.ID], h.viewers[second.ID] = first, second
	return h, first, second, reader
}

func TestControllerGenerationAndObserverIsolation(t *testing.T) {
	h, first, second, reader := testHub(t, time.Second)
	data := make([]byte, 9)
	binary.BigEndian.PutUint64(data[:8], 5)
	data[8] = 'x'
	if err := second.Input(context.Background(), data); !errors.Is(err, ErrControlDenied) {
		t.Fatalf("observer input: %v", err)
	}
	if err := second.Resize(context.Background(), 5, 120, 40); !errors.Is(err, ErrControlDenied) {
		t.Fatalf("observer resize: %v", err)
	}
	if err := first.Input(context.Background(), data); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1)
	if _, err := reader.Read(buf); err != nil || buf[0] != 'x' {
		t.Fatalf("owner received %q: %v", buf, err)
	}
	if err := second.Takeover(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	if h.generation != 6 || h.controller != second.ID {
		t.Fatalf("control did not transfer: %d %s", h.generation, h.controller)
	}
	if err := first.Input(context.Background(), data); !errors.Is(err, ErrControlDenied) {
		t.Fatalf("old controller input: %v", err)
	}
	if err := second.Input(context.Background(), data); !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("old generation input: %v", err)
	}
	binary.BigEndian.PutUint64(data[:8], 6)
	if err := second.Input(context.Background(), data); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Read(buf); err != nil || buf[0] != 'x' {
		t.Fatalf("new controller input missing: %v", err)
	}
	var event map[string]any
	if err := json.Unmarshal((<-first.queue).Data, &event); err != nil || event["role"] != "observer" {
		t.Fatalf("revocation frame: %v %v", event, err)
	}
}

func TestTerminationCancelAndSlowViewer(t *testing.T) {
	h, first, second, _ := testHub(t, 80*time.Millisecond)
	var kills atomic.Int32
	h.runtime.tmux.run = func(_ context.Context, _ int64, args ...string) ([]byte, error) {
		if args[0] == "kill-session" {
			kills.Add(1)
		}
		return nil, nil
	}
	if err := first.Terminate(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	requestID := h.pending.ID
	if err := second.CancelTermination(context.Background(), requestID); err != nil {
		t.Fatal(err)
	}
	time.Sleep(120 * time.Millisecond)
	if kills.Load() != 0 || h.pending != nil {
		t.Fatalf("cancelled request executed: kills=%d", kills.Load())
	}
	for i := 0; i < cap(second.queue); i++ {
		h.enqueueLocked(second, Frame{})
	}
	h.enqueueLocked(second, Frame{})
	select {
	case <-second.Done():
	case <-time.After(time.Second):
		t.Fatal("slow viewer was not detached")
	}
	select {
	case <-first.Done():
		t.Fatal("slow viewer affected other client")
	default:
	}
	if err := first.Terminate(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(time.Second)
	for kills.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("uncancelled request did not execute")
		case <-time.After(time.Millisecond):
		}
	}
}

func TestOwnerExitRevokesViewersAndPendingTermination(t *testing.T) {
	h, first, second, reader := testHub(t, 30*time.Millisecond)
	var kills atomic.Int32
	h.runtime.tmux.run = func(_ context.Context, _ int64, args ...string) ([]byte, error) {
		if args[0] == "kill-session" {
			kills.Add(1)
		}
		return nil, nil
	}
	if err := first.Terminate(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	if err := h.owner.file.Close(); err != nil {
		t.Fatal(err)
	}
	h.owner.file = reader
	h.drainOwner()
	if !h.closed || h.pending != nil {
		t.Fatal("owner exit did not close hub and cancel termination")
	}
	for _, viewer := range []*Viewer{first, second} {
		select {
		case <-viewer.Done():
		default:
			t.Fatal("owner exit did not revoke viewer")
		}
	}
	time.Sleep(50 * time.Millisecond)
	if kills.Load() != 0 {
		t.Fatal("cancelled termination killed the session")
	}
}

func TestTerminationIdentityAndRevocation(t *testing.T) {
	for _, revoke := range []string{"takeover", "authentication", "initiator-disconnected"} {
		t.Run(revoke, func(t *testing.T) {
			h, first, second, _ := testHub(t, time.Hour)
			var kills atomic.Int32
			h.runtime.tmux.run = func(_ context.Context, _ int64, args ...string) ([]byte, error) {
				if args[0] == "kill-session" {
					kills.Add(1)
				}
				return nil, nil
			}
			if err := second.Terminate(context.Background(), 5); !errors.Is(err, ErrControlDenied) {
				t.Fatalf("observer termination: %v", err)
			}
			if err := first.Terminate(context.Background(), 5); err != nil {
				t.Fatal(err)
			}
			pending := h.pending
			if err := first.Terminate(context.Background(), 5); !errors.Is(err, ErrTerminationPending) {
				t.Fatalf("second countdown: %v", err)
			}
			var ready struct {
				Pending *pendingTermination `json:"pending_termination"`
			}
			if err := json.Unmarshal(second.Ready(), &ready); err != nil || ready.Pending == nil || ready.Pending.ID != pending.ID || !ready.Pending.Deadline.Equal(pending.Deadline) {
				t.Fatalf("new viewer deadline: %+v %v", ready, err)
			}
			switch revoke {
			case "takeover":
				if err := second.Takeover(context.Background(), 5); err != nil {
					t.Fatal(err)
				}
			case "authentication":
				h.runtime.validate = func(context.Context, string) error { return errors.New("revoked") }
			case "initiator-disconnected":
				delete(h.viewers, first.ID)
			}
			h.executeTermination(pending)
			if kills.Load() != 0 || h.pending != nil {
				t.Fatalf("revocation executed termination: %d", kills.Load())
			}
		})
	}
}
