package files

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"syscall"
	"unicode/utf8"

	"persistty/internal/storage"
)

// PreparedPrivileged owns only its private temporary file. Preparing is not a
// publication; Close removes that file and never touches the original target.
type PreparedPrivileged struct {
	root                     storage.RegisteredFolder
	relative, name, tempName string
	dir                      *os.File
	expected                 Version
	temporary                Version
	metadata                 privilegedMetadata
	closed                   bool
	published                bool
}
type privilegedMetadata struct {
	UID, GID int
	Mode     os.FileMode
	Attrs    map[string][]byte
}

func PreparePrivileged(ctx context.Context, root storage.RegisteredFolder, relative string, expected Version, content []byte) (*PreparedPrivileged, error) {
	if !ValidRelative(relative, false) {
		return nil, ErrInvalidPath
	}
	if len(content) > 8<<20 {
		return nil, ErrTooLarge
	}
	if !utf8.Valid(content) || bytes.IndexByte(content, 0) >= 0 {
		return nil, ErrUnsupported
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dir, err := openMutationParent(root, path.Dir(relative))
	if err != nil {
		return nil, err
	}
	p := &PreparedPrivileged{root: root, relative: relative, name: path.Base(relative), dir: dir, expected: expected}
	success := false
	defer func() {
		if !success {
			p.Close()
		}
	}()
	current, err := openLeaf(dir, p.name)
	if err != nil {
		return nil, err
	}
	version, metadata, err := privilegedSnapshot(ctx, current)
	current.Close()
	if err != nil {
		return nil, err
	}
	if version != expected {
		return nil, ErrConflict
	}
	p.metadata = metadata
	random := make([]byte, 16)
	if _, err = rand.Read(random); err != nil {
		return nil, err
	}
	p.tempName = ".persistty-" + hex.EncodeToString(random) + ".tmp"
	temp, err := createTemp(dir, p.tempName)
	if err != nil {
		return nil, err
	}
	_, err = temp.Write(content)
	if err == nil {
		err = restorePrivilegedMetadata(temp, metadata)
	}
	if err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}
	// The temporary metadata is inspected via a fresh no-follow handle.
	check, err := openLeaf(dir, p.tempName)
	if err != nil {
		return nil, err
	}
	temporary, actual, err := privilegedSnapshot(ctx, check)
	check.Close()
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(content)
	if temporary.ETag != "sha256:"+hex.EncodeToString(digest[:]) || !reflect.DeepEqual(actual, metadata) {
		return nil, ErrUnsupported
	}
	p.temporary = temporary
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	success = true
	return p, nil
}
func privilegedSnapshot(ctx context.Context, file *os.File) (Version, privilegedMetadata, error) {
	info, err := file.Stat()
	if err != nil {
		return Version{}, privilegedMetadata{}, err
	}
	if info.Size() > 8<<20 {
		return Version{}, privilegedMetadata{}, ErrTooLarge
	}
	metadata, err := readPrivilegedMetadata(file)
	if err != nil {
		return Version{}, metadata, err
	}
	version, err := privilegedVersion(ctx, file)
	if err != nil {
		return version, metadata, err
	}
	// Verify metadata and a second hash, not just stat timestamps.
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return version, metadata, err
	}
	again, err := privilegedVersion(ctx, file)
	if err != nil {
		return version, metadata, err
	}
	end, err := readPrivilegedMetadata(file)
	if err != nil {
		return version, metadata, err
	}
	if again != version || !reflect.DeepEqual(end, metadata) {
		return version, metadata, ErrConflict
	}
	return version, metadata, nil
}
func (p *PreparedPrivileged) Commit(ctx context.Context) (Version, error) {
	if p.closed || p.published {
		return Version{}, ErrConflict
	}
	if err := ctx.Err(); err != nil {
		return Version{}, err
	}
	again, err := openMutationParent(p.root, path.Dir(p.relative))
	if err != nil {
		return Version{}, err
	}
	defer again.Close()
	first, err := p.dir.Stat()
	if err != nil {
		return Version{}, err
	}
	last, err := again.Stat()
	if err != nil {
		return Version{}, err
	}
	if !os.SameFile(first, last) {
		return Version{}, ErrRootChanged
	}
	file, err := openLeaf(p.dir, p.name)
	if err != nil {
		return Version{}, err
	}
	version, metadata, err := privilegedSnapshot(ctx, file)
	file.Close()
	if err != nil {
		return Version{}, err
	}
	if version != p.expected || !reflect.DeepEqual(metadata, p.metadata) {
		return Version{}, ErrConflict
	}
	temp, err := openLeaf(p.dir, p.tempName)
	if err != nil {
		return Version{}, err
	}
	temporary, tempMetadata, err := privilegedSnapshot(ctx, temp)
	temp.Close()
	if err != nil {
		return Version{}, err
	}
	if temporary != p.temporary || !reflect.DeepEqual(tempMetadata, p.metadata) {
		return Version{}, ErrConflict
	}
	if err = ctx.Err(); err != nil {
		return Version{}, err
	}
	if err = replaceLeaf(p.dir, p.tempName, p.name); err != nil {
		return Version{}, err
	}
	p.published = true
	if err = p.dir.Sync(); err != nil {
		return Version{}, err
	}
	file, err = openLeaf(p.dir, p.name)
	if err != nil {
		return Version{}, err
	}
	defer file.Close()
	version, err = privilegedVersion(ctx, file)
	return version, err
}
func (p *PreparedPrivileged) Published() bool { return p.published }
func (p *PreparedPrivileged) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	if p.tempName != "" && !p.published {
		_ = removeLeaf(p.dir, p.tempName)
	}
	return p.dir.Close()
}

// Keep privileged hashing bounded even if an external writer grows the file
// after its first stat; ordinary large-file metadata has a different limit.
type privilegedReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r privilegedReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
func privilegedVersion(ctx context.Context, file *os.File) (Version, error) {
	info, err := file.Stat()
	if err != nil {
		return Version{}, err
	}
	if !info.Mode().IsRegular() {
		return Version{}, ErrUnsupported
	}
	if info.Size() > 8<<20 {
		return Version{}, ErrTooLarge
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Nlink != 1 {
		return Version{}, ErrUnsupported
	}
	hash := sha256.New()
	count, err := io.Copy(hash, io.LimitReader(privilegedReader{ctx, file}, (8<<20)+1))
	if err != nil {
		return Version{}, err
	}
	if count > 8<<20 {
		return Version{}, ErrTooLarge
	}
	end, err := file.Stat()
	if err != nil {
		return Version{}, err
	}
	if end.Size() != info.Size() || !end.ModTime().Equal(info.ModTime()) {
		return Version{}, ErrConflict
	}
	return Version{Mtime: info.ModTime().UTC().Format("2006-01-02T15:04:05.000000000Z"), Size: count, ETag: "sha256:" + hex.EncodeToString(hash.Sum(nil)), Identity: fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)}, nil
}
