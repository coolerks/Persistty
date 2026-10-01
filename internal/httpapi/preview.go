package httpapi

import (
	"github.com/gin-gonic/gin"
	"persistty/internal/files"
)

func (a *api) inspectFile(c *gin.Context) {
	root, ok := a.folderRequest(c)
	if !ok {
		return
	}
	info, err := files.Inspect(root, c.Query("path"))
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, info)
}
func (a *api) previewFile(c *gin.Context) {
	root, ok := a.folderRequest(c)
	if !ok {
		return
	}
	data, info, err := files.ReadPreview(root, c.Query("path"))
	if err != nil {
		a.error(c, err)
		return
	}
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox; frame-ancestors 'none'")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", "inline")
	c.Header("Cache-Control", "no-store")
	c.Data(200, info.MIME, data)
}
