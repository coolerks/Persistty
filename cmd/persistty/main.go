package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/term"
	"persistty/internal/auth"
	"persistty/internal/config"
	"persistty/internal/httpapi"
	"persistty/internal/storage"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "Persistty:", err)
		os.Exit(1)
	}
}
func run(args []string, in *os.File, out, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("用法: persistty serve --config <文件> 或 persistty password")
	}
	switch args[0] {
	case "password":
		if len(args) != 1 {
			return errors.New("password不接受命令行密码或参数")
		}
		if !term.IsTerminal(int(in.Fd())) {
			return errors.New("密码必须从TTY隐藏输入")
		}
		fmt.Fprint(stderr, "请输入密码（至少12字节）: ")
		password, err := term.ReadPassword(int(in.Fd()))
		fmt.Fprintln(stderr)
		if err != nil {
			return errors.New("无法读取密码")
		}
		defer clear(password)
		fmt.Fprint(stderr, "再次输入密码: ")
		confirmation, err := term.ReadPassword(int(in.Fd()))
		fmt.Fprintln(stderr)
		if err != nil {
			return errors.New("无法读取确认密码")
		}
		defer clear(confirmation)
		if subtle.ConstantTimeCompare(password, confirmation) != 1 {
			return errors.New("两次密码不一致")
		}
		hash, err := auth.HashPassword(password)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, hash)
		return err
	case "serve":
		flags := flag.NewFlagSet("serve", flag.ContinueOnError)
		flags.SetOutput(stderr)
		path := flags.String("config", "", "配置文件")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if *path == "" || flags.NArg() != 0 {
			return errors.New("serve仅接受--config <文件>")
		}
		cfg, err := config.Load(*path)
		if err != nil {
			return err
		}
		logger := slog.New(slog.NewJSONHandler(out, nil))
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		store, err := storage.Open(ctx, cfg.Storage.Path, cfg.Auth.PasswordHash)
		if err != nil {
			return errors.New("数据库初始化失败，请检查权限和迁移版本")
		}
		defer store.Close()
		router, err := httpapi.New(cfg, store, logger)
		if err != nil {
			return errors.New("HTTP初始化失败")
		}
		server := &http.Server{Addr: cfg.Server.Listen, Handler: router, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 2 * time.Minute, WriteTimeout: 5 * time.Minute, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
		done := make(chan error, 1)
		go func() { done <- server.ListenAndServe() }()
		logger.Info("服务启动", "event", "server_start")
		select {
		case err = <-done:
			if !errors.Is(err, http.ErrServerClosed) {
				return errors.New("HTTP监听失败")
			}
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err = server.Shutdown(shutdown); err != nil {
				return errors.New("HTTP停止超时")
			}
			<-done
		}
		logger.Info("服务停止", "event", "server_stop")
		return nil
	default:
		return errors.New("未知命令，仅支持serve和password")
	}
}
