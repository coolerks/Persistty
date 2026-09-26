# 命令参考

这是 `trellis channel` 子命令的当前权威参考，已对照 `packages/cli/src/commands/channel/` 的源码验证（`index.ts` 中的 Commander 接线和各子命令处理器）。

除非另有说明，每个子命令都接受 `--scope <project|global>`；默认 `project`，根据当前 cwd 解析到对应项目分区。

## 顶层

```
trellis channel <subcommand>
```

> 多代理协作运行时：通过共享事件日志启动、协调、中断工作代理。

---

## 创建 / 列出

### `create <name>`

```bash
trellis channel create <name>
  [--scope project|global]                # 默认：project
  [--type chat|forum]                     # 默认：chat
  [--task <path>]                         # 关联的 Trellis 任务目录
  [--project <slug>]
  [--labels a,b,c]
  [--description <text>]                  # 稳定的频道描述
  [--context-file <abs-path>] ...         # 可重复
  [--context-raw  <text>]      ...        # 可重复
  [--linked-context-file <abs-path>]      # [已弃用别名]
  [--linked-context-raw  <text>]          # [已弃用别名]
  [--cwd <path>]                          # 记录在 create 事件中
  [--by <agent>]                          # 默认：main
  [--force]                               # 覆盖现有频道
  [--ephemeral]                           # 默认列表隐藏，可清理
```

行为：

- 追加 `create` 事件；`type` 不可变（之后不能在 forum↔chat 之间切换）。
- `--ephemeral` 频道默认不出现在 `channel list` 中，是 `channel prune --ephemeral` 的清理对象。
- `--linked-context-*` 归并到 `--context-*`；使用时发出弃用通知。

### `list`

```bash
trellis channel list
  [--scope project|global]
  [--json]
  [--project <slug>]                      # 对 task 字段进行子串匹配
  [--all]                                 # 包含临时频道（后缀 '*'）
  [--all-projects]                        # 扫描每个项目分区
```

行为：

- 默认范围是当前 cwd 所属项目。`--all-projects` 扫描所有分区。
- 格式化模式输出 `NAME WORKERS EVENTS LAST KIND TYPE TASK`，按最近活动排序，页脚注明隐藏的临时频道数量。
- `--json` 切换为 JSON 数组。

---

## 聊天消息

### `send <name> [text]`

```bash
trellis channel send <name> [text]
  --as <agent>                            # 必填 — 作者
  [--scope project|global]
  [--to <agents,csv>]                     # 默认：广播
  [--stdin | --text-file <path>]          # 正文来自 stdin 或文件
  [--delivery-mode appendOnly|requireKnownWorker|requireRunningWorker]
```

行为：

- 正文优先级：位置参数 `[text]` → `--stdin` → `--text-file`。
- `--to` 只有一个条目时存为字符串，多个时存为数组，省略时表示广播。
- `--delivery-mode` 选择定向投递验证：
  - `appendOnly`（近似默认行为，仅记录），
  - `requireKnownWorker`（具名目标必须存在 `spawned` 事件），
  - `requireRunningWorker`（工作代理当前必须存活）。
- 将追加事件在 stdout 上打印为一行 JSON。

