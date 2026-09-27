package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func testProbe(t *testing.T) (*probe, string, string, context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	token := strings.Repeat("a", 64)
	p := &probe{ctx: ctx, token: token, start: func() (*child, error) {
		return startPTY(exec.Command("sh", "-c", "stty raw -echo; printf PTY_READY; exec cat"))
	}}
	srv := httptest.NewServer(p)
	t.Cleanup(func() { cancel(); srv.Close(); p.workers.Wait() })
	testctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(stop)
	return p, strings.TrimPrefix(srv.URL, "http://"), token, testctx
}
func TestTokenFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")
	for _, tc := range []struct {
		name, value string
		mode        os.FileMode
		ok          bool
	}{{"valid", strings.Repeat("a", 64) + "\n", 0600, true}, {"short", "secret", 0600, false}, {"invalid", strings.Repeat("z", 64), 0600, false}, {"public", strings.Repeat("a", 64), 0644, false}, {"extra", strings.Repeat("a", 64) + "\n\n", 0600, false}} {
		t.Run(tc.name, func(t *testing.T) {
			if e := os.WriteFile(path, []byte(tc.value), 0600); e != nil {
				t.Fatal(e)
			}
			if e := os.Chmod(path, tc.mode); e != nil {
				t.Fatal(e)
			}
			_, e := readToken(path)
			if (e == nil) != tc.ok {
				t.Fatal("token validation mismatch")
			}
		})
	}
	link := filepath.Join(dir, "link")
	if e := os.Symlink(path, link); e != nil {
		t.Fatal(e)
	}
	if _, e := readToken(link); e == nil {
		t.Fatal("accepted symlink")
	}
}
func TestConfigRejectsFlags(t *testing.T) {
	for _, cfg := range []bridgeConfig{{"tmux", "/tmp/socket", "persistty_probe_123456abcdef"}, {"/usr/bin/tmux", "socket", "persistty_probe_123456abcdef"}, {"/usr/bin/tmux", "/tmp/socket", "-d"}, {"/usr/bin/tmux", "/tmp/socket", "persistty_probe_123456abcdef;exit"}} {
		if cfg.validate() == nil {
			t.Fatal("unsafe config accepted")
		}
	}
}

func TestControlStrictFields(t *testing.T) {
	for _, b := range []string{`{"type":"resize","cols":2,"cols":3,"rows":4}`, `{"type":"resize","cols":2,"r\u006fws":4,"rows":4}`, `{"Type":"resize","cols":2,"rows":4}`, `{"type":"resize","cols":2,"rows":4} true`, `{"type":"resize","cols":2}`, `null`, `[]`, `{"type":"resize","cols":1001,"rows":4}`} {
		if _, e := decodeControl([]byte(b)); e == nil {
			t.Fatal("invalid control accepted")
		}
	}
	if _, e := decodeControl([]byte(`{"type":"resize","cols":1,"rows":1000}`)); e != nil {
		t.Fatal(e)
	}
	for _, s := range []string{"Copyright (c) 2011 Keith Rarick", "Copyright (c) 2025 Coder", "Copyright 2009 The Go Authors."} {
		if !strings.Contains(notices, s) {
			t.Fatal("missing embedded notice")
		}
	}
}
func TestAuthOriginAndSingleAttach(t *testing.T) {
	p, base, token, ctx := testProbe(t)
	for _, tc := range []struct {
		auth, orig string
		status     int
	}{{"", origin, 401}, {token, "", 403}, {token, "http://127.0.0.1.evil", 403}, {token, "https://127.0.0.1", 403}} {
		c, status, e := dial(ctx, base, tc.auth, tc.orig)
		if c != nil {
			c.CloseNow()
		}
		if e == nil || status != tc.status {
			t.Fatalf("status %d want %d", status, tc.status)
		}
	}
	c, e := connect(ctx, base, token)
	if e != nil {
		t.Fatal(e)
	}
	defer c.CloseNow()
	c2, status, e := dial(ctx, base, token, origin)
	if c2 != nil {
		c2.CloseNow()
	}
	if e == nil || status != 409 {
		t.Fatal("second attach not rejected")
	}
	c.CloseNow()
	s, e := waitIdle(ctx, base, token)
	if e != nil || s.Started != 1 || s.Reaped != 1 || p.active.Load() {
		t.Fatal("attach was not reaped")
	}
}
func TestBinaryPreservesInvalidUTF8AndZero(t *testing.T) {
	_, base, token, ctx := testProbe(t)
	c, e := connect(ctx, base, token)
	if e != nil {
		t.Fatal(e)
	}
	defer c.CloseNow()
	var ready []byte
	for !bytes.Contains(ready, []byte("PTY_READY")) {
		_, b, err := c.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		ready = append(ready, b...)
	}
	payload := []byte{0xff, 0, 0xe9, 0x9b, 0xaa, '\n'}
	if e = c.Write(ctx, websocket.MessageBinary, payload[:4]); e != nil {
		t.Fatal(e)
	}
	if e = c.Write(ctx, websocket.MessageBinary, payload[4:]); e != nil {
		t.Fatal(e)
	}
	var out []byte
	for !bytes.Contains(out, payload) {
		kind, b, e := c.Read(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if kind != websocket.MessageBinary {
			t.Fatal("not binary")
		}
		out = append(out, b...)
		if len(out) > 4096 {
			t.Fatal("binary mismatch")
		}
	}
}
func TestResizeAndLimits(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		binary     bool
		code       websocket.StatusCode
	}{{"zero", `{"type":"resize","cols":0,"rows":1}`, false, 1008}, {"unknown", `{"type":"resize","cols":1,"rows":1,"extra":true}`, false, 1008}, {"trailing", `{"type":"resize","cols":1,"rows":1}{}`, false, 1008}, {"fraction", `{"type":"resize","cols":1.5,"rows":1}`, false, 1008}, {"control_limit", strings.Repeat("x", 4097), false, 1009}, {"input_limit", strings.Repeat("x", inputLimit+1), true, 1009}} {
		t.Run(tc.name, func(t *testing.T) {
			_, base, token, ctx := testProbe(t)
			c, e := connect(ctx, base, token)
			if e != nil {
				t.Fatal(e)
			}
			kind := websocket.MessageText
			if tc.binary {
				kind = websocket.MessageBinary
			}
			if e = c.Write(ctx, kind, []byte(tc.body)); e != nil {
				t.Fatal(e)
			}
			if e = expectClose(ctx, c, tc.code); e != nil {
				t.Fatal(e)
			}
			if _, e = waitIdle(ctx, base, token); e != nil {
				t.Fatal(e)
			}
		})
	}
}
func TestValidResize(t *testing.T) {
	_, base, token, ctx := testProbe(t)
	c, e := connect(ctx, base, token)
	if e != nil {
		t.Fatal(e)
	}
	defer c.CloseNow()
	if e = c.Write(ctx, websocket.MessageText, []byte(`{"type":"resize","cols":83,"rows":24}`)); e != nil {
		t.Fatal(e)
	}
	var kind websocket.MessageType
	var b []byte
	for {
		kind, b, e = c.Read(ctx)
		if e != nil || kind == websocket.MessageText {
			break
		}
	}
	var v control
	if e != nil || kind != websocket.MessageText || json.Unmarshal(b, &v) != nil || v.Type != "resized" || v.Cols != 83 || v.Rows != 24 {
		t.Fatal("resize ack mismatch")
	}
}
func TestRepeatedDisconnectReapsChildren(t *testing.T) {
	p, base, token, ctx := testProbe(t)
	for i := 0; i < 15; i++ {
		c, e := connect(ctx, base, token)
		if e != nil {
			t.Fatal(e)
		}
		c.CloseNow()
		s, e := waitIdle(ctx, base, token)
		if e != nil || s.Reaped != int64(i+1) {
			t.Fatal("child leaked")
		}
	}
	if p.started.Load() != 15 || p.active.Load() {
		t.Fatal("unexpected cleanup")
	}
}
func TestServiceCancellation(t *testing.T) {
	p, base, token, ctx := testProbe(t)
	cancelctx, cancel := context.WithCancel(p.ctx)
	p.ctx = cancelctx
	c, e := connect(ctx, base, token)
	if e != nil {
		t.Fatal(e)
	}
	defer c.CloseNow()
	cancel()
	for {
		if _, _, e = c.Read(ctx); e != nil {
			break
		}
	}
	if _, e = waitIdle(ctx, base, token); e != nil {
		t.Fatal(e)
	}
}

