# Research: 持久终端生命周期与恢复边界

- Query: 根据原需求 U01–U05，研究三个长期终端的生命周期、systemd 隔离、恢复、多客户端及明确关闭的保证范围。
- Scope: mixed；仓库规范、上游官方文档与源码；不运行 Spike、不连接 Debian、不安装依赖。
- Date: 2026-09-26
- 证据等级：用户明确要求 > 用户接受方向 > 仓库初始规范 > 调研建议。本文不是已定案设计，也不是实现完成证明。

## Findings

### 需求与仓库证据

用户故事：我在 Debian 开发机中分别运行 AI、后端和 `npm run dev`，关闭浏览器或经历网络波动后，再打开 Persistty，能找到三个未主动关闭的终端，继续操作原任务。来源为本任务 `research/requirements-source.md` 中 U01–U03。

| 文件 | 说明及具体位置 |
| --- | --- |
| `README.md` | 第 5 行明确无产品实现；第 7 行规定 Web 重启也不得停止任务；第 9 行排除进程内存跨主机重启恢复 |
| `.trellis/spec/backend/terminal-lifecycle.md` | 第 8–13 行要求独立会话、detach/close 分离和真实状态；第 16–18 行为尚未验证的 systemd 设计；第 21–22 行为历史与单写客户端初始约定；第 38–39 行为 Spike 门禁 |
| `.trellis/spec/backend/process-guidelines.md` | 第 3 行隔离任务/连接 context；第 5、13 行限制命令调用和进程组清理 |
| `.trellis/spec/backend/websocket-protocol.md` | 第 10 行禁止自动重放输入；第 25–27 行尚待确定历史切换、单写、重连和背压 |
| `.trellis/spec/frontend/editor-terminal-lifecycle.md` | 第 28–30 行要求可见尺寸、原始 bytes、卸载只 detach、每次恢复一次 |
| `.trellis/spec/backend/index.md`、`.trellis/spec/frontend/index.md` | 第 4 行/第 3 行明确规范是初始契约，依赖版本尚未锁定 |
| `.trellis/spec/guides/index.md` | 第 10、14 行要求证据及中文文档 |
| `.trellis/workflow.md` | 规划与实现分离；研究落盘；用户允许建任务不等于开始实现 |

仓库可沿用的模式是“任务属于长期管理层，连接属于临时 transport，元数据不冒充进程事实”。没有产品源码可供验证。Web 服务重启、具体历史上限、单客户端限制来自已接受架构方向或初始规范，不能把它们全当作用户逐项确认。

### 1. 三种恢复须独立验收

| 恢复对象 | 用户能观察什么 | 不能据此证明什么 |
| --- | --- | --- |
| 列表与标签 | 三个终端的身份、名称、顺序重新出现 | 原进程仍存在 |
| 会话与进程 | 原 shell/任务继续；PID、启动时间和递增心跳一致 | 所有历史 bytes 可恢复 |
| 画面与历史 | 当前 TUI 可继续操作；普通输出能回看有界历史 | exactly-once 输出、原始网络重放或无限日志归档 |

