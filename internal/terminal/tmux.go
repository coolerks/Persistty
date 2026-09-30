package terminal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var ErrUnavailable = errors.New("tmux server unavailable")
var ErrHistoryTooLarge = errors.New("terminal history exceeds configured byte limit")

type Tmux struct {
	Binary       string
	Socket       string
	Shell        string
	HistoryLines int
	RestoreLines int
	HistoryBytes int64
	run          func(context.Context, int64, ...string) ([]byte, error)
}

type History struct {
	Content       []byte
	HistorySize   int
	ReturnedLines int
	AlternateOn   bool
	Cols          int
	Rows          int
	Truncated     bool
}

func (t *Tmux) Run(ctx context.Context, limit int64, args ...string) ([]byte, error) {
	if t.run != nil {
		return t.run(ctx, limit, args...)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, t.Binary, append([]string{"-N", "-S", t.Socket}, args...)...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	var out limitedBuffer
	out.max = limit
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	err := cmd.Run()
	if out.exceeded {
		return nil, ErrHistoryTooLarge
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return out.Bytes(), err
}

type limitedBuffer struct {
	bytes.Buffer
	max      int64
	exceeded bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if int64(b.Len()+len(p)) > b.max {
		b.exceeded = true
		return 0, ErrHistoryTooLarge
	}
	return b.Buffer.Write(p)
}

func (t *Tmux) Probe(ctx context.Context) error {
	_, err := t.Run(ctx, 1024, "show-options", "-gqv", "exit-empty")
	if err != nil {
		return fmt.Errorf("tmux probe: %w", ErrUnavailable)
	}
	return nil
}

func (t *Tmux) SessionExists(ctx context.Context, name string) (bool, error) {
	if !validName(name) {
		return false, errors.New("invalid terminal name")
	}
	if err := t.Probe(ctx); err != nil {
		return false, err
	}
	_, err := t.Run(ctx, 1024, "has-session", "-t", "="+name)
	if err == nil {
		return true, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("tmux session check: %w", ErrUnavailable)
}

func (t *Tmux) SessionNames(ctx context.Context) (map[string]bool, error) {
	if err := t.Probe(ctx); err != nil {
		return nil, err
	}
	output, err := t.Run(ctx, 64<<10, "list-sessions", "-F", "#{session_name}")
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return nil, ErrUnavailable
		}
	}
	names := make(map[string]bool)
	for _, name := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if validName(name) {
			names[name] = true
		}
	}
	return names, nil
}

func (t *Tmux) Create(ctx context.Context, name, cwd, displayName, projectID string, cols, rows int) error {
	if !validName(name) || !validSize(cols, rows) {
		return errors.New("invalid terminal name or size")
	}
	if err := t.Probe(ctx); err != nil {
		return err
	}
	_, err := t.Run(ctx, 1024, "new-session", "-d", "-s", name, "-c", cwd,
		"-x", strconv.Itoa(cols), "-y", strconv.Itoa(rows),
		"-e", "PERSISTTY_INITIAL_CWD="+cwd,
		"-e", "PERSISTTY_DISPLAY_NAME="+displayName,
		"-e", "PERSISTTY_PROJECT_ID="+projectID, t.Shell)
	if err != nil {
		return fmt.Errorf("tmux create session: %w", ErrUnavailable)
	}
	_, err = t.Run(ctx, 1024, "set-option", "-w", "-t", "="+name+":0", "history-limit", strconv.Itoa(t.HistoryLines))
	if err != nil {
		return fmt.Errorf("tmux set history limit: %w", ErrUnavailable)
	}
	return nil
}

func (t *Tmux) Metadata(ctx context.Context, name string) (cwd, displayName, projectID string, err error) {
	if !validName(name) {
		return "", "", "", errors.New("invalid terminal name")
	}
	values := []*string{&cwd, &displayName, &projectID}
	options := []string{"PERSISTTY_INITIAL_CWD", "PERSISTTY_DISPLAY_NAME", "PERSISTTY_PROJECT_ID"}
	for i, option := range options {
		out, runErr := t.Run(ctx, 1024, "show-environment", "-t", "="+name, option)
		if runErr != nil {
			return "", "", "", ErrUnavailable
		}
		line := strings.TrimSuffix(string(out), "\n")
		if !strings.HasPrefix(line, option+"=") {
			return "", "", "", ErrUnavailable
		}
		*values[i] = strings.TrimPrefix(line, option+"=")
	}
	return cwd, displayName, projectID, nil
}

