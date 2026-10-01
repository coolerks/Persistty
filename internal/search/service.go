package search

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"persistty/internal/config"
	"persistty/internal/files"
	"persistty/internal/storage"
	"persistty/internal/toolrunner"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type snapshot struct {
	owner, project string
	query          Query
	result         Result
	bytes          int64
}
type previewState struct {
	mu             sync.Mutex
	owner, project string
	preview        Preview
	bytes          int64
	cancel         context.CancelFunc
	attempted      bool
	released       bool
}
type Service struct {
	Store    *storage.Store
	Runner   *toolrunner.Runner
	Config   config.Config
	mu       sync.Mutex
	searches map[string]*snapshot
	previews map[string]*previewState
}

func New(store *storage.Store, runner *toolrunner.Runner, cfg config.Config) *Service {
	return &Service{Store: store, Runner: runner, Config: cfg, searches: make(map[string]*snapshot), previews: make(map[string]*previewState)}
}
func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("random source unavailable")
	}
	return hex.EncodeToString(b)
}
func excluded(name string, values []string) bool {
	for _, v := range values {
		if name == v {
			return true
		}
	}
	return false
}
func (q Query) Valid() bool {
	if q.ProjectVersion < 1 || q.Pattern == "" || len(q.Pattern) > 4096 || strings.ContainsAny(q.Pattern, "\x00\r\n") || len(q.FolderID) > 128 || !files.ValidRelative(q.Path, true) || q.Path != "" && q.FolderID == "" || len(q.Include) > 32 || len(q.Exclude) > 32 {
		return false
	}
	for _, glob := range append(append([]string{}, q.Include...), q.Exclude...) {
		if glob == "" || len(glob) > 256 || strings.ContainsAny(glob, "\x00\r\n") || strings.HasPrefix(glob, "!") {
			return false
		}
	}
	return true
}
func (s *Service) pruneLocked() {
	now := time.Now()
	for id, v := range s.searches {
		t, _ := time.Parse(time.RFC3339Nano, v.result.ExpiresAt)
		if !t.After(now) {
			delete(s.searches, id)
		}
	}
	for id, v := range s.previews {
		v.mu.Lock()
		t, _ := time.Parse(time.RFC3339Nano, v.preview.ExpiresAt)
		expired := !t.After(now) && v.preview.State != "applying"
		v.mu.Unlock()
		if expired {
			delete(s.previews, id)
		}
	}
}
func (s *Service) pruneReleasedLocked() {
	counts := map[string]int{}
	ids := []string{}
	for id, v := range s.previews {
		v.mu.Lock()
		released := v.released
		v.mu.Unlock()
		if released {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		return s.previews[ids[i]].preview.ExpiresAt > s.previews[ids[j]].preview.ExpiresAt
	})
	for index, id := range ids {
		v := s.previews[id]
		counts[v.owner]++
		if counts[v.owner] > 4 || index >= 128 {
			delete(s.previews, id)
		}
	}
}
func (s *Service) roomLocked(owner string, amount int64) bool {
	s.pruneLocked()
	s.pruneReleasedLocked()
	n := 0
	var sum int64
	for _, v := range s.searches {
		sum += v.bytes
		if v.owner == owner {
			n++
		}
	}
	for _, v := range s.previews {
		v.mu.Lock()
		sum += v.bytes
		if v.owner == owner && !v.released {
			n++
		}
		v.mu.Unlock()
	}
	return n < 4 && sum+amount <= 128<<20
}
func (s *Service) Search(ctx context.Context, owner, project string, q Query) (Result, error) {
	if !q.Valid() {
		return Result{}, ErrInvalid
	}
	release, err := s.Runner.Acquire(ctx)
	if err != nil {
		return Result{}, err
	}
	defer release()
	ctx, cancel := context.WithTimeout(ctx, s.Config.ToolTimeout())
	defer cancel()
	p, err := s.Store.Project(ctx, project)
	if err != nil {
		return Result{}, err
	}
	if p.Version != q.ProjectVersion {
		return Result{}, storage.ErrConflict
	}
	if q.FolderID != "" {
		found := false
		for _, f := range p.Folders {
			if f.ID == q.FolderID {
				found = true
			}
		}
		if !found {
			return Result{}, storage.ErrNotFound
		}
	}
	dir, clean, err := s.Runner.Stage(s.Config.ToolStagingPath())
	if err != nil {
		return Result{}, err
	}
	defer clean()
	// Validate the pattern even if there are no candidate text files.
	if _, err = engine(ctx, s.Runner, dir, q, "", nil); err != nil {
		return Result{}, err
	}
	result := Result{ID: NewID(), ProjectVersion: p.Version, Files: []File{}, Skipped: []Skipped{}, ExpiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339Nano)}
	options := s.Config.SearchOptions()
	seen := map[string]bool{}
	var size, scanned, controlBytes int64
	entriesSeen := 0
	matches := 0
	skip := func(folder, p, reason string) {
		if len(result.Skipped) < 1000 {
			result.Skipped = append(result.Skipped, Skipped{folder, p, reason})
		} else {
			result.Truncated = true
		}
	}
	for index, folder := range p.Folders {
		if entriesSeen >= options.MaxEntries {
			result.Truncated = true
			break
		}
		if q.FolderID != "" && q.FolderID != folder.ID {
			continue
		}
		root, err := s.Store.RegisteredFolder(ctx, project, folder.ID, p.Version)
		if err != nil {
			return Result{}, err
		}
		tree := filepath.Join(dir, "tree-"+hex.EncodeToString([]byte{byte(index)}))
		if err = os.Mkdir(tree, 0700); err != nil {
			return Result{}, err
		}
		err = files.WalkSnapshot(ctx, root, "", options.MaxEntries-entriesSeen, func(relative string, e files.Entry) (bool, error) {
			entriesSeen++
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
						controlBytes += int64(b.Len())
						if controlBytes > options.MaxBytes {
							return false, toolrunner.ErrLimit
						}
						if err = os.MkdirAll(filepath.Join(target, "info"), 0700); err != nil {
							return false, err
						}
						if err = os.WriteFile(filepath.Join(target, "info/exclude"), b.Bytes(), 0600); err != nil {
							return false, err
						}
					} else if !os.IsNotExist(err) {
						skip(folder.ID, source, "ignore_unavailable")
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
				skip(folder.ID, relative, "unsupported")
				return false, nil
			}
			content := []byte{}
			if e.Name == ".gitignore" || e.Name == ".ignore" || e.Name == ".rgignore" {
				var b bytes.Buffer
				_, _, err := files.CopySnapshot(ctx, root, relative, &b, 1<<20)
				if err != nil {
					return false, err
				}
				controlBytes += int64(b.Len())
				if controlBytes > options.MaxBytes {
					return false, toolrunner.ErrLimit
				}
				content = b.Bytes()
			}
			return false, os.WriteFile(target, content, 0600)
		})
		if err != nil {
			if errors.Is(err, files.ErrTooLarge) {
				result.Truncated = true
			} else {
				return Result{}, err
			}
		}
		args := []string{"--no-config", "--files", "--hidden", "--null", "--glob", "!.git/**", "--glob", "!**/.git/**", "--", "."}
		out, code, err := s.Runner.Run(ctx, "rg", tree, nil, args...)
		if err != nil {
			return Result{}, err
		}
		if code != 0 && code != 1 {
			return Result{}, toolrunner.ErrUnavailable
		}
		allowed := map[string]bool{}
		for _, raw := range bytes.Split(out, []byte{0}) {
			relative := strings.TrimPrefix(string(raw), "./")
			if files.ValidRelative(relative, false) {
				allowed[relative] = true
			}
		}
		// Globs are applied to a second discovery, then intersected with the ignore set.
		if len(q.Include)+len(q.Exclude) > 0 {
			filtered := []string{"--no-config", "--files", "--hidden", "--null"}
			for _, glob := range q.Include {
				filtered = append(filtered, "--glob", glob)
			}
			for _, glob := range q.Exclude {
				filtered = append(filtered, "--glob", "!"+glob)
			}
			filtered = append(filtered, "--glob", "!.git/**", "--glob", "!**/.git/**", "--", ".")
			out, code, err = s.Runner.Run(ctx, "rg", tree, nil, filtered...)
			if err != nil {
				return Result{}, err
			}
			if code != 0 && code != 1 {
				return Result{}, ErrInvalid
			}
		} else {
			out = []byte(strings.Join(sortedKeys(allowed), "\x00"))
		}
		for _, raw := range bytes.Split(out, []byte{0}) {
			relative := strings.TrimPrefix(string(raw), "./")
			if !allowed[relative] || q.Path != "" && relative != q.Path && !strings.HasPrefix(relative, q.Path+"/") {
				continue
			}
			var b bytes.Buffer
			version, _, err := files.CopySnapshot(ctx, root, relative, &b, 8<<20)
			if err != nil {
				if errors.Is(err, files.ErrTooLarge) || errors.Is(err, files.ErrUnsupported) {
					skip(folder.ID, relative, "unsupported")
					continue
				}
				return Result{}, err
			}
			if seen[version.Identity] {
				continue
			}
			seen[version.Identity] = true
			text := b.String()
			if !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
				skip(folder.ID, relative, "binary")
				continue
			}
			if scanned+int64(len(text)) > options.MaxBytes {
				result.Truncated = true
				break
			}
			scanned += int64(len(text))
			found, err := engine(ctx, s.Runner, dir, q, text, nil)
			if err != nil {
				return Result{}, err
			}
			if len(found) == 0 {
				continue
			}
			file := File{ID: NewID(), FolderID: folder.ID, Path: relative, Version: version, Matches: []Match{}, Content: text}
			for _, v := range found {
				if matches >= options.MaxResults {
					result.Truncated = true
					break
				}
				v.Match.ID = NewID()
				file.Matches = append(file.Matches, v.Match)
				matches++
			}
			if len(file.Matches) > 0 {
				result.Files = append(result.Files, file)
				size += int64(len(text))
			}
			if result.Truncated || len(result.Files) >= 2000 {
				result.Truncated = true
				break
			}
		}
		if result.Truncated {
			break
		}
	}
	current, err := s.Store.Project(ctx, project)
	if err != nil {
		return Result{}, err
	}
	if current.Version != p.Version {
		return Result{}, storage.ErrConflict
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	encoded, _ := json.Marshal(result)
	size += int64(len(encoded))
	if !s.roomLocked(owner, size) {
		return Result{}, toolrunner.ErrCapacity
	}
	s.searches[result.ID] = &snapshot{owner, project, q, result, size}
	return result, nil
}
func (s *Service) getSearch(owner, project, id string) (*snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked()
	v := s.searches[id]
	if v == nil || v.owner != owner || v.project != project {
		return nil, ErrExpired
	}
	return v, nil
}
func (s *Service) CancelSearch(owner, project, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.searches[id]
	if v == nil || v.owner != owner || v.project != project {
		return ErrExpired
	}
	delete(s.searches, id)
	return nil
}
func (s *Service) Preview(ctx context.Context, owner, project string, input PreviewInput) (Preview, error) {
	source, err := s.getSearch(owner, project, input.SearchID)
	if err != nil {
		return Preview{}, err
	}
	if input.ProjectVersion != source.query.ProjectVersion || len(input.Replacement) > 4096 || strings.ContainsRune(input.Replacement, 0) {
		return Preview{}, ErrInvalid
	}
	p, err := s.Store.Project(ctx, project)
	if err != nil {
		return Preview{}, err
	}
	if p.Version != input.ProjectVersion {
		return Preview{}, storage.ErrConflict
	}
	selected, err := selection(input.SelectedMatchIDs, 5000)
	if err != nil {
		return Preview{}, err
	}
	release, err := s.Runner.Acquire(ctx)
	if err != nil {
		return Preview{}, err
	}
	defer release()
	dir, clean, err := s.Runner.Stage(s.Config.ToolStagingPath())
	if err != nil {
		return Preview{}, err
	}
	defer clean()
	preview := Preview{ID: NewID(), ProjectVersion: p.Version, Files: []PreviewFile{}, Results: []Applied{}, State: "ready", ExpiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339Nano)}
	var amount int64
	used := 0
	for _, file := range source.result.Files {
		count := 0
		for _, m := range file.Matches {
			if selected[m.ID] {
				count++
			}
		}
		if count == 0 {
			continue
		}
		values, err := engine(ctx, s.Runner, dir, source.query, file.Content, &input.Replacement)
		if err != nil {
			return Preview{}, err
		}
		expandedByOffset := map[int]expanded{}
		for _, e := range values {
			expandedByOffset[e.Match.Start] = e
		}
		var b strings.Builder
		offset := 0
		for _, m := range file.Matches {
			if !selected[m.ID] {
				continue
			}
			v, ok := expandedByOffset[m.Start]
			if !ok || v.Match.End != m.End || m.Start < offset {
				return Preview{}, toolrunner.ErrUnavailable
			}
			b.WriteString(file.Content[offset:m.Start])
			b.WriteString(v.Replacement)
			offset = m.End
			used++
			if b.Len() > 8<<20 {
				return Preview{}, files.ErrTooLarge
			}
		}
		b.WriteString(file.Content[offset:])
		if b.Len() > 8<<20 {
			return Preview{}, files.ErrTooLarge
		}
		modified := b.String()
		hash := sha256.Sum256([]byte(modified))
		preview.Files = append(preview.Files, PreviewFile{file.ID, file.FolderID, file.Path, file.Version, count, "sha256:" + hex.EncodeToString(hash[:]), file.Content, modified})
		amount += int64(len(file.Content) + len(modified))
	}
	if used != len(selected) {
		return Preview{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	encoded, _ := json.Marshal(preview)
	amount += int64(len(encoded))
	if !s.roomLocked(owner, amount) {
		return Preview{}, toolrunner.ErrCapacity
	}
	s.previews[preview.ID] = &previewState{owner: owner, project: project, preview: preview, bytes: amount}
	return preview, nil
}
func (s *Service) getPreview(owner, project, id string) (*previewState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked()
	v := s.previews[id]
	if v == nil || v.owner != owner || v.project != project {
		return nil, ErrExpired
	}
	return v, nil
}
func (s *Service) PreviewStatus(owner, project, id string) (Preview, error) {
	v, err := s.getPreview(owner, project, id)
	if err != nil {
		return Preview{}, err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	out := v.preview
	out.Files = append([]PreviewFile{}, out.Files...)
	out.Results = append([]Applied{}, out.Results...)
	return out, nil
}
func (s *Service) Comparison(owner, project, id, fileID string) (Comparison, error) {
	v, err := s.getPreview(owner, project, id)
	if err != nil {
		return Comparison{}, err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.released {
		return Comparison{}, ErrExpired
	}
	for _, f := range v.preview.Files {
		if f.ID == fileID {
			return Comparison{f.ID, f.Path, f.Original, f.Modified}, nil
		}
	}
	return Comparison{}, ErrInvalid
}
func (s *Service) CancelPreview(owner, project, id string) error {
	v, err := s.getPreview(owner, project, id)
	if err != nil {
		return err
	}
	v.mu.Lock()
	if v.cancel != nil {
		v.cancel()
	}
	if v.preview.State == "ready" {
		v.preview.State = "cancelled"
		v.attempted = true
	}
	if v.preview.State != "applying" {
		v.released = true
		for i := range v.preview.Files {
			v.preview.Files[i].Original = ""
			v.preview.Files[i].Modified = ""
		}
		encoded, _ := json.Marshal(v.preview)
		v.bytes = int64(len(encoded))
	}

	v.mu.Unlock()
	s.mu.Lock()
	s.pruneReleasedLocked()
	s.mu.Unlock()
	return nil
}
func selection(ids []string, max int) (map[string]bool, error) {
	if len(ids) < 1 || len(ids) > max {
		return nil, ErrInvalid
	}
	set := map[string]bool{}
	for _, id := range ids {
		if len(id) != 32 || set[id] {
			return nil, ErrInvalid
		}
		set[id] = true
	}
	return set, nil
}
func (s *Service) Apply(ctx context.Context, owner, project, id string, input ApplyInput) (Preview, error) {
	p, err := s.getPreview(owner, project, id)
	if err != nil {
		return Preview{}, err
	}
	selected, err := selection(input.SelectedFileIDs, 2000)
	if err != nil {
		return Preview{}, err
	}
	protected := map[string]bool{}
	for _, id := range input.ProtectedFileIDs {
		protected[id] = true
	}
	if len(protected) > 2000 {
		return Preview{}, ErrInvalid
	}
	p.mu.Lock()
	if input.ProjectVersion != p.preview.ProjectVersion {
		p.mu.Unlock()
		return Preview{}, storage.ErrConflict
	}
	if p.attempted {
		p.mu.Unlock()
		return s.PreviewStatus(owner, project, id)
	}
	known := 0
	for _, f := range p.preview.Files {
		if selected[f.ID] {
			known++
		}
	}
	if known != len(selected) {
		p.mu.Unlock()
		return Preview{}, ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	p.cancel = cancel
	p.attempted = true
	p.preview.State = "applying"
	targets := p.preview.Files
	p.mu.Unlock()
	defer cancel()
	for _, f := range targets {
		result := Applied{ID: f.ID, State: "skipped", Reason: "not_selected"}
		if selected[f.ID] {
			if protected[f.ID] {
				result.Reason = "unsaved_input"
			} else if ctx.Err() != nil {
				result.Reason = "cancelled"
			} else {
				var version files.Version
				err := s.Store.WithRegisteredFolder(ctx, project, f.FolderID, input.ProjectVersion, func(root storage.RegisteredFolder) error {
					var e error
					version, e = files.Save(ctx, root, f.Path, f.Version, f.Modified)
					return e
				})
				if err == nil {
					result.State = "applied"
					result.Reason = ""
					result.Version = &version
				} else if errors.Is(err, files.ErrConflict) || errors.Is(err, storage.ErrConflict) || errors.Is(err, files.ErrRootChanged) || errors.Is(err, storage.ErrNotFound) || os.IsNotExist(err) {
					result.State = "conflict"
					result.Reason = "version_changed"
				} else {
					result.State = "error"
					result.Reason = "save_failed"
				}
			}
		}
		p.mu.Lock()
		p.preview.Results = append(p.preview.Results, result)
		p.mu.Unlock()
	}
	p.mu.Lock()
	p.preview.State = "completed"
	p.cancel = nil
	out := p.preview
	out.Results = append([]Applied{}, out.Results...)
	p.mu.Unlock()
	return out, nil
}
