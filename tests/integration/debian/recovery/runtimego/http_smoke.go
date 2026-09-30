package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"persistty/internal/config"
	"persistty/internal/httpapi"
	"persistty/internal/storage"
	"persistty/internal/terminal"
)

type smokeEvent struct {
	Type       string `json:"type"`
	Role       string `json:"role"`
	Generation uint64 `json:"generation"`
	RequestID  string `json:"request_id"`
	Code       string `json:"code"`
}

func runHTTPWebSocketSmoke(ctx context.Context, root, tmuxBinary, socket, hash string,
	store *storage.Store, created storage.Terminal) (map[string]bool, error) {
	checks := map[string]bool{}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	origin := "http://" + listener.Addr().String()
	var cfg config.Config
	cfg.Server.Listen, cfg.Server.PublicOrigin, cfg.Server.Mode = listener.Addr().String(), origin, "development"
	cfg.Auth.PasswordHash, cfg.Auth.SessionTTL = hash, "1h"
	cfg.Storage.Path, cfg.Transfer.StagingPath = root+"/runtime-data/db.sqlite", root+"/http-staging"
	cfg.Terminal.TmuxBinary, cfg.Terminal.SocketPath, cfg.Terminal.Shell = tmuxBinary, socket, "/bin/sh"
	cfg.Terminal.HistoryLines, cfg.Terminal.RestoreLines = 120, 120
	cfg.Terminal.HistoryBytes, cfg.Terminal.TerminationSeconds = 1<<20, 2
	router, err := httpapi.New(cfg, store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		listener.Close()
		return nil, err
	}
	server := &http.Server{Handler: router, ReadHeaderTimeout: 5 * time.Second}
	go server.Serve(listener)
	defer server.Close()
	client := &http.Client{Timeout: 5 * time.Second}
	login, err := http.NewRequestWithContext(ctx, http.MethodPost, origin+"/api/v1/auth/login",
		strings.NewReader(`{"password":"isolated-test-password"}`))
	if err != nil {
		return nil, err
	}
	login.Header.Set("Origin", origin)
	login.Header.Set("Content-Type", "application/json")
	response, err := client.Do(login)
	if err != nil {
		return nil, err
	}
	var session struct {
		Data struct {
			CSRFToken string `json:"csrf_token"`
		} `json:"data"`
	}
	err = json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&session)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 || len(response.Cookies()) != 1 || session.Data.CSRFToken == "" {
		return nil, errors.New("login failed")
	}
	cookie := response.Cookies()[0]
	checks["http_login_cookie_csrf"] = cookie.HttpOnly && cookie.SameSite == http.SameSiteStrictMode
	historyRequest, err := http.NewRequestWithContext(ctx, http.MethodGet,
		origin+"/api/v1/terminals/"+created.ID+"/history", nil)
	if err != nil {
		return nil, err
	}
	historyRequest.AddCookie(cookie)
	response, err = client.Do(historyRequest)
	if err != nil {
		return nil, err
	}
	_, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
	response.Body.Close()
	checks["http_history_authenticated"] = readErr == nil && response.StatusCode == 200
	wsURL := "ws" + strings.TrimPrefix(origin, "http") + "/api/v1/terminals/" + created.ID + "/stream"
	header := http.Header{"Origin": []string{origin}, "Cookie": []string{cookie.String()}}
	first, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		return nil, err
	}
	defer first.CloseNow()
	firstReady, err := readSmokeEvent(ctx, first, "ready")
	if err != nil {
		return nil, err
	}
	checks["ws_first_controller"] = firstReady.Role == "controller" && firstReady.Generation > 0
	second, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		return nil, err
	}
	defer second.CloseNow()
	secondReady, err := readSmokeEvent(ctx, second, "ready")
	if err != nil {
		return nil, err
	}
	checks["ws_second_observer"] = secondReady.Role == "observer" && secondReady.Generation == firstReady.Generation
	command := []byte("printf 'W03_WS_MARKER\\n'\n")
	input := make([]byte, 8+len(command))
	binary.BigEndian.PutUint64(input[:8], firstReady.Generation)
	copy(input[8:], command)
	if err := first.Write(ctx, websocket.MessageBinary, input); err != nil {
		return nil, err
	}
	checks["ws_input_once"] = waitSmokeOutput(ctx, first, []byte("W03_WS_MARKER")) == nil
	if err := writeSmokeControl(ctx, second, "takeover", secondReady.Generation, ""); err != nil {
		return nil, err
	}
	firstControl, err := readSmokeEvent(ctx, first, "control")
	if err != nil {
		return nil, err
	}
	secondControl, err := readSmokeEvent(ctx, second, "control")
	if err != nil {
		return nil, err
	}
	checks["ws_control_transfer"] = firstControl.Role == "observer" && secondControl.Role == "controller" &&
		firstControl.Generation == secondControl.Generation && secondControl.Generation > firstReady.Generation
	if err := first.Write(ctx, websocket.MessageBinary, input); err != nil {
		return nil, err
	}
	denied, err := readSmokeEvent(ctx, first, "error")
	if err != nil {
		return nil, err
	}
	checks["ws_old_input_rejected"] = denied.Code == "control_denied" || denied.Code == "stale_generation"
	if err := writeSmokeControl(ctx, second, "terminate", secondControl.Generation, ""); err != nil {
		return nil, err
	}
	pending, err := readSmokeEvent(ctx, first, "termination_pending")
	if err != nil {
		return nil, err
	}
	checks["ws_termination_broadcast"] = pending.RequestID != ""
	if err := writeSmokeControl(ctx, first, "cancel_termination", 0, pending.RequestID); err != nil {
		return nil, err
	}
	if _, err := readSmokeEvent(ctx, second, "termination_cancelled"); err != nil {
		return nil, err
	}
	checks["ws_observer_cancels_termination"] = true
	logout, err := http.NewRequestWithContext(ctx, http.MethodPost, origin+"/api/v1/auth/logout", nil)
	if err != nil {
		return nil, err
	}
	logout.AddCookie(cookie)
	logout.Header.Set("Origin", origin)
	logout.Header.Set("X-CSRF-Token", session.Data.CSRFToken)
	response, err = client.Do(logout)
	if err != nil {
		return nil, err
	}
	response.Body.Close()
	checks["http_logout"] = response.StatusCode == 204
	closeCtx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	for {
		_, _, err = first.Read(closeCtx)
		if err != nil {
			break
		}
	}
	checks["ws_logout_revokes"] = err != nil && closeCtx.Err() == nil
	tmux := &terminal.Tmux{Binary: tmuxBinary, Socket: socket}
	exists, err := tmux.SessionExists(ctx, created.TmuxSessionName)
	checks["ws_logout_preserves_pane"] = err == nil && exists
	return checks, nil
}

func readSmokeEvent(ctx context.Context, conn *websocket.Conn, expected string) (smokeEvent, error) {
	readCtx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	for {
		kind, data, err := conn.Read(readCtx)
		if err != nil {
			return smokeEvent{}, err
		}
		if kind != websocket.MessageText {
			continue
		}
		var event smokeEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return smokeEvent{}, err
		}
		if event.Type == expected {
			return event, nil
		}
	}
}

func waitSmokeOutput(ctx context.Context, conn *websocket.Conn, marker []byte) error {
	readCtx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	var output []byte
	for len(output) < 1<<20 {
		kind, data, err := conn.Read(readCtx)
		if err != nil {
			return err
		}
		if kind == websocket.MessageBinary {
			output = append(output, data...)
			if bytes.Contains(output, marker) {
				return nil
			}
		}
	}
	return errors.New("output marker not seen")
}

func writeSmokeControl(ctx context.Context, conn *websocket.Conn, kind string, generation uint64, requestID string) error {
	command := map[string]any{"type": kind}
	if kind == "cancel_termination" {
		command["request_id"] = requestID
	} else {
		command["generation"] = generation
	}
	data, err := json.Marshal(command)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, data)
}
