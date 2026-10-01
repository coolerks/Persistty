package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"persistty/internal/files"
	"persistty/internal/storage"
	"persistty/internal/transfer"
)

func TestWorkspaceContractFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "tests", "contracts", "workspace-files.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	decode := func(key string, target any) {
		t.Helper()
		if err := json.Unmarshal(fixture[key], target); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
	}
	var listing struct {
		Data files.Listing `json:"data"`
	}
	decode("listing", &listing)
	if listing.Data.ProjectVersion != 2 || len(listing.Data.Items) != 2 || listing.Data.Items[0].Name != "-说明.txt" {
		t.Fatal("listing fixture drift")
	}
	var content struct {
		Data files.Content `json:"data"`
	}
	decode("content", &content)
	hash := sha256.Sum256([]byte(content.Data.Content))
	if content.Data.Version.Size != int64(len([]byte(content.Data.Content))) || content.Data.Version.ETag != "sha256:"+hex.EncodeToString(hash[:]) {
		t.Fatal("content fixture hash drift")
	}
	var metadata struct {
		Data files.Metadata `json:"data"`
	}
	decode("metadata", &metadata)
	if metadata.Data.Kind != "file" || metadata.Data.Version.Size != 5 {
		t.Fatal("metadata fixture drift")
	}
	var result struct {
		Data files.OperationResult `json:"data"`
	}
	decode("partial_operation", &result)
	if result.Data.State != "partial" || result.Data.FailureCode != "partial_failure" {
		t.Fatal("operation fixture drift")
	}
	var preview struct {
		Data struct {
			Path  string `json:"path"`
			Kind  string `json:"kind"`
			Count int    `json:"count"`
			Bytes int64  `json:"bytes"`
			Token string `json:"token"`
		} `json:"data"`
	}
	decode("delete_preview", &preview)
	if preview.Data.Kind != "directory" || preview.Data.Token == "" {
		t.Fatal("delete fixture drift")
	}
	var upload struct {
		Data transfer.UploadState `json:"data"`
	}
	decode("upload", &upload)
	if upload.Data.Status != "pending" || len(upload.Data.Received) != 1 {
		t.Fatal("upload fixture drift")
	}
	var archive struct {
		Data storage.ArchiveRecord `json:"data"`
	}
	decode("archive", &archive)
	if archive.Data.Status != "ready" || archive.Data.RelativePath != "空目录" {
		t.Fatal("archive fixture drift")
	}
	var saved struct {
		Data struct {
			Version files.Version `json:"version"`
		} `json:"data"`
	}
	decode("saved", &saved)
	if saved.Data.Version.Identity != content.Data.Version.Identity {
		t.Fatal("saved fixture drift")
	}
	for _, name := range []string{"inspection_text", "inspection_image", "inspection_binary"} {
		var inspection struct {
			Data files.Inspection `json:"data"`
		}
		decode(name, &inspection)
		if (name == "inspection_image") != inspection.Data.Previewable || (name == "inspection_text") != inspection.Data.Editable {
			t.Fatal("inspection fixture drift")
		}
	}
	var conflict errorEnvelope
	decode("conflict", &conflict)
	if conflict.Error.Code != "conflict" {
		t.Fatal("error fixture drift")
	}
}
