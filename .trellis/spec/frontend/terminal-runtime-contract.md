# W03 前端终端运行时契约

## 1. 范围与触发条件

修改工作台终端宿主、路由、连接、控制栏或历史视图时必读。W03 已验收；完整布局持久化与文件编辑/草稿仍为 W05。后端真实状态和控制权以 [终端协议](../backend/terminal-runtime-contract.md) 为准。

## 2. 签名

`TerminalRuntimeProvider` 按工作台项目/统一终端路由建立 scope；`RuntimeScope.ensure(terminal)` 返回按稳定 terminal ID 缓存的 `RuntimeEntry`，含固定 `HTMLDivElement`、DTO 和关闭意图。`TerminalSession` 是宿主包装，`TerminalSessionView` 是真实 xterm/WS。`useTerminalStateChanges` 消费状态失效通知。接口源码分别位于 `web/src/features/terminal/` 与 `web/src/lib/ws/terminal.ts`。

scope 新动作：`takeover(terminal)`、`inputIntent(id)`、`close(terminals)`、`rename(terminal)`、`register(id,actions)`、`report(id,state)`、`event(id,event)`。actions 的 takeover 返回 `Promise<void>` 等待服务端确认，不以 send 成功判完成；target 返回 `{terminal_id,viewer_id,generation}`。状态订阅与列表失效订阅分开，不能每次角色更新都触发网络刷新。

## 3. 契约

