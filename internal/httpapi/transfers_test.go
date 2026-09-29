package httpapi

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"persistty/internal/storage"
)

func TestProtectedDownloadAndArchive(t *testing.T) {
	r, cfg, _ := setup(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "folder", "empty"), 0700); err != nil {
		t.Fatal(err)
	}
	data := []byte{0, 1, 2, 255}
	if err := os.WriteFile(filepath.Join(root, "folder", "binary"), data, 0600); err != nil {
		t.Fatal(err)
	}
	cookie, csrf := login(t, r, cfg)
	created := request(r, "POST", "/api/v1/projects", fmt.Sprintf(`{"name":"归档","folder_paths":[%q],"main_index":0}`, root), cfg.Server.PublicOrigin, cookie, csrf)
	if created.Code != 201 {
		t.Fatalf("project %d %s", created.Code, created.Body)
	}
	var project struct{ Data storage.Project }
	if err := json.Unmarshal(created.Body.Bytes(), &project); err != nil {
		t.Fatal(err)
	}
	p := project.Data
	base := fmt.Sprintf("/api/v1/projects/%s/folders/%s/", p.ID, p.MainFolderID)
	fileQuery := "?project_version=1&path=" + url.QueryEscape("folder/binary")
	if got := request(r, "GET", base+"download"+fileQuery, "", "", nil, ""); got.Code != 401 {
		t.Fatal("anonymous download allowed")
	}
	metadata := request(r, "GET", base+"metadata"+fileQuery, "", "", cookie, "")
	if metadata.Code != 200 || !strings.Contains(metadata.Body.String(), `"kind":"file"`) {
		t.Fatalf("metadata %d %s", metadata.Code, metadata.Body)
	}
	download := request(r, "GET", base+"download"+fileQuery, "", "", cookie, "")
	if download.Code != 200 || !bytes.Equal(download.Body.Bytes(), data) || !strings.Contains(download.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("download %d", download.Code)
	}
	input := fmt.Sprintf(`{"project_id":%q,"folder_id":%q,"project_version":1,"path":"folder"}`, p.ID, p.MainFolderID)
	if got := request(r, "POST", "/api/v1/archives", input, "", cookie, csrf); got.Code != 403 {
		t.Fatal("archive Origin bypass")
	}
	started := request(r, "POST", "/api/v1/archives", input, cfg.Server.PublicOrigin, cookie, csrf)
	if started.Code != 202 {
		t.Fatalf("archive create %d %s", started.Code, started.Body)
	}
	var archive struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(started.Body.Bytes(), &archive); err != nil {
		t.Fatal(err)
	}
	archivePath := "/api/v1/archives/" + archive.Data.ID
	deadline := time.Now().Add(5 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		status := request(r, "GET", archivePath, "", "", cookie, "")
		if status.Code != 200 {
			t.Fatalf("status %d %s", status.Code, status.Body)
		}
		if strings.Contains(status.Body.String(), `"status":"ready"`) {
			ready = true
			break
		}
		if strings.Contains(status.Body.String(), `"status":"failed"`) {
			t.Fatalf("archive failed: %s", status.Body)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !ready {
		t.Fatal("archive timeout")
	}
	got := request(r, "GET", archivePath+"/download", "", "", cookie, "")
	if got.Code != 200 || got.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("archive download %d %s", got.Code, got.Body)
	}
	reader, err := zip.NewReader(bytes.NewReader(got.Body.Bytes()), int64(got.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	entries := make(map[string][]byte)
	for _, item := range reader.File {
		stream, err := item.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(stream)
		stream.Close()
		if err != nil {
			t.Fatal(err)
		}
		entries[item.Name] = body
	}
	if len(entries) != 3 || !bytes.Equal(entries["folder/binary"], data) || entries["folder/empty/"] == nil && !hasKey(entries, "folder/empty/") {
		t.Fatalf("archive entries: %v", entries)
	}
	if got := request(r, "DELETE", archivePath, "", cfg.Server.PublicOrigin, cookie, csrf); got.Code != 204 {
		t.Fatalf("cancel %d %s", got.Code, got.Body)
	}
	if got := request(r, "GET", archivePath+"/download", "", "", cookie, ""); got.Code != 409 {
		t.Fatalf("cancelled archive available: %d", got.Code)
	}
}

func hasKey(items map[string][]byte, key string) bool { _, exists := items[key]; return exists }
