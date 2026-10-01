package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShellAccountAndFallback(t *testing.T) {
	for _, tc := range []struct{ output, want string }{
		{"fixture:x:123:123::/nonexistent:/bin/zsh\n", "/bin/zsh"},
		{"fixture:x:456:123::/nonexistent:/bin/zsh", ""},
		{"fixture:x:123:123::/nonexistent:/bin/zsh\nother:x:1:1::/:/bin/sh", ""},
		{"invalid", ""},
	} {
		if got := passwdShell(tc.output, "123"); got != tc.want {
			t.Fatalf("account shell %q", got)
		}
	}
	dir := t.TempDir()
	executable := filepath.Join(dir, "login-shell")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if got := usableShell(executable); got != executable {
		t.Fatal(got)
	}
	if err := os.Chmod(executable, 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"", "bin/zsh", "/missing/login-shell", dir, executable} {
		if got := usableShell(path); got != "/bin/sh" {
			t.Fatal(got)
		}
	}
}

func TestLoadShellDefaultAndExplicitOverride(t *testing.T) {
	t.Setenv("SHELL", "/environment-must-not-select-shell")
	dir := t.TempDir()
	file := filepath.Join(dir, "config.yaml")
	content := configText(filepath.Join(dir, "data/db.sqlite"))
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(file)
	t.Logf("未显式配置的 shell: %s", cfg.Terminal.Shell)
	if err != nil || cfg.Terminal.Shell != defaultShell() {
		t.Fatalf("default: %q %v", cfg.Terminal.Shell, err)
	}
	if err := os.WriteFile(file, []byte(content+"terminal:\n  shell: /bin/sh\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load(file)
	if err != nil || cfg.Terminal.Shell != "/bin/sh" {
		t.Fatalf("explicit: %q %v", cfg.Terminal.Shell, err)
	}
}
