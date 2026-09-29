package files

import (
	"errors"
	"io"
	"os"
	"sort"
	"time"

	"persistty/internal/storage"
)

type directorySnapshot struct {
	mtime time.Time
	items []Entry
}

func snapshotEntries(dir *os.File, entries []os.DirEntry) (directorySnapshot, error) {
	info, err := dir.Stat()
	if err != nil {
		return directorySnapshot{}, err
	}
	items := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		item, err := statEntry(dir, entry.Name())
		if err != nil {
			return directorySnapshot{}, err
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return directorySnapshot{mtime: info.ModTime(), items: items}, nil
}

func verifyDirectorySnapshot(root storage.RegisteredFolder, relative string, original *os.File, before directorySnapshot) error {
	current, err := openMutationParent(root, relative)
	if err != nil {
		return err
	}
	defer current.Close()
	first, err := original.Stat()
	if err != nil {
		return err
	}
	last, err := current.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(first, last) || !before.mtime.Equal(last.ModTime()) {
		return ErrConflict
	}
	after, err := current.ReadDir(10001)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(after) != len(before.items) {
		return ErrConflict
	}
	newSnapshot, err := snapshotEntries(current, after)
	if err != nil {
		return err
	}
	for index, item := range before.items {
		if item != newSnapshot.items[index] {
			return ErrConflict
		}
	}
	return nil
}
