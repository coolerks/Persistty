# 初始契约证据（2026-09-26）

## 本地证据
- AGENTS.md 规定 .trellis/workflow.md/spec/tasks 为开发入口。
- README.md 只有 Persistty 标题；LICENSE 为 MIT；git ls-files 仅两文件。
- bootstrap task 已为 in_progress；所有 backend/frontend 文档仍是模板。
- 本次用户需求是初始规则来源；未来路径和示例不是现有实现。不能要求凭空找三处源码模式。

## 外部一手证据与判断
- [systemd.kill 上游手册](https://raw.githubusercontent.com/systemd/systemd/main/man/systemd.kill.xml)：默认停止策略清理整个 unit cgroup。推论：tmux 后台化本身不足以隔离 Web restart。必须在独立 unit 中预启动 tmux 并实测 PID/cgroup，不能宣称已通过。
- [Go traversal-resistant APIs](https://go.dev/blog/osroot)：先解析 symlink 再按路径打开仍存在竞态。采用根句柄限定 I/O；[os.Root API](https://pkg.go.dev/os#Root) 的版本/rename 能力由 foundation 锁定，或封装 Linux dirfd 操作。
- [Linux openat2 手册](https://man7.org/linux/man-pages/man2/openat2.2.html)：BENEATH/NO_MAGICLINKS 可约束解析，不能仅在字符串层判断。写操作须持有安全父目录句柄。
- [tmux 上游手册](https://raw.githubusercontent.com/tmux/tmux/master/tmux.1)：Spike 必须验证独立 socket、attach 客户端清理和 history 行为；应用不能让 attach 命令自动启动一个新 server 来掩盖故障。
- [Material Icon Theme LICENSE](https://github.com/material-extensions/vscode-material-icon-theme/blob/main/LICENSE) 与 [JetBrainsMono Nerd Fonts 清单](https://github.com/ryanoasis/nerd-fonts/blob/master/patched-fonts/JetBrainsMono/README.md)：本任务不导入资源；实际资产任务须锁定版本并逐项核查字体/补丁/图标许可和分发声明，不能把项目 MIT 当第三方许可。

## 待实际任务证明
Debian/systemd 隔离、history/live 切换、父目录移动/替换竞态、外部 writer 与 rename 间隙、PTY goroutine 回收、最终资产 glyph 覆盖。全部属于后续 acceptance，不是 bootstrap 已完成的运行时证据。

## 全范围审查补充
浏览器原生 WS 不暴露握手 HTTP status；规范改为 session probe/已连接 close 1008/有限未知故障重试，不要求读取不存在的浏览器状态。rg/Git 事后过滤不能防止进程先读根外内容；命令规范要求 task 实测受限执行或安全 snapshot 输入方案，不宣称单凭 cwd/结果校验解决竞态。
