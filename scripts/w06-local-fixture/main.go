package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"persistty/internal/auth"
	"persistty/internal/config"
	"persistty/internal/httpapi"
	"persistty/internal/storage"
	"syscall"
	"time"
)

func main() {
	root, err := os.MkdirTemp("/private/tmp", "persistty-w06-browser-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	repo := filepath.Join(root, "repo")
	os.Mkdir(repo, 0700)
	os.WriteFile(filepath.Join(repo, "sample.txt"), []byte("😀hit\nsecond hit\n"), 0600)
	os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("ignored.txt\n"), 0600)
	os.WriteFile(filepath.Join(repo, "ignored.txt"), []byte("hit ignored\n"), 0600)
	for _, args := range [][]string{{"init", "--quiet"}, {"add", "--", "sample.txt", ".gitignore"}, {"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--quiet", "-m", "initial"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if b, err := cmd.CombinedOutput(); err != nil {
			panic(string(b))
		}
	}
	hash, err := auth.HashPassword([]byte("local-w06-fixture"))
	if err != nil {
		panic(err)
	}
	configPath := filepath.Join(root, "config.yaml")
	os.WriteFile(configPath, []byte(fmt.Sprintf("server:\n  listen: 127.0.0.1:8089\n  public_origin: http://127.0.0.1:5179\n  mode: development\nauth:\n  password_hash: '%s'\nstorage:\n  path: %s\n", hash, filepath.Join(root, "data/db.sqlite"))), 0600)
	cfg, err := config.Load(configPath)
	if err != nil {
		panic(err)
	}
	store, err := storage.Open(context.Background(), cfg.Storage.Path, hash)
	if err != nil {
		panic(err)
	}
	defer store.Close()
	p, err := store.CreateProject(context.Background(), "W06 本地开发检查", []string{repo}, 0)
	if err != nil {
		panic(err)
	}
	unborn := filepath.Join(root, "unborn")
	if err := os.Mkdir(unborn, 0700); err != nil {
		panic(err)
	}
	cmd := exec.Command("git", "init", "--quiet", "--initial-branch=main")
	cmd.Dir = unborn
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(unborn, "new.txt"), []byte("尚无提交\n"), 0600); err != nil {
		panic(err)
	}
	if _, err := store.CreateProject(context.Background(), "W06 新仓库验收", []string{unborn}, 0); err != nil {
		panic(err)
	}
	plain := []string{filepath.Join(root, "plain-one"), filepath.Join(root, "plain-two")}
	for _, path := range plain {
		if err := os.Mkdir(path, 0700); err != nil {
			panic(err)
		}
	}
	if _, err := store.CreateProject(context.Background(), "W06 无仓库验收", plain, 0); err != nil {
		panic(err)
	}
	router, err := httpapi.New(cfg, store, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		panic(err)
	}
	server := &http.Server{Addr: cfg.Server.Listen, Handler: router}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		closing, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(closing)
	}()
	fmt.Printf("W06 fixture project=%s root=%s\n", p.ID, repo)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
