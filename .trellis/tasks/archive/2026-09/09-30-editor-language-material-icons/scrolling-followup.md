# 终端历史与滚轮、tab 悬停滚动条修复

2026-09-30 用户直接反馈历史快照逐行偏移、滚轮变成命令历史方向键，以及两类 tab 滚动条悬停消失/变粗/内容抖动。本轮继续当前已批准工作，不提交/归档，不操作用户终端。

后续触控板小数位移/连续惯性与字符网格修复见[最新补充](trackpad-followup.md)。本文记录首轮阶段结果；其中整数 scrollLines 已被历史原生 viewport 替换，最终规则以最新 owner 规范为准。

## 证据与变更边界

- `capture-pane -p -e` 输出按 LF 分行；历史 xterm 默认 convertEol=false，保留前行列坐标导致阶梯形偏移。仅历史快照改为 convertEol=true，保留 ANSI/Unicode 原字节；实时 PTY 继续 convertEol=false，不剥 ANSI、不合并历史与 live。
- 活动 xterm scrollback=0，tmux 外层 attach 通常在 alternate buffer。锁定 xterm 6.0.0 的 CoreBrowserTerminal 在没有 scrollback、没有 mouse tracking 时把 wheel 转成上下键。不能据外层 alternate 判断 pane TUI：普通/观察端滚轮由宿主 capture listener 截获，上滚进入独立有界历史，下滚到历史末尾后回实时；controller 的显式鼠标 tracking 保留给 TUI，Shift+wheel 可强制查看历史。Ctrl+wheel 保留浏览器缩放，不产生输入。快照滚轮调用公开 scrollLines，零 WS 输入、接管或重连。
- 前次 CSS 的 hover scrollbar-color 选择器比 Chromium 分支默认选择器优先级高，悬停覆盖 auto，使自定义轨道退回原生轨道。Firefox 标准属性限定在不支持 WebKit scrollbar 的分支；Chromium/WebKit 固定 4px 轨道，thumb 默认透明、hover/focus 才显示，不修改厚度/布局。Firefox 始终 thin，占用尺寸固定。

修改 owner：TerminalSession（独立历史/实时宿主）、terminal-scrolling（轮事件单位换算）、styles.css（tab 轨道）。不变更后端协议、tmux 配置、进程控制、历史采集或 live 输出缓存策略。

已读 Trellis before-dev/check 与 frontend 组件/状态/类型/质量/生命周期/终端及 backend 终端/历史/快照规范。参考 [xterm Terminal API](https://xtermjs.org/docs/api/terminal/classes/terminal/) 与 [convertEol](https://xtermjs.org/docs/api/terminal/interfaces/iterminaloptions/#converteol)，以锁定本地 xterm 源码确认实际 wheel 路径。轮事件捕获兼容 xterm 的显式 mouse tracking，避免 custom wheel API 只拦截无 tracking 的路径。

## 验证计划

- 纯单测验 pixel/line/page 滚轮换算；真实浏览器用实际 TerminalSession/xterm/WS decoder 与合成 WS/history：LF/CRLF/ANSI/中文从行首正确呈现、上下滚动改变历史 viewport 而非 WS 输入；历史查看/退出/刷新保留 live DOM/单连接，controller TUI tracking 的滚轮仍发送鼠标报告，observer/Shift 强制快照零输入。
- tab 真实浏览器在 hover 前/后/轨道 hover 对比高度、按钮位置与 scrollbar height/color，默认透明、悬停可见、无布局跳变。前次树/关闭回归保留。
- 完整前端四门禁、Go test/vet（如未改变 Go 可引用当前会话已通过结果），Markdown 链接/围栏/diff；主会话 trellis-check 与 update-spec 记录契约。

## 实际验收与自审

| 检查 | 本轮结果 |
| --- | --- |
| npm lint / typecheck / test / build | 全部通过；20 个单测文件、108 条测试，实际构建成功 |
| Go test / vet / test -race | `go test ./...`、`go vet ./...`、`go test -race ./...` 全部通过，部分缓存；本轮未修改 Go |
| controller / observer 真实 Chromium | 两个历史/滚轮场景均通过；LF/CRLF/中文/ANSI 对齐，实际 xterm 历史 viewport 改变；普通滚轮零二进制输入，Ctrl 滚轮不打开历史，单连接、原 live DOM 保留；controller 显式 tracking 发 SGR 鼠标报告，Shift 与 observer 查看历史不发送输入 |
| 工作台真实 Chromium | 一个综合交互场景通过；保留右键、目录无选中、文件定位、已结束标签关闭/重载回归；两类 tab 默认 thumb 透明、悬停可见、移开透明，轨道始终 4px，容器/标签高度与坐标不变，横向滚动有效 |
| 主会话自审 | 数据仍经原严格 history decoder，快照与 live 分离；新 capture listeners/observer/write callback 随独立实例释放；无额外依赖、调试全局、类型绕过或后端接口变更；更新 frontend 终端与组件 owner 规范 |

早期浏览器检查发现仅调整标准属性分支仍不能保证 WebKit thumb 的 hover 颜色更新，已改为容器变量 `--tab-scrollbar-thumb` 传递颜色，并加入颜色的三状态断言，全部复验通过。截图人工核对历史各行左对齐及两类 4px 滑块，原始截图位于忽略的 `web/test-results/`。jsdom canvas getContext 提示及既有大 bundle 提示保留，未屏蔽。

重现：在隔离 Vite `5175` 上，使用 `PERSISTTY_E2E_BASE_URL=http://127.0.0.1:5175 PERSISTTY_E2E_PASSWORD=fixture PERSISTTY_E2E_PROJECT=fixture npm run test:e2e -- workbench-interactions.spec.ts terminal-scrolling.spec.ts`（工作目录 `web/`）。三个用例零跳过。fixture 用真实组件/xterm/WS decoder，接口与输出为合成 HTTP/WS；此证据验证浏览器排版与事件处理，不声称真实 tmux、Debian/systemd 或手机真机重新验收。未访问或操作用户终端/进程/文件正文。

保留当前代码及任务供审阅，不提交、不归档。
