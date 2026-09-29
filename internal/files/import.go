package files

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"

	"persistty/internal/storage"
)

type ImportResult struct {
	State   string  `json:"state"`
	Version Version `json:"version"`
}

func ImportPrepared(ctx context.Context, root storage.RegisteredFolder, relative string, staged *os.File, size int64, expectedSHA string, expected *Version) (ImportResult, error) {
	if !ValidRelative(relative, false) || size < 0 || size > 20<<30 || len(expectedSHA) != 64 {
		return ImportResult{}, ErrInvalidPath
	}
	parentPath := path.Dir(relative)
	dir, err := openMutationParent(root, parentPath)
	if err != nil {
		return ImportResult{}, err
	}
	defer dir.Close()
	name := path.Base(relative)
	existing, err := openLeaf(dir, name)
	var current Version
	mode := os.FileMode(0600)
	exists := err == nil
	if exists {
		current, mode, err = versionFromFile(existing)
		existing.Close()
		if err != nil {
			return ImportResult{}, err
		}
		if current.Size == size && current.ETag == "sha256:"+expectedSHA {
			return ImportResult{State: "skipped", Version: current}, nil
		}
		if expected == nil || current != *expected {
			return ImportResult{}, ErrConflict
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return ImportResult{}, err
	} else if expected != nil {
		return ImportResult{}, ErrConflict
	}
	if _, err = staged.Seek(0, io.SeekStart); err != nil {
		return ImportResult{}, err
	}
	nameBytes := make([]byte, 16)
	if _, err = rand.Read(nameBytes); err != nil {
		return ImportResult{}, err
	}
	tempName := ".persistty-" + hex.EncodeToString(nameBytes) + ".tmp"
	temp, err := createTemp(dir, tempName)
	if err != nil {
		return ImportResult{}, err
	}
	defer removeLeaf(dir, tempName)
	hash := sha256.New()
	copied, err := io.Copy(io.MultiWriter(temp, hash), io.LimitReader(staged, size+1))
	if err == nil && copied != size {
		err = ErrConflict
	}
	if err == nil && hex.EncodeToString(hash.Sum(nil)) != expectedSHA {
		err = ErrConflict
	}
	if err == nil {
		err = temp.Chmod(mode & 0777)
	}
	if err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err != nil {
		return ImportResult{}, err
	}
	if closeErr != nil {
		return ImportResult{}, closeErr
	}
	if err = ctx.Err(); err != nil {
		return ImportResult{}, err
	}
	if err = ensureParent(root, parentPath, dir); err != nil {
		return ImportResult{}, err
	}
	check, checkErr := openLeaf(dir, name)
	if exists {
		if checkErr != nil {
			return ImportResult{}, ErrConflict
		}
		last, _, e := versionFromFile(check)
		check.Close()
		if e != nil {
			return ImportResult{}, e
		}
		if last != current {
			return ImportResult{}, ErrConflict
		}
		err = replaceLeaf(dir, tempName, name)
	} else {
		if checkErr == nil {
			check.Close()
			return ImportResult{}, ErrConflict
		}
		if !errors.Is(checkErr, os.ErrNotExist) {
			return ImportResult{}, checkErr
		}
		err = renameNoReplace(dir, tempName, dir, name)
	}
	if err != nil {
		return ImportResult{}, err
	}
	result := ImportResult{State: "uploaded"}
	if err = dir.Sync(); err != nil {
		return result, err
	}
	newFile, err := openLeaf(dir, name)
	if err != nil {
		return result, err
	}
	result.Version, _, err = versionFromFile(newFile)
	newFile.Close()
	return result, err
}
