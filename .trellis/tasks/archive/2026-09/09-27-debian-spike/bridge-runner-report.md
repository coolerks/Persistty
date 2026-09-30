# D06 Go/PTY/WS 编排首轮报告

## 文件与边界

新增 `tests/integration/debian/bridge/{probe.py,test_probe.py,README.md,evidence.json}`，扩展 `run_remote.py bridge --binary` 和现有私密入口回归。只消费私有 `.env`，不读取进程环境补齐、不回显连接参数、原始错误、token 或终端正文。probe 只在随机 0700 ROOT、私有 tmux socket 与三个精确用户 unit 中运行，不改产品服务、Nginx、WireGuard、防火墙、系统安装状态或 helper。

## 实测

首轮客户端为真实 Go WebSocket client，服务为真实 Go/PTY/tmux attach，不是浏览器/xterm。实验 workload 使用 raw stdin 的固定 UTF-8 marker；拆帧位置在汉字字节内部，只回 ACK 并计数，不执行 shell 命令。

- binary SHA256：`12bc6a54100c3adfa014cb3d4824331ca83c09cdc0c69e3cbf2e7aa64def2d93`。后续许可证嵌入版 hash 改变，须由整体 check 独立重跑，不能以本证据声称新版已实测。
- 13 次采样 server PID=914243、pane PID=914246，start ticks 与 cgroup 身份均不变；pane 在自己的 tmux-spawn scope，均不在 Web unit cgroup。
- 心跳严格逐样本增长 1→7→19→23→27→31→35→39→42→46→50→54→58；输入计数从 0 变为 1，后续跨 SSH、stop、restart、SIGKILL 和只读重连始终为 1，无自动输入重放。
- 三次操作前都有真实活动 WS/PTY attach；所采集 attach-client PID/cgroup 均归 Web unit。systemctl restart 后新 Web PID 已改变且再次真实观察到输出，原 pane 继续运行。
- exercise 的认证拒绝 401、错误 Origin 403、单活动 attach 409、正确 resize、非法 resize 1008、超限输入 1009、UTF-8 二进制往返和 detach/reconnect 回收断言均通过。7 次 started=7 次 reaped，FD=8、goroutines=6 与本次基准一致。只是有限 trace，不保证任意负载下无泄漏或完整慢客户端策略。

首轮完整去敏数据在 [evidence.json](../../../../../tests/integration/debian/bridge/evidence.json)。不是原始逐字 stdout；cgroup 身份使用稳定占位符，保留进程关联及实验判定。

## 失败与清理

首轮执行前传输曾在固定 `allocate` 阶段失败，尚未取得已知 ROOT，未执行上传或启动单位。没有回收到 mktemp stdout 不能绝对证明远端未执行目录创建；整体 check 应只读核查实验前缀目录，不用通配删除补偿。仅内存分类后的 `ssh true` 成功才继续；没有改变连接配置、密钥或 host-key 校验。未保存原始 stderr 或连接身份。

成功实验 ROOT `/tmp/persistty-bridge-avG1D0dH` 已删除；`persistty-bridge-5bb1ef152634-{client,web,tmux}.service` 与其 pane 关联 `tmux-spawn-48ec64d5-c885-4c7b-b8e2-abeb1e9bd6c3.scope` 均 inactive，原进程 identity 已结束。所有本次远端工具会话已退出。仍保留 runner 本身 SIGKILL/网络不可达可能跳过 finally、无持久恢复账本的限制，不泛化零残留。

## 本地检查与剩余门禁

bridge 本地 6 项测试、terminal/remote_config 21 项测试通过；`git diff --check` 通过。纯本地测试不建立远端连接。Go module 的 test/race/vet 归 Go owner 与整体 check。

保留浏览器/xterm/TUI、snapshot/live 时序、多设备 controller/deadline、任意二进制显示、慢消费者、主机 reboot、Landlock CLI 与提权 helper 等门禁。本轮不提交；主会话负责整体复核、规范和提交确认。
