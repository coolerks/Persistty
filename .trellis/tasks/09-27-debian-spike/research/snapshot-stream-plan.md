# Research: D08 单输出源同步截点

- Query: 固定 xterm serialize 能否在任意 UTF-8/CSI/OSC 中间截点恢复后续流；如何做 sequence/ring 的确定性正负对照？
- Scope: mixed；只读现有隔离实验与官方固定 tag 源码，不安装、不远端执行、不读取 .env。
- Date: 2026-09-27

## Findings

### 当前文件与规范

- `tests/integration/debian/history/package.json:6`：已锁 `@xterm/headless@5.5.0`，Node 原生 test，无 serialize。
- `history/analyze.mjs:7`：await write callback、Uint8Array、allowProposedApi；现有输出比较 normal/alternate cells 与 active cursor。D08 应扩展比较 wrapped、两 buffer cursor、modes，并保留既有 D07 判定。
- `history/analyze.mjs:48`：capture+attach 仅为负面对照，不是恢复入口。
- `history-check-report.md`：D07 最终证据已证明普通 capture 与 attach 间隙丢失、外层 alternate 隐藏历史；serialize 未执行。7 次真实 WS 记录的分块解析等价不证明任意 snapshot 截点安全。
- `bridgego/record.go` 与 `history/probe.py`：已有受控 synthetic/curses 记录。可复用新一次有界记录作为输入，不必修改服务或在 Debian 安装 Node。
- 相关规范：backend 的 `terminal-lifecycle.md`、`websocket-protocol.md`、`bridge-validation.md`、`remote-validation.md`；frontend 的 `editor-terminal-lifecycle.md`。single owner、controller、生产恢复协议门禁保持。

### 官方源码/API 与兼容结论

