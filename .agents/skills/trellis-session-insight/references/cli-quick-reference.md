# `trellis mem` CLI 参考

五个子命令的参数参考。实际可用能力以本地 `trellis mem --help` 为准；文档与运行时帮助不一致时，应记录版本差异而非按过时说明推断。

> 本次检查的本地 CLI 为 0.6.17，帮助已列出 OpenCode 和 `--include-children`。下方关于 `0.6.0-beta.*` 占位读取器的说明仅描述旧版本；不能据此判断当前版本不支持 OpenCode。本次只检查帮助，未读取 OpenCode 存储或验证该适配器运行结果。

## 子命令

| 命令 | 用途 |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `list` | 列出会话。不指定子命令时的默认项。 |
| `search <keyword>` | 查找内容匹配关键词的会话。 |
| `context <session-id>` | 深入查看单个会话：前 N 个命中轮次及周围上下文。配合 `--grep` 定位关键词。 |
| `extract <session-id>` | 导出清理后的对话。结合 `--phase` / `--grep` 切片。 |
| `projects` | 列出活动项目的 `cwd` 与会话数量。用于确定其他子命令应传入哪个 `--cwd`。 |

## 参数（在有意义的子命令中适用）

| 参数 | 子命令 | 含义 |
| --------------------------------------------- | ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--platform claude\|codex\|devin\|grok\|opencode\|pi\|zcode\|all` | 全部 | 默认 `all`。`devin` 指 Cognition Devin CLI（`sessions.db`），并非 `trellis init --devin`（桌面版）。 |
| `--since YYYY-MM-DD` | list / search | 包含边界的起始日期。 |
| `--until YYYY-MM-DD` | list / search | 包含边界的截止日期。 |
| `--global` | list / search | 包含本机所有项目的会话。默认仅当前项目 `cwd`。 |
| `--cwd <path>` | list / search | 强制指定项目 cwd，而非从当前位置推断。 |
| `--limit N` | list / search | 限制输出行数，默认 `50`。 |
| `--grep KW` | extract / context | 按关键词过滤轮次。空白分隔的多个词采用 AND 匹配。 |
| `--phase brainstorm\|implement\|all` | extract | 按 Trellis 任务边界切片。`brainstorm` = `[task.py create, task.py start)`；`implement` = 讨论窗口外的轮次。默认 `all`。 |
| `--turns N` | context | 返回的命中轮次数，默认 `3`。 |
| `--around N` | context | 每次命中包含的周围轮次数，默认 `1`。 |
| `--max-chars N` | context | 总字符预算，默认 `6000`（约 1500 token）。 |
| `--include-children` | search / context | 将 OpenCode 子代理会话合并到父会话。 |
| `--json` | 全部 | 输出机器可解析的 JSON，而非人类可读输出。 |

## 常用单行命令

```bash
# 本机哪些历史会话讨论过“deadlock”？
trellis mem search "deadlock" --global --limit 20

# 在指定会话中展示提及“lock contention”的前 5 个轮次，
# 以及各命中前后 2 个轮次的上下文。
trellis mem context 5842592d --grep "lock contention" --turns 5 --around 2

# 恢复会话的需求讨论窗口，适合继续用户一周前启动的任务。
trellis mem extract 5842592d --phase brainstorm

# 列出本机具有 Trellis 会话的所有项目及其数量。
trellis mem projects
```

## 输出形式

- **默认人类可读输出**（不带 `--json`）：按终端宽度换行，突出会话 id 并显示轮次标记。适合直接阅读，但粘贴到 Markdown 文件时可能杂乱。
- **`--json`**：schema 稳定，适合安全解析与处理。将 `mem` 输出通过管道交给后续步骤时（例如总结经验章节），优先使用 `--json`。

## 注意事项

- **`0.6.0-beta.*` 的 OpenCode 适配器是占位实现。** 当 `--platform` 解析为 OpenCode（或为 `all` 且会包含 OpenCode）时，`mem` 输出一行“读取器不可用”提示，并继续其他平台。适配器发布前，不要在回复中承诺支持 OpenCode。
- **`--platform devin` 是 Cognition Devin CLI**（`~/.local/share/devin/cli/sessions.db`），不是 `trellis init --devin`（Devin Desktop / Cascade），也不是 Factory Droid。
- **`--phase` 切片依赖会话记录的 bash 调用中出现 `task.py create` / `task.py start`。** 若用户在 AI 记录过程外的另一个终端运行 `task.py`，该会话没有阶段边界。`--phase all` 是安全回退。
- **`mem` 直接索引平台 JSONL 文件。** 若用户清空了 Claude / Codex / Pi 会话存储，`mem` 无法恢复已不在磁盘上的内容。
- **`mem` 是只读工具。** 不做远程同步，不编辑平台 JSONL。根据发现执行的任何写入，都是你另行调用编辑工具的后续操作。

## 需要更多参考时

在用户 shell 中运行 `trellis mem help`。运行时帮助具有权威性，在快速变化的 beta 版本中可能领先于本文。
