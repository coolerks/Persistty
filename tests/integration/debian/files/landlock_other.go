//go:build !linux

package main

func landlockABI() (int, string) { return 0, "非 Linux；未查询 Landlock" }
