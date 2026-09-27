package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestRecordRejectsLimitsBeforeDial(t *testing.T) {
	for _, tc := range []struct {
		duration   time.Duration
		cols, rows int
	}{{0, 100, 30}, {11 * time.Second, 100, 30}, {time.Second, 0, 30}, {time.Second, 100, 1001}} {
		if recordClient(context.Background(), 1, "", tc.duration, tc.cols, tc.rows) == nil {
			t.Fatal("accepted invalid record limits")
		}
	}
}

func TestRecordOutputBounds(t *testing.T) {
	for _, tc := range []struct {
		name         string
		frames, size int
	}{{"bytes", 257, 1024}, {"frames", 4097, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			done := make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer close(done)
				c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
				if err != nil {
					return
				}
				defer c.CloseNow()
				ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
				defer cancel()
				if c.Write(ctx, websocket.MessageText, []byte(`{"type":"ready"}`)) != nil {
					return
				}
				if _, _, err = c.Read(ctx); err != nil {
					return
				}
				for i := 0; i < tc.frames; i++ {
					if c.Write(ctx, websocket.MessageBinary, make([]byte, tc.size)) != nil {
						return
					}
				}
				_, _, _ = c.Read(ctx)
			}))
			defer srv.Close()
			port, _ := strconv.Atoi(strings.Split(srv.URL, ":")[2])
			err := recordClient(context.Background(), port, "synthetic", 3*time.Second, 80, 24)
			if err == nil || err.Error() != "record_output_limit" {
				t.Fatalf("limit result: %v", err)
			}
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("record connection was not closed")
			}
		})
	}
}

func TestRecordCancellationReapsAttach(t *testing.T) {
	_, base, token, testctx := testProbe(t)
	port, _ := strconv.Atoi(strings.Split(base, ":")[1])
	ctx, cancel := context.WithCancel(testctx)
	done := make(chan error, 1)
	go func() { done <- recordClient(ctx, port, token, time.Second, 80, 24) }()
	for {
		s, err := fetchStats(testctx, base, token)
		if err != nil {
			t.Fatal(err)
		}
		if s.Active {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("record ended before attach: %v", err)
		case <-testctx.Done():
			t.Fatal("record attach timeout")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	if err := <-done; err == nil {
		t.Fatal("canceled record succeeded")
	}
	if s, err := waitIdle(testctx, base, token); err != nil || s.Started != s.Reaped {
		t.Fatalf("canceled attach not reaped: %v", err)
	}
}

func TestRecordSyntheticOutputReapsAttach(t *testing.T) {
	_, base, token, ctx := testProbe(t)
	port, _ := strconv.Atoi(strings.Split(base, ":")[1])
	old := os.Stdout
	r, w, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	os.Stdout = w
	defer func() { os.Stdout = old; r.Close(); w.Close() }()
	e = recordClient(ctx, port, token, 200*time.Millisecond, 80, 24)
	w.Close()
	os.Stdout = old
	raw, _ := io.ReadAll(r)
	if e != nil {
		t.Fatal(e)
	}
	var got struct {
		Frames    []string
		Bytes     int
		ResizeAck bool `json:"resize_ack"`
		Stats     stats
	}
	if json.Unmarshal(raw, &got) != nil || got.Bytes == 0 || len(got.Frames) == 0 || !got.ResizeAck || got.Stats.Active || got.Stats.Started != got.Stats.Reaped {
		t.Fatal("record contract failed")
	}
}
