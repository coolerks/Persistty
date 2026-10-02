package elevation

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"persistty/internal/storage"
)

type Client struct {
	Enabled    bool
	SocketPath string
}

func (c Client) connect(ctx context.Context) (*net.UnixConn, func(), error) {
	if !c.Enabled || runtime.GOOS != "linux" {
		return nil, nil, ErrUnavailable
	}
	parent, err := openTrusted(filepath.Dir(c.SocketPath), true)
	if err != nil {
		return nil, nil, ErrUnavailable
	}
	parent.Close()
	info, err := os.Lstat(c.SocketPath)
	if err != nil || info.Mode()&os.ModeSocket == 0 || info.Mode().Perm()&0077 != 0 {
		return nil, nil, ErrUnavailable
	}
	raw, err := (&net.Dialer{}).DialContext(ctx, "unix", c.SocketPath)
	if err != nil {
		return nil, nil, ErrUnavailable
	}
	conn, ok := raw.(*net.UnixConn)
	if !ok {
		raw.Close()
		return nil, nil, ErrUnavailable
	}
	if err = peerAllowed(conn, os.Getuid()); err != nil {
		conn.Close()
		return nil, nil, ErrUnavailable
	}
	deadline := time.Now().Add(30 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	close := func() { stop(); conn.Close() }
	return conn, close, nil
}
func (c Client) Target(ctx context.Context, root storage.RegisteredFolder, path string) (Target, error) {
	conn, close, err := c.connect(ctx)
	if err != nil {
		return Target{}, err
	}
	defer close()
	if err = WriteFrame(conn, brokerRequest{Action: "target", Root: root, Path: path}); err != nil {
		return Target{}, ErrUnavailable
	}
	var response reply
	if ReadFrame(conn, &response) != nil {
		return Target{}, ErrUnavailable
	}
	if response.Code != "" {
		return Target{}, replyError(response.Code)
	}
	if response.Phase != "target" || response.Target == nil || !targetPattern.MatchString(response.Target.ID) || !validAbsolute(response.Target.Path) || response.Target.Path != filepath.Join(root.Path, path) {
		return Target{}, ErrUnavailable
	}
	return *response.Target, nil
}
func (c Client) Status(ctx context.Context, g Grant) (Result, error) {
	conn, close, err := c.connect(ctx)
	if err != nil {
		return Result{}, err
	}
	defer close()
	if WriteFrame(conn, brokerRequest{Action: "status", Grant: g}) != nil {
		return Result{}, ErrUnavailable
	}
	var response reply
	if ReadFrame(conn, &response) != nil || response.Result == nil || !ResultValid(*response.Result, g.ID) {
		return Result{}, ErrUnavailable
	}
	return *response.Result, nil
}
func (c Client) Run(ctx context.Context, g Grant, password, content []byte, publish Publish) (Result, error) {
	conn, close, err := c.connect(ctx)
	if err != nil {
		return Result{}, err
	}
	defer close()
	if err = WriteFrame(conn, brokerRequest{Action: "execute", Grant: g}); err != nil {
		return Result{}, ErrUnavailable
	}
	var response reply
	if ReadFrame(conn, &response) != nil {
		return Result{}, ErrUnavailable
	}
	if response.Code != "" {
		return Result{}, replyError(response.Code)
	}
	if response.Phase != "accepted" {
		return Result{}, ErrUnavailable
	}
	if WriteBytes(conn, password, 1024) != nil || WriteBytes(conn, content, MaxContent) != nil {
		return Result{}, ErrUnavailable
	}
	if ReadFrame(conn, &response) != nil {
		return Result{}, ErrUnavailable
	}
	if response.Phase == "result" && response.Result != nil && ResultValid(*response.Result, g.ID) {
		return *response.Result, nil
	}
	if response.Phase != "ready" {
		return Result{}, ErrUnavailable
	}
	return publish(func() (Result, error) {
		deadline := time.Now().Add(5 * time.Second)
		if g.ExpiresAt.Before(deadline) {
			deadline = g.ExpiresAt
		}
		_ = conn.SetDeadline(deadline)
		if !deadline.After(time.Now()) || ctx.Err() != nil {
			return Result{}, ErrExpired
		}
		if WriteFrame(conn, struct {
			Phase string `json:"phase"`
		}{"commit"}) != nil {
			return Result{}, ErrUnavailable
		}
		if ReadFrame(conn, &response) != nil || response.Phase != "result" || response.Result == nil || !ResultValid(*response.Result, g.ID) {
			return Result{}, ErrUnavailable
		}
		return *response.Result, nil
	})
}
func replyError(code string) error {
	switch code {
	case "forbidden":
		return ErrForbidden
	case "rate_limited":
		return ErrRateLimited
	case "invalid_request":
		return ErrInvalid
	default:
		return ErrUnavailable
	}
}
