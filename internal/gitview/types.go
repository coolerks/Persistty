package gitview

import (
	"errors"
	"persistty/internal/files"
)

var ErrUnavailable = errors.New("repository unavailable")
var ErrInvalid = errors.New("invalid git request")

type Repository struct {
	ID       string `json:"id"`
	FolderID string `json:"folder_id"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	State    string `json:"state"`
	Reason   string `json:"reason"`
}
type Repositories struct {
	Items     []Repository `json:"items"`
	Truncated bool         `json:"truncated"`
}
type Change struct {
	Path     string `json:"path"`
	OldPath  string `json:"old_path"`
	Index    string `json:"index"`
	Worktree string `json:"worktree"`
}
type Ref struct {
	Name     string `json:"name"`
	CommitID string `json:"commit_id"`
}
type Status struct {
	RepoID     string   `json:"repo_id"`
	Head       string   `json:"head"`
	Branch     string   `json:"branch"`
	Changes    []Change `json:"changes"`
	TotalPaths []string `json:"total_paths"`
}
type Commit struct {
	ID      string   `json:"id"`
	Parents []string `json:"parents"`
	Author  string   `json:"author"`
	Date    string   `json:"date"`
	Subject string   `json:"subject"`
}
type Log struct {
	Head       string   `json:"head"`
	Items      []Commit `json:"items"`
	NextOffset int      `json:"next_offset"`
}
type Detail struct {
	Commit   Commit   `json:"commit"`
	ParentID string   `json:"parent_id"`
	Files    []string `json:"files"`
}
type CompareInput struct {
	ProjectVersion int64  `json:"project_version"`
	Path           string `json:"path"`
	Kind           string `json:"kind"`
	Reference      string `json:"reference"`
	CommitID       string `json:"commit_id"`
	ParentID       string `json:"parent_id"`
}
type Comparison struct {
	OldPath  string `json:"old_path"`
	RepoID   string `json:"repo_id"`
	Path     string `json:"path"`
	Original string `json:"original"`
	Modified string `json:"modified"`
	Binary   bool   `json:"binary"`
	Baseline string `json:"baseline"`
}
type Baseline struct {
	State   string         `json:"state"`
	RepoID  string         `json:"repo_id"`
	Head    string         `json:"head"`
	Content string         `json:"content"`
	Version *files.Version `json:"version"`
}
