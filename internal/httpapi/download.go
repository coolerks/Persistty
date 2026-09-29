package httpapi

import (
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (a *api) downloadFile(c *gin.Context) {
	root, ok := a.folderRequest(c)
	if !ok {
		return
	}
	download, err := a.transfer.DownloadFile(c.Request.Context(), root, c.Query("path"))
	if err != nil {
		a.error(c, err)
		return
	}
	defer download.Close()
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": download.Filename})
	info, err := download.File.Stat()
	if err != nil {
		a.error(c, err)
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", disposition)
	c.Header("Cache-Control", "no-store")
	http.ServeContent(c.Writer, c.Request, download.Filename, info.ModTime(), download.File)
}
