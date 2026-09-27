# 后端质量门禁

## 当前阶段
W01已有源码，Go test/race/vet和真实SQLite/router测试必须执行。尚未实现的PTY、文件和helper按对应交付测试，不能把缺目标Debian实验写成已通过。web作为独立Go模块边界，根`go test ./...`只检查第一方Go包，不执行node_modules内的Go样例或测试。

## 每任务门禁
Go 源码 gofmt，`go test ./...`、`go vet ./...`；并发/bridge/watcher/session 用 `go test -race ./...`。unit 测试表驱动、临时目录、fake 时钟和故障注入，不写“调用过 mock 所以安全”的镜像测试。
独立实验模块 tests/integration/debian/bridgego 有自己的 manifest/锁文件，根模块测试不会遍历它；须额外在该目录执行同样的 test/race/vet，并对最终 Linux binary 留 SHA256 与远端复核证据。不得只检查根模块便声称桥接测试通过。
config、auth、session、path validation、symlink/TOCTOU、file conflict、atomic save、upload、rg parser、tmux parser、git parser、SQLite migration 都要在对应功能任务中有断言。真实 integration 使用独立 filesystem/Git/rg/SQLite/tmux/PTY，缺依赖明确 skip 原因；release CI 中 required suites skip 即验收失败。单元测试不能覆盖 Debian systemd 证明。
安全 review task 必须逐项留修复或测试证据：auth bypass、WS Origin/认证撤销、CSRF、path/symlink、shell/flag injection、upload overwrite/quota、ZIP traversal、SVG XSS、binary、session fixation/brute force、敏感日志、权限/roots。

### Debian 探针验收规则
探针的每次心跳必须比上次采样增长，PID/start time/cgroup 则与固定初始身份比较；只比初始心跳大可能误判早已停止推进的任务。Python 使用 assert 做判定时入口必须拒绝 `-O`，不能只在 README 禁止。强杀或 timeout 可能跳过 defer/finally，内部 TMPDIR 必须归入已知且校验过的自有上传目录，并提供精确 unit/scope/目录清理命令。输出内存限额不等于磁盘配额；本轮临时文件命令捕获只有时间界限，不能复用为正式服务的通用执行沙箱。

## 关键 acceptance
Terminal lifecycle Spike 先于正式 Terminal UI，真实 Debian Web restart 后 job 不变；v0.1 release 运行完整 A/B/C critical E2E。另含 file conflict、ignore/replace conflict、upload/resume/ZIP/symlink、watcher、Git CLI 对照。测试归属 task PRD，不建立另一套 TODO。
实现变更和测试同任务交付；测试隔离、deadline、有界输出、可靠清理，只清理自己的 PID/socket/session。不能用固定 sleep 当唯一就绪判断，采用心跳/health/条件轮询。

## 审查禁止项
无认证的 download/preview/WS、记录 terminal input、掉线 kill-session、无版本写入、任意 shell 参数、只 HasPrefix 路径保护、长事务执行外部进程、无界内存/队列、未解释 skip。review 根据实际行为和复现证据，不机械复述规范。
