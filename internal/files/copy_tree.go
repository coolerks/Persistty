package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"syscall"

	"persistty/internal/storage"
)

type copiedEntry struct {
	path      string
	identity  string
	version   *Version
	directory bool
}
type copyBudget struct {
	items int
	bytes int64
}

func copyDirectory(ctx context.Context, source, target storage.RegisteredFolder, operation Operation) (OperationResult, error) {
	dir, err := openMutationParent(source, operation.SourcePath)
	if err != nil {
		return OperationResult{}, err
	}
	info, err := dir.Stat()
	dir.Close()
	if err != nil {
		return OperationResult{}, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || fmt.Sprintf("%d:%d", stat.Dev, stat.Ino) != operation.ExpectedIdentity {
		return OperationResult{}, ErrConflict
	}
	result, err := createEntry(ctx, target, Operation{Kind: "create_directory", ProjectVersion: operation.ProjectVersion, TargetPath: operation.TargetPath})
	if err != nil {
		return result, err
	}
	manifest := make([]copiedEntry, 0)
	budget := copyBudget{}
	err = copyDirectoryContents(ctx, source, target, operation.SourcePath, operation.TargetPath, 0, &budget, &manifest)
	if err != nil {
		result.State = "partial"
		return result, err
	}
	if operation.Kind == "move" {
		for i := len(manifest) - 1; i >= 0; i-- {
			if err = removeCopied(ctx, source, manifest[i]); err != nil {
				result.State = "partial"
				return result, err
			}
		}
		if err = removeCopied(ctx, source, copiedEntry{path: operation.SourcePath, identity: operation.ExpectedIdentity, directory: true}); err != nil {
			result.State = "partial"
			return result, err
		}
		result.SourceRemoved = true
	}
	return result, nil
}

func copyDirectoryContents(ctx context.Context, source, target storage.RegisteredFolder, sourcePath, targetPath string, depth int, budget *copyBudget, manifest *[]copiedEntry) error {
	if depth >= 64 {
		return ErrTooLarge
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	dir, err := openMutationParent(source, sourcePath)
	if err != nil {
		return err
	}
	defer dir.Close()
	entries, err := dir.ReadDir(10001)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(entries) > 10000 {
		return ErrTooLarge
	}
	snapshot, err := snapshotEntries(dir, entries)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err = ctx.Err(); err != nil {
			return err
		}
		budget.items++
		if budget.items > 10000 {
			return ErrTooLarge
		}
		name := entry.Name()
		if !ValidRelative(name, false) || name == "." || name == ".." {
			return ErrUnsupported
		}
		from := path.Join(sourcePath, name)
		to := path.Join(targetPath, name)
		leaf, e := openLeaf(dir, name)
		if e != nil {
			return e
		}
		info, e := leaf.Stat()
		if e != nil {
			leaf.Close()
			return e
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			leaf.Close()
			return ErrUnsupported
		}
		identity := fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)
		if info.IsDir() {
			leaf.Close()
			_, e = createEntry(ctx, target, Operation{Kind: "create_directory", ProjectVersion: target.ProjectVersion, TargetPath: to})
			if e != nil {
				return e
			}
			*manifest = append(*manifest, copiedEntry{path: from, identity: identity, directory: true})
			if e = copyDirectoryContents(ctx, source, target, from, to, depth+1, budget, manifest); e != nil {
				return e
			}
			continue
		}
		if !info.Mode().IsRegular() {
			leaf.Close()
			return ErrUnsupported
		}
		version, _, e := versionFromFile(leaf)
		leaf.Close()
		if e != nil {
			return e
		}
		budget.bytes += version.Size
		if budget.bytes > 1<<30 {
			return ErrTooLarge
		}
		_, e = changeEntry(ctx, source, target, Operation{Kind: "copy", ProjectVersion: target.ProjectVersion, SourceFolderID: source.FolderID, SourcePath: from, TargetFolderID: target.FolderID, TargetPath: to, ExpectedVersion: &version, ExpectedIdentity: identity})
		if e != nil {
			return e
		}
		*manifest = append(*manifest, copiedEntry{path: from, identity: identity, version: &version})
	}
	return verifyDirectorySnapshot(source, sourcePath, dir, snapshot)
}

func removeCopied(ctx context.Context, root storage.RegisteredFolder, entry copiedEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir, err := openMutationParent(root, path.Dir(entry.path))
	if err != nil {
		return err
	}
	defer dir.Close()
	name := path.Base(entry.path)
	file, err := openLeaf(dir, name)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || fmt.Sprintf("%d:%d", stat.Dev, stat.Ino) != entry.identity {
		file.Close()
		return ErrConflict
	}
	if entry.directory {
		if !info.IsDir() {
			file.Close()
			return ErrConflict
		}
	} else {
		if entry.version == nil {
			file.Close()
			return ErrVersionRequired
		}
		version, _, e := versionFromFile(file)
		if e != nil {
			file.Close()
			return e
		}
		if version != *entry.version {
			file.Close()
			return ErrConflict
		}
	}
	file.Close()
	if err = ensureParent(root, path.Dir(entry.path), dir); err != nil {
		return err
	}
	if entry.directory {
		err = removeDirLeaf(dir, name)
	} else {
		err = removeLeaf(dir, name)
	}
	if err != nil {
		return err
	}
	return dir.Sync()
}
