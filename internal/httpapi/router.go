package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"modernc.org/sqlite"
	"persistty/internal/auth"
	"persistty/internal/config"
	"persistty/internal/storage"
)

type api struct {
	cfg    config.Config
	store  *storage.Store
	auth   *auth.Service
	logger *slog.Logger
}
type failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type errorEnvelope struct {
	Error     failure `json:"error"`
	RequestID string  `json:"request_id"`
}
type successEnvelope struct {
	Data      any    `json:"data"`
	RequestID string `json:"request_id"`
}

func New(cfg config.Config, store *storage.Store, logger *slog.Logger) (*gin.Engine, error) {
	service, err := auth.New(store, cfg.Auth.PasswordHash, cfg.TTL())
	if err != nil {
		return nil, err
	}
	a := &api{cfg, store, service, logger}
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	if err = r.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		return nil, err
	}
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(a.middleware())
	r.POST("/api/v1/auth/login", a.login)
	protected := r.Group("/api/v1")
	protected.Use(a.authenticate())
	protected.GET("/auth/session", func(c *gin.Context) { a.success(c, c.MustGet("session")) })
	protected.POST("/auth/logout", func(c *gin.Context) {
		if err := service.Logout(c.Request.Context(), c.MustGet("token").(string)); err != nil {
			a.error(c, err)
			return
		}
		a.cookie(c, "", time.Time{})
		c.Status(http.StatusNoContent)
	})
	protected.GET("/projects", func(c *gin.Context) {
		items, err := store.Projects(c.Request.Context())
		if err != nil {
			a.error(c, err)
			return
		}
		a.success(c, struct {
			Items []storage.Project `json:"items"`
		}{items})
	})
	protected.GET("/projects/:id", func(c *gin.Context) {
		id := c.Param("id")
		if len(id) > 128 {
			a.fail(c, 400, "invalid_request", "请求格式无效。")
			return
		}
		item, err := store.Project(c.Request.Context(), id)
		if err != nil {
			a.error(c, err)
			return
		}
		a.success(c, item)
	})
	protected.GET("/terminals", func(c *gin.Context) {
		items, err := store.Terminals(c.Request.Context())
		if err != nil {
			a.error(c, err)
			return
		}
		a.success(c, struct {
			Items []storage.Terminal `json:"items"`
		}{items})
	})
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			a.authenticate()(c)
			if c.IsAborted() {
				return
			}
		}
		a.fail(c, 404, "not_found", "资源不存在。")
	})
	return r, nil
}
func (a *api) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := make([]byte, 16)
		if _, err := rand.Read(raw); err != nil {
			c.AbortWithStatus(503)
			return
		}
		id := hex.EncodeToString(raw)
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Header("Cache-Control", "no-store")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		start := time.Now()
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		defer func() {
			if recover() != nil {
				a.logger.Error("请求处理异常", "event", "request_panic", "request_id", id, "error_code", "internal_error")
				a.fail(c, 500, "internal_error", "服务器内部错误。")
			}
			a.logger.Info("请求完成", "event", "http_request", "request_id", id, "status", c.Writer.Status(), "duration_ms", time.Since(start).Milliseconds())
		}()
		c.Next()
	}
}
func (a *api) origin(c *gin.Context) bool {
	return len(c.Request.Header.Values("Origin")) == 1 && c.GetHeader("Origin") == a.cfg.Server.PublicOrigin
}
func (a *api) authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookies := c.Request.CookiesNamed(a.cfg.CookieName())
		if len(cookies) != 1 {
			a.fail(c, 401, "unauthenticated", "请先登录。")
			return
		}
		token := cookies[0].Value
		session, err := a.auth.Lookup(c.Request.Context(), token)
		if err != nil {
			a.error(c, err)
			return
		}
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" && c.Request.Method != "OPTIONS" {
			if !a.origin(c) || len(c.Request.Header.Values("X-CSRF-Token")) != 1 || !auth.ValidCSRF(c.GetHeader("X-CSRF-Token"), session.CSRFToken) {
				a.fail(c, 403, "forbidden", "请求来源或授权无效。")
				return
			}
		}
		c.Set("token", token)
		c.Set("session", session)
	}
}
func (a *api) login(c *gin.Context) {
	if !a.origin(c) {
		a.fail(c, 403, "forbidden", "请求来源或授权无效。")
		return
	}
	var input struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(c.Writer, c.Request, &input, "password"); err != nil {
		if errors.Is(err, errBodyLarge) {
			a.fail(c, 413, "too_large", "请求体过大。")
		} else {
			a.fail(c, 400, "invalid_request", "请求格式无效。")
		}
		return
	}
	password := []byte(input.Password)
	input.Password = ""
	defer clear(password)
	if len(password) == 0 || len(password) > auth.MaxPasswordBytes {
		a.fail(c, 400, "invalid_request", "请求格式无效。")
		return
	}
	session, token, err := a.auth.Login(c.Request.Context(), c.ClientIP(), password)
	if err != nil {
		a.error(c, err)
		return
	}
	a.cookie(c, token, session.ExpiresAt)
	a.success(c, session)
}
func (a *api) cookie(c *gin.Context, token string, expires time.Time) {
	cookie := &http.Cookie{Name: a.cfg.CookieName(), Value: token, Path: "/", HttpOnly: true, Secure: a.cfg.SecureCookie(), SameSite: http.SameSiteStrictMode, Expires: expires}
	if token == "" {
		cookie.MaxAge = -1
	} else {
		cookie.MaxAge = int(time.Until(expires).Seconds())
	}
	http.SetCookie(c.Writer, cookie)
}
func (a *api) success(c *gin.Context, data any) {
	c.JSON(200, successEnvelope{data, c.GetString("request_id")})
}
func (a *api) fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, errorEnvelope{failure{code, message}, c.GetString("request_id")})
}
func (a *api) error(c *gin.Context, err error) {
	switch {
	case errors.Is(err, context.Canceled) || errors.Is(c.Request.Context().Err(), context.Canceled):
		c.Abort()
	case errors.Is(err, auth.ErrUnauthenticated):
		a.fail(c, 401, "unauthenticated", "请先登录。")
	case errors.Is(err, auth.ErrRateLimited):
		c.Header("Retry-After", "60")
		a.fail(c, 429, "rate_limited", "请求过于频繁，请稍后重试。")
	case errors.Is(err, storage.ErrNotFound):
		a.fail(c, 404, "not_found", "资源不存在。")
	case errors.Is(err, storage.ErrListLimit):
		a.fail(c, 413, "too_large", "资源数量超过上限。")
	default:
		var sqliteError *sqlite.Error
		if errors.As(err, &sqliteError) && (sqliteError.Code()&255 == 5 || sqliteError.Code()&255 == 6) {
			a.logger.Warn("数据库暂时不可用", "event", "dependency_error", "request_id", c.GetString("request_id"), "error_code", "unavailable")
			a.fail(c, 503, "unavailable", "服务暂时不可用。")
			return
		}
		a.logger.Error("请求处理失败", "event", "request_error", "request_id", c.GetString("request_id"), "error_code", "internal_error")
		a.fail(c, 500, "internal_error", "服务器内部错误。")
	}
}
