# 持久终端与技术验证

## 当前批准规则（覆盖下文bootstrap候选）
U06..U73要求多观察端、单输入controller，额外观察连接不返回terminal_attached。控制权显式接管与递增generation，observer不能PTY resize。终止不使用旧confirm立即kill：controller发起唯一服务器deadline，全查看端弹窗，任一鉴权查看端可取消；发起者失效/控制转移/Web重启取消pending。项目删除仅解绑元数据，shell可cd项目外，真实终端继续且统一入口可重连。上方terminal标签X/单会话trash为此终止请求，下方整面板X只收起。移动宿主不detach/runtime dispose，history默认5000行。具体WS snapshot/单输出源方案仍等待W02/W03真实实验，不以此更新宣称已经实现。

## 1. 范围 / 触发
硬性要求：浏览器关闭、UI tab 隐藏/卸载、网络或 WS 断开、重新登录、Nginx restart、Go Web Server restart 都不能终止 tmux 中任务。普通主机 reboot 不恢复进程内存；README/UI 必须明确。此契约尚未实现或验证。

## 2. 签名
初始 service 边界：`Create(ctx, input) (Terminal, error)`、`List(ctx) ([]Terminal, error)`、`Attach(ctx, id, size) (Bridge, error)`、`Close(ctx, id, confirmed bool) error`。
每 Terminal 一个后端生成的 `persistty_<随机ID>` 独立 tmux session；display name 不作为 tmux target。只操作指定私有 socket 和 Persistty 管理的精确 target，不允许用户传 session/socket/命令选项。
`Close Terminal` 对应 `POST /api/v1/terminals/{id}/close`，body `{"confirm":true}`；普通 detach 仅关闭 WS/PTY/attach client。

## 3. 契约
真实结构：Browser -> WS -> Go -> PTY -> tmux attach-session -> zsh -> job。新任务在 tmux 中创建；attach context 取消只回收 attach client 和 bridge fd/goroutines，禁止 kill-session、kill-server、对 pane 发 exit/SIGTERM、全局杀进程组。主动用户在 shell 输入 exit/任务退出是另一类真实终止原因，重新列举时如实显示。
SQLite 只保存 [metadata](database-guidelines.md)。每次列表/attach 查询 tmux；缺失为 terminated/stale，依赖故障为 unavailable；不能自动复活同 ID 会话。tmux-only 的 managed session 用标识 reconciliation，保留并允许恢复 metadata，不清理用户任务。

### systemd 隔离设计与实测门禁
拟采用同一个非 root UID 的 `persistty-tmux.service` 预启动专属 tmux server，`persistty.service` 仅连接该 socket。tmux server 和 pane job 都必须独立于 Web unit cgroup，但不能假定 pane 与 server 在同一 cgroup；Web unit 仍正常 KillMode=control-group，停止只回收 attach clients。单纯 setsid/nohup/daemonize 不能作为 cgroup 隔离证明。
Spike 确定具体启动命令、前台保持/空 server 保持、socket 目录生命周期、tmux 配置（禁 exit-unattached/意外 exit-empty），并锁定 Debian tmux/systemd 版本。Web 启动检查现有 socket/server；server 不存在时拒绝 Terminal 操作，不允许 tmux CLI 从 Web cgroup 隐式启动 server。
禁止 Web 与 tmux unit 的 PartOf/BindsTo/restart 传播，不能用 Web ExecStop kill-server。独立 tmux unit 的管理员 stop/restart 会终止任务，部署升级只重启 Web；文档须区别。不能默认自动 restart 一个失败的 tmux unit并声称旧进程恢复。

### W02 已观测能力与保留门禁
Debian 13.4/systemd 257.9/tmux 3.5a-3 的隔离 Python PTY 探针已观测：`tmux -D -S <私有socket>` 在用户 transient service 中预启动空 server；只连接使用 `-N`，缺失 socket 返回失败且不生成 server。模拟 Web 的正常停止/重启/SIGKILL 与跨 SSH 采样中，pane PID/start time 不变且心跳继续；只读 `read-only,ignore-size` 客户端没有改变控制端尺寸。可重复命令与证据归 [终端实验](../../../tests/integration/debian/terminal/README.md)。

