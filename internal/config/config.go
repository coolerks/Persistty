package config

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.yaml.in/yaml/v3"
	"persistty/internal/auth"
)

type Config struct {
	Search SearchConfig `yaml:"search"`
	Git    GitConfig    `yaml:"git"`
	Server struct {
		Listen         string   `yaml:"listen"`
		PublicOrigin   string   `yaml:"public_origin"`
		Mode           string   `yaml:"mode"`
		TrustedProxies []string `yaml:"trusted_proxies"`
	} `yaml:"server"`
	Auth struct {
		PasswordHash string `yaml:"password_hash"`
		SessionTTL   string `yaml:"session_ttl"`
	} `yaml:"auth"`
	Storage struct {
		Path string `yaml:"path"`
	} `yaml:"storage"`
	Transfer struct {
		StagingPath     string `yaml:"staging_path"`
		MaxFileBytes    int64  `yaml:"max_file_bytes"`
		MaxBatchBytes   int64  `yaml:"max_batch_bytes"`
		MaxStagingBytes int64  `yaml:"max_staging_bytes"`
		ChunkBytes      int64  `yaml:"chunk_bytes"`
		UploadTTL       string `yaml:"upload_ttl"`
		ArchiveTTL      string `yaml:"archive_ttl"`
	} `yaml:"transfer"`
	Terminal struct {
		TmuxBinary         string `yaml:"tmux_binary"`
		Shell              string `yaml:"shell"`
		SocketPath         string `yaml:"socket_path"`
		HistoryLines       int    `yaml:"history_lines"`
		RestoreLines       int    `yaml:"restore_lines"`
		HistoryBytes       int64  `yaml:"history_bytes"`
		TerminationSeconds int    `yaml:"termination_seconds"`
	} `yaml:"terminal"`
}

