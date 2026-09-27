package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/coder/websocket"
)

// recordClient is only for the fixed synthetic history experiment; never a user logger.
func recordClient(parent context.Context, port int, token string, duration time.Duration, cols, rows int) error {
	if duration <= 0 || duration > 10*time.Second || cols < 1 || cols > 1000 || rows < 1 || rows > 1000 {
		return errors.New("invalid_record_limits")
	}
	base, e := portBase(port)
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(parent, duration)
	defer cancel()
	c, e := connect(ctx, base, token)
	if e != nil {
		return e
	}
	defer c.CloseNow()
	control, _ := json.Marshal(map[string]any{"type": "resize", "cols": cols, "rows": rows})
	if e = c.Write(ctx, websocket.MessageText, control); e != nil {
		return e
	}
	frames := []string{}
	total := 0
	ack := false
	for {
		kind, b, err := c.Read(ctx)
		if err != nil {
			if ctx.Err() != context.DeadlineExceeded {
				return errors.New("record_read_failed")
			}
			break
		}
		if kind == websocket.MessageBinary {
			total += len(b)
			if total > 256*1024 || len(frames) >= 4096 {
				return errors.New("record_output_limit")
			}
			frames = append(frames, base64.StdEncoding.EncodeToString(b))
		} else if string(b) == string(mustResizeReply(cols, rows)) {
			ack = true
		}
	}
	_ = c.CloseNow()
	idle, stop := context.WithTimeout(parent, 3*time.Second)
	defer stop()
	s, e := waitIdle(idle, base, token)
	if e != nil || !ack || total == 0 {
		return errors.New("record_incomplete")
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"frames": frames, "bytes": total, "cols": cols, "rows": rows, "resize_ack": ack, "stats": s})
}

func mustResizeReply(cols, rows int) []byte {
	b, _ := json.Marshal(control{Type: "resized", Cols: cols, Rows: rows})
	return b
}
