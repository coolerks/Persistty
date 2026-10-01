package config

import (
	"errors"
	"path/filepath"
	"strings"
	"time"
)

type SearchConfig struct {
	Binary             string   `yaml:"binary"`
	TimeoutSeconds     int      `yaml:"timeout_seconds"`
	MaxEntries         int      `yaml:"max_entries"`
	MaxResults         int      `yaml:"max_results"`
	MaxBytes           int64    `yaml:"max_bytes"`
	ExcludeDirectories []string `yaml:"exclude_directories"`
}
type GitConfig struct {
	Binary   string `yaml:"binary"`
	MaxBytes int64  `yaml:"max_bytes"`
}

func (c Config) SearchOptions() SearchConfig {
	v := c.Search
	if v.Binary == "" {
		v.Binary = "rg"
	}
	if v.TimeoutSeconds == 0 {
		v.TimeoutSeconds = 15
	}
	if v.MaxEntries == 0 {
		v.MaxEntries = 50000
	}
	if v.MaxResults == 0 {
		v.MaxResults = 5000
	}
	if v.MaxBytes == 0 {
		v.MaxBytes = 64 << 20
	}
	if v.ExcludeDirectories == nil {
		v.ExcludeDirectories = []string{"node_modules", "dist", ".next", "vendor"}
	}
	return v
}
func (c Config) GitOptions() GitConfig {
	v := c.Git
	if v.Binary == "" {
		v.Binary = "git"
	}
	if v.MaxBytes == 0 {
		v.MaxBytes = 512 << 20
	}
	return v
}
func (c Config) ToolTimeout() time.Duration {
	return time.Duration(c.SearchOptions().TimeoutSeconds) * time.Second
}
func (c Config) ToolStagingPath() string {
	return filepath.Join(filepath.Dir(c.Storage.Path), "tool-staging")
}
func (c Config) validateTools() error {
	s, g := c.SearchOptions(), c.GitOptions()
	for _, b := range []string{s.Binary, g.Binary} {
		if strings.ContainsRune(b, 0) || (!filepath.IsAbs(b) && strings.ContainsAny(b, "/\\")) {
			return errors.New("工具路径必须是程序名或绝对路径")
		}
	}
	if s.TimeoutSeconds < 1 || s.TimeoutSeconds > 120 || s.MaxEntries < 1 || s.MaxEntries > 50000 || s.MaxResults < 1 || s.MaxResults > 5000 || s.MaxBytes < 1<<20 || s.MaxBytes > 64<<20 || g.MaxBytes < 1<<20 || g.MaxBytes > 512<<20 || len(s.ExcludeDirectories) > 32 {
		return errors.New("搜索/Git 容量配置无效")
	}
	for _, name := range s.ExcludeDirectories {
		if name == "" || len(name) > 128 || strings.ContainsAny(name, "/\\\x00*?[]") {
			return errors.New("依赖排除必须是目录名")
		}
	}
	return nil
}