此版本 pane 实际在独立 `tmux-spawn-UUID.scope`，server 在自身 service，二者都不在 Web cgroup。其来源是 [tmux 3.5a 上游 systemd 实现](https://raw.githubusercontent.com/tmux/tmux/3.5a/compat/systemd.c)，不是 Debian 专有补丁推断。验证和清理须记录自身 server/pane PID/start time/cgroup 及派生 scope，只操作已确认属于本次实验的精确名字，禁止全局通配停止 tmux-spawn scope。管理员停止实验 server 后本次心跳 pane 消失、scope inactive；不能据此保证任意用户任务的终止语义。

后续 D06 已通过真实 Go/PTY/WS 隔离探针：最终二进制经独立 Debian 重跑，Web stop/restart/SIGKILL 后原 server/pane PID、start time、cgroup 不变，13 次心跳逐样本增长，重连不重放输入；精确实验 unit/scope 与目录完成清理。认证/Origin、字节和控制限额、慢客户端队列拒绝及 attach 回收有回归测试，详见 [桥接验证](bridge-validation.md) 和 [独立检查](../../tasks/09-27-debian-spike/bridge-check-report.md)。

这些结果不是正式产品 Go/WS 服务、完整 TUI/history 同步或多设备控制验收。正式实现仍须验证这些路径；普通 sudo 非免密，本实验只临时解包 tmux 及必要 libevent 包，未改变系统安装状态。

### 历史、尺寸、背压
D07 机制实验已观测 raw attach 仅恢复当前画面，capture 后输出超过一屏再 attach 可遗漏间隙行；捕获历史留在 normal buffer 不等于当前 active buffer 可浏览。真实 curses 当前画面恢复与完整 scrollback 恢复必须分开判定。不得以直接拼接、过滤切屏序列或文本前缀去重定案生产方案，详见 [历史/TUI 验证](history-validation.md)。生产同步截点仍未验收。

配置 history_lines=50000、restore_lines=10000；不是无限输出。Spike 验证 capture-pane 与 attach 初始屏幕避免重复/丢失及转义安全的切换策略，之后更新 [WS](websocket-protocol.md) 的交付契约。PTY rows/cols 按连接尺寸同步并限制到 1..1000；多客户端策略初期每 Terminal 一个可写 attach，额外连接返回占用错误，不能 attach -d 无提示抢走另一客户端。
输出队列有界；慢客户端断开可重连，不阻塞/终止 pane job；重连恢复是 tmux 有界历史，不保证每一字节网络精确重放。

## 4. 验证与错误矩阵
| 触发 | 结果 |
| --- | --- |
| WS/page/attach client 消失 | job PID、start time 不变 |
| Web SIGTERM/SIGKILL 或 systemctl restart persistty | job 持续计数，重登录可 attach |
| metadata 有、tmux 无 | terminated，attach 返回 409 terminal_terminated |
| socket/command 故障 | 503 unavailable，禁止创建假状态 |
| 未 confirm 的 close | 400 confirmation_required，不 kill |
| 明确确认 A close | 仅 kill A 精确 session，B/C 持续 |

## 5. 优 / 基础 / 错误用例
优：A 日期计数、B HTTP 服务、C 心跳，反复断开后 PID/start time/心跳增量一致。基础：create/attach/detach。错误：PTY shell 本身持有长期 job、WS cleanup 执行 kill-session、单看 tab 存在就认定持久化通过。

## 6. 必需测试
正式 Terminal UI 前必须完成可重复 Spike：真实 Go + coder/websocket + creack/pty attach，断 WS、关浏览器、终止 Go、重启 Go、重 attach；记录 PID/start time、计数、HTTP 服务可达。真实 Debian systemd 测试检查 /proc/PID/cgroup，执行 systemctl restart persistty 并验证同一个 job；本地非 systemd 环境只能注明未运行，mock/容器无 systemd 不能替代。
Critical Playwright A/B/C 验证关闭页面、重连、Web restart、重新登录、metadata tabs/history 恢复，最后仅 close A。测试隔离 socket/temp cwd，仅清理自己创建的 session；失败保留脱敏证据。此门禁未通过不开始大规模 Terminal UI，不宣布 v0.1 完成。

## 7. 错误与正确
错误：`defer KillSession(id)` 放 Attach 内。正确：Attach 只释放 bridge；KillSession 只有已认证并 confirm 的 Close 路径可调用。
