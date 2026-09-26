# 论坛频道

> 版本边界：本项目本次检查的 Trellis CLI 为 0.6.17。本文保留模板原有命令示例以便对照，执行前以当前 `trellis channel <command> --help` 为准；包含 `--tag`、`forum list` 或 `thread show` 的旧示例不适用于当前 CLI。当前消息筛选使用 `--kind`，论坛列表为 `trellis channel forum <name>`，主题读取为 `trellis channel thread <name> <key>`。中断事件使用 `interrupt_requested` / `interrupted`；不要把旧文案中的 `interrupt` 当作当前事件名。

论坛频道是持久、按主题组织的频道。创建频道时通过 `--type forum` 指定，之后不可更改。它不是普通聊天流，默认阅读路径为：
**论坛摘要 -> 一个主题的时间线 -> 当前上下文**。

## 论坛与普通频道

频道类型由 `channel create` 的 `--type` 设置，之后永不改变：

- `chat`（默认）— 平铺的消息时间线。`channel messages` 始终渲染事件流。这里拒绝 `--thread` 和 `--action` 等论坛专用参数。
- `forum` — 面向主题。不带筛选条件的 `channel messages` 渲染主题看板摘要，而非原始事件。`post`、`forum`、`thread` 和 `thread rename` 子命令仅适用于论坛频道。

两种类型共享相同的范围模型（默认 `--scope project`；`--scope global` 将频道放入跨项目分区）。

## 创建论坛频道

```bash
trellis channel create design-feedback \
  --type forum \
  --scope global \
  --description "Cross-project design feedback board." \
  --context-raw "One thread per design topic; close when resolved." \
  --by main
```

单仓库看板使用 `--scope project`，跨项目看板使用 `--scope global`。

## 主题：打开、评论、状态、摘要

主题位于论坛频道内。每个主题由稳定的 `--thread <key>` 标识（惯例是小写 kebab-case）。主题的第一个操作是 `opened`；之后所有操作使用同一个 `--thread` 键。

```bash
trellis channel post design-feedback opened \
  --scope global \
  --as main \
  --thread login-empty-state \
  --title "Empty state on the login screen" \
  --description "Track design feedback for the new login empty state." \
  --labels design,login \
  --context-raw "Spotted during the 0.4 release review." \
  --text-file /tmp/thread-open.md

trellis channel post design-feedback comment \
  --scope global \
  --as reviewer \
  --thread login-empty-state \
  --text-file /tmp/review.md

trellis channel post design-feedback status \
  --scope global \
  --as main \
  --thread login-empty-state \
  --status closed

trellis channel post design-feedback summary \
  --scope global \
  --as main \
  --thread login-empty-state \
  --summary "Adopted the option-B layout; ticket TRELLIS-123 owns the fix."
```

关键区别：

- `--description` 是**持久的**主题描述（回答“这个主题讨论什么？”）。在 `opened` 时设置，再次运行带 `--description` 的 `post` 可编辑它。
- `--text` / `--stdin` / `--text-file` 是**事件正文**，即附在这一条时间线记录上的评论或载荷。
- `--labels` 和 `--assignees` 是 CSV，会**替换**当前值，不会追加。
- `--summary` 是持续更新的主题摘要。在 `status closed` 时设置摘要，是附带上下文标记主题已解决的标准方式。

除 `opened` 外，每个操作都要求 `--thread`（实际上 `opened` 也需要它，因为不存在匿名主题）。

## 阅读论坛

```bash
trellis channel messages design-feedback --scope global
trellis channel forum design-feedback --scope global --status open
trellis channel thread design-feedback login-empty-state --scope global
trellis channel messages design-feedback --scope global --raw --thread login-empty-state
```

如果协作代理说“我在论坛发表了评论”，先运行 `channel forum` 查看哪个主题发生变化，再用 `channel thread <name> <thread>` 深入该主题。不要直接临时解析 `events.jsonl`。

## 上下文

