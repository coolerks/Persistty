# 后端质量门禁

## 当前阶段
bootstrap 只有文档：检查契约覆盖、简体中文、无占位、有效链接、索引、正反例及断言点。不伪造 go test 输出；产品测试等源码和对应任务存在再执行。

## 每任务门禁
Go 源码 gofmt，`go test ./...`、`go vet ./...`；并发/bridge/watcher/session 用 `go test -race ./...`。unit 测试表驱动、临时目录、fake 时钟和故障注入，不写“调用过 mock 所以安全”的镜像测试。
config、auth、session、path validation、symlink/TOCTOU、file conflict、atomic save、upload、rg parser、tmux parser、git parser、SQLite migration 都要在对应功能任务中有断言。真实 integration 使用独立 filesystem/Git/rg/SQLite/tmux/PTY，缺依赖明确 skip 原因；release CI 中 required suites skip 即验收失败。单元测试不能覆盖 Debian systemd 证明。
安全 review task 必须逐项留修复或测试证据：auth bypass、WS Origin/认证撤销、CSRF、path/symlink、shell/flag injection、upload overwrite/quota、ZIP traversal、SVG XSS、binary、session fixation/brute force、敏感日志、权限/roots。

## 关键 acceptance
Terminal lifecycle Spike 先于正式 Terminal UI，真实 Debian Web restart 后 job 不变；v0.1 release 运行完整 A/B/C critical E2E。另含 file conflict、ignore/replace conflict、upload/resume/ZIP/symlink、watcher、Git CLI 对照。测试归属 task PRD，不建立另一套 TODO。
实现变更和测试同任务交付；测试隔离、deadline、有界输出、可靠清理，只清理自己的 PID/socket/session。不能用固定 sleep 当唯一就绪判断，采用心跳/health/条件轮询。

## 审查禁止项
无认证的 download/preview/WS、记录 terminal input、掉线 kill-session、无版本写入、任意 shell 参数、只 HasPrefix 路径保护、长事务执行外部进程、无界内存/队列、未解释 skip。review 根据实际行为和复现证据，不机械复述规范。
