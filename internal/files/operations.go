package files

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"persistty/internal/storage"
)

type Operation struct {
	Kind             string   `json:"kind"`
	ProjectVersion   int64    `json:"project_version"`
	SourceFolderID   string   `json:"source_folder_id"`
	SourcePath       string   `json:"source_path"`
	TargetFolderID   string   `json:"target_folder_id"`
	TargetPath       string   `json:"target_path"`
	ExpectedVersion  *Version `json:"expected_version"`
	ExpectedIdentity string   `json:"expected_identity"`
	DeleteToken      string   `json:"delete_token"`
}

type OperationResult struct {
	State         string `json:"state"`
	SourceRemoved bool   `json:"source_removed"`
	TargetCreated bool   `json:"target_created"`
	FailureCode   string `json:"failure_code"`
}

func Execute(ctx context.Context, source, target storage.RegisteredFolder, operation Operation) (OperationResult, error) {
	if operation.ProjectVersion != target.ProjectVersion || !ValidRelative(operation.TargetPath, false) {
		return OperationResult{}, ErrInvalidPath
	}
	switch operation.Kind {
	case "create_file", "create_directory":
		return createEntry(ctx, target, operation)
	case "rename", "copy", "move":
		if source.FolderID == "" || source.ProjectVersion != target.ProjectVersion || !ValidRelative(operation.SourcePath, false) {
			return OperationResult{}, ErrInvalidPath
		}
		if operation.Kind == "rename" && source.FolderID != target.FolderID {
			return OperationResult{}, ErrInvalidPath
		}
		if source.FolderID == target.FolderID && operation.SourcePath == operation.TargetPath {
			return OperationResult{}, ErrInvalidPath
		}
		return changeEntry(ctx, source, target, operation)
	default:
		return OperationResult{}, ErrUnsupported
	}
}

func createEntry(ctx context.Context, target storage.RegisteredFolder, operation Operation) (OperationResult, error) {
	if err := ctx.Err(); err != nil {
		return OperationResult{}, err
	}
	parent, err := openMutationParent(target, path.Dir(operation.TargetPath))
	if err != nil {
		return OperationResult{}, err
	}
	defer parent.Close()
	name := path.Base(operation.TargetPath)
	if err = ensureParent(target, path.Dir(operation.TargetPath), parent); err != nil {
		return OperationResult{}, err
	}
	if operation.Kind == "create_directory" {
		if err = mkdirLeaf(parent, name); err != nil {
			return OperationResult{}, err
		}
	} else {
		file, e := createTemp(parent, name)
		if e != nil {
			return OperationResult{}, e
		}
		if e = file.Sync(); e == nil {
			e = file.Close()
		} else {
			file.Close()
		}
		if e != nil {
			return OperationResult{State: "partial", TargetCreated: true}, e
		}
	}
	if err = parent.Sync(); err != nil {
		return OperationResult{State: "partial", TargetCreated: true}, err
	}
	return OperationResult{State: "applied", TargetCreated: true}, nil
}

