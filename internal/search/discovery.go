package search

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"persistty/internal/files"
	"persistty/internal/storage"
	"persistty/internal/toolrunner"
	"strings"
)

type discoveryBudget struct {
	entries      int
	controlBytes int64
	truncated    bool
}

// discover supplies rg only a private tree of placeholders and safely copied ignore controls.
func (s *Service) discover(ctx context.Context, root storage.RegisteredFolder, tree string, budget *discoveryBudget, skip func(string, string)) (map[string]bool, error) {
	return s.discoverTree(ctx, root, tree, budget, skip)
}

func (s *Service) discoverTree(ctx context.Context, root storage.RegisteredFolder, tree string, budget *discoveryBudget, skip func(string, string)) (map[string]bool, error) {
	options := s.Config.SearchOptions()
	if err := os.Mkdir(tree, 0700); err != nil {
		return nil, err
	}
	controlFailed := false
	err := files.WalkSnapshot(ctx, root, "", options.MaxEntries-budget.entries, func(relative string, e files.Entry) (bool, error) {
		budget.entries++
		target := filepath.Join(tree, filepath.FromSlash(relative))
		if e.Kind == "directory" {
			if excluded(e.Name, options.ExcludeDirectories) {
				return false, nil
			}
			if err := os.MkdirAll(target, 0700); err != nil {
				return false, err
			}
			if e.Name == ".git" {
				source := path.Join(relative, "info/exclude")
				var b bytes.Buffer
				_, _, err := files.CopySnapshot(ctx, root, source, &b, 1<<20)
				if err == nil {
					budget.controlBytes += int64(b.Len())
					if budget.controlBytes > options.MaxBytes {
						return false, toolrunner.ErrLimit
					}
					if err = os.MkdirAll(filepath.Join(target, "info"), 0700); err != nil {
						return false, err
					}
					if err = os.WriteFile(filepath.Join(target, "info/exclude"), b.Bytes(), 0600); err != nil {
						return false, err
					}
				} else if !os.IsNotExist(err) {
					controlFailed = true
					skip(source, "ignore_unavailable")
					return false, err
				}
				return false, nil
			}
			return true, nil
		}
		if e.Name == ".git" {
			return false, nil
		}
		if e.Kind != "file" {
			skip(relative, "unsupported")
			return false, nil
		}
		content := []byte{}
		if e.Name == ".gitignore" || e.Name == ".ignore" || e.Name == ".rgignore" {
			var b bytes.Buffer
			_, _, err := files.CopySnapshot(ctx, root, relative, &b, 1<<20)
			if err != nil {
				controlFailed = true
				return false, err
			}
			budget.controlBytes += int64(b.Len())
			if budget.controlBytes > options.MaxBytes {
				return false, toolrunner.ErrLimit
			}
			content = b.Bytes()
		}
		return false, os.WriteFile(target, content, 0600)
	})
	if err != nil {
		if errors.Is(err, files.ErrTooLarge) && !controlFailed {
			budget.truncated = true
		} else {
			return nil, err
		}
	}
	args := []string{"--no-require-git", "--no-config", "--files", "--hidden", "--null", "--glob", "!.git/**", "--glob", "!**/.git/**", "--", "."}
	out, code, err := s.Runner.Run(ctx, "rg", tree, nil, args...)
	if errors.Is(err, toolrunner.ErrUnavailable) || err == nil && code != 0 && code != 1 {
		return nativeNames(ctx, tree)
	}
	if err != nil {
		return nil, err
	}
	if code != 0 && code != 1 {
		return nil, toolrunner.ErrUnavailable
	}
	allowed := map[string]bool{}
	for _, raw := range bytes.Split(out, []byte{0}) {
		relative := strings.TrimPrefix(string(raw), "./")
		if files.ValidRelative(relative, false) {
			allowed[relative] = true
		}
	}
	return allowed, nil
}
