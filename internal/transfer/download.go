package transfer

import (
	"context"
	"os"
	"path"

	"persistty/internal/files"
	"persistty/internal/storage"
)

type Download struct {
	File        *os.File
	Filename    string
	Size        int64
	cleanupPath string
}

func (d *Download) Close() error {
	err := d.File.Close()
	var removeErr error
	if d.cleanupPath != "" {
		removeErr = os.Remove(d.cleanupPath)
	}
	if err != nil {
		return err
	}
	return removeErr
}

func (s *Service) DownloadFile(ctx context.Context, root storage.RegisteredFolder, relative string) (*Download, error) {
	if !files.ValidRelative(relative, false) {
		return nil, files.ErrInvalidPath
	}
	stage, err := os.CreateTemp(s.dir, "download-*.tmp")
	if err != nil {
		return nil, err
	}
	cleanup := func() { stage.Close(); os.Remove(stage.Name()) }
	version, err := files.CopyVerified(ctx, root, relative, stage, s.cfg.MaxFileBytes())
	if err != nil {
		cleanup()
		return nil, err
	}
	if err = stage.Sync(); err != nil {
		cleanup()
		return nil, err
	}
	if _, err = stage.Seek(0, 0); err != nil {
		cleanup()
		return nil, err
	}
	if _, err = s.store.RegisteredFolder(ctx, root.ProjectID, root.FolderID, root.ProjectVersion); err != nil {
		cleanup()
		return nil, err
	}
	return &Download{File: stage, Filename: path.Base(relative), Size: version.Size, cleanupPath: stage.Name()}, nil
}
