package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"
)

func dial(ctx context.Context, base, token, orig string) (*websocket.Conn, int, error) {
	h := http.Header{}
	if token != "" {
		h.Set("Authorization", "Bearer "+token)
	}
	h.Set("Origin", orig)
	c, r, e := websocket.Dial(ctx, "ws://"+base+"/attach", &websocket.DialOptions{HTTPHeader: h, CompressionMode: websocket.CompressionDisabled})
	status := 0
	if r != nil {
		status = r.StatusCode
		if e != nil && r.Body != nil {
			_ = r.Body.Close()
		}
	}
	return c, status, e
}
func connect(ctx context.Context, base, token string) (*websocket.Conn, error) {
	c, _, e := dial(ctx, base, token, origin)
	if e != nil {
		return nil, errors.New("attach_failed")
	}
	kind, b, e := c.Read(ctx)
	if e != nil || kind != websocket.MessageText || string(b) != `{"type":"ready"}` {
		_ = c.CloseNow()
		return nil, errors.New("ready_failed")
	}
	return c, nil
}
func fetchStats(ctx context.Context, base, token string) (stats, error) {
	var s stats
	r, e := http.NewRequestWithContext(ctx, "GET", "http://"+base+"/stats", nil)
	if e != nil {
		return s, e
	}
	r.Header.Set("Authorization", "Bearer "+token)
	resp, e := http.DefaultClient.Do(r)
	if e != nil {
		return s, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return s, errors.New("stats_failed")
	}
	e = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&s)
	return s, e
}
func waitIdle(ctx context.Context, base, token string) (stats, error) {
	for {
		s, e := fetchStats(ctx, base, token)
		if e != nil {
			return s, e
		}
		if !s.Active && s.Started == s.Reaped {
			return s, nil
		}
		select {
		case <-ctx.Done():
			return s, ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
}
func expectClose(ctx context.Context, c *websocket.Conn, code websocket.StatusCode) error {
	defer c.CloseNow()
	for {
		_, _, e := c.Read(ctx)
		if e != nil {
			if websocket.CloseStatus(e) != code {
				return errors.New("unexpected_close")
			}
			return nil
		}
	}
}
func runClient(parent context.Context, port int, token, mode, ready string, duration time.Duration) error {
	base, e := portBase(port)
	if e != nil {
		return e
	}
	if duration <= 0 || duration > time.Minute {
		return errors.New("invalid_duration")
	}
	ctx, cancel := context.WithTimeout(parent, duration)
	defer cancel()
	checks := map[string]bool{}
	if mode == "hold" || mode == "observe" {
		c, e := connect(ctx, base, token)
		if e != nil {
			return e
		}
		defer c.CloseNow()
		if mode == "hold" {
			if e = writeReady(ready, map[string]any{"pid": os.Getpid(), "attached": true}); e != nil {
				return e
			}
			for {
				_, _, e = c.Read(ctx)
				if e != nil {
					return nil
				}
			}
		}
		for {
			kind, b, e := c.Read(ctx)
			if e != nil {
				return errors.New("observe_failed")
			}
			if kind == websocket.MessageBinary && len(b) > 0 {
				checks["binary_output"] = true
				break
			}
		}
		_ = c.CloseNow()
		s, e := waitIdle(ctx, base, token)
		if e != nil {
			return e
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"checks": checks, "stats": s})
	}
	if mode != "exercise" {
		return errors.New("invalid_client_mode")
	}
	baseline, e := waitIdle(ctx, base, token)
	if e != nil {
		return e
	}
	for _, test := range []struct {
		name, auth, orig string
		status           int
	}{{"no_auth", "", origin, 401}, {"wrong_origin", token, "http://invalid.example", 403}} {
		c, status, e := dial(ctx, base, test.auth, test.orig)
		if c != nil {
			_ = c.CloseNow()
		}
		if e == nil || status != test.status {
			return errors.New("negative_handshake_failed")
		}
		checks[test.name] = true
	}
	c, e := connect(ctx, base, token)
	if e != nil {
		return e
	}
	defer c.CloseNow()
	c2, status, e := dial(ctx, base, token, origin)
	if c2 != nil {
		_ = c2.CloseNow()
	}
	if e == nil || status != 409 {
		return errors.New("busy_failed")
	}
	checks["single_attach_409"] = true
	if e = c.Write(ctx, websocket.MessageText, []byte(`{"type":"resize","cols":91,"rows":27}`)); e != nil {
		return e
	}
	for {
		kind, b, e := c.Read(ctx)
		if e != nil {
			return e
		}
		if kind == websocket.MessageText {
			var v control
			if json.Unmarshal(b, &v) != nil || v.Type != "resized" || v.Cols != 91 || v.Rows != 27 {
				return errors.New("resize_failed")
			}
			break
		}
	}
	checks["valid_resize"] = true
	// 两个消息在 UTF-8 字符内部切开；服务不得逐帧转为字符串。
	payload := []byte("BRIDGE_INPUT_雪\n")
	split := len(payload) - 3
	if e = c.Write(ctx, websocket.MessageBinary, payload[:split]); e != nil {
		return e
	}
	if e = c.Write(ctx, websocket.MessageBinary, payload[split:]); e != nil {
		return e
	}
	buf := make([]byte, 0, 4096)
	for !bytes.Contains(buf, []byte("BRIDGE_ACK_雪")) {
		kind, b, e := c.Read(ctx)
		if e != nil {
			return e
		}
		if kind == websocket.MessageBinary {
			buf = append(buf, b...)
			if len(buf) > 256*1024 {
				return errors.New("output_limit")
			}
		}
	}
	checks["split_unicode_binary_roundtrip"] = true
	if e = c.Write(ctx, websocket.MessageText, []byte(`{"type":"resize","cols":0,"rows":27}`)); e != nil {
		return e
	}
	if e = expectClose(ctx, c, websocket.StatusPolicyViolation); e != nil {
		return e
	}
	checks["invalid_resize_1008"] = true
	if _, e = waitIdle(ctx, base, token); e != nil {
		return e
	}
	c, e = connect(ctx, base, token)
	if e != nil {
		return e
	}
	if e = c.Write(ctx, websocket.MessageBinary, make([]byte, inputLimit+1)); e != nil {
		_ = c.CloseNow()
		return e
	}
	if e = expectClose(ctx, c, websocket.StatusMessageTooBig); e != nil {
		return e
	}
	checks["oversize_1009"] = true
	if _, e = waitIdle(ctx, base, token); e != nil {
		return e
	}
	for i := 0; i < 5; i++ {
		c, e = connect(ctx, base, token)
		if e != nil {
			return e
		}
		_ = c.CloseNow()
		if _, e = waitIdle(ctx, base, token); e != nil {
			return e
		}
	}
	checks["detach_reconnect_reaped"] = true
	s, e := waitIdle(ctx, base, token)
	if e != nil {
		return e
	}
	if s.FDCount >= 0 && baseline.FDCount >= 0 && s.FDCount > baseline.FDCount+2 {
		return errors.New("fd_growth")
	}
	if s.Goroutines > baseline.Goroutines+4 {
		return errors.New("goroutine_growth")
	}
	checks["resources_bounded"] = true
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"checks": checks, "stats": s, "baseline": baseline})
}
