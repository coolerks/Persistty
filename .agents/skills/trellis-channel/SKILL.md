---
name: trellis-channel
description: 使用 Trellis channel 进行实时多代理协作、启动 worker、跨代理评审、查看进度、管理论坛频道和调试频道日志。
---

# trellis-channel

> 本项目本次核对的 CLI 为 0.6.17：下文涉及 `--tag` 和保留标签的说明来自旧模板，不表示当前参数可用。当前 `send` / `messages` 帮助不提供 `--tag`，消息筛选使用 `--kind`；中断事件使用 `interrupt_requested` / `interrupted`。为保持示例对照，命令保留原文，执行前以本地 `--help` 为准。

`trellis channel` 是本地多代理协作运行时。代理需要通过持久事件日志沟通、需要将 worker 作为对等进程启动、需要中断或调试运行中的 worker，或需要将反馈记录到持久的 `--type forum` 频道时，使用它。

典型用户信号：“和 codex/claude 讨论”“与另一代理讨论方案”“启动 implement/check worker”“让代理评审”“开问题看板或变更日志论坛”“看看这个帖子”“channel 卡住了或没有输出”“进度被截断”“这个 channel 命令怎么写”。

本技能是索引。只加载当前工作需要的引用文件，不要预先加载全部文件。

## 首先运行的命令

```bash
trellis --version
trellis channel --help
trellis channel list --all
trellis channel list --scope global --all
```

若用户指出频道或帖子，先查看再询问背景：

```bash
trellis channel forum <board> --scope global
trellis channel thread <board> <thread> --scope global
trellis channel context list <board> --scope global --thread <thread>
```

## 按用户意图路由

| 用户意图 | 读取 |
|---|---|
| “和 codex/claude 讨论一下”“与另一代理讨论方案” | `references/workflows.md` |
| “派一个 implement/check agent”“让代理评审”“启动 worker” | `references/workflows.md`，然后 `references/workers.md` |
| “开 issue 区 / topic 群 / changelog / board”“建论坛” | `references/forum.md` |
| “看看这个 thread / linked context”“查看帖子” | `references/forum.md` |
| “channel 卡住了 / 没输出 / progress 被截断”“worker 停滞” | `references/progress-debugging.md` |
| “具体命令怎么写”“X 接受哪些参数” | `references/command-reference.md` |

## 核心规则

- 新论坛频道使用 `--type forum`。`thread` 是论坛频道中的一个条目。
- 使用 `--context-file` / `--context-raw` 与 `trellis channel context add/delete/list`。`--linked-context-*` 是已弃用术语。
- 长消息使用 `--stdin` 或 `--text-file`。不要把长篇中英混合文本放在 shell 的位置参数中。
- 格式化的 `messages` 输出是操作仪表板，可能截断进度。审计使用 `--raw`。
- 依命令不同，`--as` 表示发言者或 worker 标识。涉及多个代理或会话时，使用明确、稳定的名称。
- `--scope project`（默认）操作当前 cwd 的项目桶；`--scope global` 操作共享 `__global__` 桶。明确选择范围；不传 `--scope global` 时，项目列表看不到全局看板。
- 需求讨论应进行多轮压力检验。一次回答加一次确认属于评审，不是充分讨论。
- **分派者等待模式**：使用 `--kind done` / `--kind turn_finished`（Trellis 发出的系统事件），不要以用户 `--tag` 作为完成信号。CLI 帮助将 `phase_done` / `question` 列为 `--tag` 示例，但只有 `interrupt` 是带有硬编码 Trellis 行为的保留标签；其他只是任意用户标签。依赖 worker 运行 `send --tag <my_signal>` 不可靠，LLM worker 经常把标签字符串写进正文，而未实际执行 CLI 命令。参阅 `references/command-reference.md` 的“tag 与 kind”说明。
- 论坛频道采用事件溯源。不要首先解析 `events.jsonl`；使用 `forum`、`thread`、`messages --thread` 与 `context list`。
- `@mindfoldhq/trellis-core` 负责可复用的频道/帖子状态、事件追加、seq 分配、上下文与标题投影、reducer 和任务辅助工具。CLI 负责参数、终端渲染、提示词、worker 生命周期与进程退出。

## 引用文件

- `references/workflows.md` — 标准协作模式 A–F（对等讨论、启动评审、分派并等待、论坛问题记录、中断并重定向、单次运行）。
- `references/forum.md` — 论坛频道、上下文、标题、重命名、变更日志论坛和帖子过滤。
- `references/workers.md` — 启动、代理卡、上下文注入（`--file` / `--jsonl`）、中断与 kill 语义。
- `references/progress-debugging.md` — 进度与原始输出查看、停滞 worker 诊断、OOM 防护与退出码。
- `references/command-reference.md` — 当前 CLI 命令参考（全部子命令、参数、输出约定和范围/类型模型）。

## 不适用情况

- 一份 Markdown 文件加提示词已足够的单次静态评审。
- 用自我日志记录替代正常工具调用。
- 长期记忆检索。可行动问题使用持久论坛频道，会话或历史搜索使用 `trellis mem`（`trellis-session-insight` 技能）。
