package gitview

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"persistty/internal/config"
	"persistty/internal/files"
	"persistty/internal/storage"
	"persistty/internal/toolrunner"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

type Service struct {
	Store  *storage.Store
	Runner *toolrunner.Runner
	Config config.Config
	key    [32]byte
	mu     sync.Mutex
	known  map[string]repository
}
type repository struct {
	public   Repository
	root     storage.RegisteredFolder
	identity string
}
type snapshot struct {
	dir        string
	repo       repository
	versions   map[string]files.Version
	metadata   map[string]files.Version
	head       string
	branch     string
	directPath string
	githubURL  string
}

func New(store *storage.Store, r *toolrunner.Runner, cfg config.Config) *Service {
	s := &Service{Store: store, Runner: r, Config: cfg, known: map[string]repository{}}
	if _, err := rand.Read(s.key[:]); err != nil {
		panic("random source unavailable")
	}
	return s
}
func (s *Service) discover(ctx context.Context, project string, version int64) ([]repository, bool, error) {
	p, err := s.Store.Project(ctx, project)
	if err != nil {
		return nil, false, err
	}
	if p.Version != version {
		return nil, false, storage.ErrConflict
	}
	result := []repository{}
	seen := map[string]bool{}
	truncated := false
	for _, folder := range p.Folders {
		root, err := s.Store.RegisteredFolder(ctx, project, folder.ID, version)
		if err != nil {
			return nil, false, err
		}
		err = files.WalkSnapshot(ctx, root, "", s.Config.SearchOptions().MaxEntries, func(relative string, e files.Entry) (bool, error) {
			if e.Name == ".git" {
				if seen[e.Identity] {
					return false, nil
				}
				seen[e.Identity] = true
				repoPath := path.Dir(relative)
				if repoPath == "." {
					repoPath = ""
				}
				item := s.repository(project, version, root, repoPath, e).public
				result = append(result, repository{item, root, e.Identity})
				if len(result) > 100 {
					return false, files.ErrTooLarge
				}
				return false, nil
			}
			for _, name := range s.Config.SearchOptions().ExcludeDirectories {
				if name == e.Name && e.Kind == "directory" {
					return false, nil
				}
			}
			return e.Kind == "directory", nil
		})
		if err != nil {
			if err == files.ErrTooLarge {
				truncated = true
				break
			}
			if errors.Is(err, files.ErrUnsupported) {
				return nil, false, ErrUnavailable
			}
			return nil, false, err
		}
	}
	if len(result) > 100 {
		result = result[:100]
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].public.Name < result[j].public.Name })
	s.mu.Lock()
	if len(s.known)+len(result) > 1000 {
		s.known = map[string]repository{}
	}
	for _, r := range result {
		s.known[r.public.ID] = r
	}
	s.mu.Unlock()
	return result, truncated, nil
}
func (s *Service) repository(project string, version int64, root storage.RegisteredFolder, repoPath string, e files.Entry) repository {
	h := hmac.New(sha256.New, s.key[:])
	fmt.Fprintf(h, "%s:%d:%s", project, version, e.Identity)
	item := Repository{ID: hex.EncodeToString(h.Sum(nil)[:16]), FolderID: root.FolderID, Path: repoPath, Name: filepath.Base(root.Path), State: "available"}
	if repoPath != "" {
		item.Name += "/" + repoPath
	}
	if e.Kind != "directory" {
		item.State = "unavailable"
		item.Reason = "根外或链接形式的 Git 元数据尚未注册。"
	}
	return repository{item, root, e.Identity}
}

