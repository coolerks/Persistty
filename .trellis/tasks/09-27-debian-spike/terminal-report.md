# W02 终端隔离实验报告

## 结论

2026-09-27 在用户指定 Debian 13.4 amd64 机器，以非 root 用户执行隔离实验。tmux 3.5a-3 和必要 libevent-core-2.1-7t64 2.1.13-stable-1~deb13u1 从该机器配置的软件源下载解包到临时目录，未安装系统软件。systemd 为 257.9-1~deb13u1。临时目录 0700，socket 0600，未使用默认 socket，未修改 Nginx、WireGuard、防火墙或已有服务。

终端探针提供 W02 部分 D01/D02/D03/D05 证据，不代表真实 Go/WS/浏览器/TUI 终端交付验收。详见 [可重复步骤](../../../tests/integration/debian/terminal/README.md) 与 [脱敏实测 JSON](../../../tests/integration/debian/terminal/evidence.json)。连接身份及 cgroup 中的用户标识已去敏，实验关联与判定保留，不称为原始逐字记录。

## 实测结果

最终实验根 `/tmp/persistty-terminal-1lRwXVuk`，服务前缀 `persistty-spike-d7e4b828b7e6`，全部已清理。

| 场景 | 结果 |
| --- | --- |
| 私有 foreground server 空 session | exit-empty off，持续存活 |
| 缺 socket，-N new-session | 非零退出，无新 socket |
| 初始任务 | server PID 855260/start ticks 914317234，pane PID 855266/start ticks 914317355 |
| 模拟 Web attach | client 在 Web unit cgroup，server/pane 不在 Web cgroup |
| Web stop，KillMode=control-group | attach 消失，原任务身份不变，心跳 4→7 |
| 新 Web 实例与 systemctl restart | 原任务身份不变，心跳到 12 |
| Web SIGKILL | attach 消失，原任务身份不变，心跳到 15 |
| 两客户端，观察端 read-only/ignore-size | 观察 PTY 40×10，pane 仍 100×30，心跳到 19 |
| 所有 attach 消失 | 身份不变，心跳到 22，capture-pane 可见最近心跳 |
| start SSH 正常退出后新 SSH check | 同一 PID/start ticks/cgroup，心跳增长至 95 |
| 管理员 stop 实验 server | 此 heartbeat pane 消失，其派生 scope inactive；不承诺进程恢复 |

## cgroup 差异

server 位于自己的 `.../app.slice/persistty-spike-d7e4b828b7e6-tmux.service`；pane 位于 `.../app.slice/tmux-spawn-e405ecf5-1994-49ec-875e-a74d85fc1e73.scope`。这是 [tmux 3.5a 上游 systemd 实现](https://raw.githubusercontent.com/tmux/tmux/3.5a/compat/systemd.c) 中创建随机 transient scope 的行为，不能未经证明称为 Debian 补丁，也不能继续要求 pane 必然在 server unit 内。部署必须验证各实际 cgroup 独立于 Web，而非仅根据 daemon/setsid 推断。

[该版本官方手册](https://raw.githubusercontent.com/tmux/tmux/3.5a/tmux.1) 说明 -D、-N 和 read-only/ignore-size；[systemd v257 源手册](https://raw.githubusercontent.com/systemd/systemd/v257/man/systemd-run.xml) 说明 transient unit 命令。参数语义由这些资料核实，判定以上述目标机实测为准。

## 探针修正与清理

试运行发现缺 libevent（降权解包必要依赖补齐）、空 server 的 list-sessions 返回 0 且空输出、无 client 的 display-message 输出为空、pane 处于独立 scope、capture-pane 需要实际 pane ID。以上探针假设已修正，未伪记失败轮次通过。

各失败根 GZBDmlkX、jznDgrbt、yGBRbxLQ、1MvhPcVJ、WQxUkuin、nTb5NjRm 与最终 1lRwXVuk 均已确认不存在；每轮自身单位 inactive。最终 scope inactive，pane/server 原进程不存在。没有保留长期实验进程、socket、解包库或软件包。只读查询此前明确观察的 `tmux-spawn-260a9061-b38f-47d2-a019-7780eed9b99b.scope` 也为 inactive，未泛匹配清理其他 scope。

检查：本地 `python3 -m py_compile` 通过；最终远端 start、独立 SSH check、cleanup 均退出 0。尚待 check 代理复核探针安全边界与重跑。

## 未通过的产品门禁

未验证真实 Go+creack/pty+coder/websocket、浏览器关闭/鉴权重登录、Nginx/Web 产品 unit 重启、A/B/C 多任务隔离、输入 controller generation、取消终止倒计时、背压、快照切换不重复/不丢失、交互 TUI 和历史虚拟滚动。模拟 Web 使用 Python 原生 PTY，仅证明系统层隔离可行。管理员 stop 的结果只覆盖此 heartbeat，忽略 SIGHUP 或脱离终端的子进程需独立验证；主机 reboot 不恢复进程。
