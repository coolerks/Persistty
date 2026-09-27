package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestCanceledProbeCleansOwnedDirectory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, err := probe(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("取消实验应返回 context.Canceled，实际为 %v", err)
	}
	if r.Base == "" || !r.Cleanup || r.passed() {
		t.Fatalf("取消实验必须记录自身目录、完成清理且不能通过: %+v", r)
	}
	if _, err := os.Lstat(r.Base); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("实验目录仍存在或无法确认清理: %v", err)
	}
}

func TestRootProbe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	r, err := probe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range r.Checks {
		if !check.Passed {
			t.Errorf("%s: %s", check.Name, check.Error)
		}
	}
	if !r.Cleanup {
		t.Error("实验目录未清理")
	}
	if r.RaceAllowed+r.RaceRejected != r.RaceAttempts {
		t.Error("竞态统计不一致")
	}
}

func TestResultRequiresCleanupAndChecks(t *testing.T) {
	for _, r := range []result{{}, {Cleanup: true}, {Checks: []observation{{Passed: true}}}, {Cleanup: true, Checks: []observation{{Passed: false}}}} {
		if r.passed() {
			t.Fatal("不完整或失败结果不能通过")
		}
	}
}
