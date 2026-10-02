package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"persistty/internal/elevation"
)

func main() {
	if len(os.Args) != 1 || os.Geteuid() == 0 || os.Getenv("LISTEN_FDS") != "1" || os.Getenv("LISTEN_PID") != strconv.Itoa(os.Getpid()) {
		os.Exit(1)
	}
	file := os.NewFile(3, "systemd-socket")
	listener, err := net.FileListener(file)
	file.Close()
	if err != nil {
		os.Exit(1)
	}
	defer listener.Close()
	unix, ok := listener.(*net.UnixListener)
	if !ok {
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	if elevation.ServeBroker(ctx, unix) != nil {
		os.Exit(1)
	}
}
