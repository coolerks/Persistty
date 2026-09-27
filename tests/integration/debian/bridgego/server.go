package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/coder/websocket"
	"github.com/creack/pty"
)

const origin = "http://127.0.0.1"
const inputLimit = 64 * 1024

type bridgeConfig struct{ tmux, socket, session string }
type child struct {
	file *os.File
	cmd  *exec.Cmd
}
type probe struct {
	ctx     context.Context
	token   string
	start   func() (*child, error)
	active  atomic.Bool
	started atomic.Int64
	reaped  atomic.Int64
	workers sync.WaitGroup
}
type stats struct {
	Active     bool  `json:"active"`
	Started    int64 `json:"started"`
	Reaped     int64 `json:"reaped"`
	FDCount    int   `json:"fd_count"`
	Goroutines int   `json:"goroutines"`
	PID        int   `json:"pid"`
}
type control struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

func decodeControl(b []byte) (control, error) {
	var v control
	d := json.NewDecoder(bytes.NewReader(b))
	t, e := d.Token()
	if e != nil || t != json.Delim('{') {
		return v, errors.New("invalid_control")
	}
	seen := map[string]bool{}
	for d.More() {
		k, e := d.Token()
		key, ok := k.(string)
		if e != nil || !ok || seen[key] || (key != "type" && key != "cols" && key != "rows") {
			return v, errors.New("invalid_control")
		}
		seen[key] = true
		var raw json.RawMessage
		if d.Decode(&raw) != nil {
			return v, errors.New("invalid_control")
		}
	}
	if _, e = d.Token(); e != nil {
		return v, errors.New("invalid_control")
	}
	var extra any
	if d.Decode(&extra) != io.EOF || len(seen) != 3 {
		return v, errors.New("invalid_control")
	}
	if json.Unmarshal(b, &v) != nil || v.Type != "resize" || v.Cols < 1 || v.Cols > 1000 || v.Rows < 1 || v.Rows > 1000 {
		return v, errors.New("invalid_control")
	}
	return v, nil
}

func readToken(path string) (string, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", errors.New("token_file_unavailable")
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || st.Sys().(*syscall.Stat_t).Uid != uint32(os.Getuid()) {
		return "", errors.New("token_file_invalid")
	}
	b, err := io.ReadAll(io.LimitReader(f, 66))
	if err != nil {
		return "", errors.New("token_file_invalid")
	}
	b = bytes.TrimSuffix(b, []byte("\n"))
	if len(b) != 64 || !regexp.MustCompile(`^[a-f0-9]{64}$`).Match(b) {
		return "", errors.New("token_file_invalid")
	}
	return string(b), nil
}

func (cfg bridgeConfig) validate() error {
	if !filepath.IsAbs(cfg.tmux) || !filepath.IsAbs(cfg.socket) || !regexp.MustCompile(`^persistty_probe_[a-f0-9]{12}$`).MatchString(cfg.session) {
		return errors.New("invalid_bridge_config")
	}
	return nil
}
func (cfg bridgeConfig) start() (*child, error) {
	// -N 禁止隐式启动 server；这里只创建可回收的 attach client。
	c := exec.Command(cfg.tmux, "-N", "-S", cfg.socket, "attach-session", "-t", "="+cfg.session)
	c.Env = append(os.Environ(), "TERM=xterm-256color")
	return startPTY(c)
}

