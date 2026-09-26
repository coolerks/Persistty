# 本地多代理频道运行时

`trellis channel` 是 Trellis CLI 附带的本地多代理协作运行时。它让主 AI 会话启动同级工作代理（Claude Code、Codex 或 `.trellis/agents/` 中的任意代理定义），通过事件日志交换持久消息，并协调审查或头脑风暴循环，无需手动拼接 Shell 管道。

本参考说明频道如何连接到用户项目，让定制项目的 AI 知道修改位置。运行时用法（命令、论坛与线程模式、工作代理启动参数）请参阅内置能力技能 `trellis-channel`。

## 本地系统模型

频道运行时跨越三个本地部分：

1. 用户主目录中的**存储层**：持久事件日志与工作代理状态文件。
2. 项目 `.trellis/agents/` 中的**代理定义**：由 `trellis channel spawn --agent <name>` 消费的平台无关角色卡。
3. `.trellis/config.yaml` 中的**项目配置**：工作代理保护阈值与其他频道选项。

## 核心路径

| 路径 | 用途 |
| --- | --- |
| `~/.trellis/channels/<project>/<channel>/events.jsonl` | 每个频道只追加的事件日志；序列锁保护，可安全重放。 |
| `~/.trellis/channels/<project>/<channel>/<channel>.lock` | 频道级写锁。 |
| `~/.trellis/channels/<project>/<channel>/<worker>.spawnlock` | OOM 保护使用的各工作代理启动锁。 |
| `~/.trellis/channels/<project>/<channel>/.seq` | 用于有序分配事件的序列辅助文件。 |
| `~/.trellis/channels/_global/<channel>/...` | 使用 `--scope global` 创建的频道；项目分组被共享键替换。 |
| `.trellis/agents/check.md` | `--agent check` 消费的默认检查代理角色定义。 |
| `.trellis/agents/implement.md` | `--agent implement` 消费的默认实施代理角色定义。 |
| `.trellis/config.yaml`（`channel.*` 块） | 工作代理保护阈值与频道默认值。 |

项目分组名称根据项目绝对路径生成（斜杠展开，非字母数字字符替换为 `-`），与 Claude Code 的 `~/.claude/projects/<sanitized-cwd>/` 约定一致。测试或沙箱场景可使用 `TRELLIS_CHANNEL_ROOT`（根目录）或 `TRELLIS_CHANNEL_PROJECT`（分组名称）覆盖。

## 何时使用频道运行时

频道比单次 Bash 调用或一次性子代理分派更重。仅在至少满足以下一个条件时使用：

- 工作需要**两个或更多代理进行多轮对话**（跨 AI 头脑风暴、同级审查、分派者与工作代理）。
- 工作代理应作为**同级进程**运行，主会话能中断、观察进度或异步等待它。
- 对话必须**持久保存且可在之后检查**（论坛与线程频道、问题看板、决策记录）。
- 多个工作代理必须**共享事件日志**，各自可看到其他代理的报告。

以下情况应优先选择开销更小的方式：

- 单次 Bash 命令或单次 Agent 工具调用足够 → 直接执行。
- 用户只需对一个文件做静态审查 → 读取文件并直接回复。
- 需求是“回忆上周讨论了什么” → 使用 `trellis mem`，而不是频道。

## 定制位置

| 需求 | 修改位置 |
| --- | --- |
| 修改默认频道工作代理空闲超时 | `.trellis/config.yaml` 中的 `channel.worker_guard.idle_timeout`；接受 `5m`、`30s` 等值，设为 `0` 禁用空闲清理。 |
| 修改存活工作代理数量预算 | `.trellis/config.yaml` 中的 `channel.worker_guard.max_live_workers`；设为 `0` 禁用启动时预算检查。 |
| 每次启动时覆盖工作代理保护规则 | 向 `trellis channel spawn` 传递 `--idle-timeout` / `--max-live-workers`，或设置环境变量 `TRELLIS_CHANNEL_WORKER_IDLE_TIMEOUT` / `TRELLIS_CHANNEL_MAX_LIVE_WORKERS`。 |
| 修改默认检查或实施工作代理的行为 | 编辑 `.trellis/agents/check.md` 或 `.trellis/agents/implement.md`。它们是平台无关角色卡；传入 `--agent check|implement` 时由频道运行时注入。 |
| 添加角色卡 | 将 `<name>.md` 放入 `.trellis/agents/`；`trellis channel spawn --agent <name>` 会读取它。 |
| 移动频道存储位置（CI 沙箱、临时运行） | 设置 `TRELLIS_CHANNEL_ROOT=/path/to/dir`。新频道事件随之迁移，已有频道保留在旧根目录。 |
| 切换存储范围 | 每个频道子命令传入 `--scope project`（默认）或 `--scope global`。仅改变分组目录，其他行为不变。 |

工作代理保护配置的优先级：CLI 参数 > 环境变量 > `.trellis/config.yaml` > 内置默认值。内置默认值为 `idle_timeout: 5m` 和 `max_live_workers: 6`。

## 与其他本地层的关系

- **工作流层**：使用频道分派的工作流（如 `channel-driven-subagent-dispatch`）要求主代理调用 `trellis channel spawn --agent check` 或 `--agent implement`，而不是平台子代理。如果缺少 `.trellis/agents/check.md` 或 `implement.md`，`trellis workflow --template <id>` 会在安装时给出非阻断警告。误删后可用 `trellis update` 恢复。
- **任务层**：频道工作代理不拥有任务状态。负责监督的主会话通过工作代理收件箱传递活动任务路径；工作代理从磁盘解析任务产物。
- **规范层**：工作代理和主会话一样读取 `.trellis/spec/`。频道运行时不会绕过规范上下文加载。
- **平台集成层**：频道运行时与平台无关，不依赖 `.claude/`、`.codex/` 或其他平台目录。标准化提供方输出的适配器（Claude `stream-json`、Codex `app-server`）位于 Trellis CLI 二进制内部，而非项目中。
- **平台子代理文件与频道工作代理**：编辑 `.claude/agents/trellis-implement.md`（以及其他平台 `.X/agents/` 目录中的同类文件）**不会**改变频道工作代理的行为；频道工作代理加载 `.trellis/agents/<name>.md`。平台特定代理文件用于主 AI 会话直接分派子代理，而不是频道启动的工作代理。各平台代理入口见 `platform-files/agents.md`，相关职责分离规则见 `trellis-meta/SKILL.md`。

## 运行时用法

命令语法、论坛与线程模式、工作代理句柄、进度检查，以及 `--kind done` / `--kind turn_finished` 分派者等待模式，请加载内置 `trellis-channel` 技能（`trellis init` / `trellis update` 后自动安装到各平台技能目录）。本参考只介绍本地文件布局与定制选项，不重复可能随版本改变的命令语法。
