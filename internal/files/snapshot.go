package files

import (
	"context"
	"io"
	"os"
	"path"
	"persistty/internal/storage"
	"sort"
)

// SnapshotEntries never follows a directory link and verifies the opened directory
// still belongs to the registered root after enumerating it.
func SnapshotEntries(ctx context.Context, root storage.RegisteredFolder, relative string) ([]Entry, error) {
	if !ValidRelative(relative, true) {
		return nil, ErrInvalidPath
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p := relative
	if p == "" {
		p = "."
	}
	dir, err := openMutationParent(root, p)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	entries, err := dir.ReadDir(10001)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(entries) > 10000 {
		return nil, ErrTooLarge
	}
	snap, err := snapshotEntries(dir, entries)
	if err != nil {
		return nil, err
	}
	if err = verifyDirectorySnapshot(root, p, dir, snap); err != nil {
		return nil, err
	}
	for _, e := range snap.items {
		if !ValidRelative(e.Name, false) {
			return nil, ErrUnsupported
		}
	}
	sort.Slice(snap.items, func(i, j int) bool { return snap.items[i].Name < snap.items[j].Name })
	return snap.items, nil
}

// CopySnapshot additionally refuses leaf symlinks, preserves the ordinary mode,
// and compares the reopened parent/entry identity with the original handle.
func CopySnapshot(ctx context.Context, root storage.RegisteredFolder, relative string, w io.Writer, limit int64) (Version, os.FileMode, error) {
	if !ValidRelative(relative, false) {
		return Version{}, 0, ErrInvalidPath
	}
	parent := path.Dir(relative)
	dir, err := openMutationParent(root, parent)
	if err != nil {
		return Version{}, 0, err
	}
	defer dir.Close()
	file, err := snapshotLeaf(dir, path.Base(relative))
	if err != nil {
		return Version{}, 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return Version{}, 0, ErrUnsupported
	}
	version, err := copyOpenedVerified(ctx, file, w, limit)
	if err != nil {
		return Version{}, 0, err
	}
	again, err := openMutationParent(root, parent)
	if err != nil {
		return Version{}, 0, err
	}
	defer again.Close()
	a, err := dir.Stat()
	if err != nil {
		return Version{}, 0, err
	}
	b, err := again.Stat()
	if err != nil || !os.SameFile(a, b) {
		return Version{}, 0, ErrRootChanged
	}
	next, err := snapshotLeaf(again, path.Base(relative))
	if err != nil {
		return Version{}, 0, err
	}
	defer next.Close()
	c, err := next.Stat()
	if err != nil || !os.SameFile(info, c) {
		return Version{}, 0, ErrConflict
	}
	return version, info.Mode().Perm(), nil
}

func SnapshotLink(root storage.RegisteredFolder, relative string) (string, error) {
	if !ValidRelative(relative, false) {
		return "", ErrInvalidPath
	}
	dir, err := openMutationParent(root, path.Dir(relative))
	if err != nil {
		return "", err
	}
	defer dir.Close()
	return snapshotReadlink(dir, path.Base(relative))
}

func WalkSnapshot(ctx context.Context, root storage.RegisteredFolder, start string, maxEntries int, visit func(string, Entry) (bool, error)) error {
	count := 0
	var walk func(string, int) error
	walk = func(relative string, depth int) error {
		if depth > 32 {
			return ErrTooLarge
		}
		entries, err := SnapshotEntries(ctx, root, relative)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			count++
			if count > maxEntries {
				return ErrTooLarge
			}
			if err = ctx.Err(); err != nil {
				return err
			}
			p := path.Join(relative, entry.Name)
			descend, e := visit(p, entry)
			if e != nil {
				return e
			}
			if descend && entry.Kind == "directory" {
				if e = walk(p, depth+1); e != nil {
					return e
				}
			}
		}
		return nil
	}
	return walk(start, 0)
}
