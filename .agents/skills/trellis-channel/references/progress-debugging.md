# 进度与调试

> 版本边界：本项目本次检查的 Trellis CLI 为 0.6.17。本文保留模板原有命令示例以便对照，执行前以当前 `trellis channel <command> --help` 为准；包含 `--tag`、`forum list` 或 `thread show` 的旧示例不适用于当前 CLI。当前消息筛选使用 `--kind`，论坛列表为 `trellis channel forum <name>`，主题读取为 `trellis channel thread <name> <key>`。中断事件使用 `interrupt_requested` / `interrupted`；不要把旧文案中的 `interrupt` 当作当前事件名。

格式化输出供操作人员阅读，原始输出用于审计。子命令（`forum`、`thread`、`messages`、`context`）是审计*接口*；应先使用它们，再考虑手动 grep `events.jsonl`。

## 格式化输出与 `--raw`

`trellis channel messages <channel>` 渲染紧凑、便于阅读的视图：时间戳、身份、kind 和简短正文。它供操作人员浏览频道，不用于诊断。

格式化输出可能且确实会截断：

- 较长的进度增量（`text_delta`、部分工具参数）
- 工具名称与命令行
- 多行状态字段与结构化 `detail` 数据块
- 超出列宽预算的论坛主题标题

当内容看起来异常，例如工作代理似乎卡住、进度行在词中间结束、action 字段显示 `...`，切换到 `--raw`。原始模式按 `events.jsonl` 中的原貌每行输出一个 JSON 事件，不丢弃任何内容。

```bash
# 格式化输出（操作人员视图）
trellis channel messages <channel> --kind done --last 10
trellis channel messages <channel> --kind error --last 10

# 原始输出（诊断视图）— 每行一个 JSON
trellis channel messages <channel> --raw --kind progress --last 20
trellis channel messages <channel> --raw --last 50
```

经验规则：绝不要根据被截断的进度行诊断工作代理。

### 重建流式文本

要重建模型在某一轮实际流式输出的文本，拼接进度事件中的 `detail.text_delta`：

```bash
trellis channel messages <channel> --raw --kind progress --last 80 \
  | python3 -c 'import json,sys; [print((json.loads(l).get("detail") or {}).get("text_delta",""), end="") for l in sys.stdin if l.strip()]'
```

## 工作代理停滞诊断

症状：`trellis channel list` 显示工作代理正在运行，但 `messages` 没有新事件，`wait` 一直超时。

排查顺序：

1. **定位频道文件。** 如果不确定频道所在分区，使用 `list --all --all-projects`。

   ```bash
   trellis channel list --all --all-projects
   CHAN=~/.trellis/channels/<bucket>/<channel>
   ```

2. **确认监督进程与工作代理 PID 仍存活。**

   ```bash
   cat "$CHAN/<worker>.pid"            # 监督进程 PID
   cat "$CHAN/<worker>.worker-pid"     # 实际 CLI 子进程 PID
   ps -p "$(cat "$CHAN/<worker>.pid")"
   ps -p "$(cat "$CHAN/<worker>.worker-pid")"
   ```

   如果监督进程 PID 已不存在，但频道仍列出工作代理，就是幽灵条目；使用 `trellis channel kill <name> --as <worker> --force` 清理。

3. **跟踪工作代理日志尾部。** 这是查看未进入频道的提供方 / MCP / 工具启动输出的权威位置。

   ```bash
   tail -f "$CHAN/<worker>.log"
   ```

4. **检查最近的原始事件。** 发出了 `progress` 却没有 `message`/`done` 的工作代理，通常仍在流式输出或被工具调用阻塞：

   ```bash
   trellis channel messages <channel> --raw --last 50
   ```

常见“存活但沉默”的原因：

- 提供方在第一个 token 前冷启动（耗时较长，但最终会推进）。
- 启动时 MCP 服务器阻塞，可在工作代理日志中看到。
- 工作代理正在等待工具结果，而工具子进程已挂起。
- 提示词过大 / 模型受速率限制；检查工作代理日志中的提供方错误。

## 解读进度事件

`progress` 事件代表正在进行的一项工作。形状随 `action` 字段变化，但关键字段始终位于 `detail` 下：

- `detail.text_delta` — 模型输出增量（跨事件拼接可重建流式回复）。
- `detail.tool_name`、`detail.tool_input` — 即将运行或正在运行的工具调用。
- `detail.status` — 长时间操作使用的短字符串（`starting`、`running`、`flushing`、`done`）。
- `detail.action` — 语义标签（如主题心跳使用 `status`）。

进度事件按设计具有**较多噪声**。除非传入 `--include-progress`，否则 `wait` 会忽略它们。确实需要查看时，优先使用：

