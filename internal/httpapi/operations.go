package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"persistty/internal/files"
	"persistty/internal/storage"
)

func (a *api) fileOperation(c *gin.Context) {
	if len(c.Param("id")) > 128 {
		a.fail(c, 400, "invalid_request", "请求格式无效。")
		return
	}
	var input files.Operation
	if err := decodeJSON(c.Writer, c.Request, &input, "kind", "project_version", "source_folder_id", "source_path", "target_folder_id", "target_path", "expected_version", "expected_identity", "delete_token"); err != nil {
		a.decodeFailure(c, err)
		return
	}
	if input.Kind == "delete" {
		a.deleteOperation(c, input)
		return
	}
	if input.ProjectVersion < 1 || input.TargetFolderID == "" {
		a.fail(c, 428, "version_required", "需要项目版本和目标文件夹。")
		return
	}
	if input.Kind != "create_file" && input.Kind != "create_directory" && (input.SourceFolderID == "" || input.ExpectedIdentity == "") {
		a.fail(c, 428, "version_required", "需要来源身份与版本。")
		return
	}
	var result files.OperationResult
	err := a.store.WithRegisteredFolders(c.Request.Context(), c.Param("id"), input.SourceFolderID, input.TargetFolderID, input.ProjectVersion, func(source, target storage.RegisteredFolder) error {
		var err error
		result, err = files.Execute(c.Request.Context(), source, target, input)
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