1. [5.5.0 tag 的 addon manifest](https://raw.githubusercontent.com/xtermjs/xterm.js/5.5.0/addons/addon-serialize/package.json) 对应 `@xterm/addon-serialize@0.13.0` / MIT，peer 是 `@xterm/xterm:^5.0.0`，**不是 headless peer 声明**。隔离安装可能额外安装浏览器 peer，应保留实际 npm lock/完整许可证，不能为消除警告随意关闭校验，也不改 web manifest。
2. [SerializeAddon 源码](https://raw.githubusercontent.com/xtermjs/xterm.js/5.5.0/addons/addon-serialize/src/SerializeAddon.ts) 386 行接受 Terminal；454-477 行从 normal/alternate buffer 及若干 modes 生成字符串；374 行还访问 `_core._inputHandler._curAttrData`。源码有 headless fallback（503 行），说明作者考虑过 headless，但不等于所有状态承诺兼容。普通 `serialize()` 不需要 DOM；不要调用 HTML/selection 功能。**兼容候选需运行 headless loadAddon + serialize + restore smoke，不能只凭 typings 宣布通过。**
3. [addon typings](https://raw.githubusercontent.com/xtermjs/xterm.js/5.5.0/addons/addon-serialize/typings/addon-serialize.d.ts) 20-26 行要求同尺寸恢复后再 resize，45-65 行有 scrollback/excludeModes/excludeAltBuffer。实验使用有界 scrollback、不排除 alt/modes，恢复新实例 await write callback；resize 是序列事件而非 snapshot 的异步外部副作用。库仍是 experimental，完整发布包/许可证由实施者核实。
4. [WriteBuffer](https://raw.githubusercontent.com/xtermjs/xterm.js/5.5.0/src/common/input/WriteBuffer.ts) 149-203 行在本 chunk action 完成后调用 callback；它并不检查 UTF-8 interim 或 parser GROUND。因此 callback 是“已处理这个输入块”，不是“所有语义序列结束”。
5. [InputHandler](https://raw.githubusercontent.com/xtermjs/xterm.js/5.5.0/src/common/InputHandler.ts) 111-112、450-454 行使用跨调用解码器；[TextDecoder](https://raw.githubusercontent.com/xtermjs/xterm.js/5.5.0/src/common/input/TextDecoder.ts) 保存 interim；[EscapeSequenceParser](https://raw.githubusercontent.com/xtermjs/xterm.js/5.5.0/src/common/parser/EscapeSequenceParser.ts) 253-257、730-756 行持续维护状态/参数/OSC 内容。上述 pending 状态没有进入 addon 的公开 snapshot schema。推论：在中间帧 callback 后直接 serialize 新实例，再仅发送剩余字节，可能丢字符、把 CSI 后缀显示为文本或丢 OSC 事件。必须留负面回归，不把私有 `_core` 状态复制冒充公共恢复契约。

### 本轮最小实验

建议独立 `tests/integration/debian/snapshot/` Node 模块，保留 history 文件不变；锁 `@xterm/headless@5.5.0` + `@xterm/addon-serialize@0.13.0`（安装后核实 manifest、lock integrity、peer 和完整 MIT 通知），Node `--test`。无需浏览器、生产 Node sidecar、产品 API 或新 SSH 入口。

**输入与 owner：** 通过既有安全 runner 新一次获得同一真实 WS 连续受控记录，或暂以本地固定 fixture 起步。离线输入仅原 binary 帧/明确 resize 事件，不拼不同 attach 的帧为连续原任务流。一个 owner 串行处理每条事件：分配严格递增 seq，await byte write/执行 resize，再发布该事件已应用游标。snapshot 请求也进入同一串行队列，返回 `{epoch,through_seq,cols,rows,serialized}`。没有 await 间隙插入其他事件的隐含并发；未完成的应用事件不能被 snapshot through_seq 覆盖。

**序列语义：** 每个 owner 生命周期一个 epoch；seq 对 byte 和 resize 统一排序。snapshot through_seq 包含的事件不再回放；observer 仅接 `seq == expected`，旧 epoch 拒绝，同 epoch duplicate 忽略且零 xterm.write，gap 则先停止应用并请求 resync，不跨缺口继续解析。真实 WS记录的 seq 为本地 owner 序号，不得描述为 tmux 已提供 durable output offset。

**有限 ring：** 同时限制事件数与总 bytes，单事件超限单独拒绝；记录 oldest/newest 和 dropped 计数。若 desired seq 已被裁掉，明确返回 resync_required，不能悄悄跳至 newest。snapshot 后 tail 读取先验证 epoch 和 gap；确定性调度让 snapshot 生成期间 tail 填满 ring，验证 stale snapshot 不能直接接残缺 tail。resync 后比较 fresh snapshot + tail 与连续 owner 的最终状态；**若 pending 截点已证明不安全，resync 只测完整固定语义边界场景，明确并不解决任意时刻恢复。**

### 最少场景与成功标准

| 场景 | 对照与判定 |
| --- | --- |
| 基础兼容与普通 history | headless loadAddon/serialize 不依赖 DOM；完整 ASCII、SGR、中文、wrapped 行在同尺寸恢复后续完整语义帧，比较 cells/宽度/属性/wrapped、cursor、modes。 |
| alternate + resize | 完成 1049 enter 后截点，恢复后退出，再 resize；另在 resize 事件完成后截点。普通与 alternate 两 buffers 分别比较，不能只看 active text。 |
| UTF-8 中途 | prefix 为中文三个字节的前一或两字节，await callback 后 serialize；新实例只接 suffix，对照连续原实例。观测差异是成功反例；不得用 Buffer.toString 预先合并掩盖。 |
| CSI 中途 | prefix `ESC[31`，suffix `mX`；另 split 在 ESC 与 `[` 中间。比较 X 的位置/颜色及出现的多余字面文本。 |
| OSC 中途 | 固定安全 OSC title 内容拆在 BEL/ST 前或 ESC 与反斜杠之间；记录 onTitleChange 的合成事件及尾部普通文本，不更新 DOM，不发 clipboard/link 副作用。比较连续与恢复事件与 cells；只比较屏幕不足以发现 OSC 丢失。 |
| 重复/gap/epoch | 人为 duplicate、跳一 seq、旧 epoch/new epoch 从 1 起；断言拒绝/忽略发生在 write 前、无重复文本/resize；stale snapshot 不覆盖当前 owner。 |
| 慢 observer/ring | 小 event cap + byte cap 分别裁剪，gap 拒绝且无无限缓存；完整边界 resync 后与连续基线一致。随机负载不替代确定性 worst-case。 |
| 实际记录对照 | 对新同一连接真实 WS记录在帧边界及人工 byte split 截点运行矩阵；保留摘要/hash/版本/反例位置，不提交 raw正文。不预设每个真实帧边界都能无损恢复。 |

实际包行为允许发现更多失败，不应以“让测试通过”为目标过滤/补写控制序列。不得正则剥 ANSI、检查控制前缀猜 GROUND、读取私有 parser 状态来伪称完成公开恢复 API。fixture 可知道完整语义边界，但生产不能假设运输帧就是该边界。

## Caveats / Not Found

- research.jsonl 不存在；依照研究角色未读取 implement/check JSONL。已读当前 PRD/design/implement 与 D07 review。
- 尚未运行 serialize 实验或核实 addon 实际安装包；本报告兼容性为源码支持的候选，实施测试才给通过/失败。
- 离线 single owner + 多 observer 模型只验证顺序与状态对照，不是 Go live WS 服务、真实多浏览器或多控制端测试。
- 即使完整固定边界所有对照通过，任意 pending 截点、tmux capture 原子衔接、Web SIGKILL 后输出 owner/history 恢复、无网页期间输出与背压、完整终端状态/光标样式/区域/charset/自定义 OSC 都保持门禁；Addon 有限 modes 不是全状态保存。
- 不决定生产 Node sidecar、服务解析库或最终快照格式；不改变 D07 capture+attach 反例结论。负面 UTF8/CSI/OSC 证明即使 seq 无缺口也不代表 parser 恢复无损。
