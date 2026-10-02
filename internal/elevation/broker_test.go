package elevation

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// The test binary stands in for a fixed helper. It runs with only the three
// actual pipes; it cannot elevate and never installs a production bypass.
func TestProcessFixture(t *testing.T) {
	if len(os.Args) < 2 {
		return
	}
	mode := os.Args[len(os.Args)-1]
	if !strings.HasPrefix(mode, "w07-") {
		return
	}
	if mode == "w07-denied" {
		_, _ = os.Stderr.WriteString("synthetic-secret-do-not-return")
		os.Exit(1)
	}
	password, _ := io.ReadAll(os.Stdin)
	if string(password) != "synthetic-system-password\n" {
		os.Exit(2)
	}
	control, body := os.NewFile(3, "control"), os.NewFile(4, "body")
	var g Grant
	if ReadFrame(control, &g) != nil {
		os.Exit(3)
	}
	data, _ := io.ReadAll(body)
	if Digest(data) != g.ContentHash {
		os.Exit(4)
	}
	for _, value := range append(os.Args, os.Environ()...) {
		if strings.Contains(value, "synthetic-system-password") || strings.Contains(value, "synthetic-content-sentinel") {
			os.Exit(5)
		}
	}
	if mode == "w07-wait" {
		_, _ = io.Copy(io.Discard, control)
		os.Exit(0)
	}
	if WriteFrame(os.Stdout, reply{Phase: "ready"}) != nil {
		os.Exit(6)
	}
	var decision struct {
		Phase string `json:"phase"`
	}
	if ReadFrame(control, &decision) != nil || decision.Phase != "commit" {
		os.Exit(0)
	}
	r := outcome(g.ID, "rejected", "synthetic_done")
	_ = WriteFrame(os.Stdout, reply{Phase: "result", Result: &r})
	os.Exit(0)
}
func TestCommandSeparatePipesAndCancel(t *testing.T) {
	for _, mode := range []string{"w07-ready", "w07-denied", "w07-wait", "w07-abort"} {
		t.Run(mode, func(t *testing.T) {
			timeout := 5 * time.Second
			if mode == "w07-wait" {
				timeout = time.Second
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			content := bytes.Repeat([]byte("synthetic-content-sentinel"), 10000)
			g := Grant{ID: strings.Repeat("a", 64), ContentHash: Digest(content)}
			actual := mode
			if mode == "w07-abort" {
				actual = "w07-ready"
			}
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProcessFixture$", "--", actual)
			commits := 0
			r, err := runCommand(ctx, g, []byte("synthetic-system-password"), content, func(commit func() (Result, error)) (Result, error) {
				if mode == "w07-abort" {
					return outcome(g.ID, "cancelled", "cancelled"), nil
				}
				commits++
				return commit()
			}, cmd)
			switch mode {
			case "w07-ready":
				if err != nil || r.State != "rejected" || commits != 1 || *r.Code != "synthetic_done" {
					t.Fatal(r, err, commits)
				}
			case "w07-denied":
				if err != nil || r.State != "rejected" || *r.Code != "authorization_failed" {
					t.Fatal(r, err)
				}
			case "w07-abort":
				if err != nil || r.State != "cancelled" || commits != 0 {
					t.Fatal(r, err)
				}
			case "w07-wait":
				if err == nil {
					t.Fatal("timeout succeeded")
				}
			}
			if cmd.ProcessState == nil || !cmd.ProcessState.Exited() && mode != "w07-wait" && mode != "w07-denied" {
				t.Fatal("child not reaped")
			}
		})
	}
}
