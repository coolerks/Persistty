package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"persistty/internal/terminal"
)

type terminalCommand struct {
	Type       string `json:"type"`
	Generation uint64 `json:"generation"`
	Cols       int    `json:"cols"`
	Rows       int    `json:"rows"`
	RequestID  string `json:"request_id"`
	Kind       string `json:"kind"`
}

func (a *api) terminalStream(c *gin.Context) {
	if !a.origin(c) {
		a.fail(c, 403, "forbidden", "请求来源无效。")
		return
	}
	if !validTerminalID(c.Param("id")) {
		a.fail(c, 400, "invalid_request", "终端 ID 无效。")
		return
	}
	protocol := 2
	if values, exists := c.Request.URL.Query()["protocol"]; exists {
		if len(values) != 1 || (values[0] != "2" && values[0] != "3") {
			a.fail(c, 400, "invalid_request", "终端协议版本无效。")
			return
		}
		if values[0] == "3" {
			protocol = 3
		}
	}
	item, err := a.terminals.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		a.error(c, err)
		return
	}
	if item.State != "running" {
		a.fail(c, 409, "not_running", "终端已结束。")
		return
	}
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		OriginPatterns: []string{a.cfg.Server.PublicOrigin}, CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(65544)
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	viewer, err := a.runtime.ConnectProtocol(ctx, item, c.MustGet("token").(string), 80, 24, protocol)
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, "terminal_unavailable")
		return
	}
	defer viewer.Close()
	if !writeTerminalFrame(ctx, conn, terminal.Frame{Type: websocket.MessageText, Data: viewer.Ready()}) {
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-viewer.Done():
				_ = conn.Close(websocket.StatusTryAgainLater, "viewer_unavailable")
				cancel()
				return
			case frame := <-viewer.Frames():
				if !writeTerminalFrame(ctx, conn, frame) {
					cancel()
					return
				}
			}
		}
	}()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := a.auth.Lookup(ctx, viewer.Token); err != nil {
					_ = conn.Close(websocket.StatusPolicyViolation, "unauthenticated")
					cancel()
					return
				}
			}
		}
	}()
	for {
		kind, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if kind == websocket.MessageBinary {
			err = viewer.Input(ctx, data)
		} else {
			if len(data) > 4096 {
				_ = conn.Close(websocket.StatusMessageTooBig, "control_too_large")
				return
			}
			var command terminalCommand
			command, err = decodeTerminalCommand(data, protocol)
			if err != nil {
				_ = conn.Close(websocket.StatusPolicyViolation, "invalid_control")
				return
			}
			switch command.Type {
			case "takeover":
				err = viewer.Takeover(ctx, command.Generation)
			case "resize":
				err = viewer.Resize(ctx, command.Generation, command.Cols, command.Rows)
			case "terminate":
				err = viewer.Terminate(ctx, command.Generation)
			case "cancel_termination":
				err = viewer.CancelTermination(ctx, command.RequestID)
			case "device_attributes":
				err = viewer.DeviceAttributes(ctx, command.Kind)
			}
		}
		if err == nil {
			continue
		}
		switch {
		case errors.Is(err, terminal.ErrStaleGeneration):
			viewer.ReportError("stale_generation", "控制权已变化。")
		case errors.Is(err, terminal.ErrControlDenied):
			viewer.ReportError("control_denied", "当前端没有控制权。")
		case errors.Is(err, terminal.ErrTerminationPending):
			viewer.ReportError("termination_pending", "终止倒计时已开始。")
		case errors.Is(err, terminal.ErrTerminationMissing):
			viewer.ReportError("termination_missing", "终止倒计时已结束或不存在。")
		case errors.Is(err, terminal.ErrInvalidRequest):
			_ = conn.Close(websocket.StatusPolicyViolation, "invalid_input")
			return
		default:
			_ = conn.Close(websocket.StatusPolicyViolation, "unauthorized_or_unavailable")
			return
		}
	}
}

func writeTerminalFrame(ctx context.Context, conn *websocket.Conn, frame terminal.Frame) bool {
	writeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return conn.Write(writeCtx, frame.Type, frame.Data) == nil
}

func decodeTerminalCommand(data []byte, protocol int) (terminalCommand, error) {
	var command terminalCommand
	reader := json.NewDecoder(bytes.NewReader(data))
	if err := checkValue(reader, 0); err != nil {
		return command, err
	}
	if _, err := reader.Token(); err != io.EOF {
		return command, terminal.ErrInvalidRequest
	}
	if err := json.Unmarshal(data, &command); err != nil {
		return command, err
	}
	allowed := map[string]bool{"type": true}
	switch command.Type {
	case "takeover", "terminate":
		allowed["generation"] = true
	case "resize":
		allowed["generation"], allowed["cols"], allowed["rows"] = true, true, true
	case "cancel_termination":
		allowed["request_id"] = true
	case "device_attributes":
		if protocol != 3 || (command.Kind != "primary" && command.Kind != "secondary") {
			return command, terminal.ErrInvalidRequest
		}
		allowed["kind"] = true
	default:
		return command, terminal.ErrInvalidRequest
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		return command, terminal.ErrInvalidRequest
	}
	for field := range fields {
		if !allowed[field] {
			return command, terminal.ErrInvalidRequest
		}
	}
	if command.Type == "cancel_termination" && command.RequestID == "" {
		return command, terminal.ErrInvalidRequest
	}
	return command, nil
}