func Load(path string) (Config, error) {
	var cfg Config
	info, err := os.Lstat(path)
	if err != nil {
		return cfg, errors.New("无法读取配置文件")
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return cfg, errors.New("配置必须是权限0600的普通文件")
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && int(stat.Uid) != os.Geteuid() {
		return cfg, errors.New("配置必须属于服务用户")
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return cfg, errors.New("无法读取配置文件")
	}
	defer f.Close()
	openedInfo, err := f.Stat()
	if err != nil || !os.SameFile(info, openedInfo) {
		return cfg, errors.New("配置文件读取期间已变化")
	}
	cfg.Server.Listen = "127.0.0.1:8080"
	cfg.Auth.SessionTTL = "168h"
	cfg.Transfer.MaxFileBytes = 256 << 20
	cfg.Transfer.MaxBatchBytes = 1 << 30
	cfg.Transfer.MaxStagingBytes = 2 << 30
	cfg.Transfer.ChunkBytes = 4 << 20
	cfg.Transfer.UploadTTL = "24h"
	cfg.Transfer.ArchiveTTL = "1h"
	cfg.Terminal.TmuxBinary = "/usr/bin/tmux"
	cfg.Terminal.Shell = "/bin/sh"
	cfg.Terminal.HistoryLines = 5000
	cfg.Terminal.RestoreLines = 5000
	cfg.Terminal.HistoryBytes = 8 << 20
	cfg.Terminal.TerminationSeconds = 10
	decoder := yaml.NewDecoder(io.LimitReader(f, 65537))
	decoder.KnownFields(true)
	if info.Size() > 65536 {
		return cfg, errors.New("配置文件过大")
	}
	if err = decoder.Decode(&cfg); err != nil {
		return cfg, errors.New("配置格式无效或存在未知/重复字段")
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		return cfg, errors.New("配置只能包含一个文档")
	}
	return cfg, cfg.Validate()
}

func (c Config) TTL() time.Duration { d, _ := time.ParseDuration(c.Auth.SessionTTL); return d }
func (c Config) StagingPath() string {
	if c.Transfer.StagingPath != "" {
		return c.Transfer.StagingPath
	}
	return filepath.Join(filepath.Dir(c.Storage.Path), "staging")
}
func (c Config) MaxFileBytes() int64 {
	if c.Transfer.MaxFileBytes > 0 {
		return c.Transfer.MaxFileBytes
	}
	return 256 << 20
}
func (c Config) MaxBatchBytes() int64 {
	if c.Transfer.MaxBatchBytes > 0 {
		return c.Transfer.MaxBatchBytes
	}
	return 1 << 30
}
func (c Config) MaxStagingBytes() int64 {
	if c.Transfer.MaxStagingBytes > 0 {
		return c.Transfer.MaxStagingBytes
	}
	return 2 << 30
}
func (c Config) ChunkBytes() int64 {
	if c.Transfer.ChunkBytes > 0 {
		return c.Transfer.ChunkBytes
	}
	return 4 << 20
}
func (c Config) UploadDuration() time.Duration {
	d, e := time.ParseDuration(c.Transfer.UploadTTL)
	if e != nil {
		return 24 * time.Hour
	}
	return d
}
func (c Config) ArchiveDuration() time.Duration {
	d, e := time.ParseDuration(c.Transfer.ArchiveTTL)
	if e != nil {
		return time.Hour
	}
	return d
}
func (c Config) SecureCookie() bool { return c.Server.Mode == "tls" }
func (c Config) TerminalSocketPath() string {
	if c.Terminal.SocketPath != "" {
		return c.Terminal.SocketPath
	}
	return filepath.Join(filepath.Dir(c.Storage.Path), "tmux.sock")
}
func (c Config) CookieName() string {
	if c.SecureCookie() {
		return "__Host-persistty_session"
	}
	return "persistty_session"
}

func (c Config) Validate() error {
	if os.Geteuid() == 0 {
		return errors.New("Persistty 不允许以root运行")
	}
	host, port, err := net.SplitHostPort(c.Server.Listen)
	if err != nil || port == "" {
		return errors.New("listen必须为明确IP和端口")
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return errors.New("listen必须为明确IP地址")
	}
	if !validPort(port) {
		return errors.New("listen端口无效")
	}
	u, err := url.Parse(c.Server.PublicOrigin)
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return errors.New("public_origin必须为无路径的绝对源地址")
	}
	switch c.Server.Mode {
	case "development":
		if !addr.IsLoopback() || u.Scheme != "http" || (u.Hostname() != "localhost" && !isLoopback(u.Hostname())) {
			return errors.New("development仅允许loopback HTTP")
		}
	case "vpn_http":
		if u.Scheme != "http" || !addr.IsLoopback() {
			return errors.New("vpn_http必须显式HTTP与loopback反向代理监听")
		}
	case "tls":
		if u.Scheme != "https" || !addr.IsLoopback() {
			return errors.New("tls要求HTTPS源和loopback反向代理监听")
		}
	default:
		return errors.New("必须明确选择development、vpn_http或tls")
	}
	if strings.ContainsAny(u.Host, "\r\n\\") || u.Hostname() == "" {
		return errors.New("源地址无效")
	}
	if p := u.Port(); p != "" {
		if !validPort(p) {
			return errors.New("源端口无效")
		}
	}
	for _, proxy := range c.Server.TrustedProxies {
		a, e := netip.ParseAddr(proxy)
		if e != nil || !a.IsLoopback() {
			return errors.New("可信代理只能配置loopback IP")
		}
	}
	if _, err = auth.ParseHash(c.Auth.PasswordHash); err != nil {
		return errors.New("password_hash不是受支持的Argon2id hash")
	}
	d, err := time.ParseDuration(c.Auth.SessionTTL)
	if err != nil || d < time.Minute || d > 30*24*time.Hour {
		return errors.New("session_ttl必须为1分钟到30天")
	}
	if c.Storage.Path == "" || !filepath.IsAbs(c.Storage.Path) || filepath.Clean(c.Storage.Path) != c.Storage.Path {
		return fmt.Errorf("storage.path必须是规范绝对文件路径")
	}
	if c.Transfer.StagingPath != "" && (!filepath.IsAbs(c.Transfer.StagingPath) || filepath.Clean(c.Transfer.StagingPath) != c.Transfer.StagingPath || c.Transfer.StagingPath == c.Storage.Path) {
		return errors.New("transfer.staging_path必须是规范绝对目录")
	}
	if c.MaxFileBytes() < 1<<20 || c.MaxFileBytes() > 20<<30 || c.MaxBatchBytes() < c.MaxFileBytes() || c.MaxBatchBytes() > 20<<30 || c.MaxStagingBytes() < c.MaxBatchBytes() || c.MaxStagingBytes() > 40<<30 || c.ChunkBytes() < 64<<10 || c.ChunkBytes() > 16<<20 || c.ChunkBytes() > c.MaxFileBytes() {
		return errors.New("transfer容量配置无效")
	}
	if c.Transfer.UploadTTL != "" {
		d, e := time.ParseDuration(c.Transfer.UploadTTL)
		if e != nil || d < time.Minute || d > 7*24*time.Hour {
			return errors.New("transfer.upload_ttl无效")
		}
	}
	if c.Transfer.ArchiveTTL != "" {
		d, e := time.ParseDuration(c.Transfer.ArchiveTTL)
		if e != nil || d < time.Minute || d > 24*time.Hour {
			return errors.New("transfer.archive_ttl无效")
		}
	}
	for _, path := range []string{c.Terminal.TmuxBinary, c.Terminal.Shell, c.TerminalSocketPath()} {
		if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsRune(path, 0) {
			return errors.New("terminal路径必须是规范绝对路径")
		}
	}
	if c.TerminalSocketPath() == c.Storage.Path || c.TerminalSocketPath() == c.StagingPath() ||
		c.Terminal.TmuxBinary == c.Terminal.Shell || c.Terminal.TmuxBinary == c.TerminalSocketPath() ||
		c.Terminal.Shell == c.TerminalSocketPath() {
		return errors.New("terminal路径不能与存储或彼此重合")
	}
	if c.Terminal.HistoryLines < 100 || c.Terminal.HistoryLines > 50000 ||
		c.Terminal.RestoreLines < 100 || c.Terminal.RestoreLines > c.Terminal.HistoryLines ||
		c.Terminal.HistoryBytes < 1<<20 || c.Terminal.HistoryBytes > 64<<20 ||
		c.Terminal.TerminationSeconds < 1 || c.Terminal.TerminationSeconds > 120 {
		return errors.New("terminal容量或倒计时配置无效")
	}
	return c.validateTools()
}
func isLoopback(s string) bool { a, e := netip.ParseAddr(s); return e == nil && a.IsLoopback() }
func validPort(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n >= 1 && n <= 65535 && strconv.Itoa(n) == s
}
