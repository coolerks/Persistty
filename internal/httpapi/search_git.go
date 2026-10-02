package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"persistty/internal/gitview"
	"persistty/internal/search"
	"strconv"
)

func (a *api) registerSearchGit(r *gin.RouterGroup) {
	r.GET("/projects/:id/file-names", a.fileNames)
	r.POST("/projects/:id/searches", a.createSearch)
	r.DELETE("/projects/:id/searches/:searchId", a.cancelSearch)
	r.POST("/projects/:id/replace-previews", a.createReplacePreview)
	r.GET("/projects/:id/replace-previews/:previewId", a.replacePreview)
	r.POST("/projects/:id/replace-previews/:previewId/apply", a.applyReplacement)
	r.DELETE("/projects/:id/replace-previews/:previewId", a.cancelReplacement)
	r.GET("/projects/:id/repositories", a.repositories)
	r.GET("/projects/:id/repositories/:repoId/status", a.gitStatus)
	r.GET("/projects/:id/repositories/:repoId/refs", a.gitRefs)
	r.GET("/projects/:id/repositories/:repoId/log", a.gitLog)
	r.GET("/projects/:id/repositories/:repoId/commits/:commitId", a.gitDetail)
	r.POST("/projects/:id/repositories/:repoId/comparisons", a.gitComparison)
	r.GET("/projects/:id/git-baseline", a.gitBaseline)
}
func owner(c *gin.Context) string {
	hash := sha256.Sum256([]byte(c.GetString("token")))
	return hex.EncodeToString(hash[:])
}
func (a *api) projectVersion(c *gin.Context) (int64, bool) {
	v, e := strconv.ParseInt(c.Query("project_version"), 10, 64)
	if e != nil || v < 1 {
		a.fail(c, 428, "version_required", "需要项目配置版本。")
		return 0, false
	}
	if len(c.Param("id")) > 128 || len(c.Param("repoId")) > 128 {
		a.fail(c, 400, "invalid_request", "请求无效。")
		return 0, false
	}
	return v, true
}
func (a *api) createSearch(c *gin.Context) {
	var q search.Query
	if e := decodeJSONWithLimit(c.Writer, c.Request, &q, 32768, "project_version", "folder_id", "path", "pattern", "regex", "case_sensitive", "whole_word", "include", "exclude"); e != nil {
		a.decodeFailure(c, e)
		return
	}
	if q.ProjectVersion < 1 {
		a.fail(c, 428, "version_required", "需要项目配置版本。")
		return
	}
	v, e := a.search.Search(c.Request.Context(), owner(c), c.Param("id"), q)
	if e != nil {
		a.error(c, e)
		return
	}
	a.success(c, v)
}
func (a *api) cancelSearch(c *gin.Context) {
	if e := a.search.CancelSearch(owner(c), c.Param("id"), c.Param("searchId")); e != nil {
		a.error(c, e)
		return
	}
	c.Status(204)
}
func (a *api) createReplacePreview(c *gin.Context) {
	var q search.PreviewInput
	if e := decodeJSONWithLimit(c.Writer, c.Request, &q, 256<<10, "project_version", "search_id", "selected_match_ids", "replacement"); e != nil {
		a.decodeFailure(c, e)
		return
	}
	if q.ProjectVersion < 1 {
		a.fail(c, 428, "version_required", "需要项目配置版本。")
		return
	}
	v, e := a.search.Preview(c.Request.Context(), owner(c), c.Param("id"), q)
	if e != nil {
		a.error(c, e)
		return
	}
	a.success(c, v)
}
func (a *api) replacePreview(c *gin.Context) {
	if id := c.Query("file_id"); id != "" {
		v, e := a.search.Comparison(owner(c), c.Param("id"), c.Param("previewId"), id)
		if e != nil {
			a.error(c, e)
			return
		}
		a.success(c, v)
		return
	}
	v, e := a.search.PreviewStatus(owner(c), c.Param("id"), c.Param("previewId"))
	if e != nil {
		a.error(c, e)
		return
	}
	a.success(c, v)
}
func (a *api) applyReplacement(c *gin.Context) {
	var q search.ApplyInput
	if e := decodeJSONWithLimit(c.Writer, c.Request, &q, 192<<10, "project_version", "selected_file_ids", "protected_file_ids"); e != nil {
		a.decodeFailure(c, e)
		return
	}
	if q.ProjectVersion < 1 {
		a.fail(c, 428, "version_required", "需要项目配置版本。")
		return
	}
	v, e := a.search.Apply(c.Request.Context(), owner(c), c.Param("id"), c.Param("previewId"), q)
	if e != nil {
		a.error(c, e)
		return
	}
	a.success(c, v)
}
func (a *api) cancelReplacement(c *gin.Context) {
	if e := a.search.CancelPreview(owner(c), c.Param("id"), c.Param("previewId")); e != nil {
		a.error(c, e)
		return
	}
	c.Status(204)
}
func (a *api) repositories(c *gin.Context) {
	version, ok := a.projectVersion(c)
	if !ok {
		return
	}
	v, e := a.git.Repositories(c.Request.Context(), c.Param("id"), version)
	if e != nil {
		a.error(c, e)
		return
	}
	a.success(c, v)
}
func (a *api) gitStatus(c *gin.Context) {
	version, ok := a.projectVersion(c)
	if !ok {
		return
	}
	v, e := a.git.Status(c.Request.Context(), c.Param("id"), version, c.Param("repoId"))
	if e != nil {
		a.error(c, e)
		return
	}
	a.success(c, v)
}
func (a *api) gitRefs(c *gin.Context) {
	version, ok := a.projectVersion(c)
	if !ok {
		return
	}
	v, e := a.git.Refs(c.Request.Context(), c.Param("id"), version, c.Param("repoId"))
	if e != nil {
		a.error(c, e)
		return
	}
	a.success(c, gin.H{"items": v})
}
func (a *api) gitLog(c *gin.Context) {
	version, ok := a.projectVersion(c)
	if !ok {
		return
	}
	offset := 0
	var err error
	if c.Query("offset") != "" {
		offset, err = strconv.Atoi(c.Query("offset"))
		if err != nil {
			a.error(c, gitview.ErrInvalid)
			return
		}
	}
	v, e := a.git.LogAt(c.Request.Context(), c.Param("id"), version, c.Param("repoId"), offset, c.Query("head"))
	if e != nil {
		a.error(c, e)
		return
	}
	a.success(c, v)
}
func (a *api) gitDetail(c *gin.Context) {
	version, ok := a.projectVersion(c)
	if !ok {
		return
	}
	v, e := a.git.Detail(c.Request.Context(), c.Param("id"), version, c.Param("repoId"), c.Param("commitId"), c.Query("parent_id"))
	if e != nil {
		a.error(c, e)
		return
	}
	a.success(c, v)
}
func (a *api) gitComparison(c *gin.Context) {
	var q gitview.CompareInput
	if e := decodeJSONWithLimit(c.Writer, c.Request, &q, 16384, "project_version", "path", "kind", "reference", "commit_id", "parent_id"); e != nil {
		a.decodeFailure(c, e)
		return
	}
	if q.ProjectVersion < 1 {
		a.fail(c, 428, "version_required", "需要项目配置版本。")
		return
	}
	v, e := a.git.Compare(c.Request.Context(), c.Param("id"), c.Param("repoId"), q)
	if e != nil {
		a.error(c, e)
		return
	}
	a.success(c, v)
}
func (a *api) gitBaseline(c *gin.Context) {
	version, ok := a.projectVersion(c)
	if !ok {
		return
	}
	v, e := a.git.Baseline(c.Request.Context(), c.Param("id"), version, c.Query("folder_id"), c.Query("path"))
	if e != nil {
		a.error(c, e)
		return
	}
	a.success(c, v)
}

func (a *api) fileNames(c *gin.Context) {
	version, ok := a.projectVersion(c)
	if !ok {
		return
	}
	value, err := a.search.Names(c.Request.Context(), c.Param("id"), version, c.Query("query"))
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, value)
}
