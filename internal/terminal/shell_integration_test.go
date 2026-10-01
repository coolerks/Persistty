package terminal

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNewPaneUsesConfiguredShell(t *testing.T) {
	binary, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("真实 shell 测试需要 tmux")
	}
	root, err := os.MkdirTemp("", "persistty-shell-")
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(root, "tmux.sock")
	t.Setenv("HOME", root)
	t.Setenv("ZDOTDIR", root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conf := filepath.Join(root, "tmux.conf")
	if err := os.WriteFile(conf, []byte("set -g exit-empty off\nset -g exit-unattached off\n"), 0600); err != nil {
		t.Fatal(err)
	}
	started := false
	t.Cleanup(func() {
		if started {
			clean, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := exec.CommandContext(clean, binary, "-N", "-S", socket, "kill-server").Run(); err != nil {
				t.Errorf("own tmux cleanup: %v", err)
			}
		}
		if err := os.RemoveAll(root); err != nil {
			t.Error(err)
		}
	})
	if err := exec.CommandContext(ctx, binary, "-S", socket, "-f", conf, "start-server").Run(); err != nil {
		t.Fatal(err)
	}
	started = true
	for i, shell := range []string{"/bin/sh", "/bin/bash", "/bin/zsh"} {
		if _, err := os.Stat(shell); os.IsNotExist(err) {
			continue
		}
		tmux := &Tmux{Binary: binary, Socket: socket, Shell: shell, HistoryLines: 100}
		name := "persistty_shell_" + string(rune('a'+i))
		if err := tmux.Create(ctx, name, root, "fixture", "fixture", 80, 24); err != nil {
			t.Fatal(err)
		}
		for {
			output, err := tmux.Run(ctx, 1024, "display-message", "-p", "-t", "="+name+":0", "#{pane_current_command}")
			command := filepath.Base(strings.TrimSpace(string(output)))
			// macOS implements /bin/sh with bash; tmux reports its process name.
			if err == nil && (command == filepath.Base(shell) || runtime.GOOS == "darwin" && shell == "/bin/sh" && command == "bash") {
				break
			}
			if err != nil {
				t.Fatalf("shell pane probe: %s %q %v", shell, output, err)
			}
			time.Sleep(10 * time.Millisecond)
		}
		output, err := tmux.Run(ctx, 1024, "show-environment", "-t", "="+name, "SHELL")
		if err != nil || strings.TrimSpace(string(output)) != "SHELL="+shell {
			t.Fatalf("shell env: %q %v", output, err)
		}
	}
}
