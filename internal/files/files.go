package files

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"

	"persistty/internal/storage"
)

var ErrInvalidPath = errors.New("invalid relative path")
var ErrRootChanged = errors.New("registered root changed")
var ErrUnsupported = errors.New("unsupported file type")
var ErrTooLarge = errors.New("file too large")
var ErrRescan = errors.New("directory changed")
var ErrVersionRequired = errors.New("version required")

func ValidRelative(relative string, allowRoot bool) bool {
	if relative == "" {
		return allowRoot
	}
	if !utf8.ValidString(relative) || strings.ContainsRune(relative, 0) || strings.HasPrefix(relative, "/") || path.Clean(relative) != relative {
		return false
	}
	for _, part := range strings.Split(relative, "/") {
		if part == ".." || part == "." || part == "" {
			return false
		}
	}
	return true
}

func CheckTarget(root storage.RegisteredFolder, relative string) error {
	if !ValidRelative(relative, false) {
		return ErrInvalidPath
	}
	dir, err := openMutationParent(root, path.Dir(relative))
	if err != nil {
		return err
	}
	return dir.Close()
}

type Entry struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Size     int64  `json:"size"`
	Mtime    string `json:"mtime"`
	Identity string `json:"identity"`
}
type Listing struct {
	Items          []Entry `json:"items"`
	NextCursor     string  `json:"next_cursor"`
	ProjectVersion int64   `json:"project_version"`
}

func List(folder storage.RegisteredFolder, relative, cursor string, limit int) (Listing, error) {
	if !ValidRelative(relative, true) || limit < 1 || limit > 200 || len(cursor) > 80 {
		return Listing{}, ErrInvalidPath
	}
	openPath := relative
	if openPath == "" {
		openPath = "."
	}
	file, err := openConstrained(folder, openPath, true)
	if err != nil {
		return Listing{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.IsDir() {
		return Listing{}, ErrUnsupported
	}
	entries, err := file.ReadDir(10001)
	if err != nil && !errors.Is(err, io.EOF) {
		return Listing{}, err
	}
	if len(entries) > 10000 {
		return Listing{}, ErrTooLarge
	}
	items := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if !utf8.ValidString(entry.Name()) {
			return Listing{}, ErrUnsupported
		}
		item, e := statEntry(file, entry.Name())
		if e != nil {
			return Listing{}, e
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return Listing{}, ErrUnsupported
	}
	hasher := sha256.New()
	fmt.Fprintf(hasher, "%d:%d:%d:%s\n", stat.Dev, stat.Ino, folder.ProjectVersion, relative)
	for _, item := range items {
		fmt.Fprintf(hasher, "%s\x00%s\x00%d\x00%s\n", item.Name, item.Kind, item.Size, item.Identity)
	}
	revision := hex.EncodeToString(hasher.Sum(nil))
	offset := 0
	if cursor != "" {
		parts := strings.Split(cursor, ":")
		if len(parts) != 2 || parts[1] != revision {
			return Listing{}, ErrRescan
		}
		offset, err = strconv.Atoi(parts[0])
		if err != nil || offset < 0 {
			return Listing{}, ErrInvalidPath
		}
	}
	if offset > len(items) {
		return Listing{}, ErrRescan
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	result := Listing{Items: items[offset:end], ProjectVersion: folder.ProjectVersion}
	if end < len(items) {
		result.NextCursor = strconv.Itoa(end) + ":" + revision
	}
	return result, nil
}

type Version struct {
	Mtime    string `json:"mtime"`
	Size     int64  `json:"size"`
	ETag     string `json:"etag"`
	Identity string `json:"identity"`
}
type Content struct {
	Content string  `json:"content"`
	Version Version `json:"version"`
	Kind    string  `json:"kind"`
}

func OpenDirectory(folder storage.RegisteredFolder, relative string) (*os.File, error) {
	if !ValidRelative(relative, true) {
		return nil, ErrInvalidPath
	}
	if relative == "" {
		relative = "."
	}
	return openConstrained(folder, relative, true)
}

type Metadata struct {
	Kind    string  `json:"kind"`
	Version Version `json:"version"`
}

func ReadMetadata(folder storage.RegisteredFolder, relative string) (Metadata, error) {
	if !ValidRelative(relative, false) {
		return Metadata{}, ErrInvalidPath
	}
	file, err := openConstrained(folder, relative, false)
	if err != nil {
		return Metadata{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Metadata{}, err
	}
	if !info.Mode().IsRegular() {
		return Metadata{}, ErrUnsupported
	}
	version, _, err := versionFromFile(file)
	if err != nil {
		return Metadata{}, err
	}
	return Metadata{Kind: "file", Version: version}, nil
}

func ReadContent(folder storage.RegisteredFolder, relative string) (Content, error) {
	if !ValidRelative(relative, false) {
		return Content{}, ErrInvalidPath
	}
	file, err := openConstrained(folder, relative, false)
	if err != nil {
		return Content{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Content{}, err
	}
	if !info.Mode().IsRegular() {
		return Content{}, ErrUnsupported
	}
	if info.Size() > 8<<20 {
		return Content{}, ErrTooLarge
	}
	bytes, err := io.ReadAll(io.LimitReader(file, 8<<20+1))
	if err != nil {
		return Content{}, err
	}
	if len(bytes) > 8<<20 {
		return Content{}, ErrTooLarge
	}
	finalInfo, err := file.Stat()
	if err != nil {
		return Content{}, err
	}
	if finalInfo.Size() != info.Size() || !finalInfo.ModTime().Equal(info.ModTime()) {
		return Content{}, ErrConflict
	}
	if !utf8.Valid(bytes) || strings.ContainsRune(string(bytes), 0) {
		return Content{}, ErrUnsupported
	}
	hash := sha256.Sum256(bytes)
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return Content{}, ErrUnsupported
	}
	identity := fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)
	return Content{string(bytes), Version{info.ModTime().UTC().Format("2006-01-02T15:04:05.000000000Z"), int64(len(bytes)), "sha256:" + hex.EncodeToString(hash[:]), identity}, "text"}, nil
}
