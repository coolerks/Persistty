package terminal

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/coder/websocket"
	"persistty/internal/storage"
)

var ErrControlDenied = errors.New("terminal control denied")
var ErrStaleGeneration = errors.New("stale terminal generation")
var ErrTerminationPending = errors.New("terminal termination already pending")
var ErrTerminationMissing = errors.New("terminal termination request missing")

type Frame struct {
	Type websocket.MessageType
	Data []byte
}

type Runtime struct {
	tmux        *Tmux
	validate    func(context.Context, string) error
	termination time.Duration
	mu          sync.Mutex
	hubs        map[string]*hub
	controlMu   sync.Mutex
	batches     map[string]*pendingTermination
}

type hub struct {
	runtime    *Runtime
	terminal   storage.Terminal
	owner      *attachClient
	mu         sync.Mutex
	viewers    map[string]*Viewer
	controller string
	generation uint64
	cols       int
	rows       int
	pending    *pendingTermination
	closed     bool
}

type Viewer struct {
	ID              string
	Token           string
	attach          *attachClient
	hub             *hub
	queue           chan Frame
	ctx             context.Context
	cancel          context.CancelFunc
	once            sync.Once
	protocolVersion int
}

type pendingTermination struct {
	ID           string             `json:"request_id"`
	Deadline     time.Time          `json:"deadline"`
	Members      []BatchMember      `json:"members"`
	State        string             `json:"-"`
	Results      []BatchResult      `json:"-"`
	Participants []batchParticipant `json:"-"`
	Timer        *time.Timer        `json:"-"`
	Finished     time.Time          `json:"-"`
}

func NewRuntime(tmux *Tmux, termination time.Duration, validate func(context.Context, string) error) *Runtime {
	if termination <= 0 {
		termination = 10 * time.Second
	}
	return &Runtime{tmux: tmux, validate: validate, termination: termination, hubs: make(map[string]*hub), batches: make(map[string]*pendingTermination)}
}

func (r *Runtime) Connect(ctx context.Context, terminal storage.Terminal, token string, cols, rows int) (*Viewer, error) {
	return r.ConnectProtocol(ctx, terminal, token, cols, rows, 2)
}

func (r *Runtime) ConnectProtocol(ctx context.Context, terminal storage.Terminal, token string, cols, rows, protocol int) (*Viewer, error) {
	if protocol != 2 && protocol != 3 {
		return nil, ErrInvalidRequest
	}
	if !validSize(cols, rows) {
		return nil, ErrInvalidRequest
	}
	viewerID, err := randomID()
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	h := r.hubs[terminal.ID]
	closed := false
	if h != nil {
		h.mu.Lock()
		closed = h.closed
		h.mu.Unlock()
	}
	if h == nil || closed {
		owner, attachErr := r.tmux.Attach(terminal.TmuxSessionName, false, cols, rows)
		if attachErr != nil {
			r.mu.Unlock()
			return nil, ErrUnavailable
		}
		h = &hub{runtime: r, terminal: terminal, owner: owner, viewers: make(map[string]*Viewer), generation: 1, cols: cols, rows: rows}
		r.hubs[terminal.ID] = h
		go h.drainOwner()
	}
	attach, err := r.tmux.Attach(terminal.TmuxSessionName, true, cols, rows)
	if err != nil {
		h.mu.Lock()
		if len(h.viewers) == 0 {
			delete(r.hubs, terminal.ID)
			h.closed = true
			h.owner.Close()
		}
		h.mu.Unlock()
		r.mu.Unlock()
		return nil, ErrUnavailable
	}
	vctx, cancel := context.WithCancel(ctx)
	v := &Viewer{ID: viewerID, Token: token, attach: attach, hub: h,
		queue: make(chan Frame, 32), ctx: vctx, cancel: cancel, protocolVersion: protocol}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		r.mu.Unlock()
		attach.Close()
		return nil, ErrUnavailable
	}
	h.viewers[v.ID] = v
	if h.controller == "" {
		h.controller = v.ID
		h.generation++
	}
	h.mu.Unlock()
	r.mu.Unlock()
	go v.readOutput()
	return v, nil
}

func randomID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func (h *hub) drainOwner() {
	_, _ = io.Copy(io.Discard, h.owner.file)
	h.runtime.controlMu.Lock()
	defer h.runtime.controlMu.Unlock()
	h.mu.Lock()
	h.closed = true
	pending := h.pending
	for _, viewer := range h.viewers {
		viewer.cancel()
	}
	h.mu.Unlock()
	h.runtime.cancelBatchLocked(pending)
}

func (v *Viewer) readOutput() {
	buf := make([]byte, 32<<10)
	for {
		n, err := v.attach.file.Read(buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			v.hub.mu.Lock()
			v.hub.enqueueLocked(v, Frame{Type: websocket.MessageBinary, Data: data})
			v.hub.mu.Unlock()
		}
		if err != nil {
			v.cancel()
			return
		}
		select {
		case <-v.ctx.Done():
			return
		default:
		}
	}
}

func (h *hub) enqueueLocked(v *Viewer, frame Frame) {
	select {
	case v.queue <- frame:
	default:
		v.cancel()
	}
}

func (h *hub) eventLocked(v *Viewer, event any) {
	data, _ := json.Marshal(event)
	h.enqueueLocked(v, Frame{Type: websocket.MessageText, Data: data})
}

func (h *hub) broadcastLocked(event any) {
	for _, viewer := range h.viewers {
		h.eventLocked(viewer, event)
	}
}

