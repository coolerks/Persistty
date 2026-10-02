package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testHash = "$argon2id$v=19$m=65536,t=3,p=1$MDEyMzQ1Njc4OWFiY2RlZg$MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY"

func configText(path string) string {
	return "server:\n  mode: development\n  public_origin: http://127.0.0.1:5173\nauth:\n  password_hash: '" + testHash + "'\nstorage:\n  path: " + path + "\n"
}
func TestConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := configText(filepath.Join(dir, "data", "db.sqlite"))
	for _, tt := range []struct {
		name, content string
		mode          os.FileMode
		valid         bool
	}{
		{"valid", content, 0600, true}, {"unknown", content + "unexpected: true\n", 0600, false}, {"duplicate", strings.Replace(content, "mode: development", "mode: development\n  mode: tls", 1), 0600, false}, {"permissions", content, 0644, false}, {"multidoc", content + "---\n{}\n", 0600, false}, {"missingmode", strings.Replace(content, "mode: development", "", 1), 0600, false}, {"badhash", strings.Replace(content, testHash, "bad", 1), 0600, false}, {"badttl", strings.Replace(content, "password_hash:", "session_ttl: -1h\n  password_hash:", 1), 0600, false}, {"originpath", strings.Replace(content, "http://127.0.0.1:5173", "http://127.0.0.1:5173/", 1), 0600, false},
		{"terminal-default", content + "terminal:\n  history_lines: 5000\n  restore_lines: 120\n  termination_seconds: 30\n", 0600, true},
		{"terminal-invalid-limit", content + "terminal:\n  history_lines: 99\n", 0600, false},
		{"terminal-invalid-countdown", content + "terminal:\n  termination_seconds: 121\n", 0600, false},
		{"terminal-relative-socket", content + "terminal:\n  socket_path: tmux.sock\n", 0600, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(tt.content), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, tt.mode); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%v err=%v", tt.valid, err)
			}
		})
	}
}
func TestModes(t *testing.T) {
	c := Config{}
	c.Server.Listen = "127.0.0.1:8080"
	c.Auth.PasswordHash = testHash
	c.Auth.SessionTTL = "168h"
	c.Storage.Path = "/tmp/persistty/test.sqlite"
	c.Terminal.TmuxBinary = "/usr/bin/tmux"
	c.Terminal.Shell = "/bin/sh"
	c.Terminal.HistoryLines = 5000
	c.Terminal.RestoreLines = 5000
	c.Terminal.HistoryBytes = 8 << 20
	c.Terminal.TerminationSeconds = 10
	for _, tt := range []struct {
		mode, origin, listen string
		valid                bool
	}{
		{"tls", "https://ide.example.test", "127.0.0.1:8080", true}, {"tls", "http://ide.example.test", "127.0.0.1:8080", false}, {"tls", "https://ide.example.test", "0.0.0.0:8080", false}, {"vpn_http", "http://10.7.0.1:8080", "127.0.0.1:8080", true}, {"vpn_http", "http://10.7.0.1:8080", "10.7.0.1:8080", false}, {"vpn_http", "http://10.7.0.1:8080", "0.0.0.0:8080", false}, {"development", "http://evil.test", "127.0.0.1:8080", false}, {"unknown", "https://ide.example.test", "127.0.0.1:8080", false},
	} {
		c.Server.Mode = tt.mode
		c.Server.PublicOrigin = tt.origin
		c.Server.Listen = tt.listen
		if err := c.Validate(); (err == nil) != tt.valid {
			t.Errorf("%+v: %v", tt, err)
		}
	}
	c.Server.Mode = "tls"
	c.Server.PublicOrigin = "https://ide.example.test"
	c.Server.Listen = "127.0.0.1:8080"
	c.Server.TrustedProxies = []string{"10.0.0.1"}
	if c.Validate() == nil {
		t.Fatal("trusted nonloopback")
	}
}

func TestElevationConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	base := configText(filepath.Join(dir, "data/db.sqlite"))
	for _, tt := range []struct {
		suffix string
		valid  bool
	}{{"", true}, {"elevation:\n  enabled: true\n  socket_path: /run/persistty-elevation/broker.sock\n", true}, {"elevation:\n  enabled: true\n", false}, {"elevation:\n  enabled: true\n  socket_path: relative.sock\n", false}, {"elevation:\n  enabled: true\n  socket_path: /run/../unsafe.sock\n", false}, {"elevation:\n  enabled: true\n  target: /etc/shadow\n", false}} {
		if err := os.WriteFile(path, []byte(base+tt.suffix), 0600); err != nil {
			t.Fatal(err)
		}
		c, err := Load(path)
		if (err == nil) != tt.valid {
			t.Fatal(tt, err)
		}
		if tt.suffix == "" && c.Elevation.Enabled {
			t.Fatal("enabled by default")
		}
	}
}
