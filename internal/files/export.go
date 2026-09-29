package files

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"persistty/internal/storage"
)

func copyOpenedVerified(ctx context.Context, file *os.File, writer io.Writer, maxBytes int64) (Version, error) {
	info, err := file.Stat()
	if err != nil {
		return Version{}, err
	}
	if !info.Mode().IsRegular() {
		return Version{}, ErrUnsupported
	}
	if info.Size() > maxBytes {
		return Version{}, ErrTooLarge
	}
	initial, _, err := versionFromFile(file)
	if err != nil {
		return Version{}, err
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return Version{}, err
	}
	hash := sha256.New()
	count, err := io.Copy(io.MultiWriter(writer, hash), io.LimitReader(file, initial.Size+1))
	if err != nil {
		return Version{}, err
	}
	if count != initial.Size || "sha256:"+hex.EncodeToString(hash.Sum(nil)) != initial.ETag {
		return Version{}, ErrConflict
	}
	if err = ctx.Err(); err != nil {
		return Version{}, err
	}
	final, err := file.Stat()
	if err != nil {
		return Version{}, err
	}
	if final.Size() != info.Size() || !final.ModTime().Equal(info.ModTime()) {
		return Version{}, ErrConflict
	}
	return initial, nil
}

func CopyVerified(ctx context.Context, root storage.RegisteredFolder, relative string, writer io.Writer, maxBytes int64) (Version, error) {
	if !ValidRelative(relative, false) {
		return Version{}, ErrInvalidPath
	}
	file, err := openConstrained(root, relative, false)
	if err != nil {
		return Version{}, err
	}
	defer file.Close()
	return copyOpenedVerified(ctx, file, writer, maxBytes)
}

func WriteZIP(ctx context.Context, root storage.RegisteredFolder, relative string, writer io.Writer, maxBytes int64) (int, int64, error) {
	if !ValidRelative(relative, true) {
		return 0, 0, ErrInvalidPath
	}
	archiveName := path.Base(relative)
	if relative == "" {
		archiveName = filepath.Base(root.Path)
	}
	if !ValidRelative(archiveName, false) {
		return 0, 0, ErrInvalidPath
	}
	zipWriter := zip.NewWriter(writer)
	budget := copyBudget{}
	if err := zipDirectory(ctx, root, relative, archiveName, 0, &budget, zipWriter, maxBytes); err != nil {
		zipWriter.Close()
		return budget.items, budget.bytes, err
	}
	if err := zipWriter.Close(); err != nil {
		return budget.items, budget.bytes, err
	}
	return budget.items, budget.bytes, nil
}

func zipDirectory(ctx context.Context, root storage.RegisteredFolder, relative, zipName string, depth int, budget *copyBudget, writer *zip.Writer, maxBytes int64) error {
	if depth >= 64 || budget.items >= 10000 {
		return ErrTooLarge
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	openPath := relative
	if openPath == "" {
		openPath = "."
	}
	dir, err := openMutationParent(root, openPath)
	if err != nil {
		return err
	}
	defer dir.Close()
	info, err := dir.Stat()
	if err != nil || !info.IsDir() {
		return ErrUnsupported
	}
	budget.items++
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = strings.TrimSuffix(zipName, "/") + "/"
	header.Method = zip.Store
	if _, err = writer.CreateHeader(header); err != nil {
		return err
	}
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
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if err = ctx.Err(); err != nil {
			return err
		}
		name := entry.Name()
		if !ValidRelative(name, false) {
			return ErrUnsupported
		}
		child := name
		if relative != "" {
			child = path.Join(relative, name)
		}
		childZIP := path.Join(zipName, name)
		leaf, e := openLeaf(dir, name)
		if e != nil {
			return e
		}
		childInfo, e := leaf.Stat()
		if e != nil {
			leaf.Close()
			return e
		}
		if childInfo.IsDir() {
			leaf.Close()
			if e = zipDirectory(ctx, root, child, childZIP, depth+1, budget, writer, maxBytes); e != nil {
				return e
			}
			continue
		}
		if !childInfo.Mode().IsRegular() {
			leaf.Close()
			return ErrUnsupported
		}
		budget.items++
		if budget.items > 10000 || childInfo.Size() > maxBytes-budget.bytes {
			leaf.Close()
			return ErrTooLarge
		}
		fh, e := zip.FileInfoHeader(childInfo)
		if e != nil {
			leaf.Close()
			return e
		}
		fh.Name = childZIP
		fh.Method = zip.Deflate
		zipFile, e := writer.CreateHeader(fh)
		if e != nil {
			leaf.Close()
			return e
		}
		version, e := copyOpenedVerified(ctx, leaf, zipFile, maxBytes-budget.bytes)
		leaf.Close()
		if e != nil {
			return e
		}
		budget.bytes += version.Size
	}
	if err = verifyDirectorySnapshot(root, openPath, dir, snapshot); err != nil {
		return fmt.Errorf("directory changed during ZIP: %w", err)
	}
	return nil
}