// Only ancestors of the requested file can own its baseline. Examine each
// registered containing root so overlapping roots still select the innermost repo.
func (s *Service) baselineRepository(ctx context.Context, project string, version int64, absolute string) (*repository, error) {
	p, err := s.Store.Project(ctx, project)
	if err != nil {
		return nil, err
	}
	if p.Version != version {
		return nil, storage.ErrConflict
	}
	var selected *repository
	selectedDepth := -1
	for _, folder := range p.Folders {
		rel, err := filepath.Rel(folder.Path, absolute)
		if err != nil || !files.ValidRelative(filepath.ToSlash(rel), false) {
			continue
		}
		root, err := s.Store.RegisteredFolder(ctx, project, folder.ID, version)
		if err != nil {
			return nil, err
		}
		directory := path.Dir(filepath.ToSlash(rel))
		if directory == "." {
			directory = ""
		}
		for {
			entries, err := files.SnapshotEntries(ctx, root, directory)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for _, e := range entries {
				if e.Name != ".git" {
					continue
				}
				repoRoot := filepath.Join(root.Path, filepath.FromSlash(directory))
				if len(repoRoot) > selectedDepth {
					r := s.repository(project, version, root, directory, e)
					selected = &r
					selectedDepth = len(repoRoot)
				}
			}
			if directory == "" {
				break
			}
			directory = path.Dir(directory)
			if directory == "." {
				directory = ""
			}
		}
	}
	if selected != nil {
		s.mu.Lock()
		if len(s.known) >= 1000 {
			s.known = map[string]repository{}
		}
		s.known[selected.public.ID] = *selected
		s.mu.Unlock()
	}
	return selected, nil
}
func (s *Service) Repositories(ctx context.Context, project string, version int64) (Repositories, error) {
	repos, truncated, err := s.discover(ctx, project, version)
	result := Repositories{Items: []Repository{}, Truncated: truncated}
	for _, r := range repos {
		result.Items = append(result.Items, r.public)
	}
	return result, err
}
func (s *Service) find(ctx context.Context, project string, version int64, id string) (repository, error) {
	s.mu.Lock()
	cached, ok := s.known[id]
	s.mu.Unlock()
	if ok && cached.root.ProjectID == project && cached.root.ProjectVersion == version {
		root, err := s.Store.RegisteredFolder(ctx, project, cached.public.FolderID, version)
		if err != nil {
			return repository{}, err
		}
		cached.root = root
		entries, err := files.SnapshotEntries(ctx, root, cached.public.Path)
		if err != nil {
			return repository{}, err
		}
		for _, e := range entries {
			if e.Name == ".git" && e.Identity == cached.identity {
				if e.Kind != "directory" || cached.public.State != "available" {
					return repository{}, ErrUnavailable
				}
				return cached, nil
			}
		}
		return repository{}, storage.ErrNotFound
	}
	items, _, err := s.discover(ctx, project, version)
	if err != nil {
		return repository{}, err
	}
	for _, item := range items {
		if item.public.ID == id {
			if item.public.State != "available" {
				return repository{}, ErrUnavailable
			}
			return item, nil
		}
	}
	return repository{}, storage.ErrNotFound
}
func read(ctx context.Context, root storage.RegisteredFolder, relative string, limit int64) ([]byte, files.Version, os.FileMode, error) {
	var b bytes.Buffer
	v, mode, err := files.CopySnapshot(ctx, root, relative, &b, limit)
	return b.Bytes(), v, mode, err
}

type snapshotScope struct {
	worktree bool
	file     string
	kind     string
}

func worktreePath(p string) bool {
	if !files.ValidRelative(p, false) {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if strings.EqualFold(part, ".git") {
			return false
		}
	}
	return true
}

