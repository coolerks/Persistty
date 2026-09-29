package transfer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"

	"persistty/internal/config"
	"persistty/internal/files"
	"persistty/internal/storage"
)

var ErrHash = errors.New("transfer hash mismatch")
var ErrExpired = errors.New("transfer expired")
var ErrInvalid = errors.New("invalid transfer")
var ErrNotReady = errors.New("transfer not ready")
var batchPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
var shaPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type Service struct {
	cfg            config.Config
	store          *storage.Store
	dir            string
	mu             sync.Mutex
	archiveMu      sync.Mutex
	archiveCancels map[string]context.CancelFunc
}

type UploadInput struct {
	ProjectID       string         `json:"project_id"`
	FolderID        string         `json:"folder_id"`
	ProjectVersion  int64          `json:"project_version"`
	Path            string         `json:"path"`
	BatchID         string         `json:"batch_id"`
	Size            int64          `json:"size"`
	SHA256          string         `json:"sha256"`
	ExpectedVersion *files.Version `json:"expected_version"`
}

type UploadState struct {
	ID         string              `json:"id"`
	Status     string              `json:"status"`
	Size       int64               `json:"size"`
	ChunkBytes int64               `json:"chunk_bytes"`
	Received   []int64             `json:"received"`
	ExpiresAt  string              `json:"expires_at"`
	Result     *files.ImportResult `json:"result"`
}

