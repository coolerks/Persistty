package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"path"
	"sort"
	"syscall"

	"persistty/internal/storage"
)

type DeletePreview struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Count  int    `json:"count"`
	Bytes  int64  `json:"bytes"`
	Digest string `json:"-"`
}

func PreviewDelete(ctx context.Context, root storage.RegisteredFolder, relative string) (DeletePreview, error) {
	preview, _, err := scanDelete(ctx, root, relative)
	return preview, err
}

func DeleteConfirmed(ctx context.Context, root storage.RegisteredFolder, relative, expectedDigest string) (OperationResult, error) {
	preview, manifest, err := scanDelete(ctx, root, relative)
	if err != nil {
		return OperationResult{}, err
	}
	if expectedDigest == "" || preview.Digest != expectedDigest {
		return OperationResult{}, ErrConflict
	}
	result := OperationResult{State: "applied"}
	for i := len(manifest) - 1; i >= 0; i-- {
		if err = removeCopied(ctx, root, manifest[i]); err != nil {
			result.State = "partial"
			return result, err
		}
	}
	result.SourceRemoved = true
	return result, nil
}

func scanDelete(ctx context.Context, root storage.RegisteredFolder, relative string) (DeletePreview, []copiedEntry, error) {
	if !ValidRelative(relative, false) {
		return DeletePreview{}, nil, ErrInvalidPath
	}
	manifest := make([]copiedEntry, 0)
	budget := copyBudget{}
	hasher := sha256.New()
	if err := scanDeleteNode(ctx, root, relative, 0, &budget, &manifest, hasher); err != nil {
		return DeletePreview{}, nil, err
	}
	kind := "file"
	if manifest[0].directory {
		kind = "directory"
	}
	return DeletePreview{Path: relative, Kind: kind, Count: budget.items, Bytes: budget.bytes, Digest: hex.EncodeToString(hasher.Sum(nil))}, manifest, nil
}

func scanDeleteNode(ctx context.Context, root storage.RegisteredFolder, relative string, depth int, budget *copyBudget, manifest *[]copiedEntry, hasher hash.Hash) error {
	if depth >= 64 || budget.items >= 10000 {
		return ErrTooLarge
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	dir, err := openMutationParent(root, path.Dir(relative))
	if err != nil {
		return err
	}
	defer dir.Close()
	file, err := openLeaf(dir, path.Base(relative))
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		file.Close()
		return ErrUnsupported
	}
	identity := fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)
	entry := copiedEntry{path: relative, identity: identity, directory: info.IsDir()}
	budget.items++
	if entry.directory {
		fmt.Fprintf(hasher, "D\x00%s\x00%s\x00%s\n", relative, identity, info.ModTime().UTC().Format("2006-01-02T15:04:05.000000000Z"))
		*manifest = append(*manifest, entry)
		children, e := file.ReadDir(10001)
		file.Close()
		if e != nil && !errors.Is(e, io.EOF) {
			return e
		}
		if len(children) > 10000 {
			return ErrTooLarge
		}
		sort.Slice(children, func(i, j int) bool { return children[i].Name() < children[j].Name() })
		for _, child := range children {
			if !ValidRelative(child.Name(), false) {
				return ErrUnsupported
			}
			if err = scanDeleteNode(ctx, root, path.Join(relative, child.Name()), depth+1, budget, manifest, hasher); err != nil {
				return err
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		file.Close()
		return ErrUnsupported
	}
	version, _, err := versionFromFile(file)
	file.Close()
	if err != nil {
		return err
	}
	budget.bytes += version.Size
	if budget.bytes > 1<<30 {
		return ErrTooLarge
	}
	entry.version = &version
	*manifest = append(*manifest, entry)
	fmt.Fprintf(hasher, "F\x00%s\x00%s\x00%s\x00%s\x00%d\n", relative, identity, version.ETag, version.Mtime, version.Size)
	return nil
}
