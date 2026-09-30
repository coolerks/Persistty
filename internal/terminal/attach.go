package terminal

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

type attachClient struct {
	file *os.File
	cmd  *exec.Cmd
	done chan struct{}
	once sync.Once
}

func (t *Tmux) Attach(name string, readOnly bool, cols, rows int) (*attachClient, error) {
	if !validName(name) || !validSize(cols, rows) {
		return nil, errors.New("invalid attach target or size")
	}
	args := []string{"-N", "-S", t.Socket, "attach-session"}
	if readOnly {
		args = append(args, "-f", "read-only,ignore-size")
	}
	args = append(args, "-t", "="+name)
	cmd := exec.Command(t.Binary, args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	file, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, err
	}
	fd, err := syscall.Dup(int(file.Fd()))
	if err == nil {
		syscall.CloseOnExec(fd)
		err = syscall.SetNonblock(fd, true)
	}
	_ = file.Close()
	if err != nil {
		if fd >= 0 {
			_ = syscall.Close(fd)
		}
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, err
	}
	client := &attachClient{file: os.NewFile(uintptr(fd), "terminal-attach"), cmd: cmd, done: make(chan struct{})}
	go func() {
		_ = cmd.Wait()
		close(client.done)
	}()
	return client, nil
}

func (c *attachClient) Close() {
	c.once.Do(func() {
		_ = c.file.Close()
		_ = c.cmd.Process.Kill()
		<-c.done
	})
}

func (c *attachClient) Resize(cols, rows int) error {
	if !validSize(cols, rows) {
		return errors.New("invalid terminal size")
	}
	return pty.Setsize(c.file, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}
