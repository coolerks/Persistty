# Research: D07 历史恢复与 TUI 最小实验

- Query: tmux capture-pane 与真实 attach 初屏如何避免重复、遗漏和 ANSI 状态破坏；本轮能验证什么，不能放行什么？
- Scope: mixed；仓库只读检查、tmux/xterm 官方文档与固定 tag 源码，不执行远端命令。
- Date: 2026-09-27

## Findings

### 当前文件与代码模式

| 文件 | 作用与证据 |
| --- | --- |
| `web/package.json:6`、`web/package-lock.json:38` | npm 11.17.0 / Node 24 范围；Vitest 5.0.2 已锁定。未安装 xterm 或 Playwright；lock 中 Vitest 可选 peer 描述不代表安装了浏览器 runner。 |
| `tests/integration/debian/bridgego/server.go:118` | 固定参数 `tmux -N -S ... attach-session -t =...`，只是 attach；初始 PTY 固定 100x30（124 行）。 |
| `tests/integration/debian/bridgego/server.go:215` | ready 先发出，不能作为收到 tmux 全屏重绘或完成解析的边界。 |
| `tests/integration/debian/bridgego/server.go:230` | 1024-byte 任意分块 PTY 输出；无法按每帧字符串处理 ANSI 或 UTF-8。 |
| `tests/integration/debian/bridgego/client.go:101` | observe/hold 只接收；123 行只证明存在 binary bytes，未证明可读屏幕/TUI。 |
| `tests/integration/debian/bridgego/client.go:195` | 当前 ACK 搜索针对受控 fixture 原字节，不是终端解析器，也不能用它断言最终屏幕。 |
| `tests/integration/debian/run_remote.py:62`、`remote_config.py:69` | 已有私有 .env 安全传输与精确 cleanup 入口，扩展实验应复用，而非额外拼 SSH 配置。 |
| `tests/integration/debian/terminal/probe.py:179` | 早期 capture 仅证明 capture 能读取，不覆盖 snapshot/live 合并。 |

### 官方依据与风险

