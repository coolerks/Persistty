# W02 D06 独立复核

## Findings (fixed)
- 文件：`tests/integration/debian/bridgego/server.go`。问题：`syscall.Dup` 得到的 PTY fd 默认不带 close-on-exec，未来在存活 bridge 期间执行其他子进程会继承资源。修复：Dup 后设置 `CloseOnExec`，保持非阻塞包装和已有取消逻辑，新增真实 PTY fd 标志回归。
- 文件：`tests/integration/debian/bridgego/server_test.go`。补齐未授权 `/stats` 不泄露 PID、慢客户端导致有界输出关闭 1013 且 attach 已 Wait 的真实回归；修正 escaped-key 重复字段样例，使它实际测试解码后重复键，而非单纯未知键。
- 文件：`tests/integration/debian/bridge/test_probe.py`。新增最终独立证据与首轮证据的一致性断言，两份均校验逐次心跳、固定身份、输入计数及精确清理。不覆盖或篡改首轮行为版证据。

## Findings (not fixed)
- 本机 runner SIGKILL、退出或远端不可达仍可跳过 finally；没有持久恢复账本。这是编排设计范围，不能用通配删除或停止其他 scope 补偿，本轮不改为生产执行框架。早期三次 allocate 失败不能据失败信息断言没创建目录；复核前后只读检查当前用户拥有、0700、非 symlink 的 `/tmp/persistty-bridge-*` 目录均为空，目前未发现该限定集合中的残留，也没有清理未知资源。
- 仍未验收浏览器/xterm/TUI、snapshot/live 一致性、多观察端与 controller/deadline、生产认证撤销、受限 CLI/Landlock、root helper 和生产服务部署。实验的单 attach 锁不是产品方案；有限循环和慢客户端 trace 不是任意负载资源证明。这些属于保留的交付门禁，不是本轮可机械修复的问题。

## Verification
- Go 格式检查：修改文件已 gofmt；Go 静态检查（vet）：根模块、隔离 bridge 模块均通过。独立模块没有另外的 lint/typecheck 脚本，编译与 vet 作为该 Go 交付门禁，不把前端未运行门禁算通过。
- Tests：根 `go test ./...`、`go test -race ./...`、`go vet ./...` 全通过，6 个第一方包；隔离模块 12 个顶层测试，最终 `go test -count=1 ./...` 与 `go test -race -count=3 ./...`、`go vet ./...` 通过，无 skip。真实 PTY/httptest 覆盖认证/Origin/单 attach、无效 UTF-8 与零字节、resize 严格 JSON/限额、取消及 15 次 detach 回收。
- Python：bridge 6 项、terminal/remote_config 21 项通过，均不靠 mock 声称 Debian 验收。两份真实证据校验包含在 bridge 测试中。
- 依赖：锁定 creack/pty v1.1.24、coder/websocket v1.8.15；通知与实际两个模块完整 LICENSE、Go 1.26.8 LICENSE 一致，最终 Linux 二进制包含完整通知文件。独立 go.mod 不影响产品 manifest，根模块测试不包含该模块，已分别运行门禁。
- 最终 Linux amd64 二进制 SHA-256：`788be5a918f7593e63d023b77c98a7d9dc88a99ccb51d0d2a00d162cd0bbbe78`。独立 Debian 实测使用这个修正后的版本，见 [evidence-review.json](../../../tests/integration/debian/bridge/evidence-review.json)。实验错误不回显真实连接值或原始命令。
- 13 次样本心跳为 `1,7,20,24,27,31,35,39,43,47,51,55,59`；server/pane PID、start ticks、cgroup 与固定初始身份相同。一次明确输入后 input_count 始终为 1，stop/restart/SIGKILL 与重新 attach 未重放输入。真实 attach-client 位于 Web unit，server/pane 不在 Web cgroup；tmux server 为专属预启动，Go 只使用 `-N` attach。
- WS exercise 7 次 attach 的 started/reaped 均为 7，FD=8、goroutines=6 与本次基准相同；随后 hold/observe 验证正在连接时的 Go 生命周期。三个精确自有 unit 和记录的精确派生 scope 均 inactive，ROOT 已删除，未修改既有服务、网络、系统安装状态。
- 隐私：报告新增前的可提交工作树文本 280 个与 HEAD 文本 264 个，私有 `.env` 仅在内存解析，实际三个字段精确匹配扫描零命中，只输出计数/路径；`.env` 不入库。任务 JSONL validate 通过（4+4），报告完成后的文档局部链接 22 个无缺失，`git diff --check` 通过。主会话新增报告后应再次运行提交前扫描。

本报告不提交、推送或归档；真实远端实验完成且本轮所有命令会话均已结束。
