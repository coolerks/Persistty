package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"persistty/internal/transfer"
)

func (a *api) createUpload(c *gin.Context) {
	var input transfer.UploadInput
	if err := decodeJSON(c.Writer, c.Request, &input, "project_id", "folder_id", "project_version", "path", "batch_id", "size", "sha256", "expected_version"); err != nil {
		a.decodeFailure(c, err)
		return
	}
	state, err := a.transfer.Start(c.Request.Context(), input)
	if err != nil {
		a.error(c, err)
		return
	}
	c.JSON(http.StatusCreated, successEnvelope{state, c.GetString("request_id")})
}
func (a *api) uploadStatus(c *gin.Context) {
	state, err := a.transfer.Status(c.Request.Context(), c.Param("id"))
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, state)
}
func (a *api) uploadChunk(c *gin.Context) {
	if c.GetHeader("Content-Type") != "application/octet-stream" {
		a.fail(c, 400, "invalid_request", "上传块类型无效。")
		return
	}
	index, err := strconv.ParseInt(c.Param("index"), 10, 64)
	if err != nil || index < 0 {
		a.fail(c, 400, "invalid_request", "上传块序号无效。")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, a.cfg.ChunkBytes()+1))
	if err != nil {
		var max *http.MaxBytesError
		if errors.As(err, &max) {
			a.fail(c, 413, "too_large", "上传块超过上限。")
		} else {
			a.fail(c, 400, "invalid_request", "无法读取上传块。")
		}
		return
	}
	state, err := a.transfer.Chunk(c.Request.Context(), c.Param("id"), index, body, c.GetHeader("X-Chunk-SHA256"))
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, state)
}
func (a *api) completeUpload(c *gin.Context) {
	result, err := a.transfer.Complete(c.Request.Context(), c.Param("id"))
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, result)
}
func (a *api) cancelUpload(c *gin.Context) {
	if err := a.transfer.Cancel(c.Request.Context(), c.Param("id")); err != nil {
		a.error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