1. **capture 是 grid 的格式化导出，不是完整终端状态。** [tmux 3.5a man capture-pane](https://raw.githubusercontent.com/tmux/tmux/3.5a/tmux.1) 2503-2553 行：负数是 history，0 是可见首行；`-e` 增加文字属性，`-J` 合并 wrapped 行，`-P` 只导出尚未完成的转义前缀。这些开关都不构成 cursor/modes/live sequence 的同步协议。`-C` 是八进制展示转义，不能直接作为原始 ANSI 恢复流。
2. **`-a` 不等于当前 TUI 帧。** [cmd-capture-pane.c](https://raw.githubusercontent.com/tmux/tmux/3.5a/cmd-capture-pane.c) 109-120 行选择 `saved_grid`，默认选择当前 `grid`；169-177 行按 grid cell 导出并插入行尾。结合 [screen.c](https://raw.githubusercontent.com/tmux/tmux/3.5a/screen.c) 573-642 行，alternate 进入时保存进入前的可见普通屏幕，当前 grid 清空且不累计 history。因此必须同时留 `alternate_on` 判定与两种 capture 的受控对照，不能仅凭 man 的“alternate screen”描述推测内容。
3. **attach 有额外的外层终端语义。** [tty.c](https://raw.githubusercontent.com/tmux/tmux/3.5a/tty.c) 293-357 行在 attach 的外部终端发送 SMCUP、CLEAR、键盘模式、设备查询；[screen-redraw.c](https://raw.githubusercontent.com/tmux/tmux/3.5a/screen-redraw.c) 538-572、763-810 行随后绘制当前 pane grid。推论：直接 `capture(history+screen) + attach bytes` 不是“一次历史 + 原样后续输出”，会遇到重复屏幕、外层 alternate buffer、清屏及遗漏衔接期间滚动行。原任务的 TUI 切屏与 tmux 外层切屏不能混为一谈。
4. **用库判定状态，不用正则剥 ANSI。** [xterm Terminal.write API](https://xtermjs.org/docs/api/terminal/classes/terminal/#write) 支持 Uint8Array，解析异步，buffer 断言必须等待 write callback；保留分块，检查 normal/alternate buffer、cell、cursor、modes。网站 API 不是固定版本证明，实际安装后还需以锁定包 typings 复核。
5. **序列化只能序列化已经正确接收的状态。** [xterm 5.5.0 package](https://raw.githubusercontent.com/xtermjs/xterm.js/5.5.0/package.json) 确认 `@xterm/xterm` 5.5.0 / MIT；同 tag [addon-serialize package](https://raw.githubusercontent.com/xtermjs/xterm.js/5.5.0/addons/addon-serialize/package.json) 为 0.13.0 / MIT。[其 typings](https://raw.githubusercontent.com/xtermjs/xterm.js/5.5.0/addons/addon-serialize/typings/addon-serialize.d.ts) 提供 scrollback、alt buffer、modes 选项，并建议同尺寸恢复后再 resize；[README](https://raw.githubusercontent.com/xtermjs/xterm.js/5.5.0/addons/addon-serialize/README.md) 明示实验性。这可用于受控恢复对照，不自动解决 capture 与 attach 的竞态，也不构成生产服务引入 Node sidecar 的决定。
6. **安全边界留在宿主。** [xterm 官方 security](https://xtermjs.org/docs/guides/security/) 指出终端输出、标题、链接、buffer 等都是不可信数据，不使用 innerHTML/eval，不能直接照搬演示 attach addon 的认证。本实验不启用链接、clipboard、title DOM 更新、自定义 OSC handler，不执行任意输入；原始终端 fixture 不应包含凭据或业务文本。
7. **headless 引擎已确认，发布包版本仍需安装前核实。** [5.5.0 tag 的 headless 公共 Terminal 源码](https://raw.githubusercontent.com/xtermjs/xterm.js/5.5.0/src/headless/public/Terminal.ts) 88-93 行的 buffer 是 proposed API，须显式 `allowProposedApi: true`；131-133、167-168 行提供 resize / byte write callback。[对应 typings](https://raw.githubusercontent.com/xtermjs/xterm.js/5.5.0/typings/xterm-headless.d.ts) 声明 `@xterm/headless` / MIT。可以在隔离 npm 模块采用 headless 并以 Node 自带 test runner 做解析，无需 Vitest/Playwright；尚未确认 npm 发布 metadata，不能把源码 tag 自动当发布包版本锁定。使用 installed manifest/lock/许可证核实后再记录实际版本。

### 本轮最小可执行方案

建议将 D07 定为“恢复机制对照与失败边界”，不在本轮注册产品路由或冻结 snapshot/live 协议。

1. 隔离目录建立独立 npm manifest/lock，使用 npm，与 `web/` 产品 manifest 分离。固定候选 `@xterm/xterm@5.5.0`、可选 `@xterm/addon-serialize@0.13.0`；这是已核实 tag 的候选而非最新版本承诺。若需 headless，实施者先查实际官方包 manifest、版本、API/许可证及安装兼容性再锁定；本研究未确认 headless 包路径，不能凭记忆填写版本。也可直接用本地浏览器 xterm 公共 buffer API 完成全部解析对照，减少依赖。
2. 远端沿用 D06 的新 mktemp 0700 ROOT、临时 tmux/libevent 解包、随机私有 socket、独立 tmux/Web unit 和原 PID/start/cgroup 采样。仅使用 Remote 读取私有 .env，不记录值；复用精确清理，不碰现有服务。受控 workload 有固定 ANSI fixture 和场景阶段文件，阶段文件/固定脚本驱动程序输出，无需 shell keyboard 命令或重新执行 job。
3. 为实验 client 增加**限时、限字节**的 binary 帧记录与显式 resize 操作；只记录自身固定合成 fixture，不记录 token、shell 输入或环境。输出记录以 JSON 中 base64 帧 + 有界尺寸事件表示，完整原字节只保存为忽略的短期 fixture；提交证据为摘要、SHA256、计数、判定、截图中的固定文本。不要将输入和输出按采样时间猜成严格总序；write callback 只表示 xterm 解析完成。
4. 准备三条独立对照：A 仅 raw attach；B capture 后直接追加 raw attach（预期暴露限制的负面实验）；C 在同一个 xterm 连续输出源上取 serialize 快照再恢复并接后续帧（仅验证序列化/有界客户端序号模型）。C 的 source 序号与 snapshot 截点必须由同一串行 owner 生成，不能跨两个 SSH 命令的墙钟估计。若本轮尚未实现该 owner，就将 C 限于本地确定性 fixture，明确不是真实 tmux 原子恢复验收。
5. 本地 loopback 静态实验页加载锁定的 xterm/CSS，展示同一真实 Debian 记录的回放与 buffer 断言；不直接让浏览器连现有 Bearer WS：浏览器原生 WebSocket 无法设置 Authorization，当前 Origin 也固定不含端口，不能用 URL token 或放松认证临时绕过。浏览器回放不是实时远端浏览器端到端测试，两者必须分开记录。
6. 使用本地可用浏览器工具运行截图与断言；若 Playwright 不在仓库，新增独立锁定浏览器工具须先确认实际版本/浏览器可用性，不能将 Vitest optional peer 当安装成功。截图确认 ANSI 颜色、中文宽字符、光标与网格不空白；headless/jsdom 只能证解析不能证渲染。严格等待 xterm write callback 和 document.fonts.ready；跨相同尺寸比较 buffer cell，不能只比较截图 OCR。

### 场景与判定矩阵

| 场景 | 固定 fixture / 操作 | 本轮判定 |
| --- | --- | --- |
| 普通历史 | 输出 200 个带序号短行，超过 30 行屏幕；capture 限最后 50 行，再输出固定尾标记 | 区分 history 与 visible；计数去重/遗漏，原 job 不重建。A 没有完整浏览器 scrollback 不能记为通过；B 若屏幕清空或进外层 alt 必须记限制。 |
| 属性、Unicode 与任意分块 | 固定 SGR、中文、组合字符；在 UTF-8/CSI/OSC 中间切 binary 帧 | 逐帧 Uint8Array write 后 cells 与整流输入一致，不能出现替换字符或串入 HTML。 |
| alternate / TUI | 标准库 curses 或已有明确可用 TUI；先普通标记，再进入 TUI、定位写行/颜色/光标，最后退出 | raw attach 当前画面与同尺寸 tmux capture 当前 grid 一致；`-a` 为保存屏幕对照。退出回到原普通画面；人工 ANSI fixture 不能冒称真实 curses/ncurses 应用兼容性。 |
| resize | 显式 100x30 -> 80x24 -> 120x40；等待 workload 的 SIGWINCH ack 和重新绘制稳定点 | 记录尺寸/光标/宽字符、换行与 buffer；resized WS ack 不代表 TUI 已处理 SIGWINCH。不能在 resize 前后的 capture 按行号直接去重。 |
| 断连期间输出 | detach 后继续输出编号，再 reconnect，增加控制间隙使内容超过一屏 | 原 PID/start 不变、计数推进、无自动 input；当前屏幕恢复与历史恢复分别判定。中间行只因初屏不可见则是历史遗漏证据，不是失败任务。 |
| capture/attach 窗口竞态 | capture 完成后让 workload 继续写 N 行，再 attach；重复有限轮 | 明确制造遗漏/重复反例；无截点方案不能标“零丢失”。静止 workload 对照通过不代表并发输出通过。 |
| 本地 snapshot round-trip | C 在确定性帧边界 serialize，同尺寸恢复新实例，再输入尾帧；包含 alt 和 resize 独立用例 | cells/cursor/modes 与连续基线一致，snapshot 与 live 不重复。若某模式不可序列化，保留反例并继续门禁。 |

### 下一阶段门禁

- 产品单输出源在无网页、Web 重启及多观察者下的真实 owner 生命周期；单 controller 尺寸与输入 generation。
- 正式 snapshot epoch/sequence、截点、缓冲容量、慢观察者重新同步与取消；任意分块下 parser pending UTF-8/ANSI 状态。serialize 在完整字符/控制序列结束后的测试不代表任意中间截点安全。
- capture 在持续输出、滚动、resize、alternate 切换中与 live 的一致性。不能靠正则移除 smcup/rmcup、纯文本相同前缀或固定 sleep 拼接。
- 真实浏览器 authenticated WS/xterm 即时链路与密码/clipboard/URL/OSC/title 安全策略；浏览器 fixture 回放不能解除此门禁。
- 恢复量可配置、默认 5000 的真正 scrollback/内存预算及超阈值虚拟滚动；本轮 50 行窗口不是产品容量验收。
- 多端、TUI 应用覆盖与 Web SIGKILL 后历史 owner 恢复均未由以上局部对照自动证明。

### Related Specs

- `.trellis/spec/backend/terminal-lifecycle.md`：产品持久性、controller、history 与未通过的大规模 UI 门禁。
- `.trellis/spec/backend/websocket-protocol.md`：单输出源与 snapshot/live 待 Spike 固化；旧 ready/history 参数不可当现成协议。
- `.trellis/spec/backend/bridge-validation.md`：D06 隔离边界、字节/队列/Wait 与独立模块检查。
- `.trellis/spec/backend/remote-validation.md`：私有 .env、去敏、资源授权与清理。
- `.trellis/spec/frontend/editor-terminal-lifecycle.md`、`clients.md`：binary write、取消、无 input replay、历史恢复一次。

## Caveats / Not Found

- research.jsonl 不存在；遵照研究角色隔离，未读取 implement.jsonl/check.jsonl。已按原生 Active task 读取 PRD/design/implement 与相关规范。
- 本次没有安装依赖、运行远端或读取 .env 内容；以上为候选方案，所有矩阵均尚未执行，不是 D07 通过报告。
- 固定 tmux 3.5a tag 对应既有 Debian tmux 3.5a-3 的上游基础，未审计 Debian 补丁差异；真实实验应记录包版本/二进制 SHA256。
- 官方 headless README/package 路径和 npm registry 浏览访问未成功；已确认固定 tag headless 源码与 typings 的 API，但不推断发布包版本。固定 xterm 5.5.0 可作为候选，不声称最新，也不替代当前日期安全版本审查。
- 现有 Bearer bridge 不能被原生浏览器直接原样连接。实时浏览器实验若要扩展 auth，需要单独明确隔离服务契约与审查，不挤入无鉴权代理。