func changeEntry(ctx context.Context, source, target storage.RegisteredFolder, operation Operation) (OperationResult, error) {
	sourceParent, err := openMutationParent(source, path.Dir(operation.SourcePath))
	if err != nil {
		return OperationResult{}, err
	}
	defer sourceParent.Close()
	from := path.Base(operation.SourcePath)
	sourceFile, err := openLeaf(sourceParent, from)
	if err != nil {
		return OperationResult{}, err
	}
	info, err := sourceFile.Stat()
	if err != nil {
		sourceFile.Close()
		return OperationResult{}, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		sourceFile.Close()
		return OperationResult{}, ErrUnsupported
	}
	identity := fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)
	if operation.ExpectedIdentity == "" || operation.ExpectedIdentity != identity {
		sourceFile.Close()
		return OperationResult{}, ErrConflict
	}
	var initialVersion Version
	var fileMode os.FileMode
	if info.Mode().IsRegular() {
		if info.Size() > 20<<30 {
			sourceFile.Close()
			return OperationResult{}, ErrTooLarge
		}
		if operation.ExpectedVersion == nil {
			sourceFile.Close()
			return OperationResult{}, ErrVersionRequired
		}
		version, mode, e := versionFromFile(sourceFile)
		if e != nil {
			sourceFile.Close()
			return OperationResult{}, e
		}
		if version != *operation.ExpectedVersion {
			sourceFile.Close()
			return OperationResult{}, ErrConflict
		}
		initialVersion, fileMode = version, mode
	} else if !info.IsDir() {
		sourceFile.Close()
		return OperationResult{}, ErrUnsupported
	}
	sourceFile.Close()
	if info.IsDir() {
		fromPath := filepath.Join(source.Path, filepath.FromSlash(operation.SourcePath))
		toPath := filepath.Join(target.Path, filepath.FromSlash(operation.TargetPath))
		if toPath == fromPath || strings.HasPrefix(toPath, fromPath+string(os.PathSeparator)) {
			return OperationResult{}, ErrInvalidPath
		}
	}
	targetParent, err := openMutationParent(target, path.Dir(operation.TargetPath))
	if err != nil {
		return OperationResult{}, err
	}
	defer targetParent.Close()
	if err = ctx.Err(); err != nil {
		return OperationResult{}, err
	}
	if (operation.Kind == "rename" || operation.Kind == "move") && source.FolderID == target.FolderID {
		if err = ensureParent(source, path.Dir(operation.SourcePath), sourceParent); err != nil {
			return OperationResult{}, err
		}
		if err = ensureParent(target, path.Dir(operation.TargetPath), targetParent); err != nil {
			return OperationResult{}, err
		}
		if err = verifySourceLeaf(sourceParent, from, info, initialVersion); err != nil {
			return OperationResult{}, err
		}
		if err = renameNoReplace(sourceParent, from, targetParent, path.Base(operation.TargetPath)); err != nil {
			return OperationResult{}, err
		}
		if err = sourceParent.Sync(); err != nil {
			return OperationResult{State: "partial", TargetCreated: true, SourceRemoved: true}, err
		}
		if err = targetParent.Sync(); err != nil {
			return OperationResult{State: "partial", TargetCreated: true, SourceRemoved: true}, err
		}
		return OperationResult{State: "applied", TargetCreated: true, SourceRemoved: true}, nil
	}
	if info.IsDir() {
		return copyDirectory(ctx, source, target, operation)
	}
	if err = ctx.Err(); err != nil {
		return OperationResult{}, err
	}
	sourceAgain, err := openLeaf(sourceParent, from)
	if err != nil {
		return OperationResult{}, err
	}
	defer sourceAgain.Close()
	verified, _, err := versionFromFile(sourceAgain)
	if err != nil {
		return OperationResult{}, err
	}
	if verified != initialVersion {
		return OperationResult{}, ErrConflict
	}
	if _, err = sourceAgain.Seek(0, io.SeekStart); err != nil {
		return OperationResult{}, err
	}
	tempBytes := make([]byte, 16)
	if _, err = rand.Read(tempBytes); err != nil {
		return OperationResult{}, err
	}
	tempName := ".persistty-" + hex.EncodeToString(tempBytes) + ".tmp"
	temp, err := createTemp(targetParent, tempName)
	if err != nil {
		return OperationResult{}, err
	}
	defer removeLeaf(targetParent, tempName)
	count, err := io.Copy(temp, io.LimitReader(sourceAgain, initialVersion.Size+1))
	if err == nil && count != initialVersion.Size {
		err = ErrConflict
	}
	if err == nil {
		err = temp.Chmod(fileMode & 0777)
	}
	if err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err != nil {
		return OperationResult{}, err
	}
	if closeErr != nil {
		return OperationResult{}, closeErr
	}
	if err = ctx.Err(); err != nil {
		return OperationResult{}, err
	}
	sourceCheck, err := openLeaf(sourceParent, from)
	if err != nil {
		return OperationResult{}, err
	}
	lastVersion, _, err := versionFromFile(sourceCheck)
	sourceCheck.Close()
	if err != nil {
		return OperationResult{}, err
	}
	if lastVersion != initialVersion {
		return OperationResult{}, ErrConflict
	}
	if err = ensureParent(source, path.Dir(operation.SourcePath), sourceParent); err != nil {
		return OperationResult{}, err
	}
	if err = ensureParent(target, path.Dir(operation.TargetPath), targetParent); err != nil {
		return OperationResult{}, err
	}
	if err = renameNoReplace(targetParent, tempName, targetParent, path.Base(operation.TargetPath)); err != nil {
		return OperationResult{}, err
	}
	result := OperationResult{State: "applied", TargetCreated: true}
	if err = targetParent.Sync(); err != nil {
		result.State = "partial"
		return result, err
	}
	if operation.Kind == "move" {
		if err = ensureParent(source, path.Dir(operation.SourcePath), sourceParent); err != nil {
			result.State = "partial"
			return result, err
		}
		sourceFinal, e := openLeaf(sourceParent, from)
		if e != nil {
			result.State = "partial"
			return result, e
		}
		last, _, e := versionFromFile(sourceFinal)
		sourceFinal.Close()
		if e != nil || last != initialVersion {
			result.State = "partial"
			if e != nil {
				return result, e
			}
			return result, ErrConflict
		}
		if err = removeLeaf(sourceParent, from); err != nil {
			result.State = "partial"
			return result, err
		}
		result.SourceRemoved = true
		if err = sourceParent.Sync(); err != nil {
			result.State = "partial"
			return result, err
		}
	}
	return result, nil
}

func verifySourceLeaf(parent *os.File, name string, original os.FileInfo, expected Version) error {
	current, err := openLeaf(parent, name)
	if err != nil {
		return err
	}
	defer current.Close()
	info, err := current.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(original, info) || info.Mode().Type() != original.Mode().Type() {
		return ErrConflict
	}
	if original.Mode().IsRegular() {
		version, _, err := versionFromFile(current)
		if err != nil {
			return err
		}
		if version != expected {
			return ErrConflict
		}
	}
	return nil
}

func ensureParent(root storage.RegisteredFolder, relative string, original *os.File) error {
	current, err := openMutationParent(root, relative)
	if err != nil {
		return err
	}
	defer current.Close()
	a, err := original.Stat()
	if err != nil {
		return err
	}
	b, err := current.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(a, b) {
		return ErrRootChanged
	}
	return nil
}