- provider 通过 portal 将 runtime 渲染进固定 DOM element；上下移动只是将同 element 挂到另一宿主。收起面板、刷新资源列表、宿主卸载不销毁 xterm/WS、不复制 session、不自动接管。离开 scope 才 dispose 浏览器资源，仍不终止 tmux。
- StrictMode 下延后到 microtask 建 WS，并检查已清理标志，避免废弃 mount 产生连接。DTO 刷新不等于新 terminal ID；项目切换用独立 scope，旧请求 abort。
- 活动 xterm `scrollback:0`；history HTTP 返回的快照整体替换虚拟滚动视图，行数使用 `returned_lines`，不写进 live xterm。最多 1 MiB 的待渲染输出由 write callback 排空，超限关闭 1013 后重新 attach 当前画面。
- 历史 `capture-pane -p -e` 快照为 LF 分行，独立只读 xterm 必须 `convertEol:true`，保留 ANSI/Unicode，避免逐行列坐标累积；实时 PTY 仍 `convertEol:false`。历史主题变化同步独立实例，不重建 live DOM/WS。
- 普通/观察端垂直滚轮在宿主 DOM capture 阶段截获，阻止 xterm 在零 scrollback 时转成上下方向键；向上滚读取独立有界历史。仅首次进入清旧快照并请求，不能让已隐藏 live 上锁定的连续惯性事件反复清 historyBytes。等待 HTTP/解析/viewport 尺寸同步期间保留 live 画面并累加像素位移，历史模式（含加载/失败）不向 shell 发送键盘、粘贴、鼠标或手机快捷键。
- 历史字节到达不代表已绘制。TerminalSession 使用 `.terminal-screen-stack` 中的绝对定位历史覆盖层，write callback 后双rAF完成viewport同步再以 `renderedHistory === historyBytes` 显示；新快照未就绪时保留实时画面。实时仅 visibility:hidden，尺寸始终保留，避免 display:none 造成零尺寸与返回时重绘空白；手机快捷行仍在stack外。旧实例callback/rAF清理，不能迟到显示旧快照。
- 历史纵向事件交给 xterm 原生 viewport，保留小数位移与触控板分类，`smoothScrollDuration:120` 用于物理滚轮；不要把每个小于一行的事件强制放大为 `scrollLines(±1)`。锁定 live target 的事件以归一化像素转交同一历史 DOM；Chromium 合成 WheelEvent 的零值 legacy wheelDelta 字段须在转交事件上置 undefined，避免 xterm 优先读零而忽略 deltaY。首位移在 write callback 后等待 viewport 渲染同步，再转交；callback/rAF 随实例清理，不访问 `_core`。
- 历史响应 history_size 为 0、returned_lines 为 0 或解码字节仅为空白时，不能挂载空历史并隐藏 live；清除首位移、退出历史模式并在原实时画面显示“暂无历史输出”。空态提示 pointer-events:none，不遮挡首行输入点击；控制端明确输入后清除提示。capture-pane 可能在无 scrollback 时返回一个 LF，Uint8Array(0) 本身也为 truthy，不能依对象真假判断有效历史。该分支不重连、接管、创建会话或发送输入；慢请求、异常仍沿原 live 保留策略。
- 历史在底部继续向下累计一行位移才回实时，微小噪声不切模式；水平主轴（含 ±0.25px 纵向噪声）阻止默认滚动且不清内容、不请求、不切历史。历史错误保留原 live 实例与可返回/刷新入口；终端宿主 overflow:clip、overscroll-behavior:none，只允许内部 viewport 纵向滚动，不引入外层水平滚动或导航。
- tmux 外层 alternate buffer 不代表 pane 是 TUI。只有当前 controller 的显式 `mouseTrackingMode !== "none"` 才将普通滚轮交给应用鼠标协议；Shift+wheel 强制查看历史，observer 不发鼠标报告。Ctrl+wheel 保留浏览器缩放且阻止 xterm 转输入；纯横向滚轮不触发历史。不要仅使用 xterm custom wheel handler，它不覆盖显式 mouse tracking 路径。
- xterm onData 同时含用户输入与自动设备应答。live 实例用公开 parser.registerCsiHandler 对 `{final:"c"}` / `{prefix:">",final:"c"}` 消费 DA1/DA2，首参数为 0 时调用 `TerminalSocket.sendDeviceAttributes("primary"|"secondary")`，阻止默认 onData 应答误入 owner shell。参数大于 0 与原实现一样消费但不回答，分帧序列由原 parser 处理。handler 随 live 实例释放，不因 role 变化重建。
- `sendDeviceAttributes` 发送后端批准的 v3 固定枚举文本帧，允许 observer 回答自己的 read-only attach；必须已 ready、未 dispose、OPEN 且 bufferedAmount 不超过 1 MiB，失败不缓存/重放或回退 sendInput。键盘/粘贴/鼠标仍复验当前 controller/generation，粘贴与应答相同的字节也保持原输入；不能基于字符串 regex 猜输入来源。
- 可见、正尺寸且当前 controller 才 fit/发送 resize。observer live xterm 使用 ready/resized 的服务端字符网格，不按自己容器 fit，不发送 resize；较小容器裁剪、较大容器留空，不能让只读 PTY、pane 与 renderer 尺寸各异而出现 tmux 边界线/句点填充。接管后才 fit 自己尺寸；历史独立实例仍按本地容器 fit。输入必须当前 ready/controller/generation；离线立刻取消 ready，socket dispose 后晚到帧无效。输入不缓存、不重放。
- FitAddon 0.11.0 只扣除 `.xterm` 自身内边距，不扣外层宿主内边距；实时/历史宿主不放 padding，`5px 8px` 放在 height:100% 的 `.xterm` 上。controller 和历史的 screen/最后一行须在宿主内边距边界内，不能以 overflow:clip 掩盖算错的行列。observer 的服务端网格裁剪规则保持独立。
- xterm 6.0.0 只给 `.xterm-scrollable-element` 设置主题背景，绝对定位的 `.xterm-viewport` 默认黑色；宿主 owner CSS 显式将 viewport 背景设为 `var(--background)`。整数网格未铺满剩余空间或留白/内边距时不能露黑边。正常 ANSI 下划线、横线正文保留，不能全局禁用 text-decoration 或过滤输出。
- 失败后探测认证，1008/401/403 停止；其余最多 5 次指数退避（500..8000 ms 加 jitter），耗尽显示手动重试。控制被他端接管后不自动夺回。
- running 的上方标签 X/会话 trash 请求终止；下方面板 X 仅收起。明确 terminated 的标签关闭仅调整本浏览器视图，详见下文；unavailable 不当作已结束。Dialog 使用应用 body portal，不受终端 DOM 隐藏影响；显示服务器截止，新端从 ready 接收同 request/deadline。取消/失败保留入口，执行后重新查真实状态。
- 下方多组工具栏新增“合并此终端分组”：调用workspace-view.mergeTerminalGroup迁移标签与组位置，空组可取消、有会话的组并入相邻组；不调用scope.close/takeover或重建固定runtime。下方面板X整体收起契约保持，最后一组保留。位置与持久化详见[状态规范](state-management.md)。
- 手机 Ctrl/Alt 使用官方 shadcn Toggle 的 pressed 状态，其他按键用 Button，发送真实控制字节。错误用 Alert，空/结束状态用 Empty，确认用 Dialog，选择用 Select；遵守官方查找记录，禁止手写适用基础组件替代品和直接引入 Radix。
- 链接只接受 HTTP/HTTPS，桌面 Ctrl 点击/手机确认后原样新标签打开并 noopener；不改写 localhost、不提供预览代理。

