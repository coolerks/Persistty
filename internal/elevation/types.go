package elevation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"persistty/internal/files"
	"persistty/internal/storage"
)

const MaxContent = 8 << 20
const PolicyPath = "/etc/persistty-elevation/policy.json"
const HelperPath = "/usr/local/libexec/persistty-file-helper"
const BrokerPath = "/usr/local/libexec/persistty-elevatord"

var ErrUnavailable = errors.New("elevation unavailable")
var ErrInvalid = errors.New("invalid elevation request")
var ErrForbidden = errors.New("elevation forbidden")
var ErrExpired = errors.New("elevation expired")
var ErrRateLimited = errors.New("elevation rate limited")
var noncePattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var targetPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type Grant struct {
	ID          string                   `json:"id"`
	SessionHash string                   `json:"session_hash"`
	Root        storage.RegisteredFolder `json:"root"`
	Path        string                   `json:"path"`
	TargetID    string                   `json:"target_id"`
	Expected    files.Version            `json:"expected_version"`
	ContentHash string                   `json:"content_hash"`
	IssuedAt    time.Time                `json:"issued_at"`
	ExpiresAt   time.Time                `json:"expires_at"`
}
type Target struct {
	ID   string `json:"target_id"`
	Path string `json:"target_path"`
}
type Prepared struct {
	ID          string    `json:"id"`
	TargetID    string    `json:"target_id"`
	TargetPath  string    `json:"target_path"`
	ContentHash string    `json:"content_hash"`
	ExpiresAt   time.Time `json:"expires_at"`
	State       string    `json:"state"`
}
type Result struct {
	ID      string         `json:"id"`
	State   string         `json:"state"`
	Code    *string        `json:"code"`
	Version *files.Version `json:"version"`
}
type Publish func(func() (Result, error)) (Result, error)
type Backend interface {
	Target(context.Context, storage.RegisteredFolder, string) (Target, error)
	Run(context.Context, Grant, []byte, []byte, Publish) (Result, error)
	Status(context.Context, Grant) (Result, error)
}

func Digest(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}
func (g Grant) Binding() string { data, _ := json.Marshal(g); return Digest(data) }
func validAbsolute(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && len(path) <= 4096 && !strings.ContainsAny(path, "\x00\r\n") && utf8.ValidString(path)
}
func ValidContent(content []byte) bool {
	return len(content) <= MaxContent && utf8.Valid(content) && !strings.ContainsRune(string(content), 0)
}
func ValidPassword(password []byte) bool {
	return len(password) > 0 && len(password) <= 1024 && !strings.ContainsAny(string(password), "\x00\r\n") && utf8.Valid(password)
}
func (g Grant) Validate(now time.Time) error {
	if !noncePattern.MatchString(g.ID) || !noncePattern.MatchString(g.SessionHash) || !targetPattern.MatchString(g.TargetID) ||
		!validAbsolute(g.Root.Path) || !files.ValidRelative(g.Path, false) || len(g.Path) > 4096 || g.Root.ProjectVersion < 1 || g.Root.ProjectID == "" || g.Root.FolderID == "" ||
		!strings.HasPrefix(g.ContentHash, "sha256:") || !noncePattern.MatchString(strings.TrimPrefix(g.ContentHash, "sha256:")) ||
		g.Expected.Identity == "" || g.Expected.Size < 0 || g.Expected.Size > MaxContent || !strings.HasPrefix(g.Expected.ETag, "sha256:") || !noncePattern.MatchString(strings.TrimPrefix(g.Expected.ETag, "sha256:")) {
		return ErrInvalid
	}
	if _, err := time.Parse(time.RFC3339Nano, g.Expected.Mtime); err != nil {
		return ErrInvalid
	}
	if !g.ExpiresAt.After(g.IssuedAt) || g.ExpiresAt.Sub(g.IssuedAt) > time.Minute || g.IssuedAt.After(now.Add(time.Second)) {
		return ErrInvalid
	}
	if !g.ExpiresAt.After(now) {
		return ErrExpired
	}
	return nil
}
func outcome(id, state, code string) Result {
	r := Result{ID: id, State: state}
	if code != "" {
		r.Code = &code
	}
	return r
}
func ResultValid(r Result, id string) bool {
	if r.ID != id || !noncePattern.MatchString(id) {
		return false
	}
	switch r.State {
	case "applied":
		if r.Code != nil || r.Version == nil || r.Version.Size < 0 || r.Version.Size > MaxContent || r.Version.Identity == "" || !strings.HasPrefix(r.Version.ETag, "sha256:") || !noncePattern.MatchString(strings.TrimPrefix(r.Version.ETag, "sha256:")) {
			return false
		}
		_, err := time.Parse(time.RFC3339Nano, r.Version.Mtime)
		return err == nil
	case "prepared", "executing":
		return r.Code == nil && r.Version == nil
	case "rejected", "cancelled", "expired", "indeterminate":
		return r.Code != nil && len(*r.Code) > 0 && len(*r.Code) < 64 && r.Version == nil
	default:
		return false
	}
}
