package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type observation struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Error  string `json:"error,omitempty"`
}

type result struct {
	Go            string        `json:"go"`
	OS            string        `json:"os"`
	Base          string        `json:"temporary_directory"`
	Mode          string        `json:"directory_mode"`
	Checks        []observation `json:"checks"`
	RaceAttempts  int           `json:"race_attempts"`
	RaceAllowed   int           `json:"race_allowed"`
	RaceRejected  int           `json:"race_rejected"`
	RaceSwaps     int           `json:"race_swaps"`
	LandlockABI   int           `json:"landlock_abi"`
	LandlockError string        `json:"landlock_error,omitempty"`
	Cleanup       bool          `json:"cleanup_verified"`
}

func (r *result) record(name string, passed bool, err error) {
	o := observation{Name: name, Passed: passed}
	if err != nil {
		o.Error = err.Error()
	}
	r.Checks = append(r.Checks, o)
}

func (r result) passed() bool {
	if !r.Cleanup || len(r.Checks) == 0 {
		return false
	}
	for _, c := range r.Checks {
		if !c.Passed {
			return false
		}
	}
	return true
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	r, err := probe(ctx)
	if err != nil {
		r.record("probe_setup", false, err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(r); err != nil {
		os.Exit(2)
	}
	if !r.passed() {
		os.Exit(1)
	}
}

func probe(ctx context.Context) (r result, retErr error) {
	r.Go, r.OS = runtime.Version(), runtime.GOOS
	r.LandlockABI, r.LandlockError = landlockABI()
	base, err := os.MkdirTemp("", "persistty-files-")
	if err != nil {
		return r, err
	}
	r.Base = base
	defer func() {
		err := os.RemoveAll(base)
		_, statErr := os.Lstat(base)
		r.Cleanup = err == nil && errors.Is(statErr, os.ErrNotExist)
		if !r.Cleanup {
			retErr = errors.Join(retErr, fmt.Errorf("cleanup: %v, stat: %v", err, statErr))
		}
	}()
	info, err := os.Stat(base)
	if err != nil {
		return r, err
	}
	r.Mode = info.Mode().Perm().String()
	r.record("private_directory", info.Mode().Perm() == 0700, nil)
	rootPath, outside := filepath.Join(base, "root"), filepath.Join(base, "outside")
	for _, dir := range []string{rootPath, outside} {
		if err := os.Mkdir(dir, 0700); err != nil {
			return r, err
		}
	}
	const secret = "outside-sentinel-not-a-business-file"
	for _, name := range []string{"sentinel", "remove-target", "rename-source"} {
		if err := os.WriteFile(filepath.Join(outside, name), []byte(secret), 0600); err != nil {
			return r, err
		}
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return r, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	err = root.WriteFile("inside", []byte("inside"), 0600)
	r.record("create_inside", err == nil, err)
	data, err := root.ReadFile("inside")
	r.record("read_inside", err == nil && string(data) == "inside", err)
	err = root.Rename("inside", "renamed")
	r.record("rename_inside", err == nil, err)
	data, err = root.ReadFile("renamed")
	r.record("read_renamed", err == nil && string(data) == "inside", err)
	err = root.Symlink("renamed", "internal-link")
	if err != nil {
		return r, err
	}
	data, err = root.ReadFile("internal-link")
	r.record("read_internal_symlink", err == nil && string(data) == "inside", err)
	err = root.Remove("internal-link")
	r.record("remove_internal_symlink", err == nil, err)
	err = root.Remove("renamed")
	r.record("remove_inside", err == nil, err)
	err = root.Symlink("../outside", "escape")
	if err != nil {
		return r, err
	}
	for _, path := range []string{"../outside/sentinel", filepath.Join(outside, "sentinel"), "escape/sentinel"} {
		_, err = root.ReadFile(path)
		r.record("reject_read:"+path, err != nil, err)
		err = root.WriteFile(path, []byte("wrong"), 0600)
		r.record("reject_write:"+path, err != nil, err)
	}
	err = root.WriteFile("escape/new-file", []byte("wrong"), 0600)
	r.record("reject_create_external_parent", err != nil, err)
	if err = root.WriteFile("source", []byte("inside"), 0600); err != nil {
		return r, err
	}
	err = root.Rename("source", "escape/moved")
	r.record("reject_rename_external_destination", err != nil, err)
	err = root.Rename("escape/rename-source", "from-outside")
	r.record("reject_rename_external_source", err != nil, err)
	err = root.Remove("escape/remove-target")
	r.record("reject_remove_external_parent", err != nil, err)
	err = root.Remove("escape")
	r.record("remove_external_symlink_only", err == nil, err)
	if err := raceParent(ctx, root, rootPath, outside, secret, &r); err != nil {
		return r, err
	}
	// 根句柄固定目录身份，不承诺它移动后仍属于当前路径树。
	if err := root.Mkdir("anchored", 0700); err != nil {
		return r, err
	}
	anchored, err := root.OpenRoot("anchored")
	if err != nil {
		return r, err
	}
	defer func() { retErr = errors.Join(retErr, anchored.Close()) }()
	err = os.Rename(filepath.Join(rootPath, "anchored"), filepath.Join(base, "detached"))
	if err != nil {
		return r, err
	}
	err = anchored.WriteFile("after-move", []byte("owned-probe-data"), 0600)
	data, readErr := os.ReadFile(filepath.Join(base, "detached", "after-move"))
	r.record("root_handle_follows_moved_directory_identity", err == nil && readErr == nil && string(data) == "owned-probe-data", errors.Join(err, readErr))
	entries, err := os.ReadDir(outside)
	r.record("outside_entry_count_unchanged", err == nil && len(entries) == 3, err)
	for _, name := range []string{"sentinel", "remove-target", "rename-source"} {
		data, err := os.ReadFile(filepath.Join(outside, name))
		r.record("outside_unchanged:"+name, err == nil && string(data) == secret, err)
	}
	return r, nil
}

func raceParent(ctx context.Context, root *os.Root, rootPath, outside, secret string, r *result) error {
	if err := root.Mkdir("parent", 0700); err != nil {
		return err
	}
	if err := root.WriteFile("parent/sentinel", []byte("inside"), 0600); err != nil {
		return err
	}
	parent, parked := filepath.Join(rootPath, "parent"), filepath.Join(rootPath, "parked")
	raceCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	var swapErr error
	swaps := 0
	ready := make(chan struct{})
	wg.Add(1)
	// 攻击者只在本次新建目录内交换目录与根外链接，绝不访问业务路径。
	go func() {
		defer wg.Done()
		close(ready)
		for swaps < 2000 && raceCtx.Err() == nil {
			if err := os.Rename(parent, parked); err != nil {
				swapErr = err
				return
			}
			if err := os.Symlink(outside, parent); err != nil {
				swapErr = err
				return
			}
			runtime.Gosched()
			if err := os.Remove(parent); err != nil {
				swapErr = err
				return
			}
			if err := os.Rename(parked, parent); err != nil {
				swapErr = err
				return
			}
			swaps++
		}
	}()
	<-ready
	leaked := false
	for i := 0; i < 2000 && raceCtx.Err() == nil; i++ {
		data, err := root.ReadFile("parent/sentinel")
		if err == nil && string(data) == secret {
			leaked = true
		}
		outcomes := []error{err, root.WriteFile("parent/sentinel", []byte("inside"), 0600), root.Remove("parent/remove-target")}
		if err := root.WriteFile("race-source", []byte("inside"), 0600); err != nil {
			cancel()
			wg.Wait()
			return err
		}
		outcomes = append(outcomes, root.Rename("race-source", "parent/moved"), root.Rename("parent/rename-source", "race-from-outside"))
		for _, err := range outcomes {
			r.RaceAttempts++
			if err == nil {
				r.RaceAllowed++
			} else {
				r.RaceRejected++
			}
		}
	}
	cancel()
	wg.Wait()
	r.RaceSwaps = swaps
	r.record("bounded_parent_swap_no_external_read", !leaked && swaps > 0 && r.RaceAttempts > 0 && swapErr == nil, swapErr)
	return ctx.Err()
}
