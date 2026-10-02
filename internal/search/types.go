package search

import (
	"errors"
	"persistty/internal/files"
)

var ErrPattern = errors.New("invalid pattern")
var ErrExpired = errors.New("snapshot expired")
var ErrInvalid = errors.New("invalid search request")

type Query struct {
	ProjectVersion int64    `json:"project_version"`
	FolderID       string   `json:"folder_id"`
	Path           string   `json:"path"`
	Pattern        string   `json:"pattern"`
	Regex          bool     `json:"regex"`
	CaseSensitive  bool     `json:"case_sensitive"`
	WholeWord      bool     `json:"whole_word"`
	Include        []string `json:"include"`
	Exclude        []string `json:"exclude"`
}
type Match struct {
	ID        string `json:"id"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndColumn int    `json:"end_column"`
	Preview   string `json:"preview"`
	Start     int    `json:"-"`
	End       int    `json:"-"`
}
type File struct {
	ID       string        `json:"id"`
	FolderID string        `json:"folder_id"`
	Path     string        `json:"path"`
	Version  files.Version `json:"version"`
	Matches  []Match       `json:"matches"`
	Content  string        `json:"-"`
	Native   bool          `json:"-"`
}
type Skipped struct {
	FolderID string `json:"folder_id"`
	Path     string `json:"path"`
	Reason   string `json:"reason"`
}
type Result struct {
	ID             string    `json:"id"`
	ProjectVersion int64     `json:"project_version"`
	Files          []File    `json:"files"`
	Truncated      bool      `json:"truncated"`
	Skipped        []Skipped `json:"skipped"`
	ExpiresAt      string    `json:"expires_at"`
}
type PreviewInput struct {
	ProjectVersion   int64    `json:"project_version"`
	SearchID         string   `json:"search_id"`
	SelectedMatchIDs []string `json:"selected_match_ids"`
	Replacement      string   `json:"replacement"`
}
type PreviewFile struct {
	ID       string        `json:"id"`
	FolderID string        `json:"folder_id"`
	Path     string        `json:"path"`
	Version  files.Version `json:"version"`
	Count    int           `json:"count"`
	NewHash  string        `json:"new_hash"`
	Original string        `json:"-"`
	Modified string        `json:"-"`
}
type Applied struct {
	ID      string         `json:"id"`
	State   string         `json:"state"`
	Reason  string         `json:"reason"`
	Version *files.Version `json:"version"`
}
type Preview struct {
	ID             string        `json:"id"`
	ProjectVersion int64         `json:"project_version"`
	Files          []PreviewFile `json:"files"`
	Results        []Applied     `json:"results"`
	State          string        `json:"state"`
	ExpiresAt      string        `json:"expires_at"`
}
type Comparison struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Original string `json:"original"`
	Modified string `json:"modified"`
}
type ApplyInput struct {
	ProjectVersion   int64    `json:"project_version"`
	SelectedFileIDs  []string `json:"selected_file_ids"`
	ProtectedFileIDs []string `json:"protected_file_ids"`
}
