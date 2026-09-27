package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

//go:embed third-party-notices.txt
var notices string

func main() {
	if run(os.Args[1:]) != nil {
		fmt.Fprintln(os.Stderr, "bridge_probe_failed")
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		return errors.New("missing_mode")
	}
	if len(args) == 1 && args[0] == "licenses" {
		_, e := fmt.Fprint(os.Stdout, notices)
		return e
	}
	f := flag.NewFlagSet("probe", flag.ContinueOnError)
	f.SetOutput(ioDiscard{})
	tokenPath := f.String("token-file", "", "")
	readyPath := f.String("ready-file", "", "")
	tmux := f.String("tmux-bin", "", "")
	socket := f.String("socket", "", "")
	session := f.String("session", "", "")
	port := f.Int("port", 0, "")
	mode := f.String("mode", "exercise", "")
	duration := f.Duration("duration", 30*time.Second, "")
	cols := f.Int("cols", 100, "")
	rows := f.Int("rows", 30, "")
	if err := f.Parse(args[1:]); err != nil || f.NArg() != 0 {
		return errors.New("invalid_flags")
	}
	token, err := readToken(*tokenPath)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if args[0] == "client" {
		if *mode == "record" {
			return recordClient(ctx, *port, token, *duration, *cols, *rows)
		}
		return runClient(ctx, *port, token, *mode, *readyPath, *duration)
	}
	if args[0] != "serve" {
		return errors.New("invalid_mode")
	}
	cfg := bridgeConfig{*tmux, *socket, *session}
	if err = cfg.validate(); err != nil {
		return err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	p := &probe{ctx: ctx, token: token, start: cfg.start}
	srv := &http.Server{Handler: p, ReadHeaderTimeout: 3 * time.Second, IdleTimeout: 5 * time.Second, MaxHeaderBytes: 8192}
	if err = writeReady(*readyPath, map[string]any{"port": listener.Addr().(*net.TCPAddr).Port, "pid": os.Getpid()}); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(listener) }()
	select {
	case <-ctx.Done():
		sc, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sc)
		p.workers.Wait()
		return nil
	case err = <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return errors.New("serve_failed")
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(b []byte) (int, error) { return len(b), nil }
func writeReady(path string, v any) error {
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if e != nil {
		return errors.New("ready_file_failed")
	}
	e = json.NewEncoder(f).Encode(v)
	ce := f.Close()
	if e != nil || ce != nil {
		return errors.New("ready_file_failed")
	}
	return nil
}
func portBase(port int) (string, error) {
	if port < 1 || port > 65535 {
		return "", errors.New("invalid_port")
	}
	return "127.0.0.1:" + strconv.Itoa(port), nil
}
