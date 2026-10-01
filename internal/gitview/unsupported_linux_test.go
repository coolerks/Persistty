//go:build linux

package gitview

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestUnsupportedRepositoryContentsUseRepositoryError(t *testing.T) {
	s, p, root := fixture(t)
	if err := os.WriteFile(filepath.Join(root, string([]byte{0xff})), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := s.Repositories(context.Background(), p.ID, p.Version)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("repository error: %v", err)
	}
}