### 精简工作台与批次

- 项目工作台单顶栏与单终端标签行；删除旧目录/连接/接管常驻行。Tabs/菜单/Tooltip 使用官方 shadcn 源码，不混用原语。面板最右 X 只收起，running 标签 X/关闭菜单才进入带名称确认及倒计时。手机用更多菜单且不提供移动。
- 名称更新同步完整 DTO，不以名称作 key、不重建固定 runtime。默认编号由后端分配；重命名用旧名 CAS，不在客户端计数。
- 图标/菜单显式接管直接等待 ACK；观察端输入意图先确认。observer 保持 xterm disableStdin=true，DOM 捕获 keydown/beforeinput/paste/compositionstart，手机快捷操作走同确认；触发输入完全丢弃，不存字节、不在成功后补发。纯修饰键、选择复制、滚动、链接不误判；onData/onBinary 仍复验当前角色/ready，不能把 focus/mouse report 当输入确认。
- close 冻结当前上/下区域目标及名称，确认前不接管。确认后逐个 ready + takeover ACK，再一次 POST batch；任一失败零 POST，显示失败名与已接管列表，不假装回滚控制权。统一终端页没有跨项目关闭其他/全部入口。
- `RuntimeScope.close(terminals)` 先按稳定 ID 从已有 runtime 取最新 DTO：terminated 使用 `useTerminalView.dismiss` 关闭浏览器标签，零接管/终止/删除请求；running 才进入上述确认；unavailable 保留。混合关闭立即关闭 ended 标签，确认取消只影响尚未终止的 running 目标。X、右键与手机更多菜单复用此入口，不在 TerminalTab 过滤 ended。
- `terminal-view.ts` 只将 `dismissed: Record<terminalId,true>` 持久化到 `persistty.terminal-view.v1`；上下区域与统一页通过 `terminalVisible` 一致过滤，列表刷新与页面重载不重新显示已关闭 ended。未知状态或 running 总是可见，旧偏好不能隐藏真实运行进程。服务端元数据/历史保留；保留已结束 runtime 的最新 DTO，避免上方标签旧 running 快照使 closed 标签重新出现。
- scope Dialog 按 request ID 合并成员通知；隐藏宿主仍可见，不同批次分别显示取消入口。取消等服务端通知，不乐观移除；GET 查询短时结果，404 提示重新核实，不重发终止。结果逐项展示。
- 新 TerminalSocket 显式 protocol=3；decoder 保留可选 v2 exact 解析，不以未知字段降级。ApiError 只接收已定义的 terminal_id details，其他陌生 details 拒绝。
- runtime portal 的 React 事件跟随原 owner，不跟随 DOM 移动。内部拖拽用 `useDropTarget` 在目标 DOM 捕获，只拦截专用 MIME，并 stopPropagation，避免 xterm 消费标签数据；普通文本/文件拖拽不能误变为移动。监听与 scope/宿主同步释放。
- 视口布局使用 100dvh、flex min-height/min-width:0；长树 sr-only 节点的绝对定位必须有就近 positioned + overflow 边界。只允许内容内部滚动，不全局锁 body 滚动。真实手机软键盘效果须单独实机验收，模拟视口不代替实机。

## 4. 验证与错误矩阵

