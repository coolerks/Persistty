package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"persistty/internal/auth"
	"persistty/internal/config"
	"persistty/internal/storage"
)

func setup(t *testing.T) (*gin.Engine, config.Config, *bytes.Buffer) {
	t.Helper()
	hash, err := auth.HashPassword([]byte("test-only-password"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{}
	cfg.Server.Mode = "development"
	cfg.Server.PublicOrigin = "http://127.0.0.1:5173"
	cfg.Auth.PasswordHash = hash
	cfg.Auth.SessionTTL = "168h"
	cfg.Storage.Path = filepath.Join(t.TempDir(), "data", "db.sqlite")
	store, err := storage.Open(context.Background(), cfg.Storage.Path, hash)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	logs := &bytes.Buffer{}
	r, err := New(cfg, store, slog.New(slog.NewJSONHandler(logs, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return r, cfg, logs
}
func request(r http.Handler, method, path, body, origin string, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "192.0.2.1:4321"
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
func login(t *testing.T, r http.Handler, cfg config.Config) (*http.Cookie, string) {
	t.Helper()
	w := request(r, "POST", "/api/v1/auth/login", `{"password":"test-only-password"}`, cfg.Server.PublicOrigin, nil, "")
	if w.Code != 200 {
		t.Fatalf("login %d %s", w.Code, w.Body)
	}
	var envelope struct{ Data auth.Session }
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("cookie missing")
	}
	return cookies[0], envelope.Data.CSRFToken
}
func TestRouterProtectionAndLogout(t *testing.T) {
	r, cfg, logs := setup(t)
	for _, route := range r.Routes() {
		if route.Path == "/api/v1/auth/login" {
			continue
		}
		w := request(r, route.Method, strings.Replace(route.Path, ":id", "unknown", 1), "", cfg.Server.PublicOrigin, nil, "")
		if w.Code != 401 {
			t.Fatalf("unprotected %s %d", route.Path, w.Code)
		}
	}
	for _, path := range []string{"/api/v1/unknown", "/api/v1/files/content", "/api/v1/terminals/t/attach", "/api/v1/projects/secret"} {
		if w := request(r, "GET", path, "", "", nil, ""); w.Code != 401 {
			t.Fatalf("unknown bypass %s: %d", path, w.Code)
		}
	}
	cookie, csrf := login(t, r, cfg)
	if !cookie.HttpOnly || cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" || cookie.Domain != "" || cookie.MaxAge <= 0 {
		t.Fatal("unsafe cookie")
	}
	other, _ := login(t, r, cfg)
	if other.Value == cookie.Value {
		t.Fatal("fixed token")
	}
	w := request(r, "GET", "/api/v1/projects", "", "", cookie, "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"items":[]`) {
		t.Fatalf("projects list %d %s", w.Code, w.Body)
	}
	if w = request(r, "GET", "/api/v1/terminals", "", "", cookie, ""); w.Code != 503 {
		t.Fatalf("missing tmux server must not look like an empty terminal list: %d %s", w.Code, w.Body)
	}
	if w := request(r, "GET", "/api/v1/projects/missing", "", "", cookie, ""); w.Code != 404 {
		t.Fatal("unknown project faked")
	}
	for _, tt := range []struct{ origin, csrf string }{{"", csrf}, {"null", csrf}, {"http://evil.test", csrf}, {cfg.Server.PublicOrigin, ""}, {cfg.Server.PublicOrigin, "wrong"}} {
		if w := request(r, "POST", "/api/v1/auth/logout", "", tt.origin, cookie, tt.csrf); w.Code != 403 {
			t.Fatalf("CSRF bypass %d", w.Code)
		}
		if w := request(r, "POST", "/api/v1/terminals", `{}`, tt.origin, cookie, tt.csrf); w.Code != 403 {
			t.Fatalf("terminal create CSRF bypass %d", w.Code)
		}
	}
	for _, origin := range []string{"", "null", "http://evil.test"} {
		if w := request(r, "GET", "/api/v1/terminals/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/stream", "", origin, cookie, ""); w.Code != 403 {
			t.Fatalf("terminal stream Origin bypass %d", w.Code)
		}
	}
	w = request(r, "POST", "/api/v1/auth/logout", "", cfg.Server.PublicOrigin, cookie, csrf)
	if w.Code != 204 || w.Body.Len() != 0 || w.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("logout")
	}
	if w = request(r, "GET", "/api/v1/auth/session", "", "", cookie, ""); w.Code != 401 {
		t.Fatal("logout not revoked")
	}
	if w = request(r, "GET", "/api/v1/auth/session", "", "", other, ""); w.Code != 200 {
		t.Fatal("other session lost")
	}
	for _, secret := range []string{"test-only-password", cookie.Value, other.Value, csrf, cfg.Auth.PasswordHash} {
		if strings.Contains(logs.String(), secret) {
			t.Fatal("secret logged")
		}
	}
}
func TestLoginStrictJSONAndOrigin(t *testing.T) {
	r, cfg, _ := setup(t)
	for _, tt := range []struct {
		body, origin string
		code         int
	}{
		{`{"password":"test-only-password"}`, "", 403}, {`{"password":"test-only-password"}`, "null", 403}, {`{"password":"test-only-password"}`, "http://evil.test", 403},
		{`{"password":"a","password":"b"}`, cfg.Server.PublicOrigin, 400}, {`{"password":"a","extra":true}`, cfg.Server.PublicOrigin, 400}, {`{"password":null}`, cfg.Server.PublicOrigin, 400}, {`{"password":123}`, cfg.Server.PublicOrigin, 400}, {`[]`, cfg.Server.PublicOrigin, 400}, {`{} {}`, cfg.Server.PublicOrigin, 400}, {`{"password":"` + strings.Repeat("x", 9000) + `"}`, cfg.Server.PublicOrigin, 413}, {`{"password":"` + strings.Repeat("x", 1025) + `"}`, cfg.Server.PublicOrigin, 400}, {`{"password":"incorrect-password"}`, cfg.Server.PublicOrigin, 401},
		{`{"Password":"test-only-password"}`, cfg.Server.PublicOrigin, 400},
		{`{"password":"a","passw\u006frd":"b"}`, cfg.Server.PublicOrigin, 400},
	} {
		w := request(r, "POST", "/api/v1/auth/login", tt.body, tt.origin, nil, "")
		if w.Code != tt.code {
			t.Fatalf("body %d bytes: got %d expected%d", len(tt.body), w.Code, tt.code)
		}
		if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Request-ID") == "" {
			t.Fatal("response guards")
		}
	}
}
func TestForwardedHeadersNotTrustedAndRateLimit(t *testing.T) {
	r, cfg, _ := setup(t)
	for i := 0; i < 11; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"password":"incorrect-password"}`))
		req.RemoteAddr = "192.0.2.2:1234"
		req.Header.Set("Origin", cfg.Server.PublicOrigin)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "spoof-"+strings.Repeat("x", i))
		req.Header.Set("X-Forwarded-Proto", "https")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		want := 401
		if i == 10 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("attempt%d got%d", i, w.Code)
		}
		if i == 10 && w.Header().Get("Retry-After") == "" {
			t.Fatal("retry header")
		}
	}
}
func TestRecoveryDoesNotLeak(t *testing.T) {
	r, cfg, logs := setup(t)
	r.GET("/panic-test", func(c *gin.Context) { panic("secret-panic-password") })
	w := request(r, "GET", "/panic-test", "", "", nil, "")
	if w.Code != 500 || strings.Contains(logs.String(), "secret-panic-password") || strings.Contains(w.Body.String(), "secret-panic-password") {
		t.Fatal("panic leak")
	}
	a := api{cfg: cfg}
	r.GET("/cookie-test", func(c *gin.Context) { a.cfg.Server.Mode = "tls"; a.cookie(c, "fake", auth.Session{}.ExpiresAt) })
	w = request(r, "GET", "/cookie-test", "", "", nil, "")
	cookie := w.Result().Cookies()[0]
	if cookie.Name != "__Host-persistty_session" || !cookie.Secure {
		t.Fatal("TLS cookie")
	}
}
func TestCanceledRequestDoesNotReportServiceFault(t *testing.T) {
	r, cfg, logs := setup(t)
	cookie, _ := login(t, r, cfg)
	logs.Reset()
	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	req.AddCookie(cookie)
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req.WithContext(ctx))
	if w.Body.Len() != 0 || strings.Contains(logs.String(), "request_error") || strings.Contains(logs.String(), `"level":"ERROR"`) {
		t.Fatal("canceled client request was reported as a service fault")
	}
	logs.Reset()
	a := api{logger: slog.New(slog.NewJSONHandler(logs, nil))}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	a.error(c, fmt.Errorf("wrapped: %w", context.Canceled))
	if !c.IsAborted() || c.Writer.Written() || logs.Len() != 0 {
		t.Fatal("wrapped cancellation emitted a fault response or log")
	}
}
func TestSharedFixtures(t *testing.T) {
	body, err := os.ReadFile("../../tests/contracts/foundation.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]json.RawMessage
	if err = json.Unmarshal(body, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"session", "projects", "project", "empty_projects", "terminals", "empty_terminals", "unauthenticated", "not_found", "invalid_request"} {
		raw, ok := fixtures[name]
		if !ok {
			t.Fatalf("missing %s", name)
		}
		var envelope struct {
			Data      json.RawMessage `json:"data"`
			Error     *failure        `json:"error"`
			RequestID string          `json:"request_id"`
		}
		d := json.NewDecoder(bytes.NewReader(raw))
		d.DisallowUnknownFields()
		if err = d.Decode(&envelope); err != nil || envelope.RequestID == "" {
			t.Fatal("fixture envelope")
		}
		switch name {
		case "session":
			var data auth.Session
			d = json.NewDecoder(bytes.NewReader(envelope.Data))
			d.DisallowUnknownFields()
			if err = d.Decode(&data); err != nil || !data.Authenticated || data.CSRFToken == "" || data.ExpiresAt.IsZero() {
				t.Fatal("session fixture")
			}
		case "project":
			var data storage.Project
			d = json.NewDecoder(bytes.NewReader(envelope.Data))
			d.DisallowUnknownFields()
			if err = d.Decode(&data); err != nil || len(data.Folders) != 1 || data.MainFolderID != data.Folders[0].ID {
				t.Fatal("project fixture")
			}
		case "projects", "empty_projects":
			var data struct {
				Items []storage.Project `json:"items"`
			}
			d = json.NewDecoder(bytes.NewReader(envelope.Data))
			d.DisallowUnknownFields()
			if err = d.Decode(&data); err != nil || data.Items == nil {
				t.Fatal("projects fixture")
			}
		case "terminals", "empty_terminals":
			var data struct {
				Items []storage.Terminal `json:"items"`
			}
			d = json.NewDecoder(bytes.NewReader(envelope.Data))
			d.DisallowUnknownFields()
			if err = d.Decode(&data); err != nil || data.Items == nil {
				t.Fatal("terminals fixture")
			}
		default:
			if envelope.Error == nil || envelope.Error.Code == "" || len(envelope.Data) != 0 {
				t.Fatal("error fixture")
			}
		}
	}
}
