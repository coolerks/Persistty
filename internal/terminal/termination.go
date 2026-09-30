package terminal

import (
	"context"
	"time"
)

func (v *Viewer) Terminate(ctx context.Context, generation uint64) error {
	if err := v.authorized(ctx); err != nil {
		return err
	}
	id, err := randomID()
	if err != nil {
		return err
	}
	h := v.hub
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.controlLocked(v, generation); err != nil {
		return err
	}
	if h.pending != nil {
		return ErrTerminationPending
	}
	pending := &pendingTermination{
		ID: id, Deadline: time.Now().UTC().Add(h.runtime.termination),
		Initiator: v.ID, Token: v.Token, Generation: h.generation,
	}
	h.pending = pending
	pending.Timer = time.AfterFunc(h.runtime.termination, func() { h.executeTermination(pending) })
	h.broadcastLocked(map[string]any{
		"type": "termination_pending", "request_id": pending.ID, "deadline": pending.Deadline,
	})
	return nil
}

func (v *Viewer) CancelTermination(ctx context.Context, requestID string) error {
	if err := v.authorized(ctx); err != nil {
		return err
	}
	h := v.hub
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.viewers[v.ID] != v || h.pending == nil || h.pending.ID != requestID ||
		!time.Now().Before(h.pending.Deadline) {
		return ErrTerminationMissing
	}
	h.cancelPendingLocked()
	return nil
}

func (h *hub) cancelPendingLocked() {
	if h.pending == nil {
		return
	}
	pending := h.pending
	h.pending = nil
	pending.Timer.Stop()
	h.broadcastLocked(map[string]any{"type": "termination_cancelled", "request_id": pending.ID})
}

func (h *hub) executeTermination(pending *pendingTermination) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	authErr := h.runtime.validate(ctx, pending.Token)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.pending != pending {
		return
	}
	if authErr != nil || h.closed || h.controller != pending.Initiator || h.generation != pending.Generation ||
		h.viewers[pending.Initiator] == nil {
		h.cancelPendingLocked()
		return
	}
	if time.Now().Before(pending.Deadline) {
		pending.Timer = time.AfterFunc(time.Until(pending.Deadline), func() { h.executeTermination(pending) })
		return
	}
	_ = h.runtime.tmux.Kill(ctx, h.terminal.TmuxSessionName)
	exists, err := h.runtime.tmux.SessionExists(ctx, h.terminal.TmuxSessionName)
	h.pending = nil
	state := "unavailable"
	if err == nil {
		state = "running"
		if !exists {
			state = "terminated"
		}
	}
	h.broadcastLocked(map[string]any{
		"type": "termination_executed", "request_id": pending.ID, "state": state,
	})
}
