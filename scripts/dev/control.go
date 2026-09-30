package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type controlReply struct {
	Ready bool   `json:"ready"`
	URL   string `json:"url"`
	PID   int    `json:"pid"`
	Error string `json:"error,omitempty"`
}

func controlDirectory() (string, error) {
	dir := os.Getenv("PERSISTTY_DEV_STATE_DIR")
	if dir == "" {
		dir = filepath.Join(".cache", "dev")
	}
	dir, err := filepath.Abs(dir)
	if err != nil || os.MkdirAll(dir, 0700) != nil {
		return "", errors.New("无法创建私有控制目录")
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return "", errors.New("控制目录必须为私有普通目录（0700）")
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); !ok || int(stat.Uid) != os.Geteuid() {
		return "", errors.New("控制目录必须归当前用户所有")
	}
	return dir, nil
}

func dispatch(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("用法: ./scripts/dev.sh start|stop|restart [选项]")
	}
	if args[0] == "--help" {
		fmt.Println("用法: ./scripts/dev.sh start|stop|restart [选项]；stop/restart 保留 tmux 任务。")
		_, _ = parseOptions([]string{"--help"})
		return nil
	}
	if args[0] == "__serve" {
		return supervise(ctx, args[1:])
	}
	action := args[0]
	if action != "start" && action != "stop" && action != "restart" {
		return errors.New("只支持 start、stop、restart")
	}
	if action == "stop" && len(args) != 1 {
		return errors.New("stop 不接受额外选项")
	}
	dir, err := controlDirectory()
	if err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(dir, "launcher.lock"), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return errors.New("无法打开开发服务控制锁")
	}
	defer lock.Close()
	if err := lock.Chmod(0600); err != nil {
		return errors.New("无法设置控制锁私有权限")
	}
	for {
		err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			return errors.New("无法获取开发服务控制锁")
		}
		select {
		case <-ctx.Done():
			return errors.New("等待其他启动/停止操作时取消")
		case <-time.After(100 * time.Millisecond):
		}
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	if action == "stop" {
		return stopManaged(ctx, dir)
	}
	startArgs := args[1:]
	if action == "restart" {
		previous, err := rememberedArguments(dir)
		if err != nil {
			return err
		}
		startArgs = append(previous, startArgs...)
	}
	opts, err := parseOptions(startArgs)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	canonical := []string{"--config", opts.configPath, "--listen", opts.listen, "--port", strconv.Itoa(opts.port), "--tmux", opts.tmux, "--shell", opts.shell}
	if action == "restart" {
		if _, err := prepareConfig(opts); err != nil {
			return err
		}
		if err := stopManaged(ctx, dir); err != nil {
			return err
		}
	}
	return startManaged(ctx, dir, canonical)
}

func control(ctx context.Context, dir, action string) (controlReply, error) {
	var reply controlReply
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", filepath.Join(dir, "dev.sock"))
	if err != nil {
		return reply, err
	}
	defer conn.Close()
	deadline, _ := ctx.Deadline()
	_ = conn.SetDeadline(deadline)
	if err := json.NewEncoder(conn).Encode(struct {
		Action string `json:"action"`
	}{action}); err != nil {
		return reply, err
	}
	err = json.NewDecoder(io.LimitReader(conn, 4096)).Decode(&reply)
	if err == nil && reply.Error != "" {
		err = errors.New(reply.Error)
	}
	return reply, err
}

func stopManaged(ctx context.Context, dir string) error {
	_, err := control(ctx, dir, "stop")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ECONNREFUSED) {
			fmt.Println("开发服务未启动；tmux 保留。")
			return nil
		}
		return errors.New("无法确认本启动器已停止；不按 PID 或端口结束其他进程")
	}
	fmt.Println("前后端已停止；tmux 和终端任务保留。")
	return nil
}

func rememberedArguments(dir string) ([]string, error) {
	file, err := os.OpenFile(filepath.Join(dir, "launcher.json"), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.New("无法读取上次启动参数")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 16385))
	var state struct {
		Args []string `json:"args"`
	}
	if err != nil || len(data) > 16384 || json.Unmarshal(data, &state) != nil {
		return nil, errors.New("上次启动参数记录无效；原配置未修改")
	}
	return state.Args, nil
}

