package elevation

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"persistty/internal/files"
	"persistty/internal/storage"
	"strings"
	"testing"
	"time"
)

func TestBrokerBusyPeerRejectionAndShutdown(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "w07-socket-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(dir, "s")})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{}, 2)
	b := broker{policy: Policy{CallerUID: os.Getuid(), Targets: []Target{{"example", "/srv/allowed/file.txt"}}}, checkPeer: func(*net.UnixConn, int) error { return nil }, slots: make(chan struct{}, 2), run: func(ctx context.Context, g Grant, _, _ []byte, _ Publish) (Result, error) {
		started <- struct{}{}
		<-ctx.Done()
		return Result{}, ctx.Err()
	}}
	done := make(chan error, 1)
	go func() { done <- b.serve(ctx, l) }()
	now := time.Now().UTC()
	g := Grant{ID: strings.Repeat("a", 64), SessionHash: strings.Repeat("b", 64), Root: storage.RegisteredFolder{ProjectID: "p", FolderID: "f", ProjectVersion: 1, Path: "/srv/allowed"}, Path: "file.txt", TargetID: "example", ContentHash: Digest([]byte("new")), Expected: files.Version{Identity: "1:2", Mtime: now.Format(time.RFC3339Nano), Size: 3, ETag: Digest([]byte("old"))}, IssuedAt: now, ExpiresAt: now.Add(time.Minute)}
	for i := 0; i < 2; i++ {
		conn, err := net.DialUnix("unix", nil, l.Addr().(*net.UnixAddr))
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		if WriteFrame(conn, brokerRequest{Action: "execute", Grant: g}) != nil {
			t.Fatal("write")
		}
		var response reply
		if ReadFrame(conn, &response) != nil || response.Phase != "accepted" {
			t.Fatal(response)
		}
		_ = WriteBytes(conn, []byte("synthetic"), 1024)
		_ = WriteBytes(conn, []byte("new"), MaxContent)
		<-started
	}
	busy, err := net.DialUnix("unix", nil, l.Addr().(*net.UnixAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	_ = busy.SetDeadline(time.Now().Add(time.Second))
	var response reply
	if ReadFrame(busy, &response) != nil || response.Code != "rate_limited" {
		t.Fatal(response)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("broker failed to join handlers")
	}
	if len(b.slots) != 0 {
		t.Fatal("handler leaked")
	}
	// Peer failure must terminate before decoding credential or body frames.
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(dir, "peer")})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	peer, err := net.DialUnix("unix", nil, listener.Addr().(*net.UnixAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	server, err := listener.AcceptUnix()
	if err != nil {
		t.Fatal(err)
	}
	b.checkPeer = func(*net.UnixConn, int) error { return ErrForbidden }
	go b.handle(context.Background(), server)
	_ = peer.SetDeadline(time.Now().Add(time.Second))
	if ReadFrame(peer, &response) == nil {
		t.Fatal("denied peer received protocol")
	}
}
