package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"persistty/internal/files"
	"persistty/internal/storage"
)

type deleteClaim struct {
	ProjectID   string `json:"p"`
	FolderID    string `json:"f"`
	Path        string `json:"x"`
	Version     int64  `json:"v"`
	SessionHash string `json:"s"`
	Digest      string `json:"d"`
	Expires     int64  `json:"e"`
}

func (a *api) signDelete(claim deleteClaim) (string, error) {
	data, err := json.Marshal(claim)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, a.deleteKey[:])
	mac.Write(data)
	return base64.RawURLEncoding.EncodeToString(data) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
func (a *api) verifyDelete(token string) (deleteClaim, bool) {
	if len(token) > 4096 {
		return deleteClaim{}, false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return deleteClaim{}, false
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return deleteClaim{}, false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return deleteClaim{}, false
	}
	mac := hmac.New(sha256.New, a.deleteKey[:])
	mac.Write(data)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return deleteClaim{}, false
	}
	var claim deleteClaim
	if err = json.Unmarshal(data, &claim); err != nil || claim.Expires <= time.Now().Unix() {
		return deleteClaim{}, false
	}
	return claim, true
}

func (a *api) deletePreview(c *gin.Context) {
	if len(c.Param("id")) > 128 {
		a.fail(c, 400, "invalid_request", "请求格式无效。")
		return
	}
	var input struct {
		ProjectVersion int64  `json:"project_version"`
		FolderID       string `json:"folder_id"`
		Path           string `json:"path"`
	}
	if err := decodeJSON(c.Writer, c.Request, &input, "project_version", "folder_id", "path"); err != nil {
		a.decodeFailure(c, err)
		return
	}
	if input.ProjectVersion < 1 || input.FolderID == "" {
		a.fail(c, 428, "version_required", "需要项目版本和文件夹。")
		return
	}
	var preview files.DeletePreview
	err := a.store.WithRegisteredFolder(c.Request.Context(), c.Param("id"), input.FolderID, input.ProjectVersion, func(root storage.RegisteredFolder) error {
		var err error
		preview, err = files.PreviewDelete(c.Request.Context(), root, input.Path)
		return err
	})
	if err != nil {
		a.error(c, err)
		return
	}
	claim := deleteClaim{ProjectID: c.Param("id"), FolderID: input.FolderID, Path: input.Path, Version: input.ProjectVersion, SessionHash: storage.HashToken(c.MustGet("token").(string)), Digest: preview.Digest, Expires: time.Now().Add(60 * time.Second).Unix()}
	token, err := a.signDelete(claim)
	if err != nil {
		a.error(c, err)
		return
	}
	a.success(c, struct {
		Path  string `json:"path"`
		Kind  string `json:"kind"`
		Count int    `json:"count"`
		Bytes int64  `json:"bytes"`
		Token string `json:"token"`
	}{preview.Path, preview.Kind, preview.Count, preview.Bytes, token})
}

func (a *api) deleteOperation(c *gin.Context, input files.Operation) {
	claim, ok := a.verifyDelete(input.DeleteToken)
	if !ok || claim.ProjectID != c.Param("id") || claim.FolderID != input.SourceFolderID || claim.Path != input.SourcePath || claim.Version != input.ProjectVersion || claim.SessionHash != storage.HashToken(c.MustGet("token").(string)) {
		a.fail(c, 403, "forbidden", "删除确认已失效，请重新预览。")
		return
	}
	var result files.OperationResult
	err := a.store.WithRegisteredFolder(c.Request.Context(), claim.ProjectID, claim.FolderID, claim.Version, func(root storage.RegisteredFolder) error {
		var err error
		result, err = files.DeleteConfirmed(c.Request.Context(), root, claim.Path, claim.Digest)
		return err
	})
	if err != nil {
		if result.State == "partial" {
			result.FailureCode = "partial_failure"
			c.JSON(http.StatusMultiStatus, successEnvelope{result, c.GetString("request_id")})
			return
		}
		a.error(c, err)
		return
	}
	a.success(c, result)
}