func (t *Tmux) Kill(ctx context.Context, name string) error {
	if !validName(name) {
		return errors.New("invalid terminal name")
	}
	if err := t.Probe(ctx); err != nil {
		return err
	}
	_, err := t.Run(ctx, 1024, "kill-session", "-t", "="+name)
	return err
}

func (t *Tmux) DisplayName(ctx context.Context, name string) (string, error) {
	if !validName(name) {
		return "", ErrInvalidRequest
	}
	out, err := t.Run(ctx, 1024, "show-environment", "-t", "="+name, "PERSISTTY_DISPLAY_NAME")
	if err != nil {
		return "", ErrUnavailable
	}
	line := strings.TrimSuffix(string(out), "\n")
	if !strings.HasPrefix(line, "PERSISTTY_DISPLAY_NAME=") {
		return "", ErrUnavailable
	}
	return strings.TrimPrefix(line, "PERSISTTY_DISPLAY_NAME="), nil
}

func (t *Tmux) SetDisplayName(ctx context.Context, name, displayName string) error {
	if !validName(name) || !validDisplayName(displayName) {
		return ErrInvalidRequest
	}
	_, err := t.Run(ctx, 1024, "set-environment", "-t", "="+name, "PERSISTTY_DISPLAY_NAME", displayName)
	return err
}

func (t *Tmux) CaptureHistory(ctx context.Context, name string) (History, error) {
	var h History
	if !validName(name) {
		return h, errors.New("invalid terminal name")
	}
	if err := t.Probe(ctx); err != nil {
		return h, err
	}
	pane := "=" + name + ":0.0"
	meta, err := t.Run(ctx, 1024, "display-message", "-p", "-t", pane,
		"#{history_size}:#{alternate_on}:#{window_width}:#{window_height}")
	if err != nil {
		return h, err
	}
	parts := strings.Split(strings.TrimSpace(string(meta)), ":")
	if len(parts) != 4 {
		return h, errors.New("invalid tmux history metadata")
	}
	if h.HistorySize, err = strconv.Atoi(parts[0]); err != nil || h.HistorySize < 0 {
		return History{}, errors.New("invalid tmux history size")
	}
	h.AlternateOn = parts[1] == "1"
	if h.Cols, err = strconv.Atoi(parts[2]); err != nil || !validDimension(h.Cols) {
		return History{}, errors.New("invalid tmux columns")
	}
	if h.Rows, err = strconv.Atoi(parts[3]); err != nil || !validDimension(h.Rows) {
		return History{}, errors.New("invalid tmux rows")
	}
	h.Truncated = h.HistorySize > t.RestoreLines
	start := "-" + strconv.Itoa(t.RestoreLines)
	h.Content, err = t.Run(ctx, t.HistoryBytes, "capture-pane", "-p", "-e", "-t", pane,
		"-S", start, "-E", "-1")
	if err != nil {
		return History{}, err
	}
	h.ReturnedLines = bytes.Count(h.Content, []byte{'\n'})
	if len(h.Content) > 0 && h.Content[len(h.Content)-1] != '\n' {
		h.ReturnedLines++
	}
	return h, nil
}

func validName(name string) bool {
	if !strings.HasPrefix(name, "persistty_") || len(name) > 80 {
		return false
	}
	for _, c := range name[len("persistty_"):] {
		if c != '_' && c != '-' && (c < '0' || c > '9') && (c < 'a' || c > 'z') {
			return false
		}
	}
	return len(name) > len("persistty_")
}

func validDimension(n int) bool     { return n >= 1 && n <= 1000 }
func validSize(cols, rows int) bool { return validDimension(cols) && validDimension(rows) }
