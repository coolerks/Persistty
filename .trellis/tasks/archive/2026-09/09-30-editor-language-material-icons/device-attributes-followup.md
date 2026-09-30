# 终端设备属性应答误入 shell 修复

## 1. 范围与触发条件

2026-09-30 用户补充截图中的 `1;2c` / `0;276;0c` 异常输入。锁定 xterm 6.0.0 的 InputHandler 对 DA1/DA2 输出分别为 `ESC[?1;2c` 与 `ESC[>0;276;0c`，与截图一致。浏览器输出来自 viewer 的只读 PTY，但所有 onData 经二进制 Input 写到 hub owner PTY；设备应答跨 attach 误路由。此修复直接属于用户授权的终端缺陷处理，单代理执行，保留未提交修改。

## 2. 签名与修改边界

- v3 客户端文本帧新增严格 `{type:"device_attributes",kind:"primary"|"secondary"}`，无原始 payload、generation、target 或命令参数。v2 保持原协议，不接受新增帧。
- TerminalSocket 增加 sendDeviceAttributes；TerminalSession 用公开 parser.registerCsiHandler 拦截 DA1/DA2 查询，替代默认 onData 应答。普通键盘、粘贴、TUI 鼠标仍走原二进制输入。
- Viewer.DeviceAttributes(ctx,kind) 复验认证/存活/所属连接，只把服务端固定应答写到该 viewer.attach.file；绝不写 owner。
- 修改 owner 为前端 TerminalSession/WS client，后端 terminal runtime/HTTP WS decoder；新增共享 fixture、路由隔离及真实浏览器回归。无 tmux 配置、创建/终止、数据库、历史或用户进程变更。

## 3. 契约与方案依据

当前协议认证和 Origin 验证保持；两类应答固定为当前锁定 xterm 的原值。observer 可回答自己只读 attach 的能力查询，不获得 owner 输入权限。发不出的自动应答不缓存/重放。CSI 参数大于 0 与原 xterm 一样消费但不应答；分帧查询由 xterm parser 处理，不对正文做正则删减、不靠渲染期间全局禁输入。

参考 [xterm parser API](https://xtermjs.org/docs/api/terminal/interfaces/iparser/)，以本地 InputHandler 源码确认序列。后端 runtime 的 Input、只读 attach 与 owner 分离是直接根因证据。仅截获已确认 DA1/DA2；不伪造所有终端协议或扩展为任意 bytes 的 observer 写通道。

## 4. 验证与错误矩阵

| 场景 | 结果 |
| --- | --- |
| controller/observer DA1/DA2 查询 | 各自只读 attach 接收固定应答；零 owner/二进制 shell 输入 |
| kind 非枚举/额外字段/重复字段/null/v2 | 1008，不写 PTY |
| 认证撤销/连接退出/hub 关闭 | 拒绝，无应答重放 |
| 普通输入/粘贴/方向键/TUI 鼠标 | 原 generation/controller 路径保持 |

## 5. 正常 / 基础 / 错误用例

正常：两端同时 attach 各自回答自己查询，控制权不变。基础：浏览器在连接和重绘时自动回答 DA1/DA2。错误：把设备能力应答写入 owner，shell 出现设备版本数字；或正则删除用户输入/输出中的同样文本。

## 6. 所需测试

Go pipe 隔离证明两端应答只进自身 attach、owner 无写，撤销和非法 kind 不写；严格 decoder 验 v2/v3 与固定 schema。WS 单测验未 ready/离线/背压不发送、observer 可发固定枚举且不能输入。真实 Chromium/xterm 注入分帧及重复 DA1/DA2，断言只发 device_attributes，无二进制垃圾；正常 controller 键盘/粘贴不被吞，observer 不获输入权。补真实隔离 tmux/PTY 定向验证（若本机依赖可用），不据此声称 Debian/systemd 完整重验。

## 7. 错误与正确示例

错误：DA1/DA2 默认 onData → sendInput → owner.file。正确：公开 parser 的 DA handler → 固定枚举文本帧 → 当前 viewer.attach.file。失败不回退到 sendInput。

## 实际验收与自审

| 检查 | 结果 |
| --- | --- |
| 前端 lint / typecheck / test / build | 全部通过；20 个单测文件、109 条测试，实际生产构建成功 |
| Go test / vet / race | `go test ./...`、`go vet ./...`、`go test -race ./...` 通过；本轮 bridge/HTTP decoder 均检查 |
| 真实 Chromium | 新增 controller/observer 两个 DA 用例，分帧/省略/0/非零/重复/晚到查询只发固定文本帧，零二进制自动输入；真实键盘、方向键与合成 ClipboardEvent paste 中相同 DA bytes 原样保留，单连接/原 role，无 pageerror |
| 前轮回归 | 历史/滚轮两用例与工作台综合交互一用例均复验通过；合计 5 个浏览器用例，零跳过 |
| 真实 tmux / PTY | 本机 macOS tmux 3.7c，自建独立 socket/server 与 raw-input Python pane；两个 viewer 实际发出 DA1/DA2 查询，返回自身应答后 pane 只收到显式 `x`，零设备应答；仅清理本次 server/session/root |
| 安全与资源自审 | 固定枚举无任意 payload/target，认证与所属/存活连接再次校验，v2 明确拒绝，observer 无 owner 输入权限；handler 随 live 释放，无重建/新连接、输入缓存或日志；backend/frontend owner 规范与共享 fixture 已同步 |

重现浏览器：隔离 Vite 上，在 `web/` 使用 `PERSISTTY_E2E_BASE_URL=http://127.0.0.1:5175 PERSISTTY_E2E_PASSWORD=fixture PERSISTTY_E2E_PROJECT=fixture npm run test:e2e -- terminal-device-attributes.spec.ts terminal-scrolling.spec.ts workbench-interactions.spec.ts`。真实 tmux：`go test ./internal/terminal -run TestDeviceAttributesRealTmux -v`，需本机 tmux/Python；代码 fixture 不连接已有 socket、不执行接收字节为命令。完整 Go 门禁需允许 httptest 绑定本地端口；初轮沙箱禁止监听导致现有 HTTP 测试失败，允许隔离端口后完整门禁通过。

浏览器接口仍为合成 WS/HTTP，新真实 tmux 用例直接验证 Go runtime/PTY，未将其合并声称为真实端到端 WS 服务或 Debian/systemd/手机真机验收。此次新增 v3 文本帧需前后端共同更新：重启 Web 服务后刷新页面；未操作用户已有服务、tmux 或终端输入。代码/报告保持未提交，任务不归档。
