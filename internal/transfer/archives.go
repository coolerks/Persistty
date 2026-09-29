package transfer

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"persistty/internal/files"
	"persistty/internal/storage"
)

type ArchiveInput struct {
	ProjectID      string `json:"project_id"`
	FolderID       string `json:"folder_id"`
	ProjectVersion int64  `json:"project_version"`
	Path           string `json:"path"`
}

func (s *Service) archiveTemp(id string) string  { return filepath.Join(s.dir, "archive-"+id+".tmp") }
func (s *Service) archiveReady(id string) string { return filepath.Join(s.dir, "archive-"+id+".zip") }

func (s *Service) recoverArchives(ctx context.Context) error {
	if err := s.store.FailPendingArchives(ctx); err != nil {
		return err
	}
	if err := s.CleanExpiredArchives(ctx); err != nil {
		return err
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "download-") && !strings.HasPrefix(name, "archive-") {
			continue
		}
		if !strings.HasSuffix(name, ".tmp") {
			continue
		}
		if !entry.Type().IsRegular() {
			return ErrInvalid
		}
		if err = os.Remove(filepath.Join(s.dir, name)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) CleanExpiredArchives(ctx context.Context) error {
	for {
		ids, err := s.store.ExpiredArchives(ctx)
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
			for _, name := range []string{s.archiveTemp(id), s.archiveReady(id)} {
				if err = os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
					return err
				}
			}
			if err = s.store.DeleteArchive(ctx, id); err != nil {
				return err
			}
		}
	}
}

func (s *Service) CreateArchive(ctx context.Context, input ArchiveInput) (storage.ArchiveRecord, error) {
	if err := s.CleanExpiredArchives(ctx); err != nil {
		return storage.ArchiveRecord{}, err
	}
	if input.ProjectID == "" || input.FolderID == "" || input.ProjectVersion < 1 || !files.ValidRelative(input.Path, true) {
		return storage.ArchiveRecord{}, ErrInvalid
	}
	if _, err := s.store.RegisteredFolder(ctx, input.ProjectID, input.FolderID, input.ProjectVersion); err != nil {
		return storage.ArchiveRecord{}, err
	}
	id, err := newID()
	if err != nil {
		return storage.ArchiveRecord{}, err
	}
	now := time.Now().UTC()
	item := storage.ArchiveRecord{ID: id, ProjectID: input.ProjectID, FolderID: input.FolderID, ProjectVersion: input.ProjectVersion, RelativePath: input.Path, CreatedAt: now.Format("2006-01-02T15:04:05.000000000Z"), ExpiresAt: now.Add(s.cfg.ArchiveDuration()).Format("2006-01-02T15:04:05.000000000Z"), Status: "pending"}
	if err = s.store.CreateArchive(ctx, item); err != nil {
		return storage.ArchiveRecord{}, err
	}
	jobCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	s.archiveMu.Lock()
	s.archiveCancels[id] = cancel
	s.archiveMu.Unlock()
	go s.runArchive(jobCtx, item)
	return item, nil
}

func (s *Service) runArchive(ctx context.Context, item storage.ArchiveRecord) {
	defer func() {
		s.archiveMu.Lock()
		if cancel := s.archiveCancels[item.ID]; cancel != nil {
			cancel()
			delete(s.archiveCancels, item.ID)
		}
		s.archiveMu.Unlock()
	}()
	fail := func(code string) {
		os.Remove(s.archiveTemp(item.ID))
		os.Remove(s.archiveReady(item.ID))
		finishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.store.FinishArchive(finishCtx, item.ID, "failed", 0, code)
	}
	root, err := s.store.RegisteredFolder(ctx, item.ProjectID, item.FolderID, item.ProjectVersion)
	if err != nil {
		fail("unavailable")
		return
	}
	stage, err := os.OpenFile(s.archiveTemp(item.ID), os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		fail("unavailable")
		return
	}
	_, _, err = files.WriteZIP(ctx, root, item.RelativePath, stage, s.cfg.MaxBatchBytes())
	if err == nil {
		err = stage.Sync()
	}
	closeErr := stage.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		fail(archiveErrorCode(err))
		return
	}
	info, err := os.Stat(s.archiveTemp(item.ID))
	if err != nil || info.Size() > s.cfg.MaxStagingBytes() {
		fail("too_large")
		return
	}
	err = s.store.WithRegisteredFolder(ctx, item.ProjectID, item.FolderID, item.ProjectVersion, func(_ storage.RegisteredFolder) error {
		if err := os.Rename(s.archiveTemp(item.ID), s.archiveReady(item.ID)); err != nil {
			return err
		}
		return s.store.FinishArchive(ctx, item.ID, "ready", info.Size(), "")
	})
	if err != nil {
		fail(archiveErrorCode(err))
	}
}

func archiveErrorCode(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, files.ErrTooLarge):
		return "too_large"
	case errors.Is(err, files.ErrUnsupported):
		return "unsupported"
	case errors.Is(err, files.ErrConflict), errors.Is(err, files.ErrRootChanged), errors.Is(err, storage.ErrConflict):
		return "conflict"
	default:
		return "unavailable"
	}
}

func (s *Service) ArchiveStatus(ctx context.Context, id string) (storage.ArchiveRecord, error) {
	if !batchPattern.MatchString(id) {
		return storage.ArchiveRecord{}, ErrInvalid
	}
	item, err := s.store.Archive(ctx, id)
	if err != nil {
		return item, err
	}
	expires, err := time.Parse(time.RFC3339Nano, item.ExpiresAt)
	if err != nil {
		return item, err
	}
	if !time.Now().Before(expires) {
		return item, ErrExpired
	}
	return item, nil
}

func (s *Service) ArchiveDownload(ctx context.Context, id string) (*Download, error) {
	item, err := s.ArchiveStatus(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.Status != "ready" {
		return nil, ErrNotReady
	}
	root, err := s.store.RegisteredFolder(ctx, item.ProjectID, item.FolderID, item.ProjectVersion)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(s.archiveReady(id), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != item.Size {
		file.Close()
		return nil, ErrInvalid
	}
	name := path.Base(item.RelativePath)
	if item.RelativePath == "" {
		name = filepath.Base(root.Path)
	}
	return &Download{File: file, Filename: name + ".zip", Size: item.Size}, nil
}

func (s *Service) CancelArchive(ctx context.Context, id string) error {
	if !batchPattern.MatchString(id) {
		return ErrInvalid
	}
	if _, err := s.ArchiveStatus(ctx, id); err != nil {
		return err
	}
	if err := s.store.CancelArchive(ctx, id); err != nil {
		return err
	}
	s.archiveMu.Lock()
	cancel := s.archiveCancels[id]
	s.archiveMu.Unlock()
	if cancel != nil {
		cancel()
	}
	for _, name := range []string{s.archiveTemp(id), s.archiveReady(id)} {
		if err := os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
