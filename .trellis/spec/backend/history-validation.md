# 历史恢复与 TUI 机制实验

## 1. 范围与触发条件
W02 D07 对照 tmux capture 与真实 Go/PTY/WS attach，验证受控历史及 Python curses 当前画面。实验只处理自有固定合成数据，不提供用户终端日志功能，不冻结产品 snapshot/live 协议；headless 解析不能代替浏览器渲染或实时鉴权链路验收。

## 2. 签名
远端入口：`python3 -B tests/integration/debian/run_remote.py history --binary /tmp/bridge-probe`，只由 [私密连接 owner](remote-validation.md) 解析仓库根 .env。Go 的 record client 是 [独立 bridgego 模块](../../../tests/integration/debian/bridgego/README.md) 的实验扩展；解析与 fixture 属于 [history 模块](../../../tests/integration/debian/history/README.md)，独立 npm manifest/lock，不改变产品依赖。

## 3. 契约
只连接已有私有 tmux server，保留 D06 token/Origin、单 attach、取消/Wait、CLOEXEC 与限额契约。record 仅发送显式尺寸控制，不发送 shell 输入；尺寸 1..1000，采集最长 10 秒、256 KiB、4096 帧。原字节采用 base64 中间格式，只暂存已忽略的 owner-only 文件，不输出或提交；提交 hash、字节数、解析判定与脱敏资源身份。stdout 是受控实验数据通道，不能照搬为产品日志。

解析使用锁定 @xterm/headless、Uint8Array 与 write callback；normal/alternate buffer、cell 字符/宽度/颜色/样式与 cursor 分别比对。逐字节与原分块状态等价仅证明该有限记录，不涵盖任意 parser 截点。远端 curses resize 同时核对程序实际尺寸和重绘标记，WS resized 回复不是 TUI 已重绘的证明。

capture 导出 grid，不是完整 parser/mode 状态或原子 live 截点。tmux 外层 attach 可处于 alternate，即使 pane 的 curses 已退出；不能把外层 buffer 类型等同 pane alternate_on。capture -a 的 saved grid 与当前 TUI grid 必须通过实际对照判断，不能凭选项名称推断。禁止正则剥 ANSI、过滤切屏指令、按文本前缀猜测去重或固定 sleep 伪造无损恢复。

## 4. 验证与错误矩阵
| 场景 | 判定要求 |
| --- | --- |
| raw attach 到已有 200 行输出 | 当前画面可恢复不等于完整 scrollback 恢复，检查早期标记缺失 |
| capture 后继续 80 行 burst 再 attach | 检查两个 buffer 的间隙标记与 active 类型，遗漏是方案反例，不记产品通过 |
| curses 进入/resize/reconnect/退出 | actual pane 模式、程序尺寸、当前画面与固定身份一致，终端输入计数为零 |
| UTF-8/ANSI 任意分块 | 等待每次 write callback，原帧与逐字节解析状态相同 |
| record 超限/中断 | 明确失败且回收 attach，不终止原 pane、不提交 raw |
| runner 强制退出或未知 ROOT | 不能承诺零残留，精确核查已知资源，未知只读报告 |

## 5. 正常、基础、错误用例
正常：真实 curses 在 100x30、80x24、120x40 展示固定中文和颜色，重连后仍为原 pane。基础：仅解析一个固定合成帧序列。错误：只见 TUI marker 就宣称浏览器正确渲染，或将“检测到历史遗漏”的 true 检查项写成无损恢复通过。

## 6. 所需测试
根 Go 与 bridgego 模块分别 test/race/vet；record 限额与真实 PTY 回收回归。独立 npm ci/test 验证锁文件与真实库解析，Python 验证错误/输出边界/证据。最终 binary 独立 Debian 重跑应保存单独摘要证据，固定 server/pane PID/start/cgroup、逐次 heartbeat、实际 stdin 字节计数、精确 unit/scope 与 ROOT 清理；不得覆盖 D06 历史证据。

浏览器 renderer/字体/光标、实时认证 WS、多观察端、持续负载、生产 history 容量及 snapshot epoch/sequence/截点仍为门禁。确定性固定输出和本地库解析不自动解除这些门禁。

D08 后续 [快照机制实验](snapshot-validation.md) 对公开 serialize 做了单 owner/seq 对照；pending 序列、scroll-region/charset 及真实记录仍有状态差异，不能将其当作 D07 历史拼接问题的已通过替代方案。

## 7. 错误与正确示例
错误：`capture(history) + attach(bytes)` 被称为一次完整恢复，然后以 sleep 避免竞态。正确：记录 capture 与 attach 的非原子窗口，分别判断当前画面、滚动历史及间隙丢失，保存反例；产品协议只有在明确单输出 owner 和同步截点、重新同步策略经过真实验证后才定案。
