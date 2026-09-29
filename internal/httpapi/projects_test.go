package httpapi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"persistty/internal/storage"
)

func TestProjectMutationAndDirectoryProtection(t *testing.T) {
	r, cfg, _ := setup(t)
	root := t.TempDir()
	for _, name := range []string{"a", "b"} {
		if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	path := "/api/v1/directories?path=" + url.QueryEscape(root)
	if got := request(r, "GET", path, "", "", nil, ""); got.Code != 401 {
		t.Fatal("anonymous directory access")
	}
	cookie, csrf := login(t, r, cfg)
	listing := request(r, "GET", path, "", "", cookie, "")
	if listing.Code != 200 || !strings.Contains(listing.Body.String(), `"items":["a","b"]`) {
		t.Fatalf("directory %d %s", listing.Code, listing.Body)
	}
	create := fmt.Sprintf(`{"name":"示例","folder_paths":[%q,%q],"main_index":1}`, filepath.Join(root, "a"), filepath.Join(root, "b"))
	if got := request(r, "POST", "/api/v1/projects", create, "", cookie, csrf); got.Code != 403 {
		t.Fatal("Origin bypass")
	}
	if got := request(r, "POST", "/api/v1/projects", create, cfg.Server.PublicOrigin, cookie, ""); got.Code != 403 {
		t.Fatal("CSRF bypass")
	}
	created := request(r, "POST", "/api/v1/projects", create, cfg.Server.PublicOrigin, cookie, csrf)
	if created.Code != 201 {
		t.Fatalf("create %d %s", created.Code, created.Body)
	}
	var envelope struct{ Data storage.Project }
	if err := json.Unmarshal(created.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	p := envelope.Data
	if p.Version != 1 || len(p.Folders) != 2 || p.MainFolderID != p.Folders[1].ID {
		t.Fatal("invalid project response")
	}
	file := filepath.Join(root, "a", "note.txt")
	if err := os.WriteFile(file, []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	contentURL := fmt.Sprintf("/api/v1/projects/%s/folders/%s/content?project_version=1&path=note.txt", p.ID, p.Folders[0].ID)
	content := request(r, "GET", contentURL, "", "", cookie, "")
	if content.Code != 200 {
		t.Fatalf("content %d %s", content.Code, content.Body)
	}
	var snapshot struct {
		Data struct {
			Version struct {
				Mtime    string `json:"mtime"`
				Size     int64  `json:"size"`
				ETag     string `json:"etag"`
				Identity string `json:"identity"`
			} `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(content.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	versionJSON, err := json.Marshal(snapshot.Data.Version)
	if err != nil {
		t.Fatal(err)
	}
	saveBody := fmt.Sprintf(`{"project_version":1,"path":"note.txt","expected_version":%s,"content":"second"}`, versionJSON)
	saveURL := strings.Split(contentURL, "?")[0]
	if got := request(r, "PUT", saveURL, saveBody, cfg.Server.PublicOrigin, cookie, csrf); got.Code != 200 {
		t.Fatalf("save %d %s", got.Code, got.Body)
	}
	if got := request(r, "PUT", saveURL, saveBody, cfg.Server.PublicOrigin, cookie, csrf); got.Code != 409 {
		t.Fatal("stale file save accepted")
	}
	previewBody := fmt.Sprintf(`{"project_version":1,"folder_id":%q,"path":"note.txt"}`, p.Folders[0].ID)
	previewURL := "/api/v1/projects/" + p.ID + "/delete-preview"
	previewResponse := request(r, "POST", previewURL, previewBody, cfg.Server.PublicOrigin, cookie, csrf)
	if previewResponse.Code != 200 {
		t.Fatalf("preview %d %s", previewResponse.Code, previewResponse.Body)
	}
	var deleteEnvelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(previewResponse.Body.Bytes(), &deleteEnvelope); err != nil {
		t.Fatal(err)
	}
	deleteBody := fmt.Sprintf(`{"kind":"delete","project_version":1,"source_folder_id":%q,"source_path":"note.txt","delete_token":%q}`, p.Folders[0].ID, deleteEnvelope.Data.Token)
	operationURL := "/api/v1/projects/" + p.ID + "/file-operations"
	if got := request(r, "POST", operationURL, deleteBody, "", cookie, csrf); got.Code != 403 {
		t.Fatal("delete Origin bypass")
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatal("cancelled delete touched file")
	}
	if got := request(r, "POST", operationURL, deleteBody, cfg.Server.PublicOrigin, cookie, csrf); got.Code != 200 {
		t.Fatalf("delete %d %s", got.Code, got.Body)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("confirmed delete did not remove file")
	}
	update := fmt.Sprintf(`{"expected_version":1,"name":"改名","add_paths":[],"remove_folder_ids":[%q],"main_folder_id":%q}`, p.Folders[1].ID, p.Folders[0].ID)
	endpoint := "/api/v1/projects/" + p.ID
	if got := request(r, "PATCH", endpoint, update, cfg.Server.PublicOrigin, cookie, csrf); got.Code != 200 {
		t.Fatalf("update %d %s", got.Code, got.Body)
	}
	if got := request(r, "GET", contentURL, "", "", cookie, ""); got.Code != 409 {
		t.Fatal("old project version still reads file")
	}
	if got := request(r, "PATCH", endpoint, update, cfg.Server.PublicOrigin, cookie, csrf); got.Code != 409 {
		t.Fatal("stale project update accepted")
	}
	if got := request(r, "DELETE", endpoint, `{"expected_version":1}`, cfg.Server.PublicOrigin, cookie, csrf); got.Code != 409 {
		t.Fatal("stale project delete accepted")
	}
	if got := request(r, "DELETE", endpoint, `{"expected_version":2}`, cfg.Server.PublicOrigin, cookie, csrf); got.Code != 204 {
		t.Fatalf("delete %d %s", got.Code, got.Body)
	}
	if _, err := os.Stat(filepath.Join(root, "b")); err != nil {
		t.Fatal("project delete removed real folder")
	}
	if got := request(r, "GET", endpoint, "", "", cookie, ""); got.Code != 404 {
		t.Fatal("removed bookmark still resolves")
	}
}
