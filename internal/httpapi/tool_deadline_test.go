package httpapi

import (
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"persistty/internal/config"
	"testing"
	"time"
)

func TestRegisteredToolRoutesUseConfiguredDeadline(t *testing.T) {
	for _, seconds := range []int{3, 15, 25} {
		cfg := config.Config{}
		cfg.Search.TimeoutSeconds = seconds
		a := &api{cfg: cfg, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
		router := gin.New()
		router.Use(a.middleware())
		routes := []struct {
			pattern, url string
			tool         bool
		}{
			{"/api/v1/projects/:id/repositories/:repoId/status", "/api/v1/projects/p/repositories/r/status", true},
			{"/api/v1/projects/:id/git-baseline", "/api/v1/projects/p/git-baseline", true},
			{"/api/v1/projects/:id/repositories/:repoId/commits/:commitId", "/api/v1/projects/p/repositories/r/commits/c", true},
			{"/api/v1/projects/:id/file-names", "/api/v1/projects/p/file-names", true},
			{"/api/v1/projects/:id/searches", "/api/v1/projects/p/searches", true},
			{"/api/v1/projects/:id/folders/:folderId/content", "/api/v1/projects/p/folders/repositories/content", false},
		}
		for _, route := range routes {
			router.GET(route.pattern, func(c *gin.Context) {
				d, ok := c.Request.Context().Deadline()
				if !ok {
					t.Error("missing deadline")
				}
				want := 10 * time.Second
				if route.tool {
					want = time.Duration(seconds) * time.Second
				}
				remaining := time.Until(d)
				if remaining > want || remaining < want-time.Second {
					t.Errorf("%s budget %s want %s", c.FullPath(), remaining, want)
				}
				c.Status(204)
			})
		}
		for _, route := range routes {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, route.url, nil))
			if w.Code != 204 {
				t.Fatal(w.Code)
			}
		}
	}
}
