package terminal

import (
	"context"
	"sort"
	"sync"
	"time"

	"persistty/internal/storage"
)

type BatchTarget struct {
	TerminalID string `json:"terminal_id"`
	ViewerID   string `json:"viewer_id"`
	Generation uint64 `json:"generation"`
}

type BatchMember struct {
	TerminalID  string `json:"terminal_id"`
	DisplayName string `json:"display_name"`
}

type BatchResult struct {
	TerminalID string `json:"terminal_id"`
	State      string `json:"state"`
}

type BatchSnapshot struct {
	RequestID string        `json:"request_id"`
	Deadline  time.Time     `json:"deadline"`
	Members   []BatchMember `json:"members"`
	State     string        `json:"state"`
	Results   []BatchResult `json:"results"`
}

type BatchError struct {
	TerminalID string
	Cause      error
}

func (e *BatchError) Error() string { return "terminal batch rejected" }
func (e *BatchError) Unwrap() error { return e.Cause }

type batchParticipant struct {
	hub        *hub
	viewer     *Viewer
	generation uint64
	terminal   storage.Terminal
}

func (v *Viewer) protocol() int {
	if v.protocolVersion == 3 {
		return 3
	}
	return 2
}

func (p *pendingTermination) wire(protocol int) any {
	if protocol == 3 {
		return struct {
			ID       string        `json:"request_id"`
			Deadline time.Time     `json:"deadline"`
			Members  []BatchMember `json:"members"`
		}{p.ID, p.Deadline, p.Members}
	}
	return struct {
		ID       string    `json:"request_id"`
		Deadline time.Time `json:"deadline"`
	}{p.ID, p.Deadline}
}

func (p *pendingTermination) snapshot() BatchSnapshot {
	return BatchSnapshot{p.ID, p.Deadline, append([]BatchMember{}, p.Members...), p.State, append([]BatchResult{}, p.Results...)}
}

func (r *Runtime) Batch(id string) (BatchSnapshot, error) {
	r.controlMu.Lock()
	defer r.controlMu.Unlock()
	r.pruneBatchesLocked()
	p := r.batches[id]
	if p == nil {
		return BatchSnapshot{}, storage.ErrNotFound
	}
	return p.snapshot(), nil
}

func (r *Runtime) TerminateBatch(ctx context.Context, token string, targets []BatchTarget) (BatchSnapshot, error) {
	if len(targets) < 1 || len(targets) > 200 {
		return BatchSnapshot{}, ErrInvalidRequest
	}
	seen := make(map[string]bool)
	participants := make([]batchParticipant, 0, len(targets))
	r.mu.Lock()
	for _, target := range targets {
		if target.TerminalID == "" || target.ViewerID == "" || target.Generation < 1 || target.Generation > 1<<53-1 || seen[target.TerminalID] {
			r.mu.Unlock()
			return BatchSnapshot{}, ErrInvalidRequest
		}
		seen[target.TerminalID] = true
		h := r.hubs[target.TerminalID]
		if h == nil {
			r.mu.Unlock()
			return BatchSnapshot{}, &BatchError{target.TerminalID, ErrControlDenied}
		}
		h.mu.Lock()
		v := h.viewers[target.ViewerID]
		item := h.terminal
		h.mu.Unlock()
		if v == nil {
			r.mu.Unlock()
			return BatchSnapshot{}, &BatchError{target.TerminalID, ErrControlDenied}
		}
		participants = append(participants, batchParticipant{h, v, target.Generation, item})
	}
	r.mu.Unlock()
	return r.beginBatch(ctx, token, participants)
}

func (v *Viewer) Terminate(ctx context.Context, generation uint64) error {
	v.hub.mu.Lock()
	item := v.hub.terminal
	v.hub.mu.Unlock()
	_, err := v.hub.runtime.beginBatch(ctx, v.Token, []batchParticipant{{v.hub, v, generation, item}})
	return err
}

func (r *Runtime) beginBatch(ctx context.Context, token string, participants []batchParticipant) (BatchSnapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := r.validate(ctx, token); err != nil {
		return BatchSnapshot{}, err
	}
	id, err := randomID()
	if err != nil {
		return BatchSnapshot{}, err
	}
	sort.Slice(participants, func(i, j int) bool { return participants[i].terminal.ID < participants[j].terminal.ID })
	for _, member := range participants {
		exists, err := r.tmux.SessionExists(ctx, member.terminal.TmuxSessionName)
		if err != nil {
			return BatchSnapshot{}, &BatchError{member.terminal.ID, ErrUnavailable}
		}
		if !exists {
			return BatchSnapshot{}, &BatchError{member.terminal.ID, storage.ErrConflict}
		}
	}
	r.controlMu.Lock()
	defer r.controlMu.Unlock()
	r.pruneBatchesLocked()
	if len(r.batches) >= 256 {
		return BatchSnapshot{}, ErrUnavailable
	}
	lockParticipants(participants)
	defer unlockParticipants(participants)
	for _, member := range participants {
		h, v := member.hub, member.viewer
		if v.Token != token || h.closed || v.ctx.Err() != nil {
			return BatchSnapshot{}, &BatchError{h.terminal.ID, ErrControlDenied}
		}
		if err := h.controlLocked(v, member.generation); err != nil {
			return BatchSnapshot{}, &BatchError{h.terminal.ID, err}
		}
		if h.pending != nil {
			return BatchSnapshot{}, &BatchError{h.terminal.ID, ErrTerminationPending}
		}
	}
	if err := ctx.Err(); err != nil {
		return BatchSnapshot{}, err
	}
	p := &pendingTermination{ID: id, Deadline: time.Now().UTC().Add(r.termination), State: "pending", Participants: participants, Members: make([]BatchMember, 0, len(participants)), Results: []BatchResult{}}
	for _, member := range participants {
		p.Members = append(p.Members, BatchMember{member.hub.terminal.ID, member.hub.terminal.DisplayName})
		member.hub.pending = p
	}
	r.batches[id] = p
	p.Timer = time.AfterFunc(r.termination, func() { r.executeBatch(p) })
	for _, member := range participants {
		h := member.hub
		for _, viewer := range h.viewers {
			event := map[string]any{"type": "termination_pending", "request_id": p.ID, "deadline": p.Deadline}
			if viewer.protocol() == 3 {
				event["members"] = p.Members
			}
			h.eventLocked(viewer, event)
		}
	}
	return p.snapshot(), nil
}