func rememberArguments(dir string, args []string) error {
	data, err := json.Marshal(struct {
		Args []string `json:"args"`
	}{args})
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, "arguments-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), filepath.Join(dir, "launcher.json"))
}

func startManaged(ctx context.Context, dir string, args []string) error {
	if reply, err := control(ctx, dir, "status"); err == nil {
		if !reply.Ready {
			return errors.New("已有启动器正在启动，请稍后重试")
		}
		fmt.Println("开发服务已运行:", reply.URL)
		return nil
	} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, syscall.ECONNREFUSED) {
		return errors.New("已有控制 socket 无法确认，拒绝启动重复服务")
	}
	socket := filepath.Join(dir, "dev.sock")
	if info, err := os.Lstat(socket); err == nil {
		if info.Mode()&os.ModeSocket == 0 || os.Remove(socket) != nil {
			return errors.New("控制路径被其他文件占用，不删除或覆盖")
		}
	}
	log, err := os.OpenFile(filepath.Join(dir, "launcher.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return errors.New("无法打开私有启动日志")
	}
	defer log.Close()
	if err := log.Chmod(0600); err != nil {
		return errors.New("无法设置启动日志私有权限")
	}
	binary, err := os.Executable()
	if err != nil {
		return errors.New("无法定位启动器")
	}
	cmd := exec.Command(binary, append([]string{"__serve"}, args...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout, cmd.Stderr = log, log
	if err := cmd.Start(); err != nil {
		return errors.New("无法后台启动开发服务")
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	// 只向本次创建的 supervisor 发信号，不能按记录 PID 或端口杀进程。
	abort := func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	}
	timeout := time.NewTimer(90 * time.Second)
	defer timeout.Stop()
	for {
		if reply, err := control(ctx, dir, "status"); err == nil && reply.Ready {
			if err := rememberArguments(dir, args); err != nil {
				abort()
				return errors.New("无法保留启动参数，已停止本次前后端")
			}
			fmt.Printf("开发工作台: %s\n后台运行；使用 ./scripts/dev.sh stop 停止前后端。\n日志: %s\n", reply.URL, filepath.Join(dir, "launcher.log"))
			return nil
		}
		select {
		case <-done:
			return errors.New("后台启动失败，请查看私有 launcher.log；tmux 不会被停止")
		case <-ctx.Done():
			abort()
			return errors.New("启动已取消")
		case <-timeout.C:
			abort()
			return errors.New("后台启动超时")
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func supervise(parent context.Context, args []string) error {
	dir, err := controlDirectory()
	if err != nil {
		return err
	}
	listener, err := net.Listen("unix", filepath.Join(dir, "dev.sock"))
	if err != nil {
		return errors.New("控制 socket 被占用，拒绝启动")
	}
	if err := os.Chmod(filepath.Join(dir, "dev.sock"), 0600); err != nil {
		listener.Close()
		return errors.New("无法设置控制 socket 私有权限")
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	done, accepted := make(chan struct{}), make(chan struct{})
	var mu sync.Mutex
	var workers sync.WaitGroup
	state := controlReply{PID: os.Getpid()}
	go func() {
		defer close(accepted)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			workers.Add(1)
			go func() {
				defer workers.Done()
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				var request struct {
					Action string `json:"action"`
				}
				decoder := json.NewDecoder(io.LimitReader(conn, 512))
				decoder.DisallowUnknownFields()
				if decoder.Decode(&request) != nil {
					return
				}
				if request.Action == "stop" {
					_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
					cancel()
					<-done
				} else if request.Action != "status" {
					_ = json.NewEncoder(conn).Encode(controlReply{Error: "未知控制操作"})
					return
				}
				mu.Lock()
				reply := state
				mu.Unlock()
				_ = json.NewEncoder(conn).Encode(reply)
			}()
		}
	}()
	err = runServices(ctx, args, func(url string) {
		mu.Lock()
		state.Ready, state.URL = true, url
		mu.Unlock()
	})
	listener.Close()
	<-accepted
	close(done)
	workers.Wait()
	return err
}