> **注意：** `send` **没有** `--tag` 或 `--kind` 参数。参见下文 [`tag-vs-kind`](#tag-vs-kind--事件形状的实际控制方式)。

### `messages <name>`

```bash
trellis channel messages <name>
  [--scope project|global]
  [--raw]                                 # 每行一个 JSON 事件
  [--follow]                              # 流式读取新事件
  [--last <N>]                            # 最近 N 个匹配事件
  [--since <seq>]                         # seq > N（序号大于 N）
  [--kind <kind>]                         # CHANNEL_EVENT_KINDS 中的一项
  [--from <csv>]                          # 作者筛选
  [--to <target>]                         # 路由目标筛选
  [--thread <key>]                        # 仅限论坛
  [--action <thread-action>]              # 仅限论坛
  [--no-progress]                         # 隐藏进度事件
```

行为：

- 自动检测论坛频道：不带筛选时渲染主题看板，而不是事件流。`--thread` / `--action` 仅供论坛使用，用于聊天频道时会报错。
- `--kind` 根据 `CHANNEL_EVENT_KINDS` 验证（单值，不是 CSV；CSV 属于 `wait`）。

### `wait <name>`

```bash
trellis channel wait <name>
  --as <agent>                            # 必填 — 筛选上下文中的自身
  [--scope project|global]
  [--timeout <Ns|Nm|Nh|Nms>]              # 由 parseDuration 解析
  [--from <a,b>]                          # 作者 CSV
  [--kind <k1,k2>]                        # CSV，OR 语义
  [--thread <key>]                        # 论坛筛选
  [--action <thread-action>]              # 论坛筛选
  [--to <target>]                         # 默认：自身代理（广播 + 发给自己）
  [--include-progress]                    # 进度事件也会唤醒
  [--all]                                 # 要求每个 --from 都匹配
```

行为：

- 流式输出匹配事件，每行一个 JSON。
- 默认 `--to` 筛选为调用者自身代理（广播仍匹配，即广播 + 明确发给自己）。
- `--all` 要求 `--from`，并阻塞直到列出的每个代理都产生匹配事件。
- **超时以 124 退出**；使用 `--all` 时，还向 stderr 打印 `timeout: still waiting on ...`。

---

## tag-vs-kind — 事件形状的实际控制方式

v0.6.0 频道 CLI 中任何位置都**没有 `--tag` 参数**；`--kind` 也不是任何 `--tag` 参数的旧别名。

当前源码中的具体模型：

- `--kind` 是唯一的事件类型筛选器，并受 Trellis 发出事件的白名单限制（`packages/core/src/channel/internal/store/events.ts` 中的 `CHANNEL_EVENT_KINDS`）：
  - `create`、`join`、`leave`、`message`、`thread`、`context`、`channel`、`spawned`、`killed`、`respawned`、`progress`、`done`、`error`、`waiting`、`awake`、`undeliverable`、`interrupt_requested`、`turn_started`、`turn_finished`、`interrupted`、`supervisor_warning`
  - 传入其他值会抛出 `Invalid --kind '<x>'. Must be one of: …`。
- `--kind` 位于 `wait`（CSV，OR 语义）和 `messages`（单值）。`send` 和 `run` 不能发出自定义 kind；每次 `send` 都写入 `message` 事件。
- 工作代理在轮次中途终止**不是**标签。专用命令 `channel interrupt` 会追加 `interrupt_requested` / `interrupted` 事件对，并在提供方层面中断工作代理。

分派者等待工作代理时的实用规则：

- 用 `--kind done,turn_finished` 表示“工作代理完成一轮”；这些是监督进程自动发出的系统事件，不依赖工作代理 LLM 记得发出任何自定义信号。
- 只有确实需要轮次中途终止时，才使用 `trellis channel interrupt` 命令。
- **不要**发明用户侧标签作为完成信号。没有 `--tag` 筛选器；工作代理写入最终消息的自定义字符串，只是 `message` 事件中的文本，无法被 `wait` 匹配。

长正文始终通过 stdin 或文件传入：

```bash
trellis channel send T --as A --stdin < /tmp/message.md
trellis channel send T --as A --text-file /tmp/message.md
```

---

## 中断

### `interrupt <name> [text]`

```bash
trellis channel interrupt <name> [text]
  --as <agent>                            # 必填 — 调用者
  --to <agent>                            # 必填 — 目标工作代理
  [--scope project|global]
  [--stdin | --text-file <path>]
```

行为：

- 追加带 `reason: "user"` 和替代指令正文的 `interrupt` 事件；支持时，监督进程在提供方层面执行中断（Claude `/interrupt`、Codex 轮次取消）。
- 在 stdout 打印追加事件 JSON。

---

## 工作代理

### `spawn <name>`

```bash
trellis channel spawn <name>
  [--scope project|global]
  [--agent <agent-name>]                  # 加载 .trellis/agents/<name>.md
  [--provider claude|codex]               # 覆盖代理文件设置
  [--as <worker-name>]                    # 默认：代理名称
  [--cwd <path>]
  [--model <id>]
  [--resume <id>]                         # 通过会话/线程 id 恢复
  [--timeout <Ns|Nm|Nh>]                  # 到达时长后自动终止
  [--warn-before <Ns|Nm|Nh>]              # supervisor_warning 提前量
                                          # 默认 5m，0ms 禁用
  [--file <path>] ...                     # glob，可重复；注入内容
  [--jsonl <path>] ...                    # Trellis 清单，可重复
  [--by <agent>]                          # 启动事件作者
                                          # 默认：TRELLIS_CHANNEL_AS 环境变量或 'main'
  [--inbox-policy explicitOnly|broadcastAndExplicit]
                                          # 默认 explicitOnly
  [--idle-timeout <Ns|Nm|Nh>]             # OOM 保护的空闲 TTL
                                          # 默认 5m，0 禁用
  [--max-live-workers <n>]                # 启动时的存活工作代理预算
                                          # 默认 6，0 禁用
```

行为：

- 根据适配器注册表（`packages/cli/src/commands/channel/adapters/`）验证 provider；当前为 `claude`、`codex`。
- 工作代理保持收件箱空闲，直到第一次 `send --to <worker>`。
- 记录包含 `pid`、`provider`、`agent`、`files`、`manifests` 的 `spawned` 事件。
- OOM 保护优先级：CLI 参数 → 环境变量（`TRELLIS_CHANNEL_WORKER_IDLE_TIMEOUT`、`TRELLIS_CHANNEL_MAX_LIVE_WORKERS`）→ `.trellis/config.yaml#channel.worker_guard` → 内置默认值。

### `run [name]`

```bash
trellis channel run [name?]
  [--agent <name>]
  [--provider claude|codex]
  [--as <worker-name>]
  [--cwd <path>]
  [--model <id>]
  [--file <path>] ...                     # 可重复，glob
  [--jsonl <path>] ...                    # 可重复
  [--message <text> | --message-file <path> | --stdin]
  [--timeout <Ns|Nm|Nh>]                  # 默认 5m
```

行为：

- 单次运行。省略 name 时自动生成 `run-<hex>`。
- 创建临时频道（`createMode=run`），启动一个工作代理，发送提示词，等待 `done`，将最终助手文本打印到 stdout，成功后移除频道。失败时保留频道供检查，退出码为 1。

> `run` **没有** `--tag` 参数。通过监督进程发出的 `done` 事件检测完成。

### `kill <name>`

```bash
trellis channel kill <name>
  --as <agent>                            # 必填 — 工作代理名称
  [--scope project|global]
  [--force]                               # 立即 SIGKILL
```

行为：

- 默认路径：SIGTERM → 8 秒宽限 → 升级到 SIGKILL；需要 SIGKILL 时，CLI 写入 `killed` 事件，确保日志如实记录。
- 清理 `pid`、`worker-pid`、`config`、`spawnlock` 辅助文件；保留 `log`、`session-id`、`thread-id` 供取证 / 恢复。

### `rm <name>`

```bash
trellis channel rm <name>
  [--scope project|global]
```

行为：

- 终止所有存活工作代理，然后删除整个频道目录。
- 打印 `Removed channel '<name>'`。

### `prune`

```bash
trellis channel prune
  [--scope project|global]                # 省略：扫描所有项目
  [--all | --empty | --idle <Ns|Nm|Nh|Nd> | --ephemeral]   # 互斥
  [--yes]                                 # 实际删除（默认：dry-run）
  [--dry-run]                             # 默认 true；与默认值重复
  [--keep <names,csv>]                    # 排除列表
```

行为：

- 筛选参数互斥，否则报错。
- 默认 dry-run；`--yes` 切换为实际删除。
- 不带 `--scope` 时扫描**所有**项目分区（有意设计为仓库范围的清理）；带 `--scope project|global` 时仅限对应分区。
- 无论筛选条件如何，始终跳过有存活工作代理的频道。
- 输出：每个候选一行 `name  last-ts  (reason)`，最后输出摘要。

---

## 论坛频道

### `post <name> <action>`

```bash
trellis channel post <name> <action>
  --as <agent>                            # 必填
  [--scope project|global]
  [--thread <key>]                        # 除 action=opened 外必填
  [--title <text>]
  [--text <text> | --stdin | --text-file <path>]
  [--description <text>]                  # 稳定的主题描述
  [--status <status>]
  [--labels a,b]                          # 替换主题标签
  [--assignees a,b]                       # 替换负责人
  [--summary <text>]
  [--context-file <abs-path>] ...
  [--context-raw  <text>]      ...
  [--linked-context-file <abs-path>]      # [已弃用别名]
  [--linked-context-raw  <text>]          # [已弃用别名]
```

行为：

- CLI 接口中的 `<action>` 为自由格式；惯用值包括 `opened`、`comment`、`status`、`labels`、`assignees`、`summary`、`processed`。
- `action=rename` 会被拒绝，改用 `thread rename`。
- `--labels` / `--assignees` 使用替换语义，不是追加。
- 输出：stdout 上的追加事件 JSON。

### `forum <name>`

```bash
trellis channel forum <name>
  [--scope project|global]
  [--status <status>]
  [--raw]
```

行为：

- 列出主题（归约后的状态）。`--status` 按主题当前状态筛选。`--raw` 为每个主题打印一个 JSON。

### `thread <name> <thread>` / `thread rename`

```bash
trellis channel thread <name> <thread-key>
  [--scope project|global]
  [--raw]

trellis channel thread rename <name> <old-thread> <new-thread>
  --as <agent>                            # 必填
  [--scope project|global]
```

行为：

- `thread <name> <key>` 显示单个主题的时间线：标题行 `<thread> [<status>] <title>`，随后是 description / labels / assignees / summary / 时间线各行。`--raw` 切换到原始事件。
- `thread rename` 是唯一的修改操作；`post --action rename` 会被拒绝。

---

## 上下文 / 标题

### `context add` / `context delete` / `context list`

```bash
trellis channel context add <name>
  [--as <agent>]                          # 默认：main
  [--scope project|global]
  [--thread <key>]                        # 主题级，而非频道级
  [--file <abs-path>] ...                 # 可重复
  [--raw <text>]      ...                 # 可重复
                                          # --file 或 --raw 至少一个

trellis channel context delete <name>
  [--as <agent>]                          # 默认：main
  [--scope project|global]
  [--thread <key>]
  [--file <abs-path>] ...
  [--raw <text>]      ...

trellis channel context list <name>
  [--scope project|global]
  [--thread <key>]
  [--raw]                                 # 每行一个 JSON 条目
```

行为：

- `add` / `delete` 追加 `context` 事件并打印事件 JSON。
- `list` 投影当前上下文条目；格式化输出为 `file <path>` / `raw <truncated text>` 各行，空时为 `(no context)`。

### `title set <name>` / `title clear <name>`

```bash
trellis channel title set <name>
  --title <text>                          # 必填
  [--as <agent>]                          # 默认：main
  [--scope project|global]

trellis channel title clear <name>
  [--as <agent>]                          # 默认：main
  [--scope project|global]
```

行为：

- 追加 `title` 事件，为频道投影稳定的显示标题。输出为事件 JSON。

---

## 隐藏 / 内部命令

| 命令 | 用途 |
|---|---|
| `channel __supervisor <channel> <worker> <config>` | `spawn` 调用的派生入口。不要直接调用。 |
| `channel __parse-trace <adapter> <file>` | 开发辅助工具：通过匹配适配器重放录制的 stream-json / wire 追踪，并打印产生的频道事件。适配器根据提供方注册表验证。 |

---

## 事件模型

`CHANNEL_EVENT_KINDS`（由 `parseChannelKind` 强制执行的白名单）：

`create`、`join`、`leave`、`message`、`thread`、`context`、`channel`、`spawned`、`killed`、`respawned`、`progress`、`done`、`error`、`waiting`、`awake`、`undeliverable`、`interrupt_requested`、`turn_started`、`turn_finished`、`interrupted`、`supervisor_warning`。

`MEANINGFUL_EVENT_KINDS`（未显式传入 `--kind` 时，`wait` / `messages` 使用的默认可见子集）：

`create`、`join`、`leave`、`message`、`thread`、`context`、`channel`、`spawned`、`killed`、`respawned`、`done`、`error`。

不属于该子集的 kind（例如 `progress`、`waiting`、`awake`、`supervisor_warning`、`turn_*` / `interrupt*` 集合）仍进入存储；通过 `--kind` 或 `--include-progress` 主动选择查看。

论坛频道采用事件溯源；使用 CLI 归约器（`forum`、`thread`、`context list`）投影状态。

---

## 输出约定

- **修改操作**（`send`、`interrupt`、`post`、`context add/delete`、`title set/clear`、`thread rename`）在 **stdout** 上将追加事件打印为单行 JSON。
- **流式读取**（`wait`、`messages --follow`）在 stdout 上每行打印一个 JSON 事件。
- **格式化读取**（`list`、`messages`、`forum`、`thread`、`context list`）打印带颜色和对齐填充的表格 / 时间线。
- **`run`** 在 stdout 上仅打印最终助手文本（供调用者管道处理）；诊断信息写入 stderr。
- **错误**通过 `chalk.red("Error:")` 写入 stderr，并 `exit 1`。
- **`wait` 超时**专门以 **124** 退出。
