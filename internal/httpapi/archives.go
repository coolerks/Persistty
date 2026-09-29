package httpapi

import (
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"
	"persistty/internal/transfer"
)

func (a *api) createArchive(c *gin.Context) {
	var input transfer.ArchiveInput
	if err := decodeJSON(c.Writer, c.Request, &input, "project_id", "folder_id", "project_version", "path"); err != nil {
		a.decodeFailure(c, err)
		return
	}
	item, err := a.transfer.CreateArchive(c.Request.Context(), input)
	if err != nil {
		a.error(c, err)
		return
	}
	c.JSON(http.StatusAccepted, successEnvelope{item, c.GetString("request_id")})
}

func (a *api) archiveStatus(c *gin.Context) {
	item, err := a.transfer.ArchiveStatus(c.Request.Context(), c.Param("id"))
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, item)
}

func (a *api) downloadArchive(c *gin.Context) {
	download, err := a.transfer.ArchiveDownload(c.Request.Context(), c.Param("id"))
	if err != nil {
		a.error(c, err)
		return
	}
	defer download.Close()
	info, err := download.File.Stat()
	if err != nil {
		a.error(c, err)
		return
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": download.Filename}))
	c.Header("Cache-Control", "no-store")
	http.ServeContent(c.Writer, c.Request, download.Filename, info.ModTime(), download.File)
}

func (a *api) cancelArchive(c *gin.Context) {
	if err := a.transfer.CancelArchive(c.Request.Context(), c.Param("id")); err != nil {
		a.error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