```bash
trellis channel messages <channel> --raw --kind progress --last 80
```

持续稳定地产生进度，却始终不以 `done`/`error`/`message` 结束，是工具调用挂起的典型表现；检查工作代理日志中的子进程信息。

## 等待语义速查

`channel wait` 从 EOF 开始监视 `events.jsonl`，在以下事件到达时唤醒：

- `message`
- `done`
- `error`
- `killed`
- `progress`，仅当带 `--include-progress`

常用筛选条件：

```bash
trellis channel wait T --as main --from check --kind done --timeout 15m
trellis channel wait T --as main --from check,check-cx --kind done --all --timeout 15m
trellis channel wait T --as worker --tag interrupt --timeout 1h
trellis channel wait T --as main --thread release-note --action status --timeout 10m
```

退出码：`0` 表示匹配，`124` 表示超时，`1`/`2` 表示错误。`wait --all` 超时时，stderr 会列出尚未满足条件的工作代理。

## 审计 `events.jsonl`：使用子命令，避免 `grep`

每个频道在 `$CHAN/events.jsonl` 保存完整历史。调试时直接对文件使用 `tail` / `grep` / `jq` 很方便，但不要形成习惯，并且**绝不要**对论坛频道这样做。

优先子命令的原因：

- `messages` 已能通过筛选条件（`--kind`、`--from`、`--last`、`--tag`、`--thread`、`--action`）重放文件，并通过 `--raw` 提供准确 JSON。你想用单行脚本完成的操作，`messages` 已经支持。
- `wait` 以 EOF 语义消费同一文件；用 `tail -f | jq` 重新实现，会在高负载下丢失事件，并在日志轮转时打乱顺序。
- `context` 实体化工作代理的收件箱视图，包括游标状态。自行编写的筛选不会遵循 `<worker>.inbox-cursor`。

### 论坛频道：绝不直接解析 `events.jsonl`

论坛频道将多个逻辑主题复用到同一个 `events.jsonl`。每个事件携带 `thread`、`action` 和标签字段，论坛子命令知道如何归并它们。手动解析文件会：

- 混合不同主题，使某个主题显得不连贯。
- 遗漏主题生命周期事件（打开 / 状态 / 关闭），而这些事件会改变后续事件的解释方式。
- 忽略工作代理收件箱游标，导致你“看到”工作代理已经消费过的事件，却以为它们仍待处理。

改用理解论坛语义的视图：

```bash
# 列出论坛频道内的逻辑主题
trellis channel forum list <channel>

# 完整检查一个主题
trellis channel thread show <channel> <thread>

# 重放主题消息（支持 --raw、--kind、--last）
trellis channel messages <channel> --thread <thread> --raw --last 100

# 指定工作代理仍待处理的内容
trellis channel context <channel> --as <worker>
```

只有怀疑 CLI 本身有问题时才直接读取 `events.jsonl`，例如确认事件确实已持久化，或调试监督进程时与 `<worker>.inbox-cursor` 比较。

## 常见故障

| 症状 | 原因 | 修复 |
|---|---|---|
| `trellis: command not found` | CLI 未全局安装 | `npm install -g @mindfoldhq/trellis` |
| `wait` 立即退出 | 筛选错误或身份冲突 | 使用不同的 `--as`，检查原始消息 |
| zsh 对消息文本报错 | shell 解释了标点 | 使用 `--stdin` 或 `--text-file` |
| 进度行被截断 | 格式化输出截断 | 使用 `messages --raw --kind progress` |
| 工作代理始终不发言 | 提供方启动 / 提示词 / MCP 延迟 | 检查 `<worker>.log`、`ps` 和原始事件 |
| 换 cwd 后找不到频道 | 项目分区不匹配 | `cd` 到项目，使用 `--scope global` 或 `list --all-projects` |
| 列表出现幽灵工作代理 | 监督进程退出但未清理 | `trellis channel kill <name> --as <worker> --force` |
| 论坛主题看起来混乱 | 直接解析了 `events.jsonl` | 使用 `forum`、`thread`、`messages --thread` |

## 存储布局

```text
~/.trellis/channels/
└── <bucket>/
    └── <channel-name>/
        ├── events.jsonl
        ├── <channel>.lock
        ├── <worker>.log
        ├── <worker>.pid
        ├── <worker>.worker-pid
        ├── <worker>.config
        ├── <worker>.session-id
        ├── <worker>.thread-id
        ├── <worker>.inbox-cursor
        └── <worker>.spawnlock
```

代理通常应使用 CLI，而不是直接读取文件。当 CLI 视图不足时，直接读文件可用于调试；即便如此，也不要直接读取论坛频道的 `events.jsonl`。