func New(ctx context.Context, cfg config.Config, store *storage.Store) (*Service, error) {
	dir := cfg.StagingPath()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 || !ok || int(stat.Uid) != os.Geteuid() {
		return nil, ErrInvalid
	}
	s := &Service{cfg: cfg, store: store, dir: dir, archiveCancels: make(map[string]context.CancelFunc)}
	if err = s.CleanExpired(ctx); err != nil {
		return nil, err
	}
	if err = s.recoverArchives(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func newID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
func (s *Service) partPath(id string) string { return filepath.Join(s.dir, id+".part") }

func (s *Service) CleanExpired(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		ids, err := s.store.ExpiredUploads(ctx)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		for _, id := range ids {
			if !batchPattern.MatchString(id) {
				return ErrInvalid
			}
			if err = os.Remove(s.partPath(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			if err = s.store.DeleteUpload(ctx, id); err != nil {
				return err
			}
		}
	}
}

func (s *Service) Start(ctx context.Context, input UploadInput) (UploadState, error) {
	if err := s.CleanExpired(ctx); err != nil {
		return UploadState{}, err
	}
	if input.ProjectID == "" || input.FolderID == "" || input.ProjectVersion < 1 || !files.ValidRelative(input.Path, false) || input.Size < 0 || input.Size > s.cfg.MaxFileBytes() || !shaPattern.MatchString(input.SHA256) || input.BatchID != "" && !batchPattern.MatchString(input.BatchID) {
		return UploadState{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if input.BatchID == "" {
		var err error
		input.BatchID, err = newID()
		if err != nil {
			return UploadState{}, err
		}
	}
	id, err := newID()
	if err != nil {
		return UploadState{}, err
	}
	var expectedJSON *string
	if input.ExpectedVersion != nil {
		raw, e := json.Marshal(input.ExpectedVersion)
		if e != nil {
			return UploadState{}, e
		}
		value := string(raw)
		expectedJSON = &value
	}
	now := time.Now().UTC()
	item := storage.UploadRecord{ID: id, ProjectID: input.ProjectID, FolderID: input.FolderID, ProjectVersion: input.ProjectVersion, RelativePath: input.Path, BatchID: input.BatchID, Size: input.Size, SHA256: input.SHA256, ChunkBytes: s.cfg.ChunkBytes(), ExpectedVersionJSON: expectedJSON, CreatedAt: now.Format("2006-01-02T15:04:05.000000000Z"), ExpiresAt: now.Add(s.cfg.UploadDuration()).Format("2006-01-02T15:04:05.000000000Z"), Status: "pending"}
	err = s.store.WithRegisteredFolder(ctx, input.ProjectID, input.FolderID, input.ProjectVersion, func(root storage.RegisteredFolder) error { return files.CheckTarget(root, input.Path) })
	if err != nil {
		return UploadState{}, err
	}
	part, err := os.OpenFile(s.partPath(id), os.O_CREATE|os.O_EXCL|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return UploadState{}, err
	}
	defer func() {
		if err != nil {
			os.Remove(s.partPath(id))
		}
	}()
	if err = part.Truncate(input.Size); err == nil {
		err = part.Sync()
	}
	closeErr := part.Close()
	if err != nil {
		return UploadState{}, err
	}
	if closeErr != nil {
		err = closeErr
		return UploadState{}, err
	}
	if err = s.store.ReserveUpload(ctx, item, s.cfg.MaxBatchBytes(), s.cfg.MaxStagingBytes()); err != nil {
		return UploadState{}, err
	}
	return UploadState{ID: id, Status: "pending", Size: input.Size, ChunkBytes: item.ChunkBytes, Received: []int64{}, ExpiresAt: item.ExpiresAt}, nil
}

func (s *Service) state(ctx context.Context, id string) (storage.UploadRecord, UploadState, error) {
	if !batchPattern.MatchString(id) {
		return storage.UploadRecord{}, UploadState{}, ErrInvalid
	}
	item, err := s.store.Upload(ctx, id)
	if err != nil {
		return item, UploadState{}, err
	}
	expires, err := time.Parse(time.RFC3339Nano, item.ExpiresAt)
	if err != nil {
		return item, UploadState{}, err
	}
	if !time.Now().Before(expires) {
		return item, UploadState{}, ErrExpired
	}
	chunks, err := s.store.UploadChunks(ctx, id)
	if err != nil {
		return item, UploadState{}, err
	}
	state := UploadState{ID: id, Status: item.Status, Size: item.Size, ChunkBytes: item.ChunkBytes, Received: make([]int64, 0, len(chunks)), ExpiresAt: item.ExpiresAt}
	for _, chunk := range chunks {
		state.Received = append(state.Received, chunk.Index)
	}
	if item.ResultJSON != nil {
		var result files.ImportResult
		if err = json.Unmarshal([]byte(*item.ResultJSON), &result); err != nil {
			return item, UploadState{}, err
		}
		state.Result = &result
	}
	return item, state, nil
}

func (s *Service) Status(ctx context.Context, id string) (UploadState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, state, err := s.state(ctx, id)
	return state, err
}

func (s *Service) Chunk(ctx context.Context, id string, index int64, body []byte, hash string) (UploadState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, state, err := s.state(ctx, id)
	if err != nil {
		return state, err
	}
	if item.Status != "pending" || index < 0 || index >= ((item.Size+item.ChunkBytes-1)/item.ChunkBytes) || !shaPattern.MatchString(hash) {
		return state, ErrInvalid
	}
	expectedSize := item.ChunkBytes
	if left := item.Size - index*item.ChunkBytes; left < expectedSize {
		expectedSize = left
	}
	if int64(len(body)) != expectedSize {
		return state, ErrInvalid
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != hash {
		return state, ErrHash
	}
	chunks, err := s.store.UploadChunks(ctx, id)
	if err != nil {
		return state, err
	}
	for _, chunk := range chunks {
		if chunk.Index == index {
			if chunk.SHA256 == hash && chunk.Size == int64(len(body)) {
				return state, nil
			}
			return state, storage.ErrConflict
		}
	}
	file, err := os.OpenFile(s.partPath(id), os.O_RDWR|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return state, err
	}
	count, err := file.WriteAt(body, index*item.ChunkBytes)
	if err == nil && count != len(body) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return state, err
	}
	if closeErr != nil {
		return state, closeErr
	}
	if err = s.store.RecordUploadChunk(ctx, id, storage.UploadChunk{Index: index, SHA256: hash, Size: int64(len(body))}); err != nil {
		return state, err
	}
	state.Received = append(state.Received, index)
	return state, nil
}

func (s *Service) Complete(ctx context.Context, id string) (files.ImportResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, state, err := s.state(ctx, id)
	if err != nil {
		return files.ImportResult{}, err
	}
	if state.Result != nil {
		return *state.Result, nil
	}
	if item.Status != "pending" {
		return files.ImportResult{}, ErrInvalid
	}
	count := (item.Size + item.ChunkBytes - 1) / item.ChunkBytes
	if int64(len(state.Received)) != count {
		return files.ImportResult{}, ErrInvalid
	}
	for i, index := range state.Received {
		if index != int64(i) {
			return files.ImportResult{}, ErrInvalid
		}
	}
	part, err := os.OpenFile(s.partPath(id), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return files.ImportResult{}, err
	}
	defer part.Close()
	info, err := part.Stat()
	if err != nil || info.Size() != item.Size {
		return files.ImportResult{}, ErrInvalid
	}
	hash := sha256.New()
	if _, err = io.CopyN(hash, part, item.Size); err != nil {
		return files.ImportResult{}, err
	}
	if hex.EncodeToString(hash.Sum(nil)) != item.SHA256 {
		return files.ImportResult{}, ErrHash
	}
	var expected *files.Version
	if item.ExpectedVersionJSON != nil {
		var value files.Version
		if err = json.Unmarshal([]byte(*item.ExpectedVersionJSON), &value); err != nil {
			return files.ImportResult{}, err
		}
		expected = &value
	}
	var result files.ImportResult
	err = s.store.WithRegisteredFolder(ctx, item.ProjectID, item.FolderID, item.ProjectVersion, func(root storage.RegisteredFolder) error {
		var e error
		result, e = files.ImportPrepared(ctx, root, item.RelativePath, part, item.Size, item.SHA256, expected)
		return e
	})
	if err != nil {
		return result, err
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return result, err
	}
	if err = s.store.FinishUpload(ctx, id, map[bool]string{true: "skipped", false: "completed"}[result.State == "skipped"], string(resultJSON)); err != nil {
		return result, err
	}
	if err = os.Remove(s.partPath(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	return result, nil
}

func (s *Service) Cancel(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, _, err := s.state(ctx, id); err != nil {
		return err
	}
	if err := os.Remove(s.partPath(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return s.store.DeleteUpload(ctx, id)
}
