package files

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"persistty/internal/storage"
)

func rootFixture(t *testing.T) (storage.RegisteredFolder, string) {
	t.Helper()
	base := t.TempDir()
	projectDir := filepath.Join(base, "project")
	if err := os.Mkdir(projectDir, 0700); err != nil {
		t.Fatal(err)
	}
	s, err := storage.Open(context.Background(), filepath.Join(base, "data", "db.sqlite"), "testhash")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	p, err := s.CreateProject(context.Background(), "test", []string{projectDir}, 0)
	if err != nil {
		t.Fatal(err)
	}
	root, err := s.RegisteredFolder(context.Background(), p.ID, p.MainFolderID, p.Version)
	if err != nil {
		t.Fatal(err)
	}
	return root, base
}

func TestConstrainedReadAndRootIdentity(t *testing.T) {
	root, base := rootFixture(t)
	if err := os.WriteFile(filepath.Join(root.Path, "文档.txt"), []byte("你好\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root.Path, "binary"), []byte{0, 1, 2}, 0600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(base, "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root.Path, "escape")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../outside.txt", "/etc/passwd", "a/../b", ""} {
		if _, err := ReadContent(root, path); !errors.Is(err, ErrInvalidPath) {
			t.Fatalf("accepted %q: %v", path, err)
		}
	}
	read, err := ReadContent(root, "文档.txt")
	if err != nil || read.Content != "你好\r\n" || read.Version.Size != 8 || len(read.Version.ETag) != len("sha256:")+64 {
		t.Fatalf("content=%#v err=%v", read, err)
	}
	if _, err := ReadContent(root, "binary"); !errors.Is(err, ErrUnsupported) {
		t.Fatal("binary returned as text")
	}
	if _, err := ReadContent(root, "escape"); err == nil {
		t.Fatal("symlink escaped registered root")
	}
	listing, err := List(root, "", "", 2)
	if err != nil || len(listing.Items) != 2 || listing.NextCursor == "" {
		t.Fatalf("list=%#v err=%v", listing, err)
	}
	if err := os.WriteFile(filepath.Join(root.Path, "new.txt"), []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := List(root, "", listing.NextCursor, 2); !errors.Is(err, ErrRescan) {
		t.Fatal("stale cursor accepted")
	}
	if err := os.Rename(root.Path, root.Path+"-moved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root.Path, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadContent(root, "文档.txt"); !errors.Is(err, ErrRootChanged) {
		t.Fatalf("replacement root accepted: %v", err)
	}
}

func TestSaveRequiresStrongVersionAndPreservesMode(t *testing.T) {
	root, _ := rootFixture(t)
	file := filepath.Join(root.Path, "note")
	if err := os.WriteFile(file, []byte("one"), 0750); err != nil {
		t.Fatal(err)
	}
	first, err := ReadContent(root, "note")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := Save(context.Background(), root, "note", first.Version, "two\n")
	if err != nil || updated.ETag == first.Version.ETag {
		t.Fatalf("save=%#v %v", updated, err)
	}
	info, err := os.Stat(file)
	if err != nil || info.Mode().Perm() != 0750 {
		t.Fatal("mode not preserved")
	}
	if _, err := Save(context.Background(), root, "note", first.Version, "stale"); !errors.Is(err, ErrConflict) {
		t.Fatal("stale save accepted")
	}
	second, err := ReadContent(root, "note")
	if err != nil || second.Content != "two\n" {
		t.Fatal("stale save changed file")
	}
	if err := os.WriteFile(file, []byte("four"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(file, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if _, err := Save(context.Background(), root, "note", second.Version, "wrong"); !errors.Is(err, ErrConflict) {
		t.Fatal("same-size mtime content change accepted")
	}
	entries, err := os.ReadDir(root.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" {
			t.Fatal("failed save left temporary file")
		}
	}
}

func TestFileOperationsWithinRoot(t *testing.T) {
	root, _ := rootFixture(t)
	ctx := context.Background()
	create := func(kind, target string) error {
		_, err := Execute(ctx, storage.RegisteredFolder{}, root, Operation{Kind: kind, ProjectVersion: root.ProjectVersion, TargetPath: target})
		return err
	}
	if err := create("create_directory", "sub"); err != nil {
		t.Fatal(err)
	}
	if err := create("create_file", "sub/alpha"); err != nil {
		t.Fatal(err)
	}
	if err := create("create_file", "sub/alpha"); !errors.Is(err, os.ErrExist) {
		t.Fatal("existing target overwritten")
	}
	if err := os.WriteFile(filepath.Join(root.Path, "sub", "alpha"), []byte("text"), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := ReadContent(root, "sub/alpha")
	if err != nil {
		t.Fatal(err)
	}
	operation := Operation{Kind: "rename", ProjectVersion: root.ProjectVersion, SourceFolderID: root.FolderID, SourcePath: "sub/alpha", TargetFolderID: root.FolderID, TargetPath: "sub/beta", ExpectedIdentity: snapshot.Version.Identity, ExpectedVersion: &snapshot.Version}
	if result, err := Execute(ctx, root, root, operation); err != nil || !result.SourceRemoved {
		t.Fatalf("rename=%#v %v", result, err)
	}
	copySnapshot, err := ReadContent(root, "sub/beta")
	if err != nil {
		t.Fatal(err)
	}
	operation.Kind = "copy"
	operation.SourcePath = "sub/beta"
	operation.TargetPath = "sub/gamma"
	operation.ExpectedIdentity = copySnapshot.Version.Identity
	operation.ExpectedVersion = &copySnapshot.Version
	if result, err := Execute(ctx, root, root, operation); err != nil || result.SourceRemoved {
		t.Fatalf("copy=%#v %v", result, err)
	}
	if _, err := ReadContent(root, "sub/gamma"); err != nil {
		t.Fatal(err)
	}
	operation.Kind = "move"
	operation.TargetPath = "sub/delta"
	if result, err := Execute(ctx, root, root, operation); err != nil || !result.SourceRemoved {
		t.Fatalf("move=%#v %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(root.Path, "sub", "beta")); !os.IsNotExist(err) {
		t.Fatal("move retained source")
	}
	listing, err := List(root, "", "", 200)
	if err != nil {
		t.Fatal(err)
	}
	var dirIdentity string
	for _, entry := range listing.Items {
		if entry.Name == "sub" {
			dirIdentity = entry.Identity
		}
	}
	operation = Operation{Kind: "move", ProjectVersion: root.ProjectVersion, SourceFolderID: root.FolderID, SourcePath: "sub", TargetFolderID: root.FolderID, TargetPath: "sub/inside", ExpectedIdentity: dirIdentity}
	if _, err := Execute(ctx, root, root, operation); !errors.Is(err, ErrInvalidPath) {
		t.Fatal("directory moved into itself")
	}
}

func TestDirectoryCopyMoveAndPartialFailure(t *testing.T) {
	root, _ := rootFixture(t)
	for _, dir := range []string{"source", "source/empty", "source/nested"} {
		if err := os.Mkdir(filepath.Join(root.Path, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root.Path, "source/nested/data"), []byte("payload"), 0600); err != nil {
		t.Fatal(err)
	}
	listing, err := List(root, "", "", 100)
	if err != nil {
		t.Fatal(err)
	}
	var identity string
	for _, entry := range listing.Items {
		if entry.Name == "source" {
			identity = entry.Identity
		}
	}
	copyOp := Operation{Kind: "copy", ProjectVersion: root.ProjectVersion, SourceFolderID: root.FolderID, SourcePath: "source", TargetFolderID: root.FolderID, TargetPath: "copy", ExpectedIdentity: identity}
	if result, err := Execute(context.Background(), root, root, copyOp); err != nil || result.State != "applied" || result.SourceRemoved {
		t.Fatalf("copy=%#v %v", result, err)
	}
	bytes, err := os.ReadFile(filepath.Join(root.Path, "copy/nested/data"))
	if err != nil || string(bytes) != "payload" {
		t.Fatal("copy contents missing")
	}
	if info, err := os.Stat(filepath.Join(root.Path, "copy/empty")); err != nil || !info.IsDir() {
		t.Fatal("empty directory missing")
	}
	copyListing, err := List(root, "", "", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range copyListing.Items {
		if entry.Name == "copy" {
			copyOp.ExpectedIdentity = entry.Identity
		}
	}
	copyOp.Kind = "move"
	copyOp.SourcePath = "copy"
	copyOp.TargetPath = "moved"
	if result, err := Execute(context.Background(), root, root, copyOp); err != nil || !result.SourceRemoved {
		t.Fatalf("move=%#v %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(root.Path, "copy")); !os.IsNotExist(err) {
		t.Fatal("moved source remains")
	}
	if err := os.Symlink(filepath.Join(root.Path, "source/nested/data"), filepath.Join(root.Path, "source/link")); err != nil {
		t.Fatal(err)
	}
	copyOp.Kind = "copy"
	copyOp.SourcePath = "source"
	copyOp.TargetPath = "partial"
	copyOp.ExpectedIdentity = identity
	result, err := Execute(context.Background(), root, root, copyOp)
	if err == nil || result.State != "partial" || !result.TargetCreated {
		t.Fatalf("symlink copy=%#v %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(root.Path, "source/nested/data")); err != nil {
		t.Fatal("partial copy deleted source")
	}
}

func TestDeletePreviewDetectsExternalChanges(t *testing.T) {
	root, _ := rootFixture(t)
	if err := os.Mkdir(filepath.Join(root.Path, "target"), 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root.Path, "target", "file")
	if err := os.WriteFile(file, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewDelete(context.Background(), root, "target")
	if err != nil || preview.Kind != "directory" || preview.Count != 2 || preview.Bytes != 3 {
		t.Fatalf("preview=%#v %v", preview, err)
	}
	if err := os.WriteFile(file, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := DeleteConfirmed(context.Background(), root, "target", preview.Digest); !errors.Is(err, ErrConflict) {
		t.Fatal("changed target deleted")
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatal("rejected delete changed disk")
	}
	preview, err = PreviewDelete(context.Background(), root, "target")
	if err != nil {
		t.Fatal(err)
	}
	result, err := DeleteConfirmed(context.Background(), root, "target", preview.Digest)
	if err != nil || !result.SourceRemoved {
		t.Fatalf("delete=%#v %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(root.Path, "target")); !os.IsNotExist(err) {
		t.Fatal("confirmed delete retained target")
	}
}

func TestCrossRootCopyMoveAndNoReplace(t *testing.T) {
	base := t.TempDir()
	first := filepath.Join(base, "first")
	second := filepath.Join(base, "second")
	for _, directory := range []string{first, second} {
		if err := os.Mkdir(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	store, err := storage.Open(context.Background(), filepath.Join(base, "data", "db.sqlite"), "testhash")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	project, err := store.CreateProject(context.Background(), "two", []string{first, second}, 0)
	if err != nil {
		t.Fatal(err)
	}
	source, err := store.RegisteredFolder(context.Background(), project.ID, project.Folders[0].ID, project.Version)
	if err != nil {
		t.Fatal(err)
	}
	target, err := store.RegisteredFolder(context.Background(), project.ID, project.Folders[1].ID, project.Version)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(first, "树"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(first, "树", "空"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(first, "树", "-文件"), []byte("内容"), 0600); err != nil {
		t.Fatal(err)
	}
	listing, err := List(source, "", "", 100)
	if err != nil {
		t.Fatal(err)
	}
	identity := listing.Items[0].Identity
	operation := Operation{Kind: "copy", ProjectVersion: project.Version, SourceFolderID: source.FolderID, SourcePath: "树", TargetFolderID: target.FolderID, TargetPath: "副本", ExpectedIdentity: identity}
	result, err := Execute(context.Background(), source, target, operation)
	if err != nil || result.State != "applied" || result.SourceRemoved {
		t.Fatalf("copy=%#v %v", result, err)
	}
	if body, err := os.ReadFile(filepath.Join(second, "副本", "-文件")); err != nil || string(body) != "内容" {
		t.Fatalf("copied body=%q %v", body, err)
	}
	if info, err := os.Stat(filepath.Join(second, "副本", "空")); err != nil || !info.IsDir() {
		t.Fatal("empty folder lost")
	}
	if _, err := Execute(context.Background(), source, target, operation); !errors.Is(err, os.ErrExist) {
		t.Fatalf("existing destination accepted: %v", err)
	}
	operation.Kind = "move"
	operation.TargetPath = "移动"
	result, err = Execute(context.Background(), source, target, operation)
	if err != nil || result.State != "applied" || !result.SourceRemoved {
		t.Fatalf("move=%#v %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(first, "树")); !os.IsNotExist(err) {
		t.Fatal("source retained after cross-root move")
	}
	if body, err := os.ReadFile(filepath.Join(second, "移动", "-文件")); err != nil || string(body) != "内容" {
		t.Fatal("moved data lost")
	}
}

func TestTextBOMMixedLineEndingsAndCancelledSave(t *testing.T) {
	root, _ := rootFixture(t)
	body := "\ufeff首行\r\nsecond\nlast"
	file := filepath.Join(root.Path, "strange.ext")
	if err := os.WriteFile(file, []byte(body), 0640); err != nil {
		t.Fatal(err)
	}
	content, err := ReadContent(root, "strange.ext")
	if err != nil || content.Content != body || content.Version.Size != int64(len([]byte(body))) {
		t.Fatalf("read=%#v %v", content, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Save(ctx, root, "strange.ext", content.Version, "changed"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled save=%v", err)
	}
	current, err := os.ReadFile(file)
	if err != nil || string(current) != body {
		t.Fatal("cancelled save changed bytes")
	}
	updated, err := Save(context.Background(), root, "strange.ext", content.Version, body)
	if err != nil || updated.ETag != content.Version.ETag {
		t.Fatalf("same content save=%#v %v", updated, err)
	}
	info, err := os.Stat(file)
	if err != nil || info.Mode().Perm() != 0640 {
		t.Fatal("ordinary mode not retained")
	}
}

func TestVerifySourceLeafRejectsChangedContentAndIdentity(t *testing.T) {
	directory := t.TempDir()
	name := "source.txt"
	filename := filepath.Join(directory, name)
	if err := os.WriteFile(filename, []byte("old!"), 0600); err != nil {
		t.Fatal(err)
	}
	parent, err := os.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	source, err := openLeaf(parent, name)
	if err != nil {
		t.Fatal(err)
	}
	original, err := source.Stat()
	if err != nil {
		t.Fatal(err)
	}
	version, _, err := versionFromFile(source)
	source.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte("new!"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filename, original.ModTime(), original.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := verifySourceLeaf(parent, name, original, version); !errors.Is(err, ErrConflict) {
		t.Fatalf("same-size content change accepted: %v", err)
	}
	if err := os.Rename(filename, filepath.Join(directory, "old-location")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte("old!"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifySourceLeaf(parent, name, original, version); !errors.Is(err, ErrConflict) {
		t.Fatalf("source path replacement accepted: %v", err)
	}
}
