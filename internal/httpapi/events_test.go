package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"persistty/internal/storage"
)

func TestFileEventsAuthenticatedAndRescan(t *testing.T) {
	r, cfg, _ := setup(t)
	root := t.TempDir()
	cookie, csrf := login(t, r, cfg)
	created := request(r, "POST", "/api/v1/projects", fmt.Sprintf(`{"name":"监听","folder_paths":[%q],"main_index":0}`, root), cfg.Server.PublicOrigin, cookie, csrf)
	if created.Code != 201 {
		t.Fatalf("project %d %s", created.Code, created.Body)
	}
	var project struct{ Data storage.Project }
	if err := json.Unmarshal(created.Body.Bytes(), &project); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(r)
	defer server.Close()
	endpoint := strings.Replace(server.URL, "http://", "ws://", 1) + "/api/v1/events?project_id=" + url.QueryEscape(project.Data.ID) + "&folder_id=" + url.QueryEscape(project.Data.MainFolderID) + "&project_version=1&path="
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if conn, response, err := websocket.Dial(ctx, endpoint, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{cfg.Server.PublicOrigin}}}); err == nil || response == nil || response.StatusCode != 401 {
		if conn != nil {
			conn.Close(websocket.StatusNormalClosure, "")
		}
		t.Fatalf("anonymous events allowed: response=%v err=%v", response, err)
	}
	conn, response, err := websocket.Dial(ctx, endpoint, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{cfg.Server.PublicOrigin}, "Cookie": []string{cookie.String()}}})
	if err != nil {
		t.Fatalf("dial events: %v (%v)", err, response)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	read := func() fileEvent {
		t.Helper()
		_, body, err := conn.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var event fileEvent
		if err := json.Unmarshal(body, &event); err != nil {
			t.Fatal(err)
		}
		return event
	}
	initial := read()
	if !initial.Rescan || initial.ProjectID != project.Data.ID || initial.FolderID != project.Data.MainFolderID {
		t.Fatalf("initial event = %#v", initial)
	}
	if err := os.WriteFile(filepath.Join(root, "external.txt"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	next := read()
	if !next.Rescan || next.Revision <= initial.Revision {
		t.Fatalf("change event = %#v", next)
	}
}