func (s *Service) build(ctx context.Context, r repository, dir string, scope snapshotScope) (*snapshot, error) {
	snap := &snapshot{dir: dir, repo: r, versions: map[string]files.Version{}, metadata: map[string]files.Version{}}
	var size int64
	limit := s.Config.GitOptions().MaxBytes
	meta := path.Join(r.public.Path, ".git")
	// Reject object/metadata indirection before invoking Git. Metadata links never
	// become links in staging; an unsupported entry makes the whole repo unavailable.
	entries, err := files.SnapshotEntries(ctx, r.root, meta)
	if err != nil {
		return nil, err
	}
	allow := map[string]bool{"HEAD": true, "index": true, "packed-refs": true, "shallow": true, "config": true, "objects": true, "refs": true, "info": true}
	if err = os.MkdirAll(filepath.Join(dir, ".git"), 0700); err != nil {
		return nil, err
	}
	copyOne := func(relative, target string, e files.Entry, copy files.SnapshotCopy) error {
		if e.Kind != "file" {
			return ErrUnavailable
		}
		if e.Size > limit-size {
			return files.ErrTooLarge
		}
		if e.Name == "config" && e.Size > 1<<20 {
			return files.ErrTooLarge
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		destination, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		var v files.Version
		var mode os.FileMode
		var copyErr error
		if copy != nil {
			v, mode, copyErr = copy(destination, limit-size)
		} else {
			v, mode, copyErr = files.CopySnapshot(ctx, r.root, relative, destination, limit-size)
		}
		closeErr := destination.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if strings.HasSuffix(relative, "/info/attributes") {
			b, err := os.ReadFile(target)
			if err != nil {
				return err
			}
			if bytes.Contains(b, []byte("filter=")) {
				return ErrUnavailable
			}
		}
		size += v.Size
		snap.metadata[relative] = v
		return os.Chmod(target, 0600|(mode&0100))
	}
	for _, e := range entries {
		if e.Name == "commondir" || e.Name == "gitdir" {
			return nil, ErrUnavailable
		}
		if !allow[e.Name] && !strings.HasPrefix(e.Name, "sharedindex.") {
			continue
		}
		relative := path.Join(meta, e.Name)
		target := filepath.Join(dir, ".git", e.Name)
		if e.Kind == "directory" {
			// Git recognises an unborn repository only when its empty metadata
			// directories exist too, not just when there are objects/ref files.
			if err = os.MkdirAll(target, 0700); err != nil {
				return nil, err
			}
			err = files.WalkSnapshotWithCopy(ctx, r.root, relative, s.Config.SearchOptions().MaxEntries, func(p string, entry files.Entry, copy files.SnapshotCopy) (bool, error) {
				tail := strings.TrimPrefix(p, meta+"/")
				if strings.Contains(tail, "alternates") || strings.Contains(tail, "http-alternates") || strings.HasSuffix(tail, ".promisor") {
					return false, ErrUnavailable
				}
				if entry.Kind == "directory" {
					if err := os.MkdirAll(filepath.Join(dir, ".git", filepath.FromSlash(tail)), 0700); err != nil {
						return false, err
					}
					return true, nil
				}
				if strings.HasPrefix(tail, "info/") && tail != "info/exclude" && tail != "info/attributes" {
					return false, nil
				}
				return false, copyOne(p, filepath.Join(dir, ".git", filepath.FromSlash(tail)), entry, copy)
			})
			if err != nil {
				return nil, err
			}
		} else {
			if err = copyOne(relative, target, e, nil); err != nil {
				return nil, err
			}
		}
	}
	raw := filepath.Join(dir, ".git", "config")
	if _, err = os.Stat(raw); err != nil {
		return nil, ErrUnavailable
	}
	// Parse an explicit file without repository discovery or config includes.
	neutral, err := os.MkdirTemp(dir, "config-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(neutral)
	values, err := s.Runner.GitRun(ctx, neutral, "config", "--no-includes", "--null", "--file", raw, "--list")
	if err != nil {
		return nil, err
	}
	safe := map[string]string{"core.repositoryformatversion": "0", "core.bare": "false", "core.filemode": "true", "core.symlinks": "false"}
	for _, item := range bytes.Split(values, []byte{0}) {
		parts := strings.SplitN(string(item), "\n", 2)
		if len(parts) != 2 {
			continue
		}
		k, v := parts[0], parts[1]
		if strings.HasPrefix(k, "filter.") || strings.HasPrefix(k, "extensions.") && k != "extensions.objectformat" {
			return nil, ErrUnavailable
		}
		switch k {
		case "remote.origin.url":
			snap.githubURL = githubRepository(v)
		case "core.repositoryformatversion":
			if v != "0" && v != "1" {
				return nil, ErrUnavailable
			}
			safe[k] = v
		case "extensions.objectformat":
			if v != "sha1" && v != "sha256" {
				return nil, ErrUnavailable
			}
			safe[k] = v
		case "core.filemode", "core.ignorecase":
			if v != "true" && v != "false" {
				return nil, ErrUnavailable
			}
			safe[k] = v
		case "core.autocrlf":
			if v != "true" && v != "false" && v != "input" {
				return nil, ErrUnavailable
			}
			safe[k] = v
		case "core.eol":
			if v != "lf" && v != "crlf" && v != "native" {
				return nil, ErrUnavailable
			}
			safe[k] = v
		}
	}
	var sanitized strings.Builder
	keys := make([]string, 0, len(safe))
	for k := range safe {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		section, name, _ := strings.Cut(k, ".")
		fmt.Fprintf(&sanitized, "[%s]\n%s = %s\n", section, name, safe[k])
	}
	if err = os.WriteFile(raw, []byte(sanitized.String()), 0600); err != nil {
		return nil, err
	}

	head, err := s.Runner.GitRun(ctx, dir, "rev-parse", "--verify", "HEAD^{commit}")
	if err == nil {
		snap.head = strings.TrimSpace(string(head))
		if !objectID(snap.head) {
			return nil, ErrUnavailable
		}
	} else {
		rawHead, readErr := os.ReadFile(filepath.Join(dir, ".git", "HEAD"))
		ref := strings.TrimSpace(strings.TrimPrefix(string(rawHead), "ref: "))
		if readErr != nil || !strings.HasPrefix(string(rawHead), "ref: refs/heads/") {
			return nil, ErrUnavailable
		}
		refs, refsErr := s.refs(ctx, snap)
		if refsErr != nil {
			return nil, refsErr
		}
		for _, r := range refs {
			if r.Name == ref {
				return nil, ErrUnavailable
			}
		}
	}
	branch, err := s.Runner.GitRun(ctx, dir, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err == nil {
		snap.branch = strings.TrimSpace(string(branch))
	}
	if scope.kind == "head" || scope.kind == "unstaged" {
		revision := snap.head
		if scope.kind == "unstaged" {
			revision = ":"
		}
		object, e := s.blobObject(ctx, snap, revision, scope.file)
		if e != nil {
			return nil, e
		}
		if object != "" {
			scope.worktree = false
			snap.directPath = scope.file
		} else {
			scope.worktree = true
		}
	}

	if scope.worktree {
		err = s.copyWorktree(ctx, snap, &size, limit)
		if err != nil {
			return nil, err
		}
	} else if scope.file != "" {
		relative := path.Join(r.public.Path, scope.file)
		b, v, mode, e := read(ctx, r.root, relative, min(limit-size, 8<<20))
		if e == nil {
			target := filepath.Join(dir, filepath.FromSlash(scope.file))
			if e = os.MkdirAll(filepath.Dir(target), 0700); e != nil {
				return nil, e
			}
			if e = os.WriteFile(target, b, 0600|(mode&0111)); e != nil {
				return nil, e
			}
			snap.versions[scope.file] = v
		} else if !os.IsNotExist(e) {
			link, linkErr := files.SnapshotLink(r.root, relative)
			if linkErr != nil || !utf8.ValidString(link) {
				return nil, e
			}
			if int64(len(link)) > limit-size {
				return nil, files.ErrTooLarge
			}
			target := filepath.Join(dir, filepath.FromSlash(scope.file))
			if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return nil, err
			}
			if err = os.WriteFile(target, []byte(link), 0600); err != nil {
				return nil, err
			}
		}
	}

	return snap, nil
}
func (s *Service) with(ctx context.Context, project string, version int64, id string, scope snapshotScope, run func(context.Context, *snapshot) error) error {
	release, err := s.Runner.Acquire(ctx)
	if err != nil {
		return err
	}
	defer release()
	ctx, cancel := context.WithTimeout(ctx, s.Config.ToolTimeout())
	defer cancel()
	r, err := s.find(ctx, project, version, id)
	if err != nil {
		return err
	}
	dir, clean, err := s.Runner.Stage(s.Config.ToolStagingPath())
	if err != nil {
		return err
	}
	defer clean()
	snap, err := s.build(ctx, r, dir, scope)
	if err != nil {
		if errors.Is(err, files.ErrUnsupported) {
			return ErrUnavailable
		}
		return err
	}
	if err = run(ctx, snap); err != nil {
		return err
	}
	// Return no mixed snapshot if tracked metadata changed while the command ran.
	for relative, before := range snap.metadata {
		after, _, e := files.CopySnapshot(ctx, r.root, relative, io.Discard, s.Config.GitOptions().MaxBytes)
		if e != nil {
			return e
		}
		if before != after {
			return files.ErrConflict
		}
	}
	p, err := s.Store.Project(ctx, project)
	if err != nil {
		return err
	}
	if p.Version != version {
		return storage.ErrConflict
	}
	if _, err = s.find(ctx, project, version, id); err != nil {
		return err
	}
	return nil
}
func objectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
func (s *Service) Status(ctx context.Context, project string, version int64, id string) (Status, error) {
	out := Status{RepoID: id, Changes: []Change{}, TotalPaths: []string{}}
	err := s.with(ctx, project, version, id, snapshotScope{worktree: true}, func(ctx context.Context, v *snapshot) error {
		out.Head = v.head
		out.Branch = v.branch
		b, err := s.Runner.GitRun(ctx, v.dir, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignore-submodules=all")
		if err != nil {
			return err
		}
		parts := bytes.Split(b, []byte{0})
		for i := 0; i < len(parts); i++ {
			item := string(parts[i])
			if item == "" {
				continue
			}
			if len(item) < 4 || !utf8.ValidString(item) {
				return ErrUnavailable
			}
			change := Change{Path: item[3:], Index: item[:1], Worktree: item[1:2]}
			if !files.ValidRelative(change.Path, false) {
				return ErrUnavailable
			}
			if strings.ContainsAny(item[:2], "RC") {
				i++
				if i >= len(parts) {
					return ErrUnavailable
				}
				change.OldPath = string(parts[i])
			}
			out.Changes = append(out.Changes, change)
			if len(out.Changes) > 5000 {
				return files.ErrTooLarge
			}
		}
		total := map[string]bool{}
		for _, change := range out.Changes {
			if change.Index == "?" || v.head == "" {
				total[change.Path] = true
			}
		}
		if v.head != "" {
			names, err := s.Runner.GitRun(ctx, v.dir, "diff", "--no-ext-diff", "--no-textconv", "--no-renames", "--name-only", "-z", v.head, "--")
			if err != nil {
				return err
			}
			for _, name := range bytes.Split(names, []byte{0}) {
				if len(name) == 0 {
					continue
				}
				if !utf8.Valid(name) || !files.ValidRelative(string(name), false) {
					return ErrUnavailable
				}
				total[string(name)] = true
			}
		}
		if len(total) > 5000 {
			return files.ErrTooLarge
		}
		for name := range total {
			out.TotalPaths = append(out.TotalPaths, name)
		}
		sort.Strings(out.TotalPaths)

		return nil
	})
	return out, err
}
func (s *Service) refs(ctx context.Context, v *snapshot) ([]Ref, error) {
	b, err := s.Runner.GitRun(ctx, v.dir, "for-each-ref", "--format=%(refname)%00%(objectname)%00%(*objectname)", "refs/heads", "refs/tags")
	if err != nil {
		return nil, err
	}
	result := []Ref{}
	for _, line := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		if len(parts) != 3 {
			return nil, ErrUnavailable
		}
		id := parts[1]
		if parts[2] != "" {
			id = parts[2]
		}
		if !objectID(id) {
			return nil, ErrUnavailable
		}
		result = append(result, Ref{parts[0], id})
		if len(result) > 2000 {
			return nil, files.ErrTooLarge
		}
	}
	return result, nil
}
func (s *Service) Refs(ctx context.Context, project string, version int64, id string) ([]Ref, error) {
	result := []Ref{}
	err := s.with(ctx, project, version, id, snapshotScope{}, func(ctx context.Context, v *snapshot) error { var err error; result, err = s.refs(ctx, v); return err })
	return result, err
}
func parseCommits(b []byte) ([]Commit, error) {
	out := []Commit{}
	for _, row := range bytes.Split(b, []byte{0}) {
		row = bytes.Trim(row, "\n")
		if len(row) == 0 {
			continue
		}
		fields := bytes.SplitN(row, []byte{0x1f}, 5)
		if len(fields) != 5 {
			return nil, ErrUnavailable
		}
		id := string(fields[0])
		if !objectID(id) {
			return nil, ErrUnavailable
		}
		parents := append([]string{}, strings.Fields(string(fields[1]))...)
		for _, parent := range parents {
			if !objectID(parent) {
				return nil, ErrUnavailable
			}
		}
		out = append(out, Commit{id, parents, string(fields[2]), string(fields[3]), string(fields[4])})
	}
	return out, nil
}
func (s *Service) Log(ctx context.Context, project string, version int64, id string, offset int) (Log, error) {
	return s.LogAt(ctx, project, version, id, offset, "")
}
func (s *Service) LogAt(ctx context.Context, project string, version int64, id string, offset int, head string) (Log, error) {
	out := Log{Items: []Commit{}, NextOffset: -1}
	if offset < 0 || offset > 10000 || head != "" && !objectID(head) {
		return out, ErrInvalid
	}
	err := s.with(ctx, project, version, id, snapshotScope{}, func(ctx context.Context, v *snapshot) error {
		revision := head
		if revision == "" {
			revision = v.head
		}
		out.Head = revision
		if revision == "" {
			return nil
		}
		b, err := s.Runner.GitRun(ctx, v.dir, "log", "--topo-order", "--no-show-signature", "--format=%H%x1f%P%x1f%an%x1f%aI%x1f%s%x00", "--max-count=51", "--skip="+strconv.Itoa(offset), revision, "--")
		if err != nil {
			return err
		}
		commits, err := parseCommits(b)
		if err != nil {
			return err
		}
		if len(commits) > 50 {
			out.NextOffset = offset + 50
			commits = commits[:50]
		}
		out.Items = commits
		return nil
	})
	return out, err
}
func (s *Service) detail(ctx context.Context, v *snapshot, id, parent string) (Detail, error) {
	if !objectID(id) || parent != "" && !objectID(parent) {
		return Detail{}, ErrInvalid
	}
	b, err := s.Runner.GitRun(ctx, v.dir, "show", "--no-show-signature", "--no-patch", "--format=%H%x1f%P%x1f%an%x1f%aI%x1f%s%x00%B", id, "--")
	if err != nil {
		return Detail{}, err
	}
	meta, message, ok := bytes.Cut(b, []byte{0})
	if !ok || !utf8.Valid(message) || bytes.ContainsRune(message, 0) {
		return Detail{}, ErrUnavailable
	}
	if len(message) > 64<<10 {
		return Detail{}, files.ErrTooLarge
	}
	commits, err := parseCommits(meta)
	if err != nil || len(commits) != 1 {
		return Detail{}, ErrUnavailable
	}
	commit := commits[0]
	if parent == "" && len(commit.Parents) > 0 {
		parent = commit.Parents[0]
	}
	if parent != "" {
		found := false
		for _, p := range commit.Parents {
			if p == parent {
				found = true
			}
		}
		if !found {
			return Detail{}, ErrInvalid
		}
	}
	args := []string{"diff-tree", "--no-commit-id", "--no-ext-diff", "--no-textconv", "--raw", "--numstat", "--no-abbrev", "-M", "-z", "-r"}
	if parent == "" {
		args = append(args, "--root", id)
	} else {
		args = append(args, parent, id)
	}
	args = append(args, "--")
	b, err = s.Runner.GitRun(ctx, v.dir, args...)
	if err != nil {
		return Detail{}, err
	}
	stats, err := parseFileStats(b)
	if err != nil {
		return Detail{}, err
	}
	changed := make([]string, 0, len(stats))
	for _, stat := range stats {
		changed = append(changed, stat.Path)
	}
	link := ""
	if v.githubURL != "" {
		link = v.githubURL + "/commit/" + commit.ID
	}
	return Detail{Commit: commit, ParentID: parent, Files: changed, Message: strings.TrimSuffix(string(message), "\n"), Stats: stats, GitHubURL: link}, nil
}
func (s *Service) Detail(ctx context.Context, project string, version int64, repo, id, parent string) (Detail, error) {
	var out Detail
	err := s.with(ctx, project, version, repo, snapshotScope{}, func(ctx context.Context, v *snapshot) error {
		var err error
		out, err = s.detail(ctx, v, id, parent)
		return err
	})
	return out, err
}

// blobObject distinguishes absent paths from corrupt or missing objects.
func (s *Service) blobObject(ctx context.Context, v *snapshot, revision, p string) (string, error) {
	if revision == "" {
		return "", nil
	}
	args := []string{"ls-tree", "-z", revision, "--", p}
	if revision == ":" {
		args = []string{"ls-files", "--stage", "-z", "--", p}
	}
	b, err := s.Runner.GitRun(ctx, v.dir, args...)
	if err != nil {
		return "", err
	}
	for _, row := range bytes.Split(b, []byte{0}) {
		pair := bytes.SplitN(row, []byte{'\t'}, 2)
		if len(pair) != 2 || string(pair[1]) != p {
			continue
		}
		fields := strings.Fields(string(pair[0]))
		if len(fields) != 3 {
			return "", ErrUnavailable
		}
		object := fields[2]
		if revision == ":" {
			if fields[2] != "0" {
				return "", ErrUnavailable
			}
			object = fields[1]
		} else if fields[1] != "blob" {
			return "", ErrUnavailable
		}
		if !objectID(object) {
			return "", ErrUnavailable
		}
		return object, nil
	}
	return "", nil
}
func (s *Service) blob(ctx context.Context, v *snapshot, revision, p string) ([]byte, error) {
	object, err := s.blobObject(ctx, v, revision, p)
	if err != nil {
		return nil, err
	}
	if object == "" {
		return []byte{}, nil
	}
	b, err := s.Runner.GitRun(ctx, v.dir, "cat-file", "blob", object)
	if len(b) > 8<<20 {
		return nil, files.ErrTooLarge
	}
	return b, err
}
func (s *Service) originalPath(ctx context.Context, v *snapshot, kind, p string) (string, error) {
	if v.directPath == p {
		return p, nil
	}
	if kind != "head" && kind != "staged" && kind != "unstaged" {
		return p, nil
	}
	if v.head == "" && kind != "unstaged" {
		return p, nil
	}
	args := []string{"diff", "--no-ext-diff", "--no-textconv", "--find-renames", "--name-status", "-z"}
	if kind == "staged" {
		args = append(args, "--cached")
	}
	if kind != "unstaged" {
		args = append(args, v.head)
	}
	args = append(args, "--")
	b, err := s.Runner.GitRun(ctx, v.dir, args...)
	if err != nil {
		return "", err
	}
	fields := bytes.Split(b, []byte{0})
	for i := 0; i < len(fields); {
		status := string(fields[i])
		i++
		if status == "" {
			continue
		}
		if i >= len(fields) {
			return "", ErrUnavailable
		}
		before := string(fields[i])
		i++
		if status[0] == 'R' || status[0] == 'C' {
			if i >= len(fields) {
				return "", ErrUnavailable
			}
			after := string(fields[i])
			i++
			if after == p {
				if !utf8.ValidString(before) || !files.ValidRelative(before, false) {
					return "", ErrUnavailable
				}
				return before, nil
			}
		}
	}
	return p, nil
}
func (s *Service) Compare(ctx context.Context, project, repo string, input CompareInput) (Comparison, error) {
	out := Comparison{RepoID: repo, Path: input.Path}
	if !worktreePath(input.Path) {
		return out, ErrInvalid
	}
	scope := snapshotScope{kind: input.Kind}
	if input.Kind == "head" || input.Kind == "unstaged" || input.Kind == "reference" {
		scope.file = input.Path
	}
	err := s.with(ctx, project, input.ProjectVersion, repo, scope, func(ctx context.Context, v *snapshot) error {
		revision := v.head
		modifiedRevision := ""
		commitOldPath := ""
		switch input.Kind {
		case "head":
		case "unstaged":
			revision = ":"
		case "staged":
			modifiedRevision = ":"
		case "reference":
			refs, err := s.refs(ctx, v)
			if err != nil {
				return err
			}
			revision = ""
			for _, ref := range refs {
				if ref.Name == input.Reference {
					revision = ref.CommitID
				}
			}
			if revision == "" {
				return ErrInvalid
			}
		case "commit":
			detail, err := s.detail(ctx, v, input.CommitID, input.ParentID)
			if err != nil {
				return err
			}
			revision = detail.ParentID
			modifiedRevision = detail.Commit.ID
			for _, stat := range detail.Stats {
				if stat.Path == input.Path {
					commitOldPath = stat.OldPath
					break
				}
			}
		default:
			return ErrInvalid
		}
		originalPath, err := s.originalPath(ctx, v, input.Kind, input.Path)
		if err != nil {
			return err
		}
		if commitOldPath != "" {
			originalPath = commitOldPath
		}
		if originalPath != input.Path {
			out.OldPath = originalPath
		}
		original, err := s.blob(ctx, v, revision, originalPath)
		if err != nil {
			return err
		}
		modified := []byte{}
		if modifiedRevision != "" {
			modified, err = s.blob(ctx, v, modifiedRevision, input.Path)
		} else {
			b, errRead := os.ReadFile(filepath.Join(v.dir, filepath.FromSlash(input.Path)))
			if errRead != nil && !os.IsNotExist(errRead) {
				return errRead
			}
			modified = b
		}
		if err != nil {
			return err
		}
		if len(modified) > 8<<20 {
			return files.ErrTooLarge
		}
		out.Binary = !utf8.Valid(original) || !utf8.Valid(modified) || bytes.ContainsRune(original, 0) || bytes.ContainsRune(modified, 0)
		if !out.Binary {
			out.Original = string(original)
			out.Modified = string(modified)
		}
		out.Baseline = revision
		return nil
	})
	return out, err
}
func (s *Service) Baseline(ctx context.Context, project string, version int64, folder, relative string) (Baseline, error) {
	out := Baseline{State: "no_repository"}
	if !worktreePath(relative) {
		return out, ErrInvalid
	}
	folderRoot, err := s.Store.RegisteredFolder(ctx, project, folder, version)
	if err != nil {
		return out, err
	}
	absolute := filepath.Join(folderRoot.Path, filepath.FromSlash(relative))
	selected, err := s.baselineRepository(ctx, project, version, absolute)
	if err != nil {
		return out, err
	}

	if selected == nil {
		return out, nil
	}
	if selected.public.State != "available" {
		out.State = "unavailable"
		return out, nil
	}
	p, err := filepath.Rel(filepath.Join(selected.root.Path, filepath.FromSlash(selected.public.Path)), absolute)
	if err != nil {
		return out, ErrUnavailable
	}
	p = filepath.ToSlash(p)
	out.RepoID = selected.public.ID
	err = s.with(ctx, project, version, selected.public.ID, snapshotScope{file: p}, func(ctx context.Context, v *snapshot) error {
		out.Head = v.head
		if v.head == "" {
			out.State = "untracked"
			return nil
		}
		object, err := s.blobObject(ctx, v, v.head, p)
		if err != nil {
			return err
		}
		if object == "" {
			out.State = "untracked"
			return nil
		}
		b, err := s.blob(ctx, v, v.head, p)
		if err != nil {
			return err
		}
		if !utf8.Valid(b) || bytes.ContainsRune(b, 0) {
			out.State = "binary"
			return nil
		}
		out.State = "tracked"
		out.Content = string(b)
		if version, ok := v.versions[p]; ok {
			out.Version = &version
		}
		return nil
	})
	return out, err
}
