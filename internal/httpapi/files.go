package httpapi

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"persistty/internal/files"
	"persistty/internal/storage"
)

func (a *api) folderRequest(c *gin.Context) (storage.RegisteredFolder, bool) {
	if len(c.Param("id")) > 128 || len(c.Param("folderId")) > 128 || len(c.Query("path")) > 4096 {
		a.fail(c, 400, "invalid_request", "请求格式无效。")
		return storage.RegisteredFolder{}, false
	}
	version, err := strconv.ParseInt(c.Query("project_version"), 10, 64)
	if err != nil || version < 1 {
		a.fail(c, 428, "version_required", "需要项目配置版本。")
		return storage.RegisteredFolder{}, false
	}
	root, err := a.store.RegisteredFolder(c.Request.Context(), c.Param("id"), c.Param("folderId"), version)
	if err != nil {
		a.error(c, err)
		return storage.RegisteredFolder{}, false
	}
	return root, true
}

func (a *api) entries(c *gin.Context) {
	root, ok := a.folderRequest(c)
	if !ok {
		return
	}
	limit := 100
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			a.fail(c, 400, "invalid_request", "分页大小无效。")
			return
		}
		limit = parsed
	}
	listing, err := files.List(root, c.Query("path"), c.Query("cursor"), limit)
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, listing)
}

func (a *api) content(c *gin.Context) {
	root, ok := a.folderRequest(c)
	if !ok {
		return
	}
	content, err := files.ReadContent(root, c.Query("path"))
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, content)
}

func (a *api) metadata(c *gin.Context) {
	root, ok := a.folderRequest(c)
	if !ok {
		return
	}
	metadata, err := files.ReadMetadata(root, c.Query("path"))
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, metadata)
}

func (a *api) saveContent(c *gin.Context) {
	if len(c.Param("id")) > 128 || len(c.Param("folderId")) > 128 {
		a.fail(c, 400, "invalid_request", "请求格式无效。")
		return
	}
	var input struct {
		ProjectVersion  int64          `json:"project_version"`
		Path            string         `json:"path"`
		ExpectedVersion *files.Version `json:"expected_version"`
		Content         string         `json:"content"`
	}
	// JSON escaping may use six bytes per source byte; the file layer still enforces 8 MiB.
	if err := decodeJSONWithLimit(c.Writer, c.Request, &input, 6*(8<<20)+8192, "project_version", "path", "expected_version", "content"); err != nil {
		a.decodeFailure(c, err)
		return
	}
	if input.ExpectedVersion == nil || input.ProjectVersion < 1 {
		a.fail(c, 428, "version_required", "需要文件和项目版本。")
		return
	}
	var version files.Version
	err := a.store.WithRegisteredFolder(c.Request.Context(), c.Param("id"), c.Param("folderId"), input.ProjectVersion, func(root storage.RegisteredFolder) error {
		var err error
		version, err = files.Save(c.Request.Context(), root, input.Path, *input.ExpectedVersion, input.Content)
		return err
	})
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, struct {
		Version files.Version `json:"version"`
	}{version})
}