func startPTY(c *exec.Cmd) (*child, error) {
	f, err := pty.StartWithSize(c, &pty.Winsize{Cols: 100, Rows: 30})
	if err != nil {
		return nil, err
	}
	// 重新包装非阻塞 fd，让 Close 能打断 PTY Read/Write。
	fd, err := syscall.Dup(int(f.Fd()))
	if err == nil {
		syscall.CloseOnExec(fd)
		err = syscall.SetNonblock(fd, true)
	}
	_ = f.Close()
	if err != nil {
		if fd >= 0 {
			_ = syscall.Close(fd)
		}
		_ = c.Process.Kill()
		_ = c.Wait()
		return nil, err
	}
	return &child{file: os.NewFile(uintptr(fd), "probe-pty"), cmd: c}, nil
}
func (p *probe) authorized(r *http.Request) bool {
	return subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+p.token)) == 1
}
func (p *probe) snapshot() stats {
	fd := -1
	if files, err := os.ReadDir("/proc/self/fd"); err == nil {
		fd = len(files)
	}
	return stats{p.active.Load(), p.started.Load(), p.reaped.Load(), fd, runtime.NumGoroutine(), os.Getpid()}
}
func (p *probe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !p.authorized(r) {
		http.Error(w, "unauthenticated", 401)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "invalid_method", 405)
		return
	}
	if r.URL.Path == "/stats" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p.snapshot())
		return
	}
	if r.URL.Path != "/attach" || r.URL.RawQuery != "" {
		http.NotFound(w, r)
		return
	}
	if r.Header.Get("Origin") != origin {
		http.Error(w, "forbidden", 403)
		return
	}
	if !p.active.CompareAndSwap(false, true) {
		http.Error(w, "terminal_attached", 409)
		return
	}
	defer p.active.Store(false)
	p.workers.Add(1)
	defer p.workers.Done()
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{CompressionMode: websocket.CompressionDisabled, OriginPatterns: []string{"127.0.0.1"}})
	if err != nil {
		return
	}
	defer c.CloseNow()
	c.SetReadLimit(inputLimit)
	ctx, cancel := context.WithCancel(p.ctx)
	defer cancel()
	ch, err := p.start()
	if err != nil {
		_ = c.Close(websocket.StatusInternalError, "attach_unavailable")
		return
	}
	p.started.Add(1)
	waited := make(chan struct{})
	go func() { _ = ch.cmd.Wait(); p.reaped.Add(1); close(waited); cancel() }()
	var pumps sync.WaitGroup
	defer func() {
		cancel()
		_ = c.CloseNow()
		_ = ch.file.Close()
		_ = ch.cmd.Process.Kill()
		<-waited
		pumps.Wait()
	}()
	write := func(kind websocket.MessageType, b []byte) error {
		wc, stop := context.WithTimeout(ctx, 2*time.Second)
		defer stop()
		return c.Write(wc, kind, b)
	}
	if write(websocket.MessageText, []byte(`{"type":"ready"}`)) != nil {
		return
	}
	queue := make(chan []byte, 32)
	acks := make(chan []byte, 8)
	fault := make(chan websocket.StatusCode, 1)
	fail := func(code websocket.StatusCode) {
		select {
		case fault <- code:
		default:
		}
	}
	pumps.Add(2)
	go func() {
		defer pumps.Done()
		buf := make([]byte, 1024)
		for {
			n, e := ch.file.Read(buf)
			if n > 0 {
				b := append([]byte(nil), buf[:n]...)
				select {
				case queue <- b:
				case <-ctx.Done():
					return
				default:
					fail(websocket.StatusTryAgainLater)
					return
				}
			}
			if e != nil {
				cancel()
				return
			}
		}
	}()
	go func() {
		defer pumps.Done()
		for {
			kind, b, e := c.Read(ctx)
			if e != nil {
				cancel()
				return
			}
			if kind == websocket.MessageBinary {
				_ = ch.file.SetWriteDeadline(time.Now().Add(2 * time.Second))
				if _, e = ch.file.Write(b); e != nil {
					cancel()
					return
				}
				continue
			}
			if len(b) > 4096 {
				fail(websocket.StatusMessageTooBig)
				return
			}
			v, e := decodeControl(b)
			if e != nil {
				fail(websocket.StatusPolicyViolation)
				return
			}
			if pty.Setsize(ch.file, &pty.Winsize{Cols: uint16(v.Cols), Rows: uint16(v.Rows)}) != nil {
				cancel()
				return
			}
			ack, _ := json.Marshal(control{Type: "resized", Cols: v.Cols, Rows: v.Rows})
			select {
			case acks <- ack:
			case <-ctx.Done():
				return
			default:
				fail(websocket.StatusTryAgainLater)
				return
			}
		}
	}()
	for {
		select {
		case b := <-queue:
			if write(websocket.MessageBinary, b) != nil {
				return
			}
		case b := <-acks:
			if write(websocket.MessageText, b) != nil {
				return
			}
		case code := <-fault:
			_ = c.Close(code, "probe_limit")
			return
		case <-ctx.Done():
			return
		}
	}
}
