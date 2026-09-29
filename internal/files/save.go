package files

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"syscall"
	"unicode/utf8"

	"persistty/internal/storage"
)

var ErrConflict = errors.New("file version conflict")

func versionFromFile(file *os.File) (Version, os.FileMode, error) {
	info, err := file.Stat()
	if err != nil {
		return Version{}, 0, err
	}
	if !info.Mode().IsRegular() {
		return Version{}, 0, ErrUnsupported
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Nlink != 1 {
		return Version{}, 0, ErrUnsupported
	}
	if info.Size() > 20<<30 {
		return Version{}, 0, ErrTooLarge
	}
	hash := sha256.New()
	count, err := io.Copy(hash, io.LimitReader(file, 20<<30+1))
	if err != nil {
		return Version{}, 0, err
	}
	if count > 20<<30 {
		return Version{}, 0, ErrTooLarge
	}
	end, err := file.Stat()
	if err != nil {
		return Version{}, 0, err
	}
	if end.Size() != info.Size() || !end.ModTime().Equal(info.ModTime()) {
		return Version{}, 0, ErrConflict
	}
	return Version{info.ModTime().UTC().Format("2006-01-02T15:04:05.000000000Z"), count, "sha256:" + hex.EncodeToString(hash.Sum(nil)), fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)}, info.Mode().Perm(), nil
}

func Save(ctx context.Context, root storage.RegisteredFolder, relative string, expected Version, content string) (Version, error) {
	if !ValidRelative(relative, false) {
		return Version{}, ErrInvalidPath
	}
	if !utf8.ValidString(content) || len(content) > 8<<20 || containsNUL(content) {
		return Version{}, ErrUnsupported
	}
	parent := path.Dir(relative)
	name := path.Base(relative)
	dir, err := openMutationParent(root, parent)
	if err != nil {
		return Version{}, err
	}
	defer dir.Close()
	current, err := openLeaf(dir, name)
	if err != nil {
		return Version{}, err
	}
	initialInfo, err := current.Stat()
	if err != nil {
		current.Close()
		return Version{}, err
	}
	if initialInfo.Size() > 8<<20 {
		current.Close()
		return Version{}, ErrTooLarge
	}
	version, mode, err := versionFromFile(current)
	current.Close()
	if err != nil {
		return Version{}, err
	}
	if version.Size > 8<<20 {
		return Version{}, ErrTooLarge
	}
	if version != expected {
		return Version{}, ErrConflict
	}
	bytes := make([]byte, 16)
	if _, err = rand.Read(bytes); err != nil {
		return Version{}, err
	}
	tempName := ".persistty-" + hex.EncodeToString(bytes) + ".tmp"
	temp, err := createTemp(dir, tempName)
	if err != nil {
		return Version{}, err
	}
	defer removeLeaf(dir, tempName)
	if _, err = io.WriteString(temp, content); err == nil {
		err = temp.Chmod(mode & 0777)
	}
	if err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err != nil {
		return Version{}, err
	}
	if closeErr != nil {
		return Version{}, closeErr
	}
	if err = ctx.Err(); err != nil {
		return Version{}, err
	}
	parentAgain, err := openMutationParent(root, parent)
	if err != nil {
		return Version{}, err
	}
	defer parentAgain.Close()
	initialInfo, err = dir.Stat()
	if err != nil {
		return Version{}, err
	}
	finalInfo, err := parentAgain.Stat()
	if err != nil {
		return Version{}, err
	}
	if !os.SameFile(initialInfo, finalInfo) {
		return Version{}, ErrRootChanged
	}
	current, err = openLeaf(dir, name)
	if err != nil {
		return Version{}, err
	}
	finalVersion, _, err := versionFromFile(current)
	current.Close()
	if err != nil {
		return Version{}, err
	}
	if finalVersion != expected {
		return Version{}, ErrConflict
	}
	if err = replaceLeaf(dir, tempName, name); err != nil {
		return Version{}, err
	}
	if err = dir.Sync(); err != nil {
		return Version{}, err
	}
	newFile, err := openLeaf(dir, name)
	if err != nil {
		return Version{}, err
	}
	newVersion, _, err := versionFromFile(newFile)
	newFile.Close()
	return newVersion, err
}

func containsNUL(value string) bool {
	for _, character := range value {
		if character == 0 {
			return true
		}
	}
	return false
}
