package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.yaml.in/yaml/v3"
	"persistty/internal/config"
)

type options struct {
	configPath, listen, tmux, shell string
	port                            int
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := dispatch(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "开发启动:", err)
		os.Exit(1)
	}
}

func parseOptions(args []string) (options, error) {
	flags := flag.NewFlagSet("dev.sh", flag.ContinueOnError)
	var opts options
	flags.StringVar(&opts.configPath, "config", ".cache/dev/config.yaml", "现有私有开发配置")
	flags.StringVar(&opts.listen, "listen", "", "后端地址（仅 127.0.0.1）")
	flags.IntVar(&opts.port, "port", 0, "前端端口（默认使用配置的 public_origin）")
	flags.StringVar(&opts.tmux, "tmux", "", "tmux 可执行文件绝对路径")
	flags.StringVar(&opts.shell, "shell", "", "终端 shell 绝对路径，例如 /bin/zsh")
	if err := flags.Parse(args); err != nil {
		return opts, err
	}
	if flags.NArg() != 0 {
		return opts, errors.New("只接受已列出的选项，使用 --help 查看")
	}
	return opts, nil
}

func runServices(ctx context.Context, args []string, ready func(string)) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	cfg, err := prepareConfig(opts)
	if err != nil {
		return err
	}
	for _, address := range []string{cfg.Server.Listen, strings.TrimPrefix(cfg.Server.PublicOrigin, "http://")} {
		if err := checkPort(address); err != nil {
			return err
		}
	}
	for _, tool := range []string{"go", "npm", "node"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("缺少 %s，请先安装项目工具链", tool)
		}
	}
	if _, err := os.Stat("web/node_modules/vite/bin/vite.js"); err != nil {
		return errors.New("前端依赖未安装，请先运行 npm --prefix web ci")
	}
	root, err := os.Getwd()
	if err != nil {
		return errors.New("无法确定项目目录")
	}
	cache := filepath.Join(root, ".cache", "dev")
	if err := os.MkdirAll(cache, 0700); err != nil {
		return errors.New("无法创建私有开发目录")
	}
	runtimeDir, err := os.MkdirTemp(cache, "run-")
	if err != nil {
		return errors.New("无法创建私有运行目录")
	}
	defer os.RemoveAll(runtimeDir)
	configPath, err := writeRuntimeConfig(runtimeDir, cfg)
	if err != nil {
		return err
	}
	binary := filepath.Join(runtimeDir, "persistty")
	fmt.Println("构建后端...")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "./cmd/persistty")
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		return errors.New("后端构建失败或启动已取消")
	}
	if err := ensureTmux(ctx, cfg.Terminal.TmuxBinary, cfg.TerminalSocketPath(), filepath.Join(root, "deploy", "tmux.example.conf")); err != nil {
		return err
	}
	fmt.Println("私有 tmux 已就绪；退出启动器时不会终止终端任务。")
	backend, err := startChild(binary, []string{"serve", "--config", configPath}, nil)
	if err != nil {
		return errors.New("无法启动后端")
	}
	defer backend.stop()
	parsed, _ := url.Parse(cfg.Server.PublicOrigin)
	frontend, err := startChild("npm", []string{"--prefix", "web", "run", "dev", "--", "--port", parsed.Port(), "--strictPort"},
		[]string{"PERSISTTY_DEV_API_TARGET=http://" + cfg.Server.Listen})
	if err != nil {
		return errors.New("无法启动前端")
	}
	defer frontend.stop()
	if err := waitReady(ctx, []readinessTarget{{"http://" + cfg.Server.Listen + "/api/v1/auth/session", http.StatusUnauthorized}, {cfg.Server.PublicOrigin, http.StatusOK}}, backend, frontend); err != nil {
		return err
	}
	ready(cfg.Server.PublicOrigin)
	select {
	case <-ctx.Done():
		return nil
	case <-backend.done:
		return errors.New("后端已退出，停止本次前端；tmux 保留")
	case <-frontend.done:
		return errors.New("前端已退出，停止本次后端；tmux 保留")
	}
}