tmux 将 client 与 server 分开，session 可在脱离 client 后继续；`attach` 连接既有会话。此能力支持核心方向。[tmux 官方手册](https://raw.githubusercontent.com/tmux/tmux/master/tmux.1)

设计推论：生命周期应为 `Browser → WS → Go → attach PTY → tmux client`，长期 `tmux server → pane PTY → shell/job` 不由 Go 请求/服务 context 拥有。每个终端使用独立随机身份与精确 target，禁止 detach 路径触发 session 删除。关闭浏览器、隐藏标签、登出/认证撤销，只撤销控制连接；任务继续。

### 2. systemd 与 socket 的归属

官方事实：systemd 默认 `KillMode=control-group`，停止服务会清理该 unit cgroup 内进程。`setsid`、`nohup` 或 fork 不构成独立 unit 的证据。[systemd.kill](https://raw.githubusercontent.com/systemd/systemd/main/man/systemd.kill.xml)

建议保持 `persistty.service` 的正常清理行为，另由同一固定非 root UID 的 `persistty-tmux.service` 拥有 tmux server 和任务。Web 的 attach clients 仍归 Web unit，停止 Web 清理它们即可。不要通过 `KillMode=none/process` 隐藏归属错误，也不要以外部看见端口可访问代替 `/proc/<pid>/cgroup` 验证。

socket 目录应由 tmux unit 持有。`RuntimeDirectory=` 默认随所属服务 stop 删除，`RuntimeDirectoryPreserve=restart/yes` 只控制目录保留，不能恢复进程；系统 `/run` 会在重启后重新建立。该选项在上游标注自 systemd 235 起提供。[systemd.exec](https://raw.githubusercontent.com/systemd/systemd/main/man/systemd.exec.xml)

设计推论：若 socket 目录由 Web unit 拥有，重启 Web 可能删除仍运行 server 的入口，任务存活但无法连接。两个 unit 不宜共享需要相互清理的同一 `RuntimeDirectory`。固定 UID、目录权限、路径一致性及服务可见性都须实测；“私有 socket”只是隔离其他 tmux 实例，并非隔离同 UID 的任意 shell 命令。

`PartOf`/`BindsTo` 的依赖方向或部署脚本可能把 Web restart 传播为 tmux stop。验收应检查实际 unit 依赖与发布命令，允许必要启动排序但不能把排序当健康/就绪证明。[systemd.unit](https://raw.githubusercontent.com/systemd/systemd/main/man/systemd.unit.xml) `Type=simple/exec` 的进程启动也不等于 socket 可用。[systemd.service](https://raw.githubusercontent.com/systemd/systemd/main/man/systemd.service.xml)

### 3. 隐式启动 race、空 server 与 keeper

官方源码事实：tmux client 在连接遇到 `ENOENT/ECONNREFUSED` 时可进入 server 启动路径；`CLIENT_NOSTARTSERVER` 在该位置拒绝启动。因此“先检查 socket，再执行命令”仍有 server 在两步之间退出的竞争窗口。[tmux client.c](https://raw.githubusercontent.com/tmux/tmux/master/client.c)

候选解决：所有来自 Web 的 tmux 调用都带全局 `-N` 与固定 `-S`，包括创建、attach、list、close 和检查；缺失 server 时返回 unavailable，不从 Web cgroup 创建替代 server。tmux 3.3a 官方手册已含 `-N`；`-D` 为前台 server、关闭 `exit-empty`，且不能同时指定 command。[tmux 3.3a 手册](https://raw.githubusercontent.com/tmux/tmux/3.3a/tmux.1)

版本事实：官方 CHANGES 在 3.1c→3.2 段记录 `-N`、`-D` 与 `ignore-size`/read-only 分离。若采用上述机制，tmux 3.2 是能力下界候选，3.3a 是本次明确核对的历史版本；不等于已选 Debian 版本，也不保证 Debian 补丁包行为。不能照抄 master 新增参数到旧系统。[tmux CHANGES](https://raw.githubusercontent.com/tmux/tmux/master/CHANGES)

官方 server loop 会结合 `exit-empty`、`exit-unattached` 和会话/客户端状态判断退出；前台启动路径关闭 `exit-empty`。[tmux server.c](https://raw.githubusercontent.com/tmux/tmux/master/server.c)

设计推论：优先验证“独立前台 server + 空 server 保持”，不必预设隐藏 keeper session。单独 `start-server` 不能证明长期空 server 存活。若因版本/管理方式采用 keeper，必须明确其不进入用户列表、不会被清理最后一个用户终端时顺带删除、不运行用户 shell startup 副作用；同时记录 server 真实 PID/失败，而非依赖 oneshot + `RemainAfterExit` 的 active 字样。`Restart=always` 重建空 server只恢复服务能力，不会使旧任务继续。

### 4. 画面、alternate screen 与滚动历史

官方源码事实：pane 存有虚拟屏幕状态，attach 时可重新显示；pane 删除会释放屏幕和 PTY。[tmux 3.3a window.c](https://raw.githubusercontent.com/tmux/tmux/3.3a/window.c)

`capture-pane` 捕获已解释后的屏幕/历史内容；`-a` 对 alternate screen 捕获且不能访问历史，`-e` 是文本属性转义而不是原始输出录像。默认捕获可见区域，负行号可指向历史。[tmux 官方手册](https://raw.githubusercontent.com/tmux/tmux/master/tmux.1)

捕获实现通过 grid cells 生成字符串，也印证它不是原始 PTY 流。不能从该结果重建完整终端模式、所有控制序列、先前光标运动或应用内部状态。[tmux cmd-capture-pane.c](https://raw.githubusercontent.com/tmux/tmux/master/cmd-capture-pane.c)

产品语义建议：重连必须先恢复“现在能正确操作的画面”，历史作为有界回看能力，TUI alternate screen 不承诺普通日志式历史。可评估 attach 负责实时主画面、单独历史查看，或经实验锁定的一次性主缓冲预填策略；后者需证明当前屏幕不重复、TUI 模式不污染、重连中输出/resize 不乱序。不能简单 `capture-pane` 全部输出后再直接拼 attach 全屏重绘，也不能在重连时发 shell `clear` 来掩盖问题。

初始 50,000/10,000 行是仓库建议值，不是用户确认的留存承诺。应分开配置 tmux 留存、重连回看、浏览器 scrollback；行数也不是固定内存上限。超出保留范围可明确截断；需要完整审计日志时属于另一项需求。

### 5. 背压与输入不确定性

官方事实：xterm `write` 异步缓冲；高速生产者可能令缓冲增长/丢弃，文档提出基于处理回调的流控，跨 WS 需要端到端考虑。[xterm.js Flowcontrol](https://xtermjs.org/docs/guides/flowcontrol/)

设计推论：仅“Go 队列有界”不足以证明浏览器不积压，需观测 xterm 消费进度、WS 缓冲与内存。消费者持续落后时 detach 此连接，避免无限暂停长期 pane 任务；无浏览器时日志生成/HTTP 服务仍应继续。tmux 对慢 client 的表现仍需版本实测，不能凭文档保证业务毫无暂停。

键盘输入与输出不能共用重放语义。掉线时某条输入可能已到达 shell，只是反馈未到达；页面不自动重发，也不声称未执行。用户重连查看状态后决定后续输入。终端 `Ctrl-C` 属于用户向当前应用发送控制输入，既不是关闭标签，也不是 Persistty 关闭整个 session。

### 6. 多客户端、尺寸与写权限

能力事实：tmux 支持多个 client；read-only 和 ignore-size 可控制写入/尺寸影响，窗口大小可由 largest/smallest/latest/manual 等策略决定。[tmux 官方手册](https://raw.githubusercontent.com/tmux/tmux/master/tmux.1)

产品决策尚未确认：初始规范每 terminal 一个可写连接、第二个拒绝占用错误，是合理首版候选；用户是否期望桌面/手机并行访问仍未知。另可选择“一写多只读”，只读应同时不参与尺寸，避免手机查看改变桌面 AI TUI 布局。不能静默 `attach -d` 抢走原连接。

一个 pane 只有一套真实尺寸，并非每客户端一份布局。客户端先 fit 再发送非零 rows/cols，隐藏视图不发 0×0；缩放、字体加载、移动设备方向变化都需重算。若一写多只读，查看者显示空白边缘或 viewport 裁切是可预期取舍，须明确。外部 SSH tmux attach 也是客户端，Web 内互斥本身不能约束它。

单写占用是连接租约而非终端生命周期：掉线且旧 WS 未被识别时，新连接可能暂时占用失败。必须有明确 heartbeat/deadline、释放和手动恢复机制，验证 server restart 后无永远占用，也不能为了重连删除 session。若 close 与 attach 同时发生，close 应撤销租约/连接且阻止旧重连逻辑继续送入输入。

### 7. 明确关闭的保证范围

zsh 文档说明：退出时是否给 job 发 HUP 受 HUP 选项影响，`nohup`/`disown` 可以避免 shell 对运行 job 的终止行为。[zsh Jobs & Signals](https://zsh.sourceforge.io/Doc/Release/Jobs-_0026-Signals.html)

由 PTY/会话清理机制推论：`kill-session` 可以保证该 tmux session 消失，不保证用户在其内创建的任意脱离进程、忽略 HUP 的 job、daemon、另建 systemd service 或外部任务全部停止。后台 `&` 也不应一概视为“关闭必杀”，具体取决于 job group、信号与文件描述符。AI CLI 发起的远程任务更不归本地 session 清理。

推荐将两个承诺分开：a. 明确关闭后该终端不能再连接，不自动复活；b. 若首版承诺 AI/Go/npm 前台任务也停止，必须逐项实验并说明允许的退出等待时间/行为。若要“所有本地后代都强制终止且仅影响 A”，需要每终端独立进程归属/监督方案，属于额外设计；不能杀共享 tmux unit cgroup，否则 B/C 也停止。即便独立 cgroup，也不能承诺终止已移交其他管理员服务或远端系统的工作。

API 确认关闭与页面关闭应分开；关闭响应丢失后，重新查询真实状态，不自动重新创建同 ID。没有合适隔离证据时，文案应表达“终止终端会话”，不要表达“全部后台进程已清理”。

### 8. zsh、PATH、locale、TERM 与 Oh My Zsh

zsh 按 interactive/login 条件读取不同启动文件，interactive 会读取 `.zshrc`，login 还会读 profile/login 文件；安装路径和 `ZDOTDIR` 影响配置位置。[zsh Files](https://zsh.sourceforge.io/Doc/Release/Files.html)

tmux 3.3a spawn 源码显示：无命令时启动 login shell，单个命令字符串走 shell `-c`，多参数使用 exec；它还可能从未 attach 的创建 client 覆盖 pane PATH。因此只配置 server 环境不足以保证每个新终端环境。[tmux 3.3a spawn.c](https://raw.githubusercontent.com/tmux/tmux/3.3a/spawn.c)

设计推论：真实部署需验证非 root UID 的 HOME、SHELL、工作目录、UTF-8 locale、合法 terminfo 和初始化后的 PATH；不得把服务器调用 git/rg 的受限 env 白名单直接套入用户交互 shell，也不得借创建终端绕过既定 env 来源。node/npm/AI CLI 可能依赖 nvm/fnm/asdf、用户目录与 shell 初始化，能在 SSH 启动不代表 systemd 条件下相同。

外侧 attach PTY 的 TERM 描述浏览器终端能力，内侧 pane TERM 描述 tmux 能力；不要一律覆盖成 `xterm-256color`。是否使用 `tmux-256color` 取决于实际 terminfo 是否存在及应用测试。`-u` 只影响 tmux 编码判断，不能修复不存在的系统 locale 或字形宽度不一致。

Oh My Zsh 使用 `.zshrc` 配置。[Oh My Zsh FAQ](https://github.com/ohmyzsh/ohmyzsh/wiki/FAQ) 兼容性推论：主题、插件可能有更新检查、启动输出或快捷键影响。实测用户现有配置；故障时给出可定位原因，不静默退回干净 shell再声称兼容。Nerd Font 能覆盖图标仍不保证中英文/组合字符/emoji 的 cell width 一致，需在真实 xterm 与 tmux 下看截图和光标位置。此处不锁定 codex/opencode/pi 的版本或行为。

### 9. 失败与恢复语义建议

| 触发 | 可见状态与恢复动作 | 不应执行 |
| --- | --- | --- |
| WS/页面消失、认证过期 | 控制断开；重新认证、查询真状态、attach | kill-session、自动输入重放 |
| Web 重启 | 服务恢复后重登录/连接；原任务继续 | 启动替代任务冒充原 PID |
| 元数据有、session 已无 | 已终止；用户可创建全新终端 | 自动复活同 ID |
| session 有、元数据未落盘 | 用管理标识恢复列表；保留原任务 | 清理“孤儿”任务 |
| socket 不可达/server 状态不明 | unavailable；恢复管理入口后重查 | 把依赖故障标成 terminated |
| socket 被删而 server 存活 | 管理端查证 PID/目录后修复入口，属操作手册 | Web 重启 tmux、清空所有 session |
| 最后一个终端显式关闭 | server 可继续健康，下一次创建正常 | keeper 误入列表或隐式 server 重建 |
| 管理员停止 tmux unit/主机 reboot | 原任务真实终止；启动服务只恢复创建能力 | 声称进程内存恢复 |
| 慢客户端/高输出 | 可见断开，后台继续，重新连接得到有限恢复 | 无限缓冲或阻塞所有任务 |

### 10. A/B/C 验收矩阵（后续实验，未执行）

基线应先用确定性 fixture，再复测真实工具。A：交互 AI/TUI（先用确定性 TUI，后选实际 codex/opencode/pi）；B：前台 HTTP 服务；C：`npm run dev` 与递增心跳。另用 shell 计数 fixture 保证不依赖工具自身偶然行为。记录 shell PID、前台实际任务 PID、启动时间、server PID、cgroup、计数/HTTP 身份与终端 ID；仅保留标签或端口能访问不足以通过。

| 步骤 | A | B | C | 共通断言 |
| --- | --- | --- | --- | --- |
| 创建三会话 | TUI/输入可用 | HTTP 身份正确 | dev 服务/心跳递增 | 独立身份，工作目录正确 |
| 隐藏/切换标签、组件卸载 | 仍可重新操作 | 请求继续 | 心跳继续 | 零 close 请求，尺寸非零 |
| 断网、浏览器关闭后重开 | 当前画面可操作 | 原实例响应 | 原任务继续 | 原 PID+start time，一次历史恢复 |
| 网络抖动+输入在途 | 用户检查实际命令结果 | 同上 | 同上 | 不重放输入，不假称未执行 |
| Nginx restart | 重连后继续 | 持续可观测 | 持续可观测 | 不影响 tmux/pane 生命周期 |
| Web SIGTERM/SIGKILL/restart | 同一 TUI | 同一 HTTP 实例 | 同一心跳/服务 | tmux/job 独立 cgroup，bridge 回收 |
| 退出登录/会话撤销 | 连接撤销，重新登录可连 | 任务继续 | 任务继续 | 撤销控制权而不终止任务 |
| 高输出+慢/冻结浏览器 | 可重连 | 无被拖慢证据 | 心跳继续 | 浏览器/Go内存有界，长任务不被 kill |
| 桌面+手机同时连接 | 按确定策略拒绝/只读 | 不变 | 不变 | 只读输入无效且不改尺寸；不抢连接 |
| 关闭请求未确认/响应丢失 | 无确认不删；确认后查询真状态 | 持续 | 持续 | 只影响 A，无同 ID 复活 |
| 明确关闭 A | session 消失；前台退出按定案语义验证 | PID/start time 不变 | PID/start time 不变 | 独立目标；另测后台/daemon 边界 |
| 最后用户 session 关闭，再创建 | 新会话全新身份 | — | — | server 健康，不从 Web unit 隐式启动 |
| server 在检查后退出 | unavailable | — | — | -N 防隐式启动，元数据不得伪造成功 |
| tmux unit stop/reboot | 已终止 | 已终止 | 已终止 | 负向边界；不算持久核心通过项 |

## 待澄清（本轮不提问）

1. “关闭终端”表示关闭视图、detach、终止会话，还是连已脱离终端的后台任务也须杀死？是否另设隐藏/恢复终端列表？
2. 多设备是否需要同时查看/控制？单写拒绝、显式接管、一写多只读的优先级，以及 SSH 外部 attach 是否纳入支持范围？
3. 重连需哪些历史：普通日志最近 N 行、完整日志下载、TUI 当前屏幕，还是应用自身历史？完整日志不是 tmux 现有历史的同义词。
4. AI 终端需要同时兼容 codex/opencode/pi 哪些版本、常用主题/插件和输入方式？三者都为首版验收还是选一个主流程？
5. 接受的 Web 更新中断控制连接时间、假连接占用等待时间、历史截断提示与管理员恢复职责是什么？
6. 实际 Debian/tmux/systemd、浏览器/移动设备、UID、shell 初始化、开发工具安装路径、用户级或系统级 unit 环境是什么？
7. 主机重启后是否仅恢复标签并显示已终止？自动重跑命令可能产生副作用，不属于内存恢复，需独立确认。

## 待实验（后续授权的 Spike）

- 锁定实际版本与 unit 方式；验证 `-D/-N`、空 server、私有配置、`exit-empty/exit-unattached/destroy-unattached`、最后会话清理；记录就绪与错误分流。
- 验证 tmux server/pane PID 全在 tmux unit cgroup，attach client 在 Web unit；重复 Web stop/restart、SIGKILL；检查依赖传播和 RuntimeDirectory 删除路径。
- 主动制造 socket/server 检查与命令执行间 race；保证没有 Web cgroup 新 server、没有假成功元数据；验证 socket 丢失/故障恢复。
- capture 与 attach 在持续输出/resize 下的切换；普通屏幕、alternate screen、鼠标、bracketed paste、Ctrl/Alt/功能键、宽字符、颜色、TUI退出后返回 shell、重复重连。
- 端到端背压、冻结浏览器、高输出、断开后计数/HTTP继续、内存有界；不得只靠 server 队列长度。
- 并发连接/半开 WS/租约释放、外部 attach、close/attach race；只读既不写也不改变工作 pane 尺寸。
- foreground、shell `&`、忽略 HUP、nohup/disown、fork/daemon、已移交其他 service 分组实验；只清理自己的隔离 fixture，验证 B/C 无影响。
- 真正用户 zsh/Oh My Zsh 启动、PATH 初始化、HOME、locale、TERM/terminfo、AI CLI 登录状态与 npm 工具链；不读取或记录敏感凭证。

## Caveats / Not Found

- 没有真实 Debian/浏览器/库版本，没有产品代码；所有集成测试均未执行。上游 master 是调研时点能力，不能代替部署版本手册。
- 3.3a 源码中的 PATH 行为提示必须实测各候选版本，不能把“同一个 UID”当成“和 SSH 环境完全一致”。
- systemd 在线 HTML 页访问失败，本次改读官方仓库 XML；没有推定 Debian 默认配置。
- 有界屏幕/历史恢复不能保证所有离线输出、所有 TUI 模式和原始 bytes 的精确复现；方案须 Spike 定案。
- 共用一个 tmux server 不足以证明“关闭 A 强制清除全部后台后代且 B/C 不动”。若用户要求该语义，应重新评估每终端的进程监督边界。
- 本文只写本任务研究文件，未修改现有规范、代码或任务状态。
