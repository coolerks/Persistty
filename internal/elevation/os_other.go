//go:build !linux

package elevation

import (
	"net"
	"os"
	"time"

	"persistty/internal/files"
)

func timeNow() time.Time { return time.Now() }
func validRelativeTarget(path string) bool {
	return len(path) <= 4096 && files.ValidRelative(path, false)
}
func openTrusted(string, bool) (*os.File, error) { return nil, ErrUnavailable }
func peerAllowed(*net.UnixConn, int) error       { return ErrUnavailable }
func disableCore() error                         { return ErrUnavailable }
func helperFD(int) (*os.File, error)             { return nil, ErrUnavailable }