func lockParticipants(members []batchParticipant) {
	for _, member := range members {
		member.hub.mu.Lock()
	}
}
func unlockParticipants(members []batchParticipant) {
	for i := len(members) - 1; i >= 0; i-- {
		members[i].hub.mu.Unlock()
	}
}

func (v *Viewer) CancelTermination(ctx context.Context, requestID string) error {
	if err := v.authorized(ctx); err != nil {
		return err
	}
	r, h := v.hub.runtime, v.hub
	r.controlMu.Lock()
	defer r.controlMu.Unlock()
	h.mu.Lock()
	p := h.pending
	valid := h.viewers[v.ID] == v && v.ctx.Err() == nil && p != nil && p.ID == requestID && time.Now().Before(p.Deadline)
	h.mu.Unlock()
	if !valid || p.State != "pending" {
		return ErrTerminationMissing
	}
	r.cancelBatchLocked(p)
	return nil
}

// The coordinator lock is held, but no hub lock may be held on entry.
func (r *Runtime) cancelBatchLocked(p *pendingTermination) {
	if p == nil || p.State != "pending" {
		return
	}
	p.State = "cancelled"
	p.Finished = time.Now().UTC()
	p.Timer.Stop()
	lockParticipants(p.Participants)
	defer unlockParticipants(p.Participants)
	for _, member := range p.Participants {
		h := member.hub
		if h.pending == p {
			h.pending = nil
			h.broadcastLocked(map[string]any{"type": "termination_cancelled", "request_id": p.ID})
		}
	}
}

func (h *hub) executeTermination(p *pendingTermination) { h.runtime.executeBatch(p) }

func (r *Runtime) executeBatch(p *pendingTermination) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	authErr := r.validate(ctx, p.Participants[0].viewer.Token)
	r.controlMu.Lock()
	if p.State != "pending" {
		r.controlMu.Unlock()
		return
	}
	lockParticipants(p.Participants)
	valid := authErr == nil
	for _, member := range p.Participants {
		h, v := member.hub, member.viewer
		if h.closed || v.ctx.Err() != nil || h.pending != p || h.controlLocked(v, member.generation) != nil {
			valid = false
		}
	}
	unlockParticipants(p.Participants)
	if !valid {
		r.cancelBatchLocked(p)
		r.controlMu.Unlock()
		return
	}
	if time.Now().Before(p.Deadline) {
		p.Timer.Stop()
		p.Timer = time.AfterFunc(time.Until(p.Deadline), func() { r.executeBatch(p) })
		r.controlMu.Unlock()
		return
	}
	p.State = "executing"
	r.controlMu.Unlock()
	results := make([]BatchResult, len(p.Participants))
	jobs := make(chan int, len(results))
	for i := range results {
		jobs <- i
	}
	close(jobs)
	var workers sync.WaitGroup
	for range min(4, len(results)) {
		workers.Go(func() {
			for i := range jobs {
				member := p.Participants[i]
				state := "unavailable"
				if ctx.Err() == nil {
					_ = r.tmux.Kill(ctx, member.terminal.TmuxSessionName)
					exists, err := r.tmux.SessionExists(ctx, member.terminal.TmuxSessionName)
					if err == nil {
						state = "running"
						if !exists {
							state = "terminated"
						}
					}
				}
				results[i] = BatchResult{member.terminal.ID, state}
			}
		})
	}
	workers.Wait()
	r.controlMu.Lock()
	defer r.controlMu.Unlock()
	lockParticipants(p.Participants)
	defer unlockParticipants(p.Participants)
	p.Results = results
	p.State = "completed"
	p.Finished = time.Now().UTC()
	for i, member := range p.Participants {
		h := member.hub
		if h.pending == p {
			h.pending = nil
		}
		for _, viewer := range h.viewers {
			event := map[string]any{"type": "termination_executed", "request_id": p.ID, "state": results[i].State}
			if viewer.protocol() == 3 {
				event["results"] = results
			}
			h.eventLocked(viewer, event)
		}
	}
}

func (r *Runtime) pruneBatchesLocked() {
	for id, p := range r.batches {
		if !p.Finished.IsZero() && time.Since(p.Finished) > 5*time.Minute {
			delete(r.batches, id)
		}
	}
	if len(r.batches) < 256 {
		return
	}
	var oldest *pendingTermination
	for _, p := range r.batches {
		if !p.Finished.IsZero() && (oldest == nil || p.Finished.Before(oldest.Finished)) {
			oldest = p
		}
	}
	if oldest != nil {
		delete(r.batches, oldest.ID)
	}
}
