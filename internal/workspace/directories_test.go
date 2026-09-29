package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowseAndResolve(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "子目录"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "plain.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	listing, err := Browse(root)
	canonical, resolveErr := filepath.EvalSymlinks(root)
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}
	if err != nil || listing.Path != canonical || listing.Parent != filepath.Dir(canonical) || len(listing.Items) != 1 || listing.Items[0] != "子目录" {
		t.Fatalf("listing=%#v err=%v", listing, err)
	}
	for _, path := range []string{"relative", root + "/..", root + "\x00"} {
		if _, err := Resolve(path); !errors.Is(err, ErrInvalidPath) {
			t.Fatalf("accepted invalid path %q: %v", path, err)
		}
	}
	if _, err := Resolve(filepath.Join(root, "plain.txt")); !errors.Is(err, ErrUnreachable) {
		t.Fatal("regular file accepted as directory")
	}
}
