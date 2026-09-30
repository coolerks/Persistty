package terminal

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

func TestDeviceAttributesStayInTheirViewerAttach(t *testing.T) {
	h, controller, observer, owner := testHub(t, time.Second)
	for _, viewer := range []*Viewer{controller, observer} {
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { reader.Close(); writer.Close() })
		viewer.attach = &attachClient{file: writer}
		viewer.protocolVersion = 3
		for _, kind := range []string{"primary", "secondary"} {
			if err := viewer.DeviceAttributes(context.Background(), kind); err != nil {
				t.Fatal(err)
			}
		}
		want := "\x1b[?1;2c\x1b[>0;276;0c"
		if err := reader.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		got := make([]byte, len(want))
		if _, err := io.ReadFull(reader, got); err != nil || string(got) != want {
			t.Fatalf("viewer response %q: %v", got, err)
		}
	}
	if h.controller != controller.ID || h.generation != 5 {
		t.Fatal("device query changed control")
	}
	// A real keyboard byte is still the first and only byte in the owner pipe.
	payload := make([]byte, 9)
	binary.BigEndian.PutUint64(payload, 5)
	payload[8] = 'x'
	if err := controller.Input(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	if err := owner.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1)
	if _, err := owner.Read(buf); err != nil || buf[0] != 'x' {
		t.Fatalf("owner got protocol data instead of keyboard input: %q %v", buf, err)
	}
	if err := owner.SetReadDeadline(time.Now()); err != nil {
		t.Fatal(err)
	}
	if n, err := owner.Read(buf); n != 0 || !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("extra owner bytes: %q %v", buf[:n], err)
	}
}

func TestDeviceAttributesRejectInvalidOrRevokedViewer(t *testing.T) {
	for _, state := range []string{"kind", "v2", "authentication", "closed", "cancelled", "removed"} {
		t.Run(state, func(t *testing.T) {
			h, viewer, _, _ := testHub(t, time.Second)
			viewer.protocolVersion = 3
			kind := "primary"
			switch state {
			case "kind":
				kind = "pwd\n"
			case "v2":
				viewer.protocolVersion = 2
			case "authentication":
				h.runtime.validate = func(context.Context, string) error { return ErrControlDenied }
			case "closed":
				h.closed = true
			case "cancelled":
				viewer.cancel()
			case "removed":
				delete(h.viewers, viewer.ID)
			}
			// No attach is assigned: any attempted PTY write would fail this test.
			if err := viewer.DeviceAttributes(context.Background(), kind); err == nil {
				t.Fatal("invalid/revoked viewer wrote a device response")
			}
		})
	}
}
