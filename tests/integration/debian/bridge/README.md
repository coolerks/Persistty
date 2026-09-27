# Debian Go/PTY/WebSocket 基础链路探针

仅在明确授权的 Debian 13 amd64 环境运行，不是产品 API 或完整终端验收。二进制来自独立 [bridgego 模块](../bridgego/)，远端无需安装 Go。

## 入口与隐私

```sh
python3 -B tests/integration/debian/run_remote.py bridge --binary /tmp/bridge-probe
```

先按 bridgego 文档构建名为 `bridge-probe` 的 Linux amd64 二进制。入口仅安全解析仓库根私有 `.env` 的三个 `DEBIAN_*` 字段，校验和错误约定归 [连接规范](../../../../.trellis/spec/backend/remote-validation.md)。不使用进程环境变量补齐，不回显值、展开的 SSH/SCP 参数或原始错误。stdout 是脱敏观察结果，不是原始逐字证据。

## 资源与判定

独立 `mktemp` 0700 目录中临时解包固定 tmux 3.5a-3 与 libevent-core-2.1-7t64 2.1.13-stable-1~deb13u1，不执行 sudo、不安装全局包。64 字节随机 token 存在 0600 私有文件中，不放进 argv、URL 或报告。Go 监听 loopback 动态端口，检查 Bearer 与固定 loopback Origin；独立用户级 unit 采用 `KillMode=control-group`。tmux 独立 foreground server、私有 socket，不连接默认 server；Go 只 attach 已有精确 session。

专用 raw-mode Python workload 不执行 shell 命令，只接收固定实验输入。一次跨 UTF-8 字节拆分的二进制输入对应一次计数并回显 ACK；重连、服务 stop/restart/SIGKILL 不发送此输入。每次采样检查 server/pane PID、start ticks、cgroup 不变，心跳严格比上次增长，input_count 始终为 1。真实 Go attach-client 必须在 Web unit cgroup，server 和 pane 则不能在其中。

Go client `exercise` 自测认证/Origin、单活动 attach、resize/非法控制、超大输入、二进制收发和资源回收；`observe` 只读周期输出并断开；`hold` 在独立 client unit 中保持真实活动 WS，供 Web 生命周期测试。`start` 与 `check` 经独立 SSH 连接运行，禁用连接复用。

每条子命令最长 25 秒，stdout 返回内存最多 1 MiB，stderr 丢弃；临时 stdout 捕获文件仅受时间限制，不是磁盘配额。就绪条件最多轮询 8 秒（单命令另受 25 秒上限）。单位最多 300 秒，workload 最多 280 秒，hold client 最多 30 秒。服务日志丢弃，实验未采集用户输入或业务文件。

## 清理与保留门禁

runner finally 调用本次明确 ROOT 的 `cleanup`，只 stop 记录的三个精确 `persistty-bridge-<随机ID>` unit 和与本次 pane 关联的精确 `tmux-spawn-*.scope`，检查原 PID/start ticks 不再存活及目录确已删除。不通配停止单位，不修改 Nginx、WireGuard、防火墙或已有业务服务。失败只报固定类别，清理失败不能称无残留。

本机 runner SIGKILL、主机退出或不可达可跳过 finally；没有持久恢复账本。单位 RuntimeMaxSec 不能删除临时目录，也不能替代 cleanup；需要按本次已知 ROOT 精确复核残留，不扩大清理范围。入口拒绝 `python -O`。

这里只验证单活动 attach 的基础桥接和生命周期，不证明浏览器/xterm/TUI、snapshot/live 无缝衔接、多设备 controller/deadline、慢客户端产品策略、主机重启恢复或完整终端 API。tmux 可能解释部分控制字节，UTF-8 跨帧证明不等于任意八位字节在 tmux 显示层原样回显。

## 本地回归

```sh
python3 -B -m unittest discover -s tests/integration/debian/bridge -p 'test_*.py' -v
python3 -B -m unittest discover -s tests/integration/debian/terminal -p 'test_*.py' -v
```

本地测试使用虚构参数及 mock，不建立远端连接；覆盖优化模式拒绝、输出限额、固定错误、逐采样心跳和非实验 unit 拒绝，以及传输失败后精确 cleanup。

首轮脱敏实测见 [evidence.json](evidence.json)：13 次采样原 server/pane 身份一致，心跳 1→58，输入仅执行一次；7 次 attach 已回收，FD=8、goroutines=6 与本次基准一致，三个 unit 和派生 scope 均 inactive、目录已删除。真实 WS 客户端是 Go client，不是浏览器。资源采样只描述此有限 trace，不证明任意负载下无泄漏或背压策略已验收。首轮二进制 hash 随证据保存；后续版本（包括许可证嵌入）须独立复核，不能冒充已观测版本。

最终版本独立复核见 [evidence-review.json](evidence-review.json)：二进制包含完整第三方通知与 PTY fd 的 close-on-exec 修正，hash 随证据保存。13 次采样心跳 1→59，明确输入之后计数始终为 1，原 server/pane 身份保持；7 次 attach 全部回收，三个 unit 与精确派生 scope inactive，目录已删除。复核前后只读检查当前用户拥有的 0700 非 symlink `/tmp/persistty-bridge-*` 目录，均未发现残留；这不是对其他目录或任意异常路径零残留的保证。