func (v *Viewer) Ready() []byte {
	h := v.hub
	h.mu.Lock()
	defer h.mu.Unlock()
	role := "observer"
	if h.controller == v.ID {
		role = "controller"
	}
	var pending any
	if h.pending != nil {
		pending = h.pending.wire(v.protocol())
	}
	data, _ := json.Marshal(struct {
		Type               string `json:"type"`
		Protocol           int    `json:"protocol"`
		TerminalID         string `json:"terminal_id"`
		ViewerID           string `json:"viewer_id"`
		Role               string `json:"role"`
		Generation         uint64 `json:"generation"`
		Cols               int    `json:"cols"`
		Rows               int    `json:"rows"`
		PendingTermination any    `json:"pending_termination"`
	}{"ready", v.protocol(), h.terminal.ID, v.ID, role, h.generation, h.cols, h.rows, pending})
	return data
}

func (v *Viewer) Frames() <-chan Frame  { return v.queue }
func (v *Viewer) Done() <-chan struct{} { return v.ctx.Done() }

func (r *Runtime) UpdateMetadata(item storage.Terminal) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h := r.hubs[item.ID]; h != nil {
		h.mu.Lock()
		h.terminal = item
		for _, viewer := range h.viewers {
			if viewer.protocol() == 3 {
				h.eventLocked(viewer, map[string]any{"type": "metadata", "terminal_id": item.ID, "display_name": item.DisplayName})
			}
		}
		h.mu.Unlock()
	}
}

func (v *Viewer) ReportError(code, message string) {
	v.hub.mu.Lock()
	defer v.hub.mu.Unlock()
	v.hub.eventLocked(v, map[string]any{"type": "error", "code": code, "message": message})
}

func (v *Viewer) authorized(ctx context.Context) error {
	return v.hub.runtime.validate(ctx, v.Token)
}

func (h *hub) controlLocked(v *Viewer, generation uint64) error {
	if h.viewers[v.ID] != v || h.controller != v.ID {
		return ErrControlDenied
	}
	if generation != h.generation {
		return ErrStaleGeneration
	}
	return nil
}

func (v *Viewer) Input(ctx context.Context, payload []byte) error {
	if len(payload) < 9 || len(payload) > 65544 {
		return ErrInvalidRequest
	}
	if err := v.authorized(ctx); err != nil {
		return err
	}
	h := v.hub
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.controlLocked(v, binary.BigEndian.Uint64(payload[:8])); err != nil {
		return err
	}
	if err := h.owner.file.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	n, err := h.owner.file.Write(payload[8:])
	if err == nil && n != len(payload)-8 {
		return io.ErrShortWrite
	}
	return err
}

func (v *Viewer) Resize(ctx context.Context, generation uint64, cols, rows int) error {
	if !validSize(cols, rows) {
		return ErrInvalidRequest
	}
	if err := v.authorized(ctx); err != nil {
		return err
	}
	h := v.hub
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.controlLocked(v, generation); err != nil {
		return err
	}
	if err := h.owner.Resize(cols, rows); err != nil {
		return err
	}
	h.cols, h.rows = cols, rows
	h.eventLocked(v, map[string]any{"type": "resized", "cols": cols, "rows": rows})
	return nil
}

func (v *Viewer) Takeover(ctx context.Context, generation uint64) error {
	if err := v.authorized(ctx); err != nil {
		return err
	}
	h := v.hub
	r := h.runtime
	r.controlMu.Lock()
	defer r.controlMu.Unlock()
	h.mu.Lock()
	if h.closed || v.ctx.Err() != nil || h.viewers[v.ID] != v {
		h.mu.Unlock()
		return ErrControlDenied
	}
	if h.generation != generation {
		h.mu.Unlock()
		return ErrStaleGeneration
	}
	if h.controller == v.ID {
		h.mu.Unlock()
		return nil
	}
	pending := h.pending
	h.mu.Unlock()
	r.cancelBatchLocked(pending)
	h.mu.Lock()
	h.generation++
	h.controller = v.ID
	h.broadcastControlLocked()
	h.mu.Unlock()
	return nil
}

func (h *hub) broadcastControlLocked() {
	for _, viewer := range h.viewers {
		role := "observer"
		if h.controller == viewer.ID {
			role = "controller"
		}
		h.eventLocked(viewer, map[string]any{"type": "control", "role": role, "generation": h.generation})
	}
}

func (v *Viewer) Close() {
	v.once.Do(func() {
		v.cancel()
		h := v.hub
		r := h.runtime
		r.controlMu.Lock()
		h.mu.Lock()
		pending := h.pending
		terminalID := h.terminal.ID
		initiator := false
		if pending != nil {
			for _, member := range pending.Participants {
				if member.viewer == v {
					initiator = true
				}
			}
		}
		delete(h.viewers, v.ID)
		if h.controller == v.ID {
			h.controller = ""
			h.generation++
			h.broadcastControlLocked()
		}
		empty := len(h.viewers) == 0
		if empty {
			h.closed = true
		}
		h.mu.Unlock()
		if initiator || empty {
			r.cancelBatchLocked(pending)
		}
		r.controlMu.Unlock()
		v.attach.Close()
		if empty {
			h.owner.Close()
			r.mu.Lock()
			if r.hubs[terminalID] == h {
				delete(r.hubs, terminalID)
			}
			r.mu.Unlock()
		}
	})
}
