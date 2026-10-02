package httpapi

import (
	"errors"
	"syscall"

	"github.com/gin-gonic/gin"
	"persistty/internal/elevation"
	"persistty/internal/files"
)

func (a *api) registerElevation(r *gin.RouterGroup) {
	r.POST("/projects/:id/folders/:folderId/elevation-requests", a.prepareElevation)
	r.POST("/elevation-requests/:requestId/execute", a.executeElevation)
	r.GET("/elevation-requests/:requestId", a.elevationStatus)
	r.DELETE("/elevation-requests/:requestId", a.cancelElevation)
}
func (a *api) prepareElevation(c *gin.Context) {
	var input struct {
		ProjectVersion int64          `json:"project_version"`
		Path           string         `json:"path"`
		Expected       *files.Version `json:"expected_version"`
		Content        string         `json:"content"`
	}
	if len(c.Param("id")) > 128 || len(c.Param("folderId")) > 128 {
		a.fail(c, 400, "invalid_request", "请求格式无效。")
		return
	}
	if err := decodeJSONWithLimit(c.Writer, c.Request, &input, 6*elevation.MaxContent+8192, "project_version", "path", "expected_version", "content"); err != nil {
		a.decodeFailure(c, err)
		return
	}
	if input.ProjectVersion < 1 || input.Expected == nil {
		a.fail(c, 428, "version_required", "需要文件和项目版本。")
		return
	}
	root, err := a.store.RegisteredFolder(c.Request.Context(), c.Param("id"), c.Param("folderId"), input.ProjectVersion)
	if err != nil {
		a.error(c, err)
		return
	}
	content := []byte(input.Content)
	input.Content = ""
	defer clear(content)
	prepared, err := a.elevation.Prepare(c.Request.Context(), c.MustGet("token").(string), root, input.Path, *input.Expected, content)
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, prepared)
}
func (a *api) executeElevation(c *gin.Context) {
	var input struct {
		Password string `json:"password"`
		Content  string `json:"content"`
	}
	if err := decodeJSONWithLimit(c.Writer, c.Request, &input, 6*elevation.MaxContent+8192, "password", "content"); err != nil {
		a.decodeFailure(c, err)
		return
	}
	password, content := []byte(input.Password), []byte(input.Content)
	input.Password = ""
	input.Content = ""
	defer clear(password)
	defer clear(content)
	result, err := a.elevation.Execute(c.Request.Context(), c.MustGet("token").(string), c.Param("requestId"), password, content)
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, result)
}
func (a *api) elevationStatus(c *gin.Context) {
	result, err := a.elevation.Status(c.Request.Context(), c.MustGet("token").(string), c.Param("requestId"))
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, result)
}
func (a *api) cancelElevation(c *gin.Context) {
	result, err := a.elevation.Cancel(c.Request.Context(), c.MustGet("token").(string), c.Param("requestId"))
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, result)
}

func permissionFailure(err error) bool {
	return errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM)
}
