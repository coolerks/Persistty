package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"persistty/internal/auth"
	"persistty/internal/config"
	"persistty/internal/elevation"
	"persistty/internal/files"
	"persistty/internal/storage"
	"strings"
	"testing"
)

type httpElevationBackend struct{}

func (httpElevationBackend) Target(_ context.Context, r storage.RegisteredFolder, path string) (elevation.Target, error) {
	return elevation.Target{ID: "example", Path: filepath.Join(r.Path, path)}, nil
}
func (httpElevationBackend) Run(_ context.Context, g elevation.Grant, _, _ []byte, p elevation.Publish) (elevation.Result, error) {
	return p(func() (elevation.Result, error) {
		code := "authorization_failed"
		return elevation.Result{ID: g.ID, State: "rejected", Code: &code}, nil
	})
}
func (httpElevationBackend) Status(context.Context, elevation.Grant) (elevation.Result, error) {
	return elevation.Result{}, elevation.ErrUnavailable
}
func TestElevationHTTPProtectionSingleUseAndNoSecretLogs(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	hash, err := auth.HashPassword([]byte("test-only-password"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{}
	cfg.Server.Mode = "development"
	cfg.Server.PublicOrigin = "http://127.0.0.1:5173"
	cfg.Auth.PasswordHash = hash
	cfg.Auth.SessionTTL = "168h"
	cfg.Storage.Path = filepath.Join(dir, "data/db.sqlite")
	store, err := storage.Open(ctx, cfg.Storage.Path, hash)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	project, err := store.CreateProject(ctx, "测试", []string{root}, 0)
	if err != nil {
		t.Fatal(err)
	}
	registered, err := store.RegisteredFolder(ctx, project.ID, project.MainFolderID, 1)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := files.ReadContent(registered, "file.txt")
	if err != nil {
		t.Fatal(err)
	}
	logs := &bytes.Buffer{}
	router, err := newRouter(cfg, store, slog.New(slog.NewJSONHandler(logs, nil)), httpElevationBackend{})
	if err != nil {
		t.Fatal(err)
	}
	cookie, csrf := login(t, router, cfg)
	routes := []struct{ method, path string }{{"POST", fmt.Sprintf("/api/v1/projects/%s/folders/%s/elevation-requests", project.ID, project.MainFolderID)}, {"POST", "/api/v1/elevation-requests/" + strings.Repeat("a", 64) + "/execute"}, {"GET", "/api/v1/elevation-requests/" + strings.Repeat("a", 64)}, {"DELETE", "/api/v1/elevation-requests/" + strings.Repeat("a", 64)}}
	for _, route := range routes {
		if got := request(router, route.method, route.path, "{}", cfg.Server.PublicOrigin, nil, ""); got.Code != 401 {
			t.Fatal(route, got.Code, got.Body)
		}
		if route.method != "GET" {
			for _, auth := range []struct{ origin, csrf string }{{"https://evil.invalid", csrf}, {cfg.Server.PublicOrigin, ""}} {
				if got := request(router, route.method, route.path, "{}", auth.origin, cookie, auth.csrf); got.Code != 403 {
					t.Fatal(route, got.Code)
				}
			}
		}
	}
	data, _ := json.Marshal(map[string]any{"project_version": 1, "path": "file.txt", "expected_version": snapshot.Version, "content": "synthetic-body-sentinel"})
	prepare := routes[0].path
	for _, body := range []string{`{"project_version":1,"project_version":1}`, `{"unknown":true}`, strings.Replace(string(data), `"identity":`, `"unknown":1,"identity":`, 1)} {
		if got := request(router, "POST", prepare, body, cfg.Server.PublicOrigin, cookie, csrf); got.Code != 400 {
			t.Fatal(got.Code, got.Body)
		}
	}
	got := request(router, "POST", prepare, string(data), cfg.Server.PublicOrigin, cookie, csrf)
	if got.Code != 200 {
		t.Fatal(got.Code, got.Body)
	}
	var envelope struct{ Data elevation.Prepared }
	if err = json.Unmarshal(got.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	other, _ := login(t, router, cfg)
	path := "/api/v1/elevation-requests/" + envelope.Data.ID
	if got = request(router, "GET", path, "", cfg.Server.PublicOrigin, other, ""); got.Code != 404 {
		t.Fatal(got.Code)
	}
	body := `{"password":"synthetic-password-sentinel","content":"synthetic-body-sentinel"}`
	got = request(router, "POST", path+"/execute", body, cfg.Server.PublicOrigin, cookie, csrf)
	if got.Code != 200 || !strings.Contains(got.Body.String(), "authorization_failed") {
		t.Fatal(got.Code, got.Body)
	}
	if got = request(router, "POST", path+"/execute", body, cfg.Server.PublicOrigin, cookie, csrf); got.Code != 409 {
		t.Fatal(got.Code, got.Body)
	}
	if got = request(router, "GET", "/api/v1/auth/session", "", cfg.Server.PublicOrigin, cookie, ""); got.Code != http.StatusOK {
		t.Fatal("system failure logged out app")
	}
	if strings.Contains(logs.String(), "synthetic-password-sentinel") || strings.Contains(logs.String(), "synthetic-body-sentinel") {
		t.Fatal("secret leaked to logs")
	}
}
func TestElevationDefaultUnavailable(t *testing.T) {
	r, cfg, _ := setup(t)
	cookie, csrf := login(t, r, cfg)
	w := request(r, "POST", "/api/v1/elevation-requests/"+strings.Repeat("a", 64)+"/execute", `{"password":"synthetic","content":"text"}`, cfg.Server.PublicOrigin, cookie, csrf)
	if w.Code != 404 {
		t.Fatal(w.Code, w.Body)
	}
}
