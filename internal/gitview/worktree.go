package gitview

import (
	"bytes"
	"context"
	"os"
	"path"
	"path/filepath"
	"persistty/internal/files"
	"strings"
	"unicode/utf8"
)

// Build breadth-first so one bounded check-ignore command per depth can prune
// ignored directories before enumerating them. Tracked paths always survive.
func (s *Service) copyWorktree(ctx context.Context, snap *snapshot, size *int64, limit int64) error {
	trackedBytes, err := s.Runner.GitRun(ctx, snap.dir, "ls-files", "--cached", "-z")
	if err != nil {
		return err
	}
	tracked := map[string]bool{}
	ancestors := map[string]bool{}
	for _, raw := range bytes.Split(trackedBytes, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		p := string(raw)
		if !worktreePath(p) {
			return ErrUnavailable
		}
		tracked[p] = true
		for parent := path.Dir(p); parent != "."; parent = path.Dir(parent) {
			ancestors[parent] = true
		}
	}
	type candidate struct {
		relative, tail string
		entry          files.Entry
	}
	copied := map[string]bool{}
	copyFile := func(c candidate, copy files.SnapshotCopy) error {
		if copied[c.tail] {
			return nil
		}
		target := filepath.Join(snap.dir, filepath.FromSlash(c.tail))
		if c.entry.Size > limit-*size {
			return files.ErrTooLarge
		}
		if c.entry.Kind == "unsupported" {
			link, err := files.SnapshotLink(snap.repo.root, c.relative)
			if err != nil || !utf8.ValidString(link) {
				return ErrUnavailable
			}
			if int64(len(link)) > limit-*size {
				return files.ErrTooLarge
			}
			*size += int64(len(link))
			return os.WriteFile(target, []byte(link), 0600)
		}
		var b bytes.Buffer
		var v files.Version
		var mode os.FileMode
		var err error
		if copy != nil {
			v, mode, err = copy(&b, limit-*size)
		} else {
			v, mode, err = files.CopySnapshot(ctx, snap.repo.root, c.relative, &b, limit-*size)
		}
		if err != nil {
			return err
		}
		if v.Identity != c.entry.Identity {
			return files.ErrConflict
		}
		if c.entry.Name == ".gitattributes" && bytes.Contains(b.Bytes(), []byte("filter=")) {
			return ErrUnavailable
		}
		*size += v.Size
		snap.versions[c.tail] = v
		copied[c.tail] = true
		return os.WriteFile(target, b.Bytes(), 0600|(mode&0111))
	}
	selected := map[string]bool{}
	directories := []string{snap.repo.public.Path}
	count := 0
	for depth := 0; len(directories) > 0; depth++ {
		if depth > 32 {
			return files.ErrTooLarge
		}
		candidates := []candidate{}
		var input bytes.Buffer
		for _, directory := range directories {
			entries, err := files.SnapshotEntries(ctx, snap.repo.root, directory)
			if err != nil {
				return err
			}
			for _, e := range entries {
				count++
				if count > s.Config.SearchOptions().MaxEntries {
					return files.ErrTooLarge
				}
				if e.Name == ".git" {
					continue
				}
				if strings.EqualFold(e.Name, ".git") {
					return ErrUnavailable
				}
				relative := path.Join(directory, e.Name)
				tail := strings.TrimPrefix(strings.TrimPrefix(relative, snap.repo.public.Path), "/")
				c := candidate{relative, tail, e}
				// All rules for this breadth are present before the batched ignore query.
				if (e.Name == ".gitignore" || e.Name == ".gitattributes") && e.Kind == "file" {
					if err := copyFile(c, nil); err != nil {
						return err
					}
				}
				candidates = append(candidates, c)
				input.WriteString(tail)
				if e.Kind == "directory" {
					input.WriteByte('/')
				}
				input.WriteByte(0)
				if input.Len() > 4<<20 {
					return files.ErrTooLarge
				}
			}
		}
		ignoredBytes, err := s.Runner.GitIgnored(ctx, snap.dir, input.Bytes())
		if err != nil {
			return err
		}
		ignored := map[string]bool{}
		for _, p := range bytes.Split(ignoredBytes, []byte{0}) {
			ignored[strings.TrimSuffix(string(p), "/")] = true
		}
		next := []string{}
		for _, c := range candidates {
			if err := ctx.Err(); err != nil {
				return err
			}
			if ignored[c.tail] && !tracked[c.tail] && !ancestors[c.tail] {
				continue
			}
			if c.entry.Kind == "directory" {
				if err := os.MkdirAll(filepath.Join(snap.dir, filepath.FromSlash(c.tail)), 0700); err != nil {
					return err
				}
				next = append(next, c.relative)
			}
			selected[c.tail] = true
		}
		directories = next
	}
	// Copy selected leaves through the established directory-scoped safe reader,
	// avoiding a new root/parent traversal for every ordinary file.
	return files.WalkSnapshotWithCopy(ctx, snap.repo.root, snap.repo.public.Path, s.Config.SearchOptions().MaxEntries, func(relative string, e files.Entry, copy files.SnapshotCopy) (bool, error) {
		tail := strings.TrimPrefix(strings.TrimPrefix(relative, snap.repo.public.Path), "/")
		if e.Name == ".git" {
			return false, nil
		}
		if strings.EqualFold(e.Name, ".git") {
			return false, ErrUnavailable
		}
		if !selected[tail] {
			return false, nil
		}
		if e.Kind == "directory" {
			return true, nil
		}
		return false, copyFile(candidate{relative, tail, e}, copy)
	})
}
