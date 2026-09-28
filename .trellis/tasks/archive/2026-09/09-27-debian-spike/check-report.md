# W02 检查报告

## 文件探针审查

- 已按 check.jsonl、PRD、design、implement 读取批准范围，未修改产品 API、manifest、前端或现有目标服务。
- 未发现静态路径解析越界；正常路径、外部读写/创建/rename/remove、父目录交换及三个外部哨兵的结果均由真实操作断言，25 项结果不构成所有调度的形式证明。
- `os.Root` 移出目录后仍绑定原身份，报告明确保留这一限制；Landlock 只查询 ABI，不宣称受限 CLI 已验收。
- 已修复复现说明的超时清理缺口：最终 JSON 在正常结束后才出现，强制终止无法得知内部随机目录。远端复现现在将 TMPDIR 限制在本次已校验的上传目录内，并要求失败后只精确清理该目录；SSH/SCP 均禁止未知主机自动接受。
- 增加已取消 context 的回归，断言返回取消错误、结果不能通过、自有目录已删除。普通文件系统调用不能被 context 强制打断，说明已明确外层 timeout 的职责。

## 文件真实 Linux 复核

- 独立交叉编译测试二进制，上传到本次新建 `/tmp/persistty-check-files.gW4BWL0B`；TMPDIR 同样限制在该目录内。
- Debian 执行 `timeout 25s ... -test.v -test.timeout=20s`，3 项测试通过，RootProbe 约 0.23 秒。
- 二进制与上传目录精确删除，远端不存在断言通过。没有启动服务、socket 或持续运行的进程，没有访问业务目录。
- 首次本地 race 因默认 Go cache 的 sandbox 权限拒绝失败；使用独立 `/private/tmp/persistty-debian-check-cache` 重跑后，完整 `go test ./...`、`go test -race ./...`、`go vet ./...` 均通过（6 个第一方包）。本机 race 结果不是 Linux 文件系统竞态证明。

## 终端探针修复

- 每个 snapshot 原只要求心跳大于初始 baseline，后续任务停住仍可能通过。已改为心跳严格大于上一 sample，同时 PID/start ticks/cgroup 仍绑定初始身份；新增停滞回归。
- 命令 capture_output 原无内存上限；改为捕获到自身目录临时文件，每流读取最多 1 MiB，超过即拒绝。磁盘捕获仍受 20 秒命令超时而非字节配额限制，此限制明确记录，不宣称完整磁盘配额防护。
- 入口主动拒绝 Python 优化模式，防止 `-O` 关闭实验断言；新增拒绝回归。README 增加 mktemp ROOT 格式校验和新 SSH 连接不复用 master 的选项。
- 清理仅使用本次 state 精确记录的随机服务和关联 pane scope，没有泛匹配或触碰其他 tmux scope；权限、私有 socket、命令参数列表与失败路径按实际代码核对。

## 终端真实 Linux 复核

- 自有目录 `/tmp/persistty-terminal-luRHNXKL`，单位前缀 `persistty-spike-a3c0a42d7df0`，独立 scope `tmux-spawn-b23fc9e7-2147-4ed2-82d2-1daacc379995.scope`。
- 非 root 下载解包固定 tmux 3.5a-3 与 libevent-core 依赖，仅局部 LD_LIBRARY_PATH；没有系统安装或服务配置变更。
- server PID 856481/start ticks 914345156，pane PID 856486/start ticks 914345271；Web stop/restart/SIGKILL 后身份不变，逐 sample 心跳增长；40×10 只读观察 PTY 不改变 100×30 pane。
- start SSH 退出、新 SSH check 心跳从上一采样 24 增到 101，固定 PID/start ticks/cgroup 不变；缺 socket 的 `-N new-session` 非零退出且 socket 不存在。
- 真实 Linux `py_compile` 与 3 项 Python 判定回归通过，编译缓存只在自身目录删除。正常 cleanup 3 个服务及精确 pane scope 均 inactive，原 server/pane 消失，目录不存在。
- 另在 `/tmp/persistty-terminal-RyoI9DxJ` 注入首次 apt 下载失败，start 非零退出，自动 cleanup 对自身随机单位均 inactive，目录删除；该失败没有下载软件或启动任何单位。两个 terminal 根及文件复核根最终不存在断言通过。
- 未更改原始 evidence.json；本节为独立复核证据，不将模拟 Python PTY 等同于产品 Go/WS/TUI。

## 最终门禁与剩余范围

- Go test / race / vet：通过，6 个第一方包；文件真实 Debian 3 项测试通过。
- Python：本地与真实 Debian 3 项回归通过，Linux 编译通过。
- gofmt 与 git diff --check：通过；无前端修改，本任务前端检查不适用。
- 远端资源：本次创建的 3 个临时根均不存在，自有单位与派生 scope inactive；没有残留持续探针。

真实 WebSocket、浏览器/TUI、CLI Landlock、原子版本保存、提权 helper 仍未验收；不将系统隔离探针通过等同于完整 W02 或 MVP 完成。

## 私密配置与去敏复核（仅本地）

- 本轮只按最终契约读取仓库根已忽略 `.env`，没有 SSH、安装、远端重跑、stage、commit 或 push。上方真实 Linux 证据属于此前授权实验，不能用来宣称新 runner 已经实机验收。
- 文件读取核对为 O_NOFOLLOW/O_NONBLOCK、普通文件、当前 owner、0600、4 KiB 上限、UTF-8；解析不 source/eval，不执行命令或插值，不从同名进程环境补齐/覆盖。重复、未知字段和非法连接值在传输前失败。
- SSH/SCP 使用参数数组、严格 host key、禁用配置文件及连接复用；IPv6 SCP 目标括号化。stderr 丢弃，stdout 在内存有界并经结构化去敏后输出，失败只输出固定类别。
- 修复 start 传输在远端 probe 创建 state 前失败时的清理缺口：没有 state 时仅删除本次上传文件并 rmdir 自有根；有 state 时仍调用精确单位/scope 清理，禁止泛删除。
- 修复连接端口字段去敏：port/DEBIAN_PORT/ssh_port/remote_port 的字符串或数字值去敏，不改写恰好同值的 PID/start ticks/heartbeat。此规则覆盖当前已知观测结构的显式连接字段，不宣称可安全归档任意未知原始传输正文。
- 脱敏证据有效 JSON，声明不是原始输出；UID/HOME 关联使用稳定占位符，PID/start ticks、samples 心跳和 cgroup 隔离关系保留。
- 私密扫描只在内存安全解析 `.env`，对全部 tracked 与非忽略 untracked UTF-8 可提交文本、HEAD 文本分别扫描：命中文件均为 0，仅输出计数；`.env` 被忽略且未跟踪。没有打印实际值或匹配行。
- Python 本地回归 18 项通过，新增端口/身份数值碰撞、start 未启动清理、非法 UTF-8、端口边界与环境不能补缺字段。Python 语法、JSON/JSONL、shell 片段语法、本地文档链接、task.py validate（implement/check 各 3 条）及 git diff --check 均通过。无产品 Go/前端变化，本轮不重跑这些门禁。
- 仍有明确限制：runner 本身被 SIGKILL 或主机退出时 finally 不保证执行，没有持久清理恢复账本；README 已同步，不宣称此类中断后零残留。新增 runner 本轮未连接真实目标。
- 遵循最新用户要求：本批保留为工作区修改，不提交。
