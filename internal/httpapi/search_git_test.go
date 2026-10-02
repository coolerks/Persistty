package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"persistty/internal/gitview"
	"persistty/internal/search"
	"persistty/internal/storage"
	"reflect"
	"strings"
	"testing"
)

func TestSearchGitSharedDTO(t *testing.T) {
	data, err := os.ReadFile("../../tests/fixtures/search-git.json")
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]json.RawMessage
	if err = json.Unmarshal(data, &values); err != nil {
		t.Fatal(err)
	}
	dtos := map[string]any{"file_names": &search.FileNames{}, "search": &search.Result{}, "preview": &search.Preview{}, "repositories": &gitview.Repositories{}, "status": &gitview.Status{}, "log": &gitview.Log{}, "detail": &gitview.Detail{}, "baseline": &gitview.Baseline{}, "comparison": &gitview.Comparison{}}
	for name, dto := range dtos {
		t.Run(name, func(t *testing.T) {
			if err := json.Unmarshal(values[name], dto); err != nil {
				t.Fatal(err)
			}
			got, err := json.Marshal(dto)
			if err != nil {
				t.Fatal(err)
			}
			var before, after any
			json.Unmarshal(values[name], &before)
			json.Unmarshal(got, &after)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("DTO drift %s", got)
			}
		})
	}
}
func TestSearchReplaceHTTPBoundToSessionAndCSRF(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg unavailable")
	}
	r, cfg, logs := setup(t)
	cookie, csrf := login(t, r, cfg)
	other, otherCSRF := login(t, r, cfg)
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "secret.txt"), []byte("private marker"), 0600)
	body, _ := json.Marshal(map[string]any{"name": "fixture", "folder_paths": []string{root}, "main_index": 0})
	created := request(r, "POST", "/api/v1/projects", string(body), cfg.Server.PublicOrigin, cookie, csrf)
	var p struct{ Data storage.Project }
	json.Unmarshal(created.Body.Bytes(), &p)
	if created.Code != 201 {
		t.Fatal(created.Body)
	}
	prefix := "/api/v1/projects/" + p.Data.ID
	query := `{"project_version":1,"folder_id":"","path":"","pattern":"private","regex":false,"case_sensitive":false,"whole_word":false,"include":[],"exclude":[]}`
	for _, got := range []int{request(r, "POST", prefix+"/searches", query, "", cookie, csrf).Code, request(r, "POST", prefix+"/searches", query, cfg.Server.PublicOrigin, cookie, "").Code} {
		if got != 403 {
			t.Fatalf("bypass %d", got)
		}
	}
	found := request(r, "POST", prefix+"/searches", query, cfg.Server.PublicOrigin, cookie, csrf)
	var result struct{ Data search.Result }
	json.Unmarshal(found.Body.Bytes(), &result)
	if found.Code != 200 || len(result.Data.Files) != 1 {
		t.Fatalf("search %d %s", found.Code, found.Body)
	}
	previewBody, _ := json.Marshal(search.PreviewInput{ProjectVersion: 1, SearchID: result.Data.ID, SelectedMatchIDs: []string{result.Data.Files[0].Matches[0].ID}, Replacement: "replacement"})
	if got := request(r, "POST", prefix+"/replace-previews", string(previewBody), cfg.Server.PublicOrigin, other, otherCSRF); got.Code != 410 {
		t.Fatalf("session isolation %d", got.Code)
	}
	preview := request(r, "POST", prefix+"/replace-previews", string(previewBody), cfg.Server.PublicOrigin, cookie, csrf)
	var prepared struct{ Data search.Preview }
	json.Unmarshal(preview.Body.Bytes(), &prepared)
	if preview.Code != 200 {
		t.Fatal(preview.Body)
	}
	endpoint := prefix + "/replace-previews/" + prepared.Data.ID
	if got := request(r, "GET", endpoint, "", "", other, ""); got.Code != 410 {
		t.Fatalf("GET session isolation %d", got.Code)
	}
	applyBody := fmt.Sprintf(`{"project_version":1,"selected_file_ids":[%q],"protected_file_ids":[]}`, prepared.Data.Files[0].ID)
	applied := request(r, "POST", endpoint+"/apply", applyBody, cfg.Server.PublicOrigin, cookie, csrf)
	if applied.Code != 200 || !strings.Contains(applied.Body.String(), `"state":"applied"`) {
		t.Fatal(applied.Body)
	}
	again := request(r, "POST", endpoint+"/apply", applyBody, cfg.Server.PublicOrigin, cookie, csrf)
	if again.Code != 200 || again.Body.String() == "" {
		t.Fatal("retry status")
	}
	disk, _ := os.ReadFile(filepath.Join(root, "secret.txt"))
	if string(disk) != "replacement marker" {
		t.Fatal(string(disk))
	}
	for _, private := range []string{"private marker", "replacement marker", "replacement", "secret.txt", root} {
		if strings.Contains(logs.String(), private) {
			t.Fatalf("logged content/path %s", private)
		}
	}
	if got := request(r, "GET", prefix+"/repositories?project_version=1", "", "", cookie, ""); got.Code != 200 {
		t.Fatal(got.Body)
	}
	if got := request(r, "GET", prefix+"/repositories", "", "", cookie, ""); got.Code != 428 {
		t.Fatal(got.Code)
	}
}

func TestFileNamesHTTPAuthenticationVersionAndNoBodyLeak(t *testing.T) {
	r, cfg, logs := setup(t)
	cookie, csrf := login(t, r, cfg)
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "SecretName.txt"), []byte("PRIVATE_BODY_MARKER"), 0600)
	body, _ := json.Marshal(map[string]any{"name": "names", "folder_paths": []string{root}, "main_index": 0})
	created := request(r, "POST", "/api/v1/projects", string(body), cfg.Server.PublicOrigin, cookie, csrf)
	var p struct{ Data storage.Project }
	json.Unmarshal(created.Body.Bytes(), &p)
	if created.Code != 201 {
		t.Fatal(created.Body)
	}
	prefix := "/api/v1/projects/" + p.Data.ID + "/file-names"
	for _, tc := range []struct {
		suffix string
		cookie *http.Cookie
		code   int
	}{
		{"?project_version=1&query=secret", nil, 401},
		{"?query=secret", cookie, 428},
		{"?project_version=2&query=secret", cookie, 409},
		{"?project_version=1&query=", cookie, 400},
		{"?project_version=1&query=secret%0A", cookie, 400},
	} {
		got := request(r, "GET", prefix+tc.suffix, "", "", tc.cookie, "")
		if got.Code != tc.code {
			t.Fatalf("%s: %d %s", tc.suffix, got.Code, got.Body)
		}
	}
	got := request(r, "GET", prefix+"?project_version=1&query=secret", "", "", cookie, "")
	var result struct{ Data search.FileNames }
	json.Unmarshal(got.Body.Bytes(), &result)
	if got.Code != 200 || len(result.Data.Items) != 1 || result.Data.Items[0].Path != "SecretName.txt" || strings.Contains(got.Body.String(), "PRIVATE_BODY_MARKER") {
		t.Fatalf("response %d %s", got.Code, got.Body)
	}
	for _, secret := range []string{"SecretName.txt", "PRIVATE_BODY_MARKER", root, "query=secret"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("logged %s", secret)
		}
	}
}
