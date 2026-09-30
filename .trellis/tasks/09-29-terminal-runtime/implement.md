# W03 单代理执行计划

全程按 `codex.dispatch_mode: inline`，不调用子代理或 channel worker。W03 依赖已归档 W01/W02/W04；先读 [PRD](prd.md)、[设计](design.md)、W02 原始报告及 owner 规范。用户已批准实施，任务处于 `in_progress`。

## 有序清单

1. **历史/TUI 门禁**：建立隔离 Linux 控制模式/attach/capture 对照实验，固定 Debian tmux 与 xterm 版本；验证持续输出中的 snapshot/live 顺序、pending UTF-8/CSI/OSC、alternate/current/normal history、resize、Web owner 重建与慢端。保存可复现 fixture、失败样本及脱敏报告。T02 无解时停止并回设计审查，不开放假终端。
2. **协议与规范**：将经证明的同步、epoch/seq/generation、输入一次性、背压、错误码与终止状态写入共享 JSON/WS fixture 和 owner spec；修正 `terminal-lifecycle.md`、`websocket-protocol.md` 里的旧单客户端/立即 close/默认行数候选。先定协议再接 UI。
3. **持久服务**：复用已足够的既有 schema，新增 repository、tmux adapter、固定 socket/`-N`/精确 target、真实状态/孤儿恢复、项目 folder 创建裁决和独立 systemd 模板。假 CLI 与独立 tmux fixture 测创建失败、元数据失败、缺 server、项目解绑，再在 Debian 私有 socket 验原 PID/cgroup。
4. **WS 控制与输出**：按最终设计使用每查看端只读 attach 和一个可写控制 attach、有界 observer 队列、鉴权/Origin/到期撤销、generation 复验的输入/resize/显式接管；三端、慢端、断线、旧帧、Web 重启有并发与 race 测试。输入不持久化、不重发、不记日志。
5. **终止倒计时**：服务器唯一 deadline、发起者/控制失效取消、任意观察端取消、到期精确 kill 与结果复查；API/WS 和应用级 Dialog 同步。可控时钟测双发起、取消边界、取消成功零终止及其他会话不受影响。
6. **前端终端**：替换下方占位和统一 `/terminals`，用已锁定 `@xterm/xterm` 与必要官方 addon，接主/其他 folder 创建、真实状态、宿主切换、手机快捷栏、安全链接。桌面/手机渲染非空且尺寸正确，observer 禁止输入/resize。
7. **全范围验收**：Go test/race/vet、前端 lint/typecheck/test/build、共享 fixture、真实 Debian/systemd 私有测试与桌面/手机浏览器；逐项验证关页/断网/登出/Web restart 原 PID、权限、历史/TUI、倒计时、项目移除。只清理精确自有 socket/session/unit/scope；未执行项如实记录。完成后 trellis-check、update-spec、3.4 提交确认、finish-work。

## 当前进度

- 第 1 步机制实验和真实产品 Debian 探针已通过，证据见 `tests/integration/debian/recovery/`。控制模式拼接方案被 pending UTF-8 反例否定；正式实现采用普通 PTY attach + 独立有界历史视图。
- 第 2–6 步完成 Go 存储、tmux、HTTP/WS、独立 systemd 模板及 React/xterm 界面；共享协议、并发、严格权限与元数据恢复测试通过。runtime 与宿主解耦，上下移动/隐藏/刷新不新增 WS、不自动接管。
- 第 7 步完成，2026-09-30 真实 Debian 恢复/产品探针 40 项检查及完整 9 条浏览器路径通过；三类负载跨正常/SIGKILL Web 重启原 PID/start/cgroup 不变，并验证关页、离线、登出后 TUI 继续交互。最终 Go test/vet/race 与前端 lint/typecheck/49 单测/build 通过，详见 [验收记录](check-report.md)。
- 用户已批准三批工作提交及 W03 归档/日志，不推送；正式部署和 W05 编辑/完整布局不纳入本次完成声明。

## 回滚与门禁

- 正式终端 UI/写接口等待第 1 步通过；W02 D06 成功或 D08 负面结果不等于生产恢复通过。失败时 W03 不归档，不删测试或缩窄 T02 声明。
- attach/视图卸载/登录撤销不调用 kill-session；只有到期裁决可精确终止。Go/Web 重启不自动启动 tmux server 或重跑 shell 命令。
- 远端验证沿用已忽略 `.env` 与受限 runner，不记录连接值；不安装系统 tmux、重启用户真实服务或清理非自有资源，除非另有授权。
- 若技术方案改变历史/TUI 可见语义、多端权限或终止确认，回 PRD/design 重新评审，不能因实现方便悄悄改变承诺。
