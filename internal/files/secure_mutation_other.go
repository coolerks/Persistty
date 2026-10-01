//go:build !linux && !darwin

package files

import (
	"os"
	"persistty/internal/storage"
	"strings"
)

func openMutationParent(folder storage.RegisteredFolder, relative string) (*os.File, error) {
	root, err := os.OpenRoot(folder.Path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	if relative != "." {
		current := ""
		for _, part := range strings.Split(relative, "/") {
			if current == "" {
				current = part
			} else {
				current += "/" + part
			}
			info, e := root.Lstat(current)
			if e != nil {
				return nil, e
			}
			if !info.IsDir() {
				return nil, ErrUnsupported
			}
		}
	}
	return openConstrained(folder, relative, true)
}
