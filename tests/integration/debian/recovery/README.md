# W03 终端恢复门禁实验

在用户明确授权的 Debian 环境上运行：

```sh
python3 -B tests/integration/debian/run_remote.py recovery
```

runner 只读取仓库根已忽略的 `.env`，在目标机创建 `0700` 的私有 `/tmp/persistty-recovery-*`、独立 tmux server/session 和 systemd 临时 unit。探针只输出合成编号、拆分 CSI/UTF-8/OSC 与 curses 画面；PTY 原始片段仅在本机已忽略的 `.cache/` 中以 `0600` 临时文件交给固定版本 xterm headless 解析，随后删除。所有实验 unit、scope 和目录按随机身份精确清理，不操作用户终端。

控制模式同一连接内的 `capture-pane` 与 `%output` 对 400 条持续编号输出没有重漏，但 `capture-pane -P` 只补回 pending CSI/OSC；拆开的 UTF-8 前缀没有出现在快照中，因此不能把控制模式画面快照和原始 `%output` 拼成正式重连流。

候选恢复模型将**活动画面**和**普通历史**分开：活动画面使用普通只读 PTY attach，让 tmux 自己重绘完整终端状态；xterm 的本地 scrollback 为 0，避免重复积累。普通历史单独从 `capture-pane -p -e -S - -E -1` 获取，作为有界、可替换的虚拟滚动快照，而不写入活动 xterm。tmux 3.5a 不支持较新手册中的 `capture-pane -L`；使用 `#{history_size}` 与窗口配置的 `history-limit` 校验边界。alternate screen 期间普通历史仍可单独读取。只读观察端的尺寸变化不改变 pane，可写控制端的尺寸变化会触发 curses 正确重绘。

慢端负例：故意不读只读 attach 的 PTY 时，5,000 行 burst 与原 pane 身份均保持，但该 attach 的温和关闭超时。正式桥接必须持续读取 PTY、限制 WS 写入队列，并在超限时有界强制回收**仅这个 attach 客户端**，不依赖 tmux 自动隔离，也不 kill pane/server。

正式产品探针需先将 `./tests/integration/debian/recovery/runtimego` 编译为文件名严格为 `runtime-probe` 的 Linux 二进制，再执行 `python3 -B tests/integration/debian/run_remote.py recovery --binary /private/tmp/runtime-probe`。runner 通过文件名选择产品探针；换名会退回纯机制实验。产品检查覆盖创建与元数据故障恢复、项目解绑、精确终止、有效 pane history-limit、HTTP Cookie/CSRF/历史、两端 WS 接管、旧代输入拒绝、终止广播与观察端取消、登出撤销，以及缺 tmux server 不隐式重启。2026-09-30 最近一次机制/产品探针合计 40 项全部为 true，私有 unit/scope 与目录已清理。

另有 [隔离浏览器 E2E](../browser/README.md)，2026-09-30 完整九条通过：三种计数/HTTP/确定性 TUI 负载跨正常/SIGKILL Web 重启保留 PID/start/cgroup、端口和任务输出，关页/离线/登出后继续交互；桌面/手机真实画面、历史、方向/鼠标/控制字节/粘贴、链接、接管、三端倒计时、精确终止及宿主移动零新 WS 全部通过。T01..T07 证据见 [W03 验收](../../../../.trellis/tasks/archive/2026-09/09-29-terminal-runtime/check-report.md)。`evidence.json` 保留 09-29 机制阶段尚未集成的原始观察，不代表最新产品状态。
