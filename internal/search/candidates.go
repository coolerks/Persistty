package search

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path"
	"persistty/internal/files"
	"persistty/internal/storage"
	"persistty/internal/toolrunner"
)

type discoveryBudget struct {
	entries      int
	controlBytes int64
	truncated    bool
}

// Discover in memory and prune before entering ignored directories. Ordinary
// contents and thousands of empty staging files are unnecessary for names.
func (s *Service) discoverCandidates(ctx context.Context, root storage.RegisteredFolder, budget *discoveryBudget, skip func(string, string)) (map[string]bool, error) {
	allowed := map[string]bool{}
	err := s.walkCandidates(ctx, root, budget, skip, func(relative string, e files.Entry, _ files.SnapshotCopy) (bool, error) {
		if e.Kind == "file" {
			allowed[relative] = true
		}
		return e.Kind == "directory", nil
	})
	return allowed, err
}

func (s *Service) walkCandidates(ctx context.Context, root storage.RegisteredFolder, budget *discoveryBudget, skip func(string, string), visit func(string, files.Entry, files.SnapshotCopy) (bool, error)) error {
	options := s.Config.SearchOptions()
	scopes := map[string]ignoreRules{}
	rulesBudget := ignoreBudget{}
	controlFailed := false
	outputBytes := 0
	prepare := func(directory string, entries []files.Entry, copy func(string, io.Writer, int64) (files.Version, os.FileMode, error)) error {
		parent := path.Dir(directory)
		if parent == "." {
			parent = ""
		}
		rules := scopes[parent]
		if directory == "" {
			rules = ignoreRules{}
		}
		kinds := map[string]string{}
		for _, e := range entries {
			kinds[e.Name] = e.Kind
			if e.Name == ".git" && e.Kind == "directory" {
				rules[0], rules[1] = nil, nil
			}
		}
		var err error
		scopes[directory], err = loadIgnoreRules(ctx, directory, rules, &rulesBudget, func(filename string) ([]byte, error) {
			var b bytes.Buffer
			var err error
			if filename == ".git/info/exclude" {
				if kinds[".git"] != "directory" {
					return nil, nil
				}
				_, _, err = files.CopySnapshot(ctx, root, path.Join(directory, filename), &b, 1<<20)
				if os.IsNotExist(err) {
					return nil, nil
				}
			} else {
				if kinds[filename] != "file" {
					return nil, nil
				}
				_, _, err = copy(filename, &b, 1<<20)
			}
			if err != nil {
				controlFailed = true
				return nil, err
			}
			budget.controlBytes += int64(b.Len())
			if budget.controlBytes > options.MaxBytes {
				return nil, toolrunner.ErrLimit
			}
			return b.Bytes(), nil
		})
		return err
	}
	err := files.WalkSnapshotPrepared(ctx, root, "", options.MaxEntries-budget.entries, prepare, func(relative string, e files.Entry, copy files.SnapshotCopy) (bool, error) {
		budget.entries++
		if e.Name == ".git" || e.Kind == "directory" && excluded(e.Name, options.ExcludeDirectories) {
			return false, nil
		}
		if e.Kind != "directory" && e.Kind != "file" {
			skip(relative, "unsupported")
			return false, nil
		}
		parent := path.Dir(relative)
		if parent == "." {
			parent = ""
		}
		ignored, err := ignoredName(ctx, relative, e.Kind == "directory", scopes[parent], &rulesBudget)
		if err != nil || ignored {
			return false, err
		}
		if e.Kind == "directory" {
			return visit(relative, e, copy)
		}
		outputBytes += len(relative) + 1
		if outputBytes > 32<<20 {
			return false, toolrunner.ErrLimit
		}
		return visit(relative, e, copy)
	})
	if errors.Is(err, files.ErrTooLarge) && !controlFailed {
		budget.truncated = true
		err = nil
	}
	return err
}
