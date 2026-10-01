package httpapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"persistty/internal/files"
	"persistty/internal/storage"
	"strings"
	"testing"
)

func TestPreviewProtectionAndHeaders(t *testing.T) {
	r, cfg, _ := setup(t)
	root := t.TempDir()
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10"/></svg>`
	if err := os.WriteFile(filepath.Join(root, "safe.svg"), []byte(svg), 0600); err != nil {
		t.Fatal(err)
	}
	cookie, csrf := login(t, r, cfg)
	created := request(r, "POST", "/api/v1/projects", fmt.Sprintf(`{"name":"预览","folder_paths":[%q],"main_index":0}`, root), cfg.Server.PublicOrigin, cookie, csrf)
	var project struct{ Data storage.Project }
	if err := json.Unmarshal(created.Body.Bytes(), &project); err != nil {
		t.Fatal(err)
	}
	base := fmt.Sprintf("/api/v1/projects/%s/folders/%s/", project.Data.ID, project.Data.MainFolderID)
	for _, endpoint := range []string{"inspect", "preview"} {
		if got := request(r, "GET", base+endpoint+"?project_version=1&path=safe.svg", "", "", nil, ""); got.Code != 401 {
			t.Fatal("anonymous preview")
		}
	}
	inspected := request(r, "GET", base+"inspect?project_version=1&path=safe.svg", "", "", cookie, "")
	var envelope struct{ Data files.Inspection }
	if err := json.Unmarshal(inspected.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if inspected.Code != 200 || !envelope.Data.Previewable {
		t.Fatal(inspected.Body)
	}
	preview := request(r, "GET", base+"preview?project_version=1&path=safe.svg", "", "", cookie, "")
	if preview.Code != 200 || preview.Body.String() != svg || preview.Header().Get("Content-Type") != "image/svg+xml" || preview.Header().Get("Cache-Control") != "no-store" || preview.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(preview.Header().Get("Content-Security-Policy"), "sandbox") {
		t.Fatalf("preview %d %v", preview.Code, preview.Header())
	}
	if got := request(r, "GET", base+"preview?project_version=2&path=safe.svg", "", "", cookie, ""); got.Code != 409 {
		t.Fatalf("stale project %d", got.Code)
	}
	if err := os.WriteFile(filepath.Join(root, "safe.svg"), []byte(`<svg><script>alert(1)</script></svg>`), 0600); err != nil {
		t.Fatal(err)
	}
	if got := request(r, "GET", base+"preview?project_version=1&path=safe.svg", "", "", cookie, ""); got.Code != 415 {
		t.Fatalf("unsafe preview %d", got.Code)
	}
}

func TestSaveBodyAllowsEscapedTextWithoutRaisingContentLimit(t *testing.T) {
	r, cfg, _ := setup(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	cookie, csrf := login(t, r, cfg)
	created := request(r, "POST", "/api/v1/projects", fmt.Sprintf(`{"name":"保存","folder_paths":[%q],"main_index":0}`, root), cfg.Server.PublicOrigin, cookie, csrf)
	var project struct{ Data storage.Project }
	if err := json.Unmarshal(created.Body.Bytes(), &project); err != nil {
		t.Fatal(err)
	}
	base := fmt.Sprintf("/api/v1/projects/%s/folders/%s/content", project.Data.ID, project.Data.MainFolderID)
	current := request(r, "GET", base+"?project_version=1&path=note", "", "", cookie, "")
	var snapshot struct{ Data files.Content }
	if err := json.Unmarshal(current.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("<", 2<<20)
	body, err := json.Marshal(map[string]any{"project_version": 1, "path": "note", "expected_version": snapshot.Data.Version, "content": text})
	if err != nil {
		t.Fatal(err)
	}
	if len(body) <= 8<<20 {
		t.Fatal("fixture did not exceed old JSON limit")
	}
	saved := request(r, "PUT", base, string(body), cfg.Server.PublicOrigin, cookie, csrf)
	if saved.Code != 200 {
		t.Fatalf("escaped UTF8 rejected %d", saved.Code)
	}
	data, err := os.ReadFile(filepath.Join(root, "note"))
	if err != nil || string(data) != text {
		t.Fatal("raw text changed")
	}
}
