package httpapi

import (
	"encoding/json"
	"errors"

	"github.com/gin-gonic/gin"
	"persistty/internal/storage"
	"persistty/internal/terminal"
)

func (a *api) terminateBatch(c *gin.Context) {
	var input struct {
		Members []json.RawMessage `json:"members"`
	}
	if err := decodeJSONWithLimit(c.Writer, c.Request, &input, 64<<10, "members"); err != nil {
		a.decodeFailure(c, err)
		return
	}
	if len(input.Members) < 1 || len(input.Members) > 200 {
		a.fail(c, 400, "invalid_request", "批次成员数量无效。")
		return
	}
	members := make([]terminal.BatchTarget, 0, len(input.Members))
	for _, raw := range input.Members {
		var fields map[string]json.RawMessage
		var member terminal.BatchTarget
		if json.Unmarshal(raw, &fields) != nil || len(fields) != 3 || fields["terminal_id"] == nil || fields["viewer_id"] == nil || fields["generation"] == nil || json.Unmarshal(raw, &member) != nil {
			a.fail(c, 400, "invalid_request", "批次成员字段无效。")
			return
		}
		if !validTerminalID(member.TerminalID) || !validTerminalID(member.ViewerID) {
			a.fail(c, 400, "invalid_request", "批次成员身份无效。")
			return
		}
		members = append(members, member)
	}
	item, err := a.runtime.TerminateBatch(c.Request.Context(), c.MustGet("token").(string), members)
	if err != nil {
		var rejected *terminal.BatchError
		if errors.As(err, &rejected) {
			code, message, status := "unavailable", "目标终端暂时不可用。", 503
			switch {
			case errors.Is(err, terminal.ErrControlDenied):
				code, message, status = "control_denied", "目标终端尚未由当前端接管。", 409
			case errors.Is(err, terminal.ErrStaleGeneration):
				code, message, status = "stale_generation", "目标终端控制权已变化。", 409
			case errors.Is(err, terminal.ErrTerminationPending):
				code, message, status = "termination_pending", "目标终端已有终止倒计时。", 409
			case errors.Is(err, storage.ErrConflict):
				code, message, status = "conflict", "目标终端已不再运行。", 409
			}
			c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message, "details": gin.H{"terminal_id": rejected.TerminalID}}, "request_id": c.GetString("request_id")})
			return
		}
		a.error(c, err)
		return
	}
	a.success(c, item)
}

func (a *api) getTerminationBatch(c *gin.Context) {
	if !validTerminalID(c.Param("id")) {
		a.fail(c, 400, "invalid_request", "终止批次 ID 无效。")
		return
	}
	item, err := a.runtime.Batch(c.Param("id"))
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, item)
}
