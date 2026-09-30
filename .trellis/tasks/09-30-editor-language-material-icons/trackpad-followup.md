# 触控板滚动精度与横向惯性白屏修复

2026-09-30 用户反馈终端滚动跳跃不跟手、左右滑动后白屏以及底部黑线。继续已批准工作，单代理实施与检查，不提交、不归档。

## 证据与变更边界

- 白屏复现：历史显示后，向原 live xterm 发送横向为主且带 -0.25px 纵向位移的惯性事件，controller/observer 两个浏览器用例都从已有行变成空数组。旧 live capture handler 无模式守卫，重复清空 historyBytes，但 historyVisible 未改变，不触发请求，因而永久空白。
- 不跟手：旧 wheelLines 对每个 0.5px 位移强制返回 1 行，累计微小位移严重放大；每次 scrollLines 为整数，丢弃原生 xterm Viewport 的精细像素位置、触控板分类与物理滚轮平滑处理。
- 用户随后反馈句点，进一步定位到跨层字符网格错配：后端 Resize 只调整输入 owner，read-only 输出 attach 固定 80x24，浏览器则按容器 fit。迟加入端也从固定尺寸 attach，observer renderer 自行 fit。归属扩大到 runtime 的 attach/resize 同步与前端 observer 网格；沿用既有 resized 形状，向所有同步成功端广播，不新增协议或观察端进程 resize 权限。不改 tmux 配置/会话/进程，不过滤合法正文。
- 历史加载时保留原 live 画面，不提前显示空 surface；连续事件在同一次请求内累加位移，读取完成后交给历史原生 viewport；已进入历史的 latched live 事件只转交，不清快照、不重新读取。横向为主事件忽略，不切模式、不改变内容，不触发浏览器导航/外层 overscroll。
- 精细纵向滚动由 xterm 6.0.0 的原生 viewport 处理，使用公开 smoothScrollDuration 保留物理滚轮动画，触控板原始精度不经整行放大。不访问 `_core` 私有 API；手势转交仅针对 xterm 宿主 DOM。底部继续下滚回 live 的行为保持，忽略不足一行的边界惯性噪声。

