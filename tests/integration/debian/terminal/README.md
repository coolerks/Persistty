# Debian 终端隔离探针

仅用于明确授权的 Debian 13 amd64 实验机，需要 Python 3、apt/dpkg、用户 systemd manager、既有 SSH key。不是产品终端实现。当前固定软件源版本为 tmux 3.5a-3 与 libevent-core-2.1-7t64 2.1.13-stable-1~deb13u1；候选改变时先审查并调整版本，不静默换包。

## 重跑

仅在授权重跑时执行。连接参数唯一来源是仓库根已忽略的 `.env`，原键为 `DEBIAN_USER`、`DEBIAN_IP`、`DEBIAN_PORT`；不读同名进程环境变量、不使用默认值。文件必须是当前用户所有的普通非链接文件、权限 0600、最多 4 KiB。安全字面解析支持 `KEY=VALUE`、单/双引号、空白和注释，不支持 `source`、`export`、插值、转义或执行语法；缺失、重复、未知字段或非法值在连接前失败。禁止打印 `.env`、展开后的 SSH argv 或原始传输 stderr。

```sh
python3 -B tests/integration/debian/run_remote.py terminal
```

runner 创建独立 0700 临时目录，上传探针，并通过分别建立的 SSH 连接执行 start/check/cleanup；连接之间等待一秒用于心跳比较，禁用复用 master。SSH key 使用默认路径，关闭 SSH 配置文件避免覆盖目标参数；不会请求密码或接受未知主机 key。失败返回固定类别，不回显连接信息。每次传输最多 60 秒、stdout 内存上限 1 MiB、stderr 丢弃；仅输出脱敏 JSON。cleanup 始终针对本次校验过的 ROOT，无法连接完成清理时报告失败，不宣称清理成功。本地 runner 被 SIGKILL 或主机退出时 finally 不保证执行，当前没有持久化的清理恢复账本；远端单位自限也不能保证临时目录自动删除。

实验单位自限 180 秒；runner 应在 start 后 180 秒内完成 check/cleanup。每个命令超时 20 秒，就绪轮询截止 8 秒（正在执行的单条命令仍受其自身 20 秒上限）；单位 RuntimeMaxSec=180，attach 自限 120 秒，心跳自限 170 秒。命令输出先落入本实验目录临时文件，返回内存时每流最多 1 MiB，溢出拒绝；磁盘捕获受命令超时而非字节配额限制。start 普通异常自动清理；若 SSH 中断、主进程被强制杀死或超时，使用同一明确 ROOT 执行 cleanup，并核实报告的精确 unit/scope/进程已结束。若下载前失败且没有 state.json，确认仅包含本次上传文件后精确删除该 ROOT，不套用到既有目录。单位自限不能代替临时目录清理。禁止使用 Python `-O`，探针以断言执行实验判定。不要手工 stop 泛匹配的 tmux/persistty 服务。

## 判定与边界

- `tmux -D -S <private> -f <private-config>` 与 `exit-empty off`、`exit-unattached off` 在空 session 时存活。缺失 socket 时 `-N new-session` 拒绝且未创建 socket。
- pane/server 身份按 `/proc/PID/stat` start ticks 与 cgroup 比较，心跳增长必须满足。模拟 Web unit 使用 `KillMode=control-group`，只负责原生 PTY attach。
- Debian 所用 tmux 的 pane 可在 `tmux-spawn-UUID.scope`，不强求与 server 同一个 cgroup；二者必须独立于 Web cgroup。只清理实际记录且与自身 pane 关联的精确 scope。
- 40×10 观察 PTY 使用 `read-only,ignore-size`，100×30 pane 不被缩小；此项不等于应用控制权、generation 或 WebSocket observer 验收。
- capture-pane 只展示有界历史可读取，没有验证 snapshot/attach 零重复、零丢失或完整 TUI 恢复。没有测试真实浏览器、Go Web 服务、Nginx、HTTP 服务任务或主机 reboot。
- cleanup 停止实验 tmux server，记录 heartbeat pane 是否消失，然后清理精确派生 scope。该心跳任务消失不能推导所有忽略 SIGHUP 的任意子进程都会结束。

脱敏实测记录见 [evidence.json](evidence.json)，不是原始输出。cgroup 中同一用户 UID 使用稳定占位符，保留 PID/start ticks、心跳和 cgroup 隔离关系。重跑的原始 stdout/stderr、HOME、UID、SSH argv 或环境不得直接归档；只保存去标识后的结果。失败报告仅使用阶段、退出码和固定错误类别，不回显完整连接命令或环境。

## 判定回归

`python3 -B -m unittest discover -s tests/integration/debian/terminal -p 'test_*.py' -v` 验证迟滞心跳拒绝、命令输出内存限额、`-O` 拒绝、私密文件缺失/非法/权限/非链接校验、进程环境无法覆盖、IPv4/IPv6 传输参数、失败清理和脱敏身份比较。测试使用虚构 fixture 与 subprocess mock，不连接远端、不创建服务。真实 Linux 编译可在自身实验目录执行 `python3 -m py_compile probe.py test_probe.py`，随后精确删除该目录内的 `__pycache__`。入口主动拒绝优化模式，不仅依赖操作者遵守说明。
