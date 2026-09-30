package httpapi

import (
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"persistty/internal/storage"
	"persistty/internal/terminal"
)

func (a *api) listTerminals(c *gin.Context) {
	items, err := a.terminals.List(c.Request.Context())
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, struct {
		Items []storage.Terminal `json:"items"`
	}{items})
}

func (a *api) createTerminal(c *gin.Context) {
	var input terminal.CreateRequest
	err := decodeJSON(c.Writer, c.Request, &input,
		"project_id", "project_version", "folder_id", "display_name", "cols", "rows")
	if err != nil {
		if errors.Is(err, errBodyLarge) {
			a.fail(c, http.StatusRequestEntityTooLarge, "too_large", "请求体过大。")
		} else {
			a.fail(c, http.StatusBadRequest, "invalid_request", "请求格式无效。")
		}
		return
	}
	if input.Cols == 0 {
		input.Cols = 80
	}
	if input.Rows == 0 {
		input.Rows = 24
	}
	item, err := a.terminals.Create(c.Request.Context(), input)
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, item)
}

func (a *api) getTerminal(c *gin.Context) {
	if !validTerminalID(c.Param("id")) {
		a.fail(c, 400, "invalid_request", "终端 ID 无效。")
		return
	}
	item, err := a.terminals.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, item)
}

func (a *api) terminalHistory(c *gin.Context) {
	if !validTerminalID(c.Param("id")) {
		a.fail(c, 400, "invalid_request", "终端 ID 无效。")
		return
	}
	history, err := a.terminals.History(c.Request.Context(), c.Param("id"))
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, struct {
		ContentBase64 string `json:"content_base64"`
		HistorySize   int    `json:"history_size"`
		ReturnedLines int    `json:"returned_lines"`
		AlternateOn   bool   `json:"alternate_on"`
		Cols          int    `json:"cols"`
		Rows          int    `json:"rows"`
		Truncated     bool   `json:"truncated"`
	}{base64.StdEncoding.EncodeToString(history.Content), history.HistorySize,
		history.ReturnedLines, history.AlternateOn, history.Cols, history.Rows, history.Truncated})
}

func validTerminalID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, char := range id {
		if char < '0' || char > '9' {
			if char < 'a' || char > 'f' {
				return false
			}
		}
	}
	return true
}

func (a *api) renameTerminal(c *gin.Context) {
	if !validTerminalID(c.Param("id")) {
		a.fail(c, 400, "invalid_request", "终端 ID 无效。")
		return
	}
	var input terminal.RenameRequest
	if err := decodeJSON(c.Writer, c.Request, &input, "expected_display_name", "display_name"); err != nil {
		if errors.Is(err, errBodyLarge) {
			a.fail(c, 413, "too_large", "请求体过大。")
		} else {
			a.fail(c, 400, "invalid_request", "请求格式无效。")
		}
		return
	}
	item, err := a.terminals.Rename(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		a.error(c, err)
		return
	}
	a.runtime.UpdateMetadata(item)
	a.success(c, item)
}
