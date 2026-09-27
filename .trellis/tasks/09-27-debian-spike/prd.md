# Persistty W02 Debian 隔离环境实验

## 授权与目标
依赖 W01 提交 4113358 与父计划 W02。用户提供指定 SSH 环境，并在隔离实验范围总结之后明确“可以，继续推进”。目标是获取真实 Debian 终端 cgroup/进程持久性和根句柄安全证据，供 W03/W04 设计使用，不实现正式终端或文件 API。

## 已确认环境与边界
SSH 连接从仓库根已忽略 .env 的 DEBIAN_USER/DEBIAN_IP/DEBIAN_PORT 字段读取，使用既有 key，不记录实际值；Debian 13.4、systemd 257.9、用户 manager running/linger yes。只允许 tmux 安装与独立临时目录/socket/用户级 transient unit，清理自己的资源。sudo -n 安装需要密码，未安装成功；使用软件源 Debian tmux 包解包到用户临时目录的降权方案，不修改系统安装状态。不得索取密码，不改 Nginx/WireGuard/防火墙或已有服务，不安装 helper，不使用默认 tmux socket。

## 验收
- D01 记录实际工具版本与临时资源身份；资源权限私有，清理仅精确自身路径/unit/socket。
- D02 tmux server/pane job 独立于模拟 Web unit cgroup，取消 attach、停止/重启模拟 Web、断 SSH 后原 PID/start time 不变且任务继续；停止实验 tmux unit 不宣称进程可恢复。
- D03 明确预启动空 server、无 server 时只 attach 不隐式创建、多客户端/尺寸/恢复能力与剩余真实 PTY/WS/TUI 验证边界。
- D04 Linux 根句柄读取/创建/rename/remove 对根外 symlink/父目录交换的拒绝证据，合法根内访问有效；不把有限竞态实验宣称无竞态证明。CLI 根安全与提权仍未验收。
- D05 可重复探针、限时/有界输出/精确清理与检查报告入库；缺依赖/未执行不标通过。
- D06 当前继续阶段验证真实 Go + creack/pty + coder/websocket 实验 bridge：二进制数据传输、明确输入、连接取消/重连、服务正常停止与 SIGKILL/重启后原 pane 身份及任务推进；依赖锁定在隔离实验模块，不改产品 manifest。
- D07 继续验证历史恢复机制与限制：固定合成普通历史、持续输出间隙和真实 Python curses TUI，在同一授权隔离环境经 Go/PTY/WS 取证，使用锁定 xterm 库解析。对照 raw attach、capture 后追加 attach 的缓冲区/重复/遗漏；记录 resize 后真实 TUI 重绘与原进程身份。负面结果同样留证，不以 sleep/正则剥 ANSI 拼成生产恢复方案。

## 修改范围与非目标
新增 tests/integration/debian 的隔离探针及本任务报告；主会话维护任务及 owner spec。不得修改产品 API/schema/前端，不改现有系统级部署。D06 基础链路已通过；D07 只验证恢复机制与一个受控真实 TUI。浏览器实时鉴权链路、完整 TUI 覆盖、生产 snapshot/live 一致性、多设备 controller/deadline、Landlock CLI 和 root helper 仍待后续专门实验，不因局部探针成功解除其门禁。