| 场景 | 行为 |
| --- | --- |
| 上下移动/面板隐藏/资源刷新 | 同 runtime/element/socket，零新增连接或终止请求 |
| 掉线/认证撤销 | 立即禁输入，不重发，不修改 server session |
| observer/旧 generation | 前端不写入；后端仍复验，不以 UI 当权限边界 |
| 1013/真实状态改变 | 刷新 DTO，显示已结束/不可用，而非无限 loading |
| terminated 标签 X/关闭其他/全部 | 本浏览器移除并持久化，零终止请求；上下区域均生效 |
| unavailable 标签关闭 | 禁用，不能推断真实进程已结束 |
| running + terminated 混合关闭 | ended 标签直接关闭；仅 running 弹确认，取消零终止请求 |
| 历史请求失败/超限 | 可见错误，不清空或伪造实时画面 |
| LF/CRLF/ANSI/中文历史 | 各行从行首正确呈现，保留颜色与字符 |
| 普通/观察端上下滚轮 | 浏览历史 viewport，零输入/接管/新增 WS；底部继续下滚回实时 |
| controller 显式鼠标 tracking | 普通滚轮为应用鼠标报告；Shift 滚轮只读历史；不以 alternate 推断 TUI |
| DA1/DA2 查询/重复/分帧 | 仅固定 device_attributes 文本帧，零二进制垃圾输入，role 不变 |
| 正常输入或粘贴相同 DA bytes | 原样二进制输入，controller/generation 仍必需，不当作自动应答 |
| 手机链接取消 | 无新标签，无控制权变化 |
| 批量部分接管失败 | 不 POST、不开始计时，显示已取得控制权的对象 |
| 改名/菜单刷新 | 同固定 DOM/WS；刷新只读状态、不清画面 |
| 观察端触发输入确认 | 丢弃触发字符/粘贴/IME/控制字节；确认后只接受新输入 |

## 5. 正常 / 基础 / 错误用例

正常：controller 在上下宿主之间按钮/拖拽移动、收起展开，其他端始终 observer；任意端取消全端倒计时。基础：手机单视图输入方向键/控制键，深色和窄屏无溢出。错误：卸载宿主关闭 WS 后自动 takeover，或者面板 X 直接终止 shell。

## 6. 所需测试

`TerminalRuntime.test.tsx` 在 StrictMode 验同实例、同 element 和仅 scope 退出释放。WS 单测验证 decoder、8 字节 generation、离线立即失效、晚到帧忽略、零重放。Playwright 验真实非空桌面/手机画面、所有快捷字节、粘贴、链接、主题、双端/新第三端、宿主移动时零新 WS、取消与到期终止。Debian 正常/SIGKILL Web 重启验证原 PID/start/cgroup、三负载及恢复后 TUI 按键/鼠标/resize；详见 [浏览器步骤](../../../tests/integration/debian/browser/README.md)。

UI 调整新增 provider 部分接管失败零 POST、共享 v3 fixture、改名同 DOM/零 WS、上下区域与混合协议取消、120 文件/双根三主题/四视口 document 几何与内部滚动。移动必须同时覆盖按钮和真实 dragTo 落在活跃 xterm 上，不能用人工 dispatchEvent 代替拖拽回归。

已结束关闭回归由 `terminal-view.test.ts`、`TerminalRuntime.test.tsx` 与 `workbench-interactions.spec.ts` 验证：上下标签 X/关闭其他/全部、持久偏好重载、running/unavailable 保护、混合批次取消零 POST。合成接口工作台测试只验 UI 状态流，不代替真实 PTY、tmux、倒计时执行和 Debian 持久性验收。

历史/滚轮回归由 `terminal-scrolling.test.ts` 验 pixel/line/page、小数与主轴换算，`terminal-scrolling.spec.ts` 用真实 xterm/WS decoder 与合成 HTTP/WS 分别验 controller/observer：LF/CRLF/ANSI/中文对齐、原 live target 的连续惯性、40 次 0.5px 位移不过度放大、纯横向/带纵向噪声横向、慢 HTTP 期间保留画面/单请求/零输入、历史不重建、底部噪声与返回 live、Ctrl 缩放、单 WS、TUI 鼠标及 Shift 强制历史。`terminal-device-attributes.spec.ts` 同时验 observer ready/resized 网格固定，隔离真实 tmux 录制中的边界线/句点经同步重绘消失；录制解析与当次真实后端 PTY 尺寸测试配套，不能用录制代替运行时或 Debian 验收。

