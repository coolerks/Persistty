# D07 历史恢复与真实 curses 实施报告

## 范围与实际结果

仅新增隔离实验；没有产品 API、认证、前端或根 Go module 修改，没有远端系统安装/sudo/helper。连接只用私有 .env 和现有安全 transport；未记录连接值。D06 两份既有 evidence 保持原样。

最终 Linux amd64 CGO=0 binary SHA256：`c294742fb8848421b3da04b956199327df981c36ebcdf5b9ee2c2b6319e16501`。真实 Debian tmux 3.5a，临时解包既定 3.5a-3/libevent 包；远端不安装 Go/Node。结果见 [摘要证据](../../../../../tests/integration/debian/history/evidence.json)。

| 场景 | 实际判定 |
| --- | --- |
| 普通 200 行历史 | capture 包含 HIST_0001；raw attach 只有当前画面，不含该首行，解析 active buffer 为 tmux 外层 alternate。不是完整历史恢复通过。 |
| capture 后输出 80 行再 attach | 拼接后旧 history 存在 normal，但 active alternate 不显示它；GAP_0080 可见、GAP_0001 不在任一 buffer。这是截点间滚动遗漏反例，不是无损方案。 |
| 真实 Python curses | CURSES_D07、颜色、中文宽字符和尺寸文本被 xterm 解析识别；当前 capture 有 TUI，`capture -a` 的 saved grid 无 TUI；tmux alternate_on=1。 |
| 两种 resize 与 reconnect | 80x24、120x40 都在 curses workload 的实际尺寸文件和解析 grid 中一致；重连仍是同一 pane、TUI marker/尺寸/宽字符/颜色存在。WS resized ack 单独不足，实验同时核对实际程序重绘。 |
| TUI 退出 | tmux alternate_on=0，raw attach 含 TUI_EXIT_D07 且无当前 TUI marker；tmux attach 外层依旧 alternate，不能将其误认为 workload 未退出。 |
| 任意分块 | 7 次真实记录逐字节重分块后，normal/alternate cells（字符、宽度、颜色、基本样式）及光标与原帧解析一致；等待 write callback，未自制 ANSI parser 或剥控制序列。 |
| 进程与资源 | 6 个样本 server/pane PID/start_ticks/cgroup 不变，heartbeat 2→50→95→116→137→161；实际 stdin 字节计数均 0，7 attach 均回收。3 个自身 units、派生 pane scope inactive，最终 ROOT 删除。 |

所有 8 条实验检查为 true；其中历史丢失/外层 alternate 的 true 表示成功观测限制，而不是通过产品恢复验收。

## 文件与依赖

- `tests/integration/debian/history/`：固定 workload、独立 npm manifest/lock、headless 解析及本地测试、摘要 evidence、README/notice。
- `tests/integration/debian/bridgego/`：main CLI 最小 record 分支，record.go/record_test.go，以及 README；不改 server 或既有 D06 client 行为。
- `tests/integration/debian/run_remote.py`：history 入口；不打印 raw frames，暂存 `.cache` 内 0600 文件后删除，只输出脱敏摘要。

实际 npm manifest/registry 确认 `@xterm/headless@5.5.0`、MIT、integrity 已锁。包是 CommonJS，使用 default import 再取 Terminal。实际 typings 确认 proposed buffer/Uint8Array write callback API。发布包没有 LICENSE，完整通知来自固定官方 5.5.0 tag，见 [notice](../../../../../tests/integration/debian/history/third-party-notices.txt)。不声称该版本为最新版本。

## 检查

- 根 `go test ./...`、`go test -race ./...`、`go vet ./...` 通过。
- 独立 bridgego `go test ./...`、`go test -race ./...`、`go vet ./...` 通过；14 个 top-level Go tests，新增 2 个 record limits/实际 PTY 回收测试，未 skip。
- `npm ci --ignore-scripts`、`npm test` 通过，2 个 Node tests；真实记录另有 7 项逐字节状态对照。
- Python history 4、bridge 6、terminal/remote_config 21 tests，共 31 项通过，无远端依赖 mock 冒称实测。
- Linux binary 构建及最终远端实测通过；`git diff --check` 通过。没有执行产品前端检查，因为未改产品前端。

## 失败与限制

两次早期运行在 capture 阶段失败：`capture-pane` pane target 不接受 attach/list-panes 采用的 `=session` 候选形式，换为仅服务器生成的固定 session 名后通过；两轮 finally 清理未报清理错误，未保存各轮完整摘要，不能作为各轮逐字证据。之后完整实验成功，增加 cell/实际 alternate/stdin 断言后重新实测保存最终证据。此修正不允许客户端传 target。

本轮没有 serialize、生产单输出 owner/snapshot sequence、多设备 controller/deadline、browser renderer、真实 browser authenticated WS、Landlock 或 root helper 验收。当前 Bearer/Origin 不改，没有 URL token 或无鉴权代理。持续窗口采用明确阶段间的 80 行固定 burst，证明直接拼接存在反例，不代表任意长期高速输出负载覆盖。D07 进程样本在同一远端执行中；跨 SSH 续存证据仍归 D06，不能重复声称本轮增加该覆盖。

record 仅限 10 秒、256 KiB、4096 帧，分析输入由 Remote 1 MiB 边界和固定程序约束；子命令临时 stdout 文件仍是时间约束而非磁盘 quota。正常 finally 验证已知资源清理；runner SIGKILL/机器退出/网络失败仍无持久恢复账本，不能宣称所有故障下零残留。结果待独立 check；不提交、推送、归档。