已读 before-dev/check、前端 runtime/hooks/组件/生命周期/质量规范及 [xterm 滚动选项](https://xtermjs.org/docs/api/terminal/interfaces/iterminaloptions/#smoothscrollduration)，具体行为以锁定源码 Viewport/ScrollableElement 为准。本轮使用 trellis-break-loop，按根因和上轮测试缺口记录预防机制。

## 最终实现与验证

- 单测验方向主轴、pixel/line/page 与小数位移保留。
- 浏览器验原 live target 惯性、纯横向/带纵向噪声横向、快速连续小数纵向、慢历史请求、模式边界、历史不重建/单请求/单 WS/零输入，以及此前 LF、DA、TUI、tab/树/已结束关闭回归。
- 检查底部真实 DOM/截图，区分宿主越界、原生框/焦点和终端输出；不删除合法输出/下划线掩盖问题。
- 前端 lint/typecheck/test/build 实际通过：20 文件、110 条单测，构建成功。原 jsdom canvas 提示与已有大 chunk 提示保留；没有通过安装无关依赖或提高警告阈值消除提示。
- 同一隔离 5175 Vite 上 `terminal-scrolling.spec.ts` 两角色、`terminal-device-attributes.spec.ts` 两角色、`workbench-interactions.spec.ts` 共 5 用例通过。新增真实网格录制后重跑设备应答两例通过，并再跑 lint/typecheck。保留 ANSI/LF/中文、TUI/Shift/Ctrl、历史加载期间零输入、同 live DOM/单 WS/单请求、标签/树/已结束关闭回归。
- 后端 gofmt、go test ./...、go vet ./...、go test -race ./... 实际通过，无 tmux 依赖跳过。`TestDeviceAttributesRealTmux -count=1 -v` 在 macOS tmux 3.7c 的独立 socket/config/raw pane 上通过：三次尺寸同步所有 PTY、真实 pane 同尺寸/PID 不变、迟加入 observer 继承网格、禁止 observer resize、全 viewer 收广播、pane 只收到显式 x，精确清理所有自有客户端/server/root。
- 黑线/句点实证：独立固定 cat pane 的 owner 从 80x24 缩放成 140x12，故意保留只读输出为 80x24，真实输出新增 80 个 `─` 和 880 个 `·`；将只读 PTY 同步 140x12 后重绘输出两者为 0。录制保存到 `web/tests/fixtures/terminal-grid-recording.json`（只有自有合成 pane，无用户输入/正文），浏览器真实 xterm 回放复现横线/句点并在 resized+重绘后消失。黑线是 tmux 边界字符，未删除正文或调整 fill-character 掩盖问题。[tmux 官方 FAQ](https://github.com/tmux/tmux/wiki/FAQ#why-do-i-see-dots-around-a-session-when-i-attach-to-it) 解释了窗口尺寸与句点的关系；本轮结论以真实录制和 PTY 查询为主。
- owner Resize 在 hub 锁内先更新输入 PTY，再更新所有输出 PTY并通知，读输出经同锁排队；单个输出 PTY 失败只取消该 viewer。Connect 持相同锁按当前 hub cols/rows 创建只读 attach。观察端 live 使用 ready/resized 网格，容器变小裁剪、变大留白，接管后才 fit；历史仍本地 fit。
- 证据边界：Chromium 原生滚轮与合成小数事件覆盖逻辑/渲染，不能声称所有真实触控板或 Safari 手感已验收；macOS 私有 tmux 测试不代替 Debian/systemd 持久性验收。更新运行中的版本需要重启 Web 服务并刷新浏览器，不终止既有 tmux 会话，不自动重跑命令。

## 缺陷分析：连续手势与字符网格传播

### 1. 根因类别

- **E 隐含假设 / D 测试覆盖缺口**：假设隐藏 live 后所有 wheel 都转到历史，且每个事件至少代表一行；上轮只测单次大滚轮与模式状态，漏掉同一 target 上持续惯性、小数和主轴噪声。
- **B 跨层契约 / C 变更传播失败**：只将尺寸传给输入端，忽略只读输出客户端和观察端 renderer。前端窗口尺寸、后端 cols/rows ACK、真实 pane 相等不是同一证明。
- **A 缺失规范**：旧规范允许 observer 本地 fit，未规定各输出 attach 与 owner 同网格；已替换为明确一致性契约。

### 2. 修复为何失败

1. 上轮用整数 scrollLines 阻止方向键输入，修复零输入但没有解决精度；清空快照没有同步模式守卫，惯性可在已显示历史时再次清数据。
2. 本轮初始转交使用合成 WheelEvent，Chromium 自动提供零 legacy wheelDelta，xterm 优先读零而忽略 deltaY；仅对转交事件置 undefined 后修正。
3. write callback 不代表 viewport 已完成尺寸渲染同步，首位移可能被 queued sync 重置；改为两帧后转交，并清理 rAF。不能仅等待 parser marker 判断滚动完成。
4. 原测试假设原生 100000px 必定直达底部；Chrome legacy wheel 归一化不保证该对应关系。改为明确检查原生滚动、精细事件及转交位移，分开到达底部和继续下滚两个动作。
5. 首个真实尺寸探针用 display-message 的会话 target 未取到 pane 字段；改为明确 list-panes -F 后，实际尺寸和 PID 条件通过。失败运行均清理了自有资源，没有将空查询当验收。

### 3. 预防机制

| 优先级 | 机制 | 实际行动 | 状态 |
| --- | --- | --- | --- |
| P0 | 模式/生命周期 | 同步 mode ref、首次请求守卫、加载保留画面、连续事件转交同一实例、callback/rAF 清理 | DONE |
| P0 | 精度与边界 | 原生历史 viewport、方向主轴、小数位移不整行放大、历史零输入、边界噪声守卫 | DONE |
| P0 | 跨层一致性 | hub 内统一所有 PTY/迟加入网格、向所有 viewer 通知、observer renderer 跟随 | DONE |
| P0 | 测试 | 两角色连续/加载/精细/横向浏览器回归、真实 PTY/进程/拒绝权限、真实录制复现黑线/句点 | DONE |
| P1 | 规范/审查 | owner specs 与跨层检查指南固化目标锁定、渲染同步、网格传播的检查点 | DONE |

### 4. 系统性扩展

- 搜索并核对其他尺寸/角色消费者，更新 lifecycle 中 observer fit 的旧描述；原历史报告保留阶段证据，并链接本次替代方案。
- 实时 PTY 与独立历史继续分离；不把 TUI alternate 当普通 shell 是否可滚动的判断，也不修改 readonly attach 的 ignore-size 以让观察端影响进程。
- 异常画面先检查真实输出、PTY 网格与 DOM，再判 CSS；黑线和句点相同来源，不追加两个正文过滤修补。
- 编译/单测无法发现浏览器的手势锁定或 tmux 填充；相应需求验收必须有真实库、真实 PTY 和连续事件证据。

### 5. 知识记录

- [x] 前端 terminal-runtime-contract：原生精度、惯性转交、加载只读、observer 服务端网格。
- [x] 后端 terminal-runtime-contract / websocket-protocol：输出 attach 尺寸传播、全端 resized、迟加入网格、单 viewer 失败边界。
- [x] editor-terminal-lifecycle 与 cross-layer-thinking-guide：替换旧 observer fit 规则，固化跨层与连续手势检查。
- [x] 父 PRD/check-report 和阶段 scrolling-followup 链接本次记录；无上游模板目录，不创建空模板、不提交或归档。

## 收尾检查与资源

实际检查 27 个 Markdown 文件、82 个本地链接、1 个本地标题锚点及代码围栏，全部通过；规范搜索确认 observer 本地 fit/整数 scrollLines 的旧最终规则已替换，git diff --check 通过。最后的前端四门禁再次全部通过，浏览器五例含真实网格回放全部通过；相关资源、许可与忽略规则保持有效。

专属 Vite 5175 已停止（原运行 session 94795，退出 130），两项 /private/tmp 探针文件已精确删除；每次真实 tmux fixture 按自有 socket/client/root 清理。保留合成录制和忽略目录内的 Playwright 证据，未重启/停止用户服务、未清理用户终端、未提交或归档。