`terminal-geometry.spec.ts` 在三个桌面视口与浅/深主题下，用真实 xterm/FitAddon、隔离 HTTP/WS 验 controller 最后一行/光标和历史底部行完整容纳、viewport 与 scrollable 背景一致、正常 ANSI 下划线保留、单 WS/零输入。不能仅检查文字存在或可见来判定没有局部裁剪；须比较 screen 边界与宿主内边距边界。

设备应答由 WS 单测验 observer 可发固定枚举且无键盘权限、未 ready/离线/背压零发送；`terminal-device-attributes.spec.ts` 在真实 xterm 中分别验 controller/observer 的省略/0/非零参数、分帧/重复/晚到 DA 查询仅发固定文本帧，真实键盘/方向键及合成 paste 事件保留原字节；配合后端 pipe/真实隔离 tmux 测试验证 attach 归属，不把合成 WS 浏览器证据称为真实后端集成。

## 7. 错误与正确示例

### 下方面板全屏与恢复（2026-10-03）

ProjectWorkbench 管理全部下方终端组的临时最大化；TerminalWorkspace 接收 `maximized?: boolean` 与 `onToggleMaximize?(): void`，复用 Button/Maximize2/Minimize2。编辑器 Panel 可折叠到0且仍挂载，竖向 Group 使用 useGroupRef。按钮放大前、分隔器 pointerdown/keydown 前记住常规布局；恢复用 setLayout。编辑器 minSize=38px，collapsedThreshold=1px（库按低于 minSize 的距离解释；1px容差避免浮点尺寸在标签边缘提前收起）。分隔器连续经过160px直到顶部38px标签行，继续向上才吸附；不能保留160px最小值使拖动提前停住，也不能在同一拖动中动态切换约束。

最大化填满右侧 main 区域，侧栏/顶栏/状态栏保留；水平分隔器在最大化时0px且保留顶部8px热区；只有终端面板被收起时才 disabled 并移除热区。顶部向下拖重新展开并连续调整，按钮退出仍恢复进入前比例。onResize 同步最大化及终端可见性。最大化退出恢复进入前比例；全屏时关闭面板先恢复常规比例再折叠终端，重新打开保持该比例。useDefaultLayout 的 onLayoutChanged 仅在 editors>0 时接收布局，不能保存临时0/100并让刷新后丢失文件区域。手机仍为单内容视图，无额外全屏入口。

正常：按钮或真实分隔器拖动→全屏→按钮恢复，原 xterm/Monaco DOM 和常规尺寸保留。基础：空终端面板、多下方分组同样放大；刷新恢复常规布局。错误：移动/重建 provider 来全屏、保存0/100覆盖常规布局、收起时发终止。正确：保留 provider/Panel/runtime，仅 setLayout/collapse 调整视图。

`workbench-modern-ui.spec.ts` 覆盖真实指针连续140/90/48/38px缩小、继续上拖吸附、顶部向下展开、浅/深截图、多组、收起再打开、刷新、单 WS/零输入/零写入、固定 DOM 标记和常规尺寸持久化。需先等待 separator 的实际0px状态再测上下对齐，不能只等待 editors=0 后立刻采样仍未提交的 React 状态。已有 terminal-geometry 与 editor-recovery 继续验证网格及缓冲区；合成 WS 不记为 Debian/tmux 或物理设备验收。

错误：`useEffect(() => new Terminal(), [host])`，cleanup 断连，再用 `wasController` 自动接管。正确：provider 以 terminal ID 保留 runtime，host 只 append/remove 固定 element；连接断开后只按服务器 ready/control 恢复权限，不绕过显式接管。

2026-10-01 截图回归增加零字节/无 scrollback 的 LF/仅空白三种空态与短历史重复切换，均保持 live 内容、单 WS 和零输入。真实 Debian 隔离实例的 w05-feedback-live 同时验新会话空历史与明确输出 120 行后长历史切换；合成滚轮不能代替真实触控板手感或手机软键盘验收。

2026-10-03 追加滚动反馈：`terminal-scrolling.spec.ts` 对历史首次绘制/刷新/三视口变化/返回实时逐rAF检查可见非空行，同时比较实时宿主尺寸、固定DOM、单WS及零输入。旧实现对照记录10个空白采样帧；修复后应为0。该采样验证DOM renderer行与可见性，不代替物理触控板或真实Debian输出负载。