上下文条目是持久背景，阅读频道或主题时应始终纳入范围。它们**不是**时间线事件，而是单独投影，并为每个读者重放。

使用 `context` 子命令。`create` 和 `post` 上旧的 `--linked-context-file` / `--linked-context-raw` 参数是已弃用的别名，会归并到标准的 `--context-file` / `--context-raw`。

### 添加上下文

```bash
# 频道级上下文（整个论坛）
trellis channel context add design-feedback \
  --scope global \
  --raw "Upstream feedback board; please link tasks before opening threads."

# 主题级上下文（一个主题）
trellis channel context add design-feedback \
  --scope global \
  --thread login-empty-state \
  --file "$PWD/.trellis/tasks/05-13-login-redesign/design.md"
```

- `--thread <key>` 用于切换频道级和主题级上下文。
- `--file` 路径**必须是绝对路径**；相对路径会被拒绝。
- `--raw` 是内联纯文本内容。
- 两个参数都可重复；`add` / `delete` 至少需要一个。
- `--as <agent>` 记录作者身份，默认 `main`。

### 列出上下文

```bash
trellis channel context list design-feedback --scope global
trellis channel context list design-feedback --scope global --thread login-empty-state --raw
```

`list` 的 `--raw` 每行输出一个 JSON 条目（便于管道处理）；不带它时，输出可读的 `file <path>` / `raw <truncated text>` 列表。空存储输出 `(no context)`。

### 删除上下文

```bash
trellis channel context delete design-feedback \
  --scope global \
  --thread login-empty-state \
  --raw "stale note"
```

删除依据是**值**，不是 id：传入添加时相同的 `--file` 或 `--raw` 值。重复参数可在一次调用中删除多个条目。

### 阅读顺序

阅读主题时，自上而下进行：

1. 主题 `description`（持久的“这讨论什么”说明）。
2. 上下文条目（频道级 + 主题级）。
3. 时间线（`opened`、`comment`、`status`、`summary`）。

如果上下文文件缺失或不可读，明确说明，并继续处理剩余数据；不要编造内容。

## 标题投影

`title` 为频道投影稳定的显示标题，不会重命名存储地址。传给所有命令的频道 `name` 保持不变。

```bash
trellis channel title set design-feedback \
  --scope global \
  --title "Design feedback board"

trellis channel title clear design-feedback --scope global
```

- `title set` 要求 `--title`。
- `--as <agent>` 记录作者身份，默认 `main`。
- 这是展示层变更。工具和脚本继续使用原频道名称。

## 主题重命名

当主题创建时使用了错误键（拼写错误、slug 约定错误等），`thread rename` 是修正路径。主题不支持硬删除，重命名是受支持的修正操作。

```bash
trellis channel thread rename design-feedback old-key new-key \
  --scope global \
  --as main
```

- `--as <agent>` **必填**。
- `post <name> rename` 会被拒绝，必须使用 `thread rename`。

## 删除规则

不要把单条评论删除或主题硬删除视为常规工作流。论坛主题是仅追加的协作历史。修正状态时使用：

- `post ... status` 将主题标记为关闭、阻塞等状态。
- `post ... summary` 记录解决结果。
- `post ... --labels` 重新设置标签（替换整个集合）。
- `thread rename` 修正错误的主题键。

## 内部变更日志模式

全局论坛频道常用于内部发布/运行时变更日志。每项重要变更建立一个主题，便于检索历史：

```bash
trellis channel create release-notes \
  --type forum \
  --scope global \
  --description "Internal release and runtime changelog." \
  --context-raw "One thread per notable change; close when shipped." \
  --by main

trellis channel post release-notes opened \
  --scope global \
  --as main \
  --thread release-2026-q1 \
  --title "Channel threads and forum UX in 0.6" \
  --description "Forum channel UX shipped in the 0.6 line." \
  --labels channel,release \
  --text-file /tmp/release-notes.md
```

使用稳定、有描述性的主题键（如 `release-2026-q1`、`runtime-event-schema-change`），便于后续读者按名称找到主题。