func prepareConfig(opts options) (config.Config, error) {
	cfg, err := config.Load(opts.configPath)
	if err != nil {
		return cfg, fmt.Errorf("开发配置无效: %w", err)
	}
	if cfg.Server.Mode != "development" {
		return cfg, errors.New("此脚本仅用于 development，不管理正式部署")
	}
	if opts.listen != "" {
		cfg.Server.Listen = opts.listen
	}
	host, _, err := net.SplitHostPort(cfg.Server.Listen)
	if err != nil || host != "127.0.0.1" {
		return cfg, errors.New("开发后端必须监听 127.0.0.1 的明确端口")
	}
	origin, _ := url.Parse(cfg.Server.PublicOrigin)
	port := opts.port
	if port == 0 {
		port, _ = strconv.Atoi(origin.Port())
	}
	if port < 1 || port > 65535 {
		return cfg, errors.New("前端端口必须为 1..65535")
	}
	cfg.Server.PublicOrigin = "http://127.0.0.1:" + strconv.Itoa(port)
	if opts.tmux != "" {
		cfg.Terminal.TmuxBinary = opts.tmux
	} else if cfg.Terminal.TmuxBinary == "/usr/bin/tmux" && !executable(cfg.Terminal.TmuxBinary) {
		found, err := exec.LookPath("tmux")
		if err != nil {
			return cfg, errors.New("找不到 tmux，请先安装或指定 --tmux")
		}
		cfg.Terminal.TmuxBinary, err = filepath.Abs(found)
		if err != nil {
			return cfg, errors.New("无法解析 tmux 路径")
		}
	}
	if opts.shell != "" {
		cfg.Terminal.Shell = opts.shell
	}
	if !executable(cfg.Terminal.TmuxBinary) || !executable(cfg.Terminal.Shell) {
		return cfg, errors.New("tmux/shell 必须为存在且可执行的绝对路径")
	}
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("运行配置无效: %w", err)
	}
	return cfg, nil
}

func executable(path string) bool {
	info, err := os.Stat(path)
	return filepath.IsAbs(path) && err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0
}

func checkPort(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("端口 %s 不可用，请先停止旧服务或指定 --listen/--port；不会自动结束其他进程", address)
	}
	return listener.Close()
}

func writeRuntimeConfig(dir string, cfg config.Config) (string, error) {
	content, err := yaml.Marshal(cfg)
	if err != nil {
		return "", errors.New("无法生成运行配置")
	}
	defer clear(content)
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, content, 0600); err != nil {
		return "", errors.New("无法写入私有运行配置")
	}
	return path, nil
}

func ensureTmux(ctx context.Context, binary, socket, settings string) error {
	if err := os.MkdirAll(filepath.Dir(socket), 0700); err != nil {
		return errors.New("无法创建私有 socket 目录")
	}
	parent, err := os.Stat(filepath.Dir(socket))
	if err != nil || !parent.IsDir() || parent.Mode().Perm()&0077 != 0 {
		return errors.New("socket 父目录必须为当前用户的私有目录（0700）")
	}
	if stat, ok := parent.Sys().(*syscall.Stat_t); !ok || int(stat.Uid) != os.Geteuid() {
		return errors.New("socket 父目录必须归当前用户所有")
	}
	probe := func() bool {
		probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return exec.CommandContext(probeCtx, binary, "-N", "-S", socket, "show-options", "-gqv", "exit-empty").Run() == nil
	}
	if probe() {
		return nil
	}
	if _, err := os.Lstat(socket); err == nil || !errors.Is(err, os.ErrNotExist) {
		return errors.New("私有 socket 已存在但不可连接；不会删除 socket 或重启已有 server")
	}
	startCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// tmux 自行 daemonize，不属于前后端进程组，且不用 ~/.tmux.conf。
	if err := exec.CommandContext(startCtx, binary, "-S", socket, "-f", settings, "start-server").Run(); err != nil {
		return errors.New("独立 tmux 启动失败，请检查项目 tmux 配置")
	}
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		if probe() {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("启动已取消；不清理持久 tmux")
		case <-deadline.C:
			return errors.New("tmux 未在限时内就绪")
		case <-tick.C:
		}
	}
}

type child struct {
	cmd  *exec.Cmd
	done chan struct{}
	once sync.Once
}

func startChild(binary string, args, extraEnv []string) (*child, error) {
	cmd := exec.Command(binary, args...)
	cmd.Env = os.Environ()
	for _, extra := range extraEnv {
		key := strings.SplitN(extra, "=", 2)[0] + "="
		cmd.Env = append(removeEnv(cmd.Env, key), extra)
	}
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	c := &child{cmd: cmd, done: make(chan struct{})}
	go func() { _ = cmd.Wait(); close(c.done) }()
	return c, nil
}

func removeEnv(env []string, prefix string) []string {
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func (c *child) stop() {
	c.once.Do(func() {
		defer syscall.Kill(-c.cmd.Process.Pid, syscall.SIGKILL)
		_ = syscall.Kill(-c.cmd.Process.Pid, syscall.SIGTERM)
		select {
		case <-c.done:
		case <-time.After(12 * time.Second):
			_ = syscall.Kill(-c.cmd.Process.Pid, syscall.SIGKILL)
			<-c.done
		}
	})
}

type readinessTarget struct {
	url    string
	status int
}

func waitReady(ctx context.Context, targets []readinessTarget, children ...*child) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	client := &http.Client{Timeout: time.Second}
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		for _, child := range children {
			select {
			case <-child.done:
				return errors.New("前后端启动失败，请检查上方输出；不会停止 tmux")
			default:
			}
		}
		ready := true
		for _, target := range targets {
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, target.url, nil)
			response, err := client.Do(req)
			if err != nil {
				ready = false
				break
			}
			_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
			response.Body.Close()
			ready = ready && response.StatusCode == target.status
		}
		if ready {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("前后端就绪等待超时或已取消")
		case <-tick.C:
		}
	}
}
