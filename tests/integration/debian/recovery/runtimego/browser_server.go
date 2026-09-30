package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"persistty/internal/auth"
	"persistty/internal/config"
	"persistty/internal/httpapi"
	"persistty/internal/storage"
)

func serveBrowser() error {
	if len(os.Args) != 6 {
		return errors.New("invalid arguments")
	}
	root, binary, socket := os.Args[2], os.Args[3], os.Args[4]
	port, err := strconv.Atoi(os.Args[5])
	if filepath.Dir(root) != "/tmp" || !strings.HasPrefix(filepath.Base(root), "persistty-browser-") ||
		!filepath.IsAbs(binary) || socket != filepath.Join(root, "tmux.sock") || err != nil || port < 1024 || port > 65535 {
		return errors.New("invalid probe paths")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()
	hashPath := filepath.Join(root, "test.hash")
	rawHash, err := os.ReadFile(hashPath)
	if errors.Is(err, os.ErrNotExist) {
		hash, hashErr := auth.HashPassword([]byte("isolated-test-password"))
		if hashErr != nil {
			return hashErr
		}
		if err = os.WriteFile(hashPath, []byte(hash), 0600); err != nil {
			return err
		}
		rawHash = []byte(hash)
	} else if err != nil {
		return err
	}
	hash := string(rawHash)
	dbPath := filepath.Join(root, "data", "db.sqlite")
	store, err := storage.Open(ctx, dbPath, hash)
	if err != nil {
		return err
	}
	defer store.Close()
	projectPath := filepath.Join(root, "project")
	if err := os.MkdirAll(projectPath, 0700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(projectPath, "welcome.txt"), []byte("W03 isolated browser probe\n"), 0600); err != nil {
		return err
	}
	projects, err := store.Projects(ctx)
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		if _, err := store.CreateProject(ctx, "W03 隔离终端", []string{projectPath}, 0); err != nil {
			return err
		}
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return err
	}
	defer listener.Close()
	var cfg config.Config
	cfg.Server.Listen, cfg.Server.PublicOrigin, cfg.Server.Mode = listener.Addr().String(), "http://127.0.0.1:5174", "development"
	cfg.Auth.PasswordHash, cfg.Auth.SessionTTL = hash, "1h"
	cfg.Storage.Path, cfg.Transfer.StagingPath = dbPath, filepath.Join(root, "staging")
	cfg.Terminal.TmuxBinary, cfg.Terminal.SocketPath, cfg.Terminal.Shell = binary, socket, "/bin/sh"
	cfg.Terminal.HistoryLines, cfg.Terminal.RestoreLines = 5000, 5000
	cfg.Terminal.HistoryBytes, cfg.Terminal.TerminationSeconds = 8<<20, 5
	router, err := httpapi.New(cfg, store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		return err
	}
	server := &http.Server{Handler: router, ReadHeaderTimeout: 5 * time.Second}
	ready, err := os.OpenFile(filepath.Join(root, "ready.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	err = json.NewEncoder(ready).Encode(map[string]int{"port": listener.Addr().(*net.TCPAddr).Port})
	if closeErr := ready.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case err = <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			return err
		}
		<-done
	}
	return nil
}
