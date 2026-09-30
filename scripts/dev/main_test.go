package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/coder/websocket"
	"go.yaml.in/yaml/v3"
	"persistty/internal/auth"
	"persistty/internal/config"
	"persistty/internal/storage"
)

const testHash = "$argon2id$v=19$m=65536,t=3,p=1$MDEyMzQ1Njc4OWFiY2RlZg$MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY"

func sourceConfig(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "source.yaml")
	content := "server:\n  mode: development\n  listen: 127.0.0.1:8080\n  public_origin: http://127.0.0.1:5173\nauth:\n  password_hash: '" + testHash + "'\nstorage:\n  path: " + filepath.Join(dir, "metadata.db") + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRuntimeConfigPreservesSource(t *testing.T) {
	dir := t.TempDir()
	source := sourceConfig(t, dir)
	before, _ := os.ReadFile(source)
	cfg, err := prepareConfig(options{configPath: source, tmux: "/bin/sh", shell: "/bin/bash", listen: "127.0.0.1:8081", port: 5175})
	if err != nil {
		t.Fatal(err)
	}
	generated, err := writeRuntimeConfig(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(generated)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(source)
	info, _ := os.Stat(generated)
	if !bytes.Equal(before, after) || loaded.Auth.PasswordHash != testHash || info.Mode().Perm() != 0600 ||
		loaded.Server.PublicOrigin != "http://127.0.0.1:5175" || loaded.Server.Listen != "127.0.0.1:8081" ||
		loaded.Terminal.TmuxBinary != "/bin/sh" || loaded.TerminalSocketPath() != filepath.Join(dir, "tmux.sock") {
		t.Fatal("source/secret/permissions/terminal/origin changed unexpectedly")
	}
	for _, opts := range []options{
		{configPath: source, tmux: "/bin/sh", shell: "/bin/bash", listen: "0.0.0.0:8081"},
		{configPath: source, tmux: "/bin/sh", shell: "/bin/bash", port: 65536},
		{configPath: source, tmux: "relative"},
		{configPath: source, tmux: "/bin/sh", shell: "/not/a/shell"},
	} {
		if _, err := prepareConfig(opts); err == nil {
			t.Fatal("unsafe overrides accepted")
		}
	}
}

func TestPortConflictAndEnvironment(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if checkPort(listener.Addr().String()) == nil {
		t.Fatal("occupied port accepted")
	}
	filtered := removeEnv([]string{"A=1", "PERSISTTY_DEV_API_TARGET=old", "B=2"}, "PERSISTTY_DEV_API_TARGET=")
	if strings.Join(filtered, ",") != "A=1,B=2" {
		t.Fatal("environment replacement failed")
	}
}

func TestAuthenticatedReadiness(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/session" {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer server.Close()
	targets := []readinessTarget{{server.URL + "/api/v1/auth/session", http.StatusUnauthorized}, {server.URL, http.StatusOK}}
	if err := waitReady(context.Background(), targets); err != nil {
		t.Fatal("anonymous 401 should prove authenticated endpoint is ready")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if waitReady(ctx, []readinessTarget{{server.URL + "/api/v1/auth/session", http.StatusOK}}) == nil {
		t.Fatal("wrong readiness status accepted")
	}
}

func TestOwnedProcessGroupCleanup(t *testing.T) {
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	c, err := startChild(path, []string{"-test.run=^TestLauncherChild$"}, []string{"PERSISTTY_DEV_TEST_CHILD=1"})
	if err != nil {
		t.Fatal(err)
	}
	c.stop()
	if err := syscall.Kill(-c.cmd.Process.Pid, 0); err != syscall.ESRCH {
		t.Fatal("owned process group remains")
	}
}

func TestLauncherChild(t *testing.T) {
	if os.Getenv("PERSISTTY_DEV_TEST_CHILD") != "1" {
		return
	}
	for {
		time.Sleep(time.Hour)
	}
}

func freePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	return address
}

// 显式开启的真实本机验收，只管理随机私有 socket 和自身前后端进程组。
func TestLocalStartupPersistence(t *testing.T) {
	if os.Getenv("PERSISTTY_DEV_INTEGRATION") != "1" {
		t.Skip("设置 PERSISTTY_DEV_INTEGRATION=1 执行真实 tmux/npm/Go 联调")
	}
	tmux, err := exec.LookPath("tmux")
	if err != nil {
		t.Fatal("required tmux unavailable")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	source := sourceConfig(t, dir)
	cfg, err := config.Load(source)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Auth.PasswordHash, err = auth.HashPassword([]byte("isolated-dev-test-password"))
	if err != nil {
		t.Fatal(err)
	}
	content, _ := yaml.Marshal(cfg)
	if err := os.WriteFile(source, content, 0600); err != nil {
		t.Fatal(err)
	}
	socket := cfg.TerminalSocketPath()
	tmuxCommand := func(args ...string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		output, err := exec.CommandContext(ctx, tmux, append([]string{"-N", "-S", socket}, args...)...).Output()
		return strings.TrimSpace(string(output)), err
	}
	t.Cleanup(func() { _, _ = tmuxCommand("kill-server") })
	launcher := filepath.Join(dir, "dev-launcher")
	build := exec.Command("go", "build", "-race", "-o", launcher, "./scripts/dev")
	build.Dir = root
	if err := build.Run(); err != nil {
		t.Fatal("launcher build failed")
	}
	backend := freePort(t)
	_, port, _ := net.SplitHostPort(freePort(t))
	frontend := "http://127.0.0.1:" + port
	client := &http.Client{Timeout: 2 * time.Second}
	command := func(args ...string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, launcher, args...)
		cmd.Dir = root
		cmd.Env = append(removeEnv(os.Environ(), "PERSISTTY_DEV_STATE_DIR="), "PERSISTTY_DEV_STATE_DIR="+dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("launcher command failed: %s: %v", out, err)
		}
	}
	t.Cleanup(func() { command("stop") })
	command("start", "--config", source, "--listen", backend, "--port", port, "--tmux", tmux)
	first, err := control(context.Background(), dir, "status")
	if err != nil || !first.Ready {
		t.Fatal("background supervisor unavailable")
	}
	command("start")
	duplicate, err := control(context.Background(), dir, "status")
	if err != nil || duplicate.PID != first.PID {
		t.Fatal("repeated start created a duplicate")
	}
	invalid := exec.Command(launcher, "restart", "--config", filepath.Join(dir, "missing.yaml"))
	invalid.Dir = root
	invalid.Env = append(removeEnv(os.Environ(), "PERSISTTY_DEV_STATE_DIR="), "PERSISTTY_DEV_STATE_DIR="+dir)
	if invalid.Run() == nil {
		t.Fatal("invalid restart configuration accepted")
	}
	preserved, err := control(context.Background(), dir, "status")
	if err != nil || !preserved.Ready || preserved.PID != first.PID {
		t.Fatal("invalid restart interrupted running services")
	}
	login, _ := http.NewRequest(http.MethodPost, frontend+"/api/v1/auth/login", strings.NewReader(`{"password":"isolated-dev-test-password"}`))
	login.Header.Set("Origin", frontend)
	login.Header.Set("Content-Type", "application/json")
	response, err := client.Do(login)
	if err != nil {
		t.Fatal("login transport failed")
	}
	var session struct {
		Data struct {
			CSRFToken string `json:"csrf_token"`
		} `json:"data"`
	}
	loginErr := json.NewDecoder(response.Body).Decode(&session)
	response.Body.Close()
	if response.StatusCode != 200 || len(response.Cookies()) != 1 || loginErr != nil || session.Data.CSRFToken == "" {
		t.Fatal("login through frontend proxy failed")
	}
	cookie := response.Cookies()[0]
	request, _ := http.NewRequest(http.MethodGet, frontend+"/api/v1/terminals", nil)
	request.AddCookie(cookie)
	response, err = client.Do(request)
	if err != nil {
		t.Fatal("terminal request failed")
	}
	var terminals struct {
		Data struct {
			Items []json.RawMessage `json:"items"`
		} `json:"data"`
	}
	decodeErr := json.NewDecoder(response.Body).Decode(&terminals)
	response.Body.Close()
	if response.StatusCode != 200 || decodeErr != nil || terminals.Data.Items == nil {
		t.Fatal("real terminal dependency unavailable")
	}
	post := func(path string, payload any, result any) {
		t.Helper()
		data, _ := json.Marshal(payload)
		request, _ := http.NewRequest(http.MethodPost, frontend+path, bytes.NewReader(data))
		request.Header.Set("Origin", frontend)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-CSRF-Token", session.Data.CSRFToken)
		request.AddCookie(cookie)
		response, err := client.Do(request)
		if err != nil {
			t.Fatal("product create request failed")
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode > 299 || json.NewDecoder(response.Body).Decode(result) != nil {
			t.Fatal("product create response failed")
		}
	}
	projectDir := filepath.Join(dir, "project")
	if err := os.Mkdir(projectDir, 0700); err != nil {
		t.Fatal(err)
	}
	var project struct {
		Data storage.Project `json:"data"`
	}
	post("/api/v1/projects", map[string]any{"name": "开发启动验收", "folder_paths": []string{projectDir}, "main_index": 0}, &project)
	var terminal struct {
		Data storage.Terminal `json:"data"`
	}
	post("/api/v1/terminals", map[string]any{"project_id": project.Data.ID, "project_version": project.Data.Version}, &terminal)
	streamCtx, cancelStream := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStream()
	conn, _, err := websocket.Dial(streamCtx, strings.Replace(frontend, "http://", "ws://", 1)+"/api/v1/terminals/"+terminal.Data.ID+"/stream",
		&websocket.DialOptions{HTTPHeader: http.Header{"Origin": {frontend}, "Cookie": {cookie.String()}}})
	if err != nil {
		t.Fatal("real terminal WebSocket unavailable")
	}
	defer conn.CloseNow()
	_, ready, err := conn.Read(streamCtx)
	var state struct {
		Type       string `json:"type"`
		Role       string `json:"role"`
		Generation uint64 `json:"generation"`
	}
	if err != nil || json.Unmarshal(ready, &state) != nil || state.Type != "ready" || state.Role != "controller" {
		t.Fatal("terminal controller readiness failed")
	}
	input := []byte("printf 'dev-%s\\n' 'startup-proof'\r")
	frame := make([]byte, 8+len(input))
	binary.BigEndian.PutUint64(frame, state.Generation)
	copy(frame[8:], input)
	if err := conn.Write(streamCtx, websocket.MessageBinary, frame); err != nil {
		t.Fatal("terminal input failed")
	}
	var output []byte
	for !bytes.Contains(output, []byte("dev-startup-proof")) {
		kind, data, err := conn.Read(streamCtx)
		if err != nil || len(output) > 1<<20 {
			t.Fatal("real shell output not observed")
		}
		if kind == websocket.MessageBinary {
			output = append(output, data...)
		}
	}
	_ = conn.Close(websocket.StatusNormalClosure, "test_complete")
	target := "=persistty_" + terminal.Data.ID + ":0.0"
	before, err := tmuxCommand("display-message", "-p", "-t", target, "#{pane_pid}")
	if err != nil {
		t.Fatal(err)
	}
	pid, _ := strconv.Atoi(before)
	command("stop")
	command("stop")
	after, err := tmuxCommand("display-message", "-p", "-t", target, "#{pane_pid}")
	if err != nil || before != after || syscall.Kill(pid, 0) != nil {
		t.Fatal("launcher exit killed persistent pane")
	}
	if checkPort(backend) != nil || checkPort(strings.TrimPrefix(frontend, "http://")) != nil {
		t.Fatal("frontend/backend ports remain occupied")
	}
	command("restart")
	second, err := control(context.Background(), dir, "status")
	if err != nil || !second.Ready || second.PID == first.PID {
		t.Fatal("restart did not start a new supervisor")
	}
	command("restart")
	after, err = tmuxCommand("display-message", "-p", "-t", target, "#{pane_pid}")
	if err != nil || before != after {
		t.Fatal("second launch did not reuse existing server")
	}
	command("stop")
	log, err := os.ReadFile(filepath.Join(dir, "launcher.log"))
	if err != nil || bytes.Contains(log, []byte("isolated-dev-test-password")) || bytes.Contains(log, []byte(cfg.Auth.PasswordHash)) || bytes.Contains(log, []byte("dev-startup-proof")) {
		t.Fatal("startup log missing or contains credential/terminal content")
	}
	fmt.Println("真实联调通过：后台启动、重复 start/stop、restart 参数恢复、代理终端输入输出、pane PID 保留。")
}

func TestRememberedArguments(t *testing.T) {
	dir := t.TempDir()
	args := []string{"--config", "/private/config.yaml", "--port", "5175", "--shell", "/bin/zsh"}
	if err := rememberArguments(dir, args); err != nil {
		t.Fatal(err)
	}
	got, err := rememberedArguments(dir)
	info, _ := os.Stat(filepath.Join(dir, "launcher.json"))
	if err != nil || strings.Join(got, "|") != strings.Join(args, "|") || info.Mode().Perm() != 0600 {
		t.Fatal("private argument round trip failed")
	}
	opts, err := parseOptions(append(got, "--port", "5176"))
	if err != nil || opts.port != 5176 || opts.shell != "/bin/zsh" {
		t.Fatal("restart override did not preserve other arguments")
	}
	if err := os.WriteFile(filepath.Join(dir, "launcher.json"), bytes.Repeat([]byte("x"), 16385), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := rememberedArguments(dir); err == nil {
		t.Fatal("oversized arguments accepted")
	}
}

func TestControlDirectoryAndCommands(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PERSISTTY_DEV_STATE_DIR", dir)
	if err := dispatch(context.Background(), []string{"stop"}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{nil, {"unknown"}, {"stop", "--port", "5173"}} {
		if dispatch(context.Background(), args) == nil {
			t.Fatal("invalid command accepted")
		}
	}
	path := filepath.Join(dir, "dev.sock")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if startManaged(context.Background(), dir, nil) == nil {
		t.Fatal("regular control path accepted")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "keep" {
		t.Fatal("unowned file overwritten")
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := controlDirectory(); err == nil {
		t.Fatal("public control directory accepted")
	}
}
