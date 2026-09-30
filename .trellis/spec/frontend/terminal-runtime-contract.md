# W03 前端终端运行时契约

## 1. 范围与触发条件

修改工作台终端宿主、路由、连接、控制栏或历史视图时必读。W03 已验收；完整布局持久化与文件编辑/草稿仍为 W05。后端真实状态和控制权以 [终端协议](../backend/terminal-runtime-contract.md) 为准。

## 2. 签名

`TerminalRuntimeProvider` 按工作台项目/统一终端路由建立 scope；`RuntimeScope.ensure(terminal)` 返回按稳定 terminal ID 缓存的 `RuntimeEntry`，含固定 `HTMLDivElement`、DTO 和关闭意图。`TerminalSession` 是宿主包装，`TerminalSessionView` 是真实 xterm/WS。`useTerminalStateChanges` 消费状态失效通知。接口源码分别位于 `web/src/features/terminal/` 与 `web/src/lib/ws/terminal.ts`。

## 3. 契约

- provider 通过 portal 将 runtime 渲染进固定 DOM element；上下移动只是将同 element 挂到另一宿主。收起面板、刷新资源列表、宿主卸载不销毁 xterm/WS、不复制 session、不自动接管。离开 scope 才 dispose 浏览器资源，仍不终止 tmux。
- StrictMode 下延后到 microtask 建 WS，并检查已清理标志，避免废弃 mount 产生连接。DTO 刷新不等于新 terminal ID；项目切换用独立 scope，旧请求 abort。
- 活动 xterm `scrollback:0`；history HTTP 返回的快照整体替换虚拟滚动视图，行数使用 `returned_lines`，不写进 live xterm。最多 1 MiB 的待渲染输出由 write callback 排空，超限关闭 1013 后重新 attach 当前画面。
- 可见、正尺寸才 fit；observer 可本地 fit，不发送服务器 resize。输入必须当前 ready/controller/generation；离线立刻取消 ready，socket dispose 后晚到帧无效。输入不缓存、不重放。
- 失败后探测认证，1008/401/403 停止；其余最多 5 次指数退避（500..8000 ms 加 jitter），耗尽显示手动重试。控制被他端接管后不自动夺回。
- 上方标签 X/会话 trash 请求终止；下方面板 X 仅收起。Dialog 使用应用 body portal，不受终端 DOM 隐藏影响；显示服务器截止，新端从 ready 接收同 request/deadline。取消/失败保留入口，执行后重新查真实状态。
- 手机 Ctrl/Alt 使用官方 shadcn Toggle 的 pressed 状态，其他按键用 Button，发送真实控制字节。错误用 Alert，空/结束状态用 Empty，确认用 Dialog，选择用 Select；遵守官方查找记录，禁止手写适用基础组件替代品和直接引入 Radix。
- 链接只接受 HTTP/HTTPS，桌面 Ctrl 点击/手机确认后原样新标签打开并 noopener；不改写 localhost、不提供预览代理。

## 4. 验证与错误矩阵

| 场景 | 行为 |
| --- | --- |
| 上下移动/面板隐藏/资源刷新 | 同 runtime/element/socket，零新增连接或终止请求 |
| 掉线/认证撤销 | 立即禁输入，不重发，不修改 server session |
| observer/旧 generation | 前端不写入；后端仍复验，不以 UI 当权限边界 |
| 1013/真实状态改变 | 刷新 DTO，显示已结束/不可用，而非无限 loading |
| 历史请求失败/超限 | 可见错误，不清空或伪造实时画面 |
| 手机链接取消 | 无新标签，无控制权变化 |

## 5. 正常 / 基础 / 错误用例

正常：controller 在上下宿主之间按钮/拖拽移动、收起展开，其他端始终 observer；任意端取消全端倒计时。基础：手机单视图输入方向键/控制键，深色和窄屏无溢出。错误：卸载宿主关闭 WS 后自动 takeover，或者面板 X 直接终止 shell。

## 6. 所需测试

`TerminalRuntime.test.tsx` 在 StrictMode 验同实例、同 element 和仅 scope 退出释放。WS 单测验证 decoder、8 字节 generation、离线立即失效、晚到帧忽略、零重放。Playwright 验真实非空桌面/手机画面、所有快捷字节、粘贴、链接、主题、双端/新第三端、宿主移动时零新 WS、取消与到期终止。Debian 正常/SIGKILL Web 重启验证原 PID/start/cgroup、三负载及恢复后 TUI 按键/鼠标/resize；详见 [浏览器步骤](../../../tests/integration/debian/browser/README.md)。

## 7. 错误与正确示例

错误：`useEffect(() => new Terminal(), [host])`，cleanup 断连，再用 `wasController` 自动接管。正确：provider 以 terminal ID 保留 runtime，host 只 append/remove 固定 element；连接断开后只按服务器 ready/control 恢复权限，不绕过显式接管。
