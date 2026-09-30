# W03 技术设计审查稿

## 边界与数据流

`Browser xterm -> authenticated WS -> Go terminal service -> tmux client/PTY -> isolated tmux server -> pane shell/job`。tmux/systemd 拥有持久资源，WS/Go attach 是可丢弃通道。SQLite 保存稳定 terminal ID、随机 tmux target、名称、可空 project_id、初始 cwd 和创建时间；列表状态由当前 tmux 观测，不能凭元数据推断 running。W01 `unavailable` DTO 扩为 running/terminated/unavailable、控制与待终止投影；只新增实际所需迁移，不改写 0001–0004。

所有 tmux 调用固定 `-S <私有socket> -N`、精确 target 和参数数组，缺服务即 unavailable，不走 `sh -c` 或默认 socket。创建先验证 W04 当前项目/folder 版本及根身份，再在独立 tmux server 建 session；若 tmux 成功、SQLite 失败，应保留 session 并按受管理标识恢复元数据，不能补偿删除真实任务。项目锁仅保护创建时的配置裁决，不重绑既有任务 cwd。

## 恢复机制先行实验

W02 已证明 raw attach 能恢复当前画面，但普通历史不足；capture 后再 attach 会漏间隙行；公开 xterm serialize 在 parser pending、scroll region、charset 与真实 TUI 截点不等价。因此不把任一方案直接当生产协议。

隔离 tmux 3.5a 实验及脱敏证据见 [恢复实验](../../../../../tests/integration/debian/recovery/README.md) 与 [结果](../../../../../tests/integration/debian/recovery/evidence.json)。控制模式同一流的 `capture-pane`/`%output` 在 400 条连续编号中无重漏，`capture-pane -P` 可以补 pending CSI/OSC，却不能补拆开的 UTF-8 前缀。**因此禁止以控制模式画面快照加原始 `%output` 实施正式恢复**；这不是 T02 的合格方案。

候选方案改为两种不相拼接的视图：当前画面只使用普通 tmux PTY attach 输出，由 tmux 重绘解析状态；活动 xterm 的 scrollback 为 0。普通历史以 `capture-pane -p -e -S - -E -1` 从 tmux 的有界历史读取，在同一终端界面的虚拟滚动视图中**整体替换快照**，绝不追加进活动 xterm。用户回到实时画面直接看持续更新的 attach。该视图切换不能中断 pane、重发输入或制造第二个 session。历史上限在创建 session 时配置；tmux 3.5a 无 `capture-pane -L`，只能使用支持的 `-S/-E` 与 `#{history_size}`。实验中 120 行上限的最后历史编号 371 与实时画面首编号 372 连续，alternate screen 时普通历史仍可读，观察 owner 重建后 TUI 和原 PID/cgroup 不变，xterm headless 5.5.0 单元格检查通过。只读观察 attach 的 `ignore-size` 不改变 pane；可写控制 attach 缩放到 80×24 后 curses 正确重绘并能恢复 100×30。

该结果只通过恢复**机制**的实验门禁，未替代产品集成验收。正式 Web 仍需对持续输出中的历史视图刷新、任意帧分割、resize、慢端、Web restart、输入一次性、浏览器桌面/手机真实画面逐项检查，全部通过才开放生产入口。历史视图必须与同一终端表面结合，不能退化成独立日志页；超过留存量淘汰旧行，用户滚动时保持当前快照，刷新时整体替换以免内容重复。

## 多端控制与终止

每 terminal 一个后端协调器与递增 generation，连接有独立 observer ID。一个 controller 由服务端持有；观察端需显式接管，旧端收到撤销。输入与 resize 每帧在同一串行裁决中复验 generation。**不使用旧设计的一条 owner 输出源广播**：每个查看端拥有独立 `read-only,ignore-size` PTY attach，tmux 为该客户端重绘当前画面；另有一个 Web 进程持有的可写控制 attach 负责经串行授权的输入与 resize，持续排空其输出。每条 WS 输出队列有界，慢端迅速断开并只回收对应 attach，不能阻塞 pane。隔离负例表明完全不读 PTY 的慢 attach 在 5,000 行 burst 后需要有界强制回收该**客户端进程**，但原 pane 继续运行。WS 握手验证 Cookie/Origin，连接期间撤销/到期失效；认证失效不杀 tmux。

终止状态 `{terminal_id,request_id,generation,deadline}` 由服务协调器串行裁决；仅 controller 可发起，任一有效查看端可取消。Web 重启、发起者认证或控制失效取消 pending。到期对精确 target 执行 `kill-session` 后复查；响应丢失只查询真实状态，不自动重建或重发输入。浏览器显示服务端 UTC 截止，不由本地计时决定执行。

## 前端与兼容

在现有 `ProjectWorkbench` 下方面板和 `/terminals` 统一入口接真实 xterm。runtime 按 terminal ID 管连接/控制/输出，React 宿主不拥有 tmux 生命周期；上/下宿主切换不创建第二 session 或重新播放输入。W05 的完整标签组/边缘折叠/布局持久化后续消费同一 runtime。手机快捷项产生明确控制动作，observer 不发送输入/resize。应用级终止 Dialog 独立于下方面板，因此面板收起不遮蔽倒计时。

正式 Debian 模板采用独立同 UID、非 root tmux unit 与私有 socket；Web unit 不持 socket 运行目录，也不通过 `PartOf/BindsTo` 传播 Web restart。实际 tmux/systemd/terminfo、zsh 启动及工具路径以目标机验证；未获安装授权不修改系统服务。旧 W01 terminal 元数据可保持 terminated/unavailable，不复活旧 target。系统重启及管理员停 tmux unit 是终止边界。回滚只停新 API/WS，不 kill 已有 tmux；不能靠 Git 回滚恢复真实进程。