func TestStatsRequiresAuthentication(t *testing.T) {
	_, base, token, ctx := testProbe(t)
	for _, auth := range []string{"", "Bearer wrong", "Bearer " + token} {
		r, e := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+base+"/stats", nil)
		if e != nil {
			t.Fatal(e)
		}
		r.Header.Set("Authorization", auth)
		response, e := http.DefaultClient.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		body, e := io.ReadAll(response.Body)
		response.Body.Close()
		if e != nil {
			t.Fatal(e)
		}
		if auth == "Bearer "+token {
			var s stats
			if response.StatusCode != 200 || json.Unmarshal(body, &s) != nil || s.PID == 0 {
				t.Fatal("authenticated stats unavailable")
			}
		} else if response.StatusCode != 401 || bytes.Contains(body, []byte("pid")) {
			t.Fatal("unauthenticated stats leaked")
		}
	}
}

func TestPTYDescriptorDoesNotSurviveExec(t *testing.T) {
	ch, e := startPTY(exec.Command("sh", "-c", "exec cat"))
	if e != nil {
		t.Fatal(e)
	}
	defer func() { ch.file.Close(); ch.cmd.Process.Kill(); ch.cmd.Wait() }()
	flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, ch.file.Fd(), syscall.F_GETFD, 0)
	if errno != 0 || flags&syscall.FD_CLOEXEC == 0 {
		t.Fatal("PTY descriptor can leak through exec")
	}
}

func TestSlowClientDisconnectDoesNotLeakAttach(t *testing.T) {
	p, base, token, ctx := testProbe(t)
	p.start = func() (*child, error) {
		return startPTY(exec.Command("sh", "-c", "exec yes"))
	}
	c, e := connect(ctx, base, token)
	if e != nil {
		t.Fatal(e)
	}
	defer c.CloseNow()
	// 暂不读取，令无限输出填满有界队列；不以该 trace 代替浏览器负载验收。
	time.Sleep(300 * time.Millisecond)
	if e = expectClose(ctx, c, websocket.StatusTryAgainLater); e != nil {
		t.Fatal(e)
	}
	s, e := waitIdle(ctx, base, token)
	if e != nil || s.Started != 1 || s.Reaped != 1 {
		t.Fatal("slow client attach leaked")
	}
}
