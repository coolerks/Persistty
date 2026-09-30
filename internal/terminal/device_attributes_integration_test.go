package terminal

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"persistty/internal/storage"
)

// This owns a private tmux socket and raw-input fixture; it never attaches to
// an existing server or executes any received input as shell commands.
func TestDeviceAttributesRealTmux(t *testing.T) {
	tmuxBinary, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("真实设备应答集成需要 tmux")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("真实设备应答集成需要 Python 3 fixture")
	}
	// Keep the UNIX socket below macOS's path length limit.
	root, err := os.MkdirTemp("", "persistty-da-")
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(root, "tmux.sock")
	started := false
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if started {
			if err := exec.CommandContext(ctx, tmuxBinary, "-N", "-S", socket, "kill-server").Run(); err != nil {
				t.Errorf("private tmux cleanup: %v", err)
			}
		}
		if err := os.RemoveAll(root); err != nil {
			t.Errorf("private fixture cleanup: %v", err)
		}
	})
	config := filepath.Join(root, "tmux.conf")
	script := filepath.Join(root, "pane.py")
	inputFile := filepath.Join(root, "input")
	if err := os.WriteFile(config, []byte("set -g exit-empty off\nset -g exit-unattached off\nset -g status off\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	const pane = `import os, sys, tty
tty.setraw(0)
with open(sys.argv[1], "wb", buffering=0) as output:
    os.write(1, b"fixture ready\r\n")
    while True:
        data = os.read(0, 65536)
        if not data: break
        output.write(data)
`
	if err := os.WriteFile(script, []byte(pane), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	const name = "persistty_device_fixture"
	if err := exec.CommandContext(ctx, tmuxBinary, "-S", socket, "-f", config, "new-session", "-d", "-s", name, "-x", "80", "-y", "24", python, script, inputFile).Run(); err != nil {
		t.Fatalf("private tmux fixture startup: %v", err)
	}
	started = true
	wait := func(check func() bool) {
		t.Helper()
		for !check() {
			select {
			case <-ctx.Done():
				t.Fatal("private tmux fixture timed out")
			case <-time.After(10 * time.Millisecond):
			}
		}
	}
	wait(func() bool { _, err := os.Stat(inputFile); return err == nil })
	runtime := NewRuntime(&Tmux{Binary: tmuxBinary, Socket: socket}, time.Second, func(context.Context, string) error { return nil })
	var controller *Viewer
	for range 2 {
		viewer, err := runtime.ConnectProtocol(ctx, storage.Terminal{ID: "device-fixture", TmuxSessionName: name}, "fixture-token", 80, 24, 3)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(viewer.Close)
		if controller == nil {
			controller = viewer
		}
		var output []byte
		for !bytes.Contains(output, []byte("\x1b[c")) || !bytes.Contains(output, []byte("\x1b[>c")) {
			select {
			case <-ctx.Done():
				t.Fatal("tmux attach did not emit DA1/DA2 queries")
			case frame := <-viewer.Frames():
				output = append(output, frame.Data...)
				if len(output) > 1<<20 {
					t.Fatal("tmux fixture output exceeded bound")
				}
			}
		}
		for _, kind := range []string{"primary", "secondary"} {
			if err := viewer.DeviceAttributes(ctx, kind); err != nil {
				t.Fatal(err)
			}
		}
	}
	// Controller resize must keep every output PTY and late observer on one grid.
	{
		panePID, err := exec.CommandContext(ctx, tmuxBinary, "-N", "-S", socket, "list-panes", "-t", "="+name, "-F", "#{pane_pid}").Output()
		if err != nil {
			t.Fatal(err)
		}
		checkGrid := func(cols, rows int) {
			t.Helper()
			h := controller.hub
			h.mu.Lock()
			defer h.mu.Unlock()
			clients := []*attachClient{h.owner}
			for _, viewer := range h.viewers {
				clients = append(clients, viewer.attach)
			}
			for _, client := range clients {
				y, x, err := pty.Getsize(client.file)
				if err != nil || x != cols || y != rows {
					t.Fatalf("attach grid=%dx%d, want=%dx%d: %v", x, y, cols, rows, err)
				}
			}
		}
		for index, size := range [][2]int{{140, 12}, {100, 30}, {160, 10}} {
			controller.hub.mu.Lock()
			generation := controller.hub.generation
			controller.hub.mu.Unlock()
			if err := controller.Resize(ctx, generation, size[0], size[1]); err != nil {
				t.Fatal(err)
			}
			checkGrid(size[0], size[1])
			wait(func() bool {
				out, err := exec.CommandContext(ctx, tmuxBinary, "-N", "-S", socket, "list-panes", "-t", "="+name, "-F", "#{pane_width}x#{pane_height}:#{pane_pid}").Output()
				if err != nil {
					t.Fatalf("pane grid query: %v", err)
				}
				return err == nil && strings.TrimSpace(string(out)) == fmt.Sprintf("%dx%d:%s", size[0], size[1], strings.TrimSpace(string(panePID)))
			})
			controller.hub.mu.Lock()
			viewers := make([]*Viewer, 0, len(controller.hub.viewers))
			for _, viewer := range controller.hub.viewers {
				viewers = append(viewers, viewer)
			}
			controller.hub.mu.Unlock()
			for _, viewer := range viewers {
				found := false
				for !found {
					select {
					case frame := <-viewer.queue:
						found = bytes.Contains(frame.Data, []byte(`"type":"resized"`))
					default:
						t.Fatal("viewer 未收到尺寸广播")
					}
				}
			}
			if index == 0 {
				late, err := runtime.ConnectProtocol(ctx, storage.Terminal{ID: "device-fixture", TmuxSessionName: name}, "fixture-token", 80, 24, 3)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(late.Close)
				checkGrid(size[0], size[1])
				if err := late.Resize(ctx, generation, 80, 24); err != ErrControlDenied {
					t.Fatalf("observer resize=%v", err)
				}
			}
		}
	}
	payload := make([]byte, 9)
	controller.hub.mu.Lock()
	binary.BigEndian.PutUint64(payload, controller.hub.generation)
	controller.hub.mu.Unlock()
	payload[8] = 'x'
	if err := controller.Input(ctx, payload); err != nil {
		t.Fatal(err)
	}
	wait(func() bool { data, _ := os.ReadFile(inputFile); return len(data) > 0 })
	if data, err := os.ReadFile(inputFile); err != nil || string(data) != "x" {
		t.Fatalf("pane input included device responses: %q %v", data, err)
	}
}
