package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"persistty/internal/storage"
	"persistty/internal/workspace"
)

func (a *api) directories(c *gin.Context) {
	if len(c.Query("path")) > 4096 {
		a.fail(c, 400, "invalid_request", "目录路径过长。")
		return
	}
	listing, err := workspace.Browse(c.Query("path"))
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, listing)
}

func (a *api) createProject(c *gin.Context) {
	var input struct {
		Name        string   `json:"name"`
		FolderPaths []string `json:"folder_paths"`
		MainIndex   *int     `json:"main_index"`
	}
	if err := decodeJSON(c.Writer, c.Request, &input, "name", "folder_paths", "main_index"); err != nil {
		a.decodeFailure(c, err)
		return
	}
	if input.MainIndex == nil {
		a.fail(c, 400, "invalid_request", "请选择主文件夹。")
		return
	}
	project, err := a.store.CreateProject(c.Request.Context(), input.Name, input.FolderPaths, *input.MainIndex)
	if err != nil {
		a.error(c, err)
		return
	}
	c.JSON(http.StatusCreated, successEnvelope{project, c.GetString("request_id")})
}

func (a *api) updateProject(c *gin.Context) {
	if len(c.Param("id")) > 128 {
		a.fail(c, 400, "invalid_request", "请求格式无效。")
		return
	}
	var input storage.ProjectChange
	if err := decodeJSON(c.Writer, c.Request, &input, "expected_version", "name", "add_paths", "remove_folder_ids", "main_folder_id", "main_added_index"); err != nil {
		a.decodeFailure(c, err)
		return
	}
	project, err := a.store.UpdateProject(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, project)
}

func (a *api) deleteProject(c *gin.Context) {
	if len(c.Param("id")) > 128 {
		a.fail(c, 400, "invalid_request", "请求格式无效。")
		return
	}
	var input struct {
		ExpectedVersion *int64 `json:"expected_version"`
	}
	if err := decodeJSON(c.Writer, c.Request, &input, "expected_version"); err != nil {
		a.decodeFailure(c, err)
		return
	}
	if input.ExpectedVersion == nil {
		a.fail(c, 428, "version_required", "需要项目版本。")
		return
	}
	if err := a.store.DeleteProject(c.Request.Context(), c.Param("id"), *input.ExpectedVersion); err != nil {
		a.error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *api) decodeFailure(c *gin.Context, err error) {
	if errors.Is(err, errBodyLarge) {
		a.fail(c, 413, "too_large", "请求体过大。")
	} else {
		a.fail(c, 400, "invalid_request", "请求格式无效。")
	}
}
