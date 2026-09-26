# 代理

Trellis 代理文件定义专门角色。用户项目中常见的 Trellis 代理包括：

- `trellis-research`
- `trellis-implement`
- `trellis-check`

文件位置与格式因平台而异，但职责边界应保持一致。

## 代理职责

| 代理 | 职责 |
| --- | --- |
| `trellis-research` | 调研问题，将发现写入当前任务的 `research/`。 |
| `trellis-implement` | 按 `prd.md`、可选的 `design.md` / `implement.md`、`implement.jsonl` 与相关规范、研究实施。 |
| `trellis-check` | 审查变更、修复发现的问题并运行必要检查。 |

代理文件不应变成通用聊天提示。它们应定义输入来源、写入边界、是否可修改代码，以及如何报告结果。

## 常见路径

| 平台 | 代理路径 |
| --- | --- |
| Claude Code | `.claude/agents/trellis-*.md` |
| Cursor | `.cursor/agents/trellis-*.md` |
| OpenCode | `.opencode/agents/trellis-*.md` |
| Codex | `.codex/agents/trellis-*.toml` |
| Kiro | `.kiro/agents/trellis-*.json` |
| Gemini CLI | `.gemini/agents/trellis-*.md` |
| Qoder | `.qoder/agents/trellis-*.md` |
| CodeBuddy | `.codebuddy/agents/trellis-*.md` |
| Factory Droid | `.factory/droids/trellis-*.md` |
| Pi Agent | `.pi/agents/trellis-*.md` |
| Reasonix | `.reasonix/skills/trellis-*/SKILL.md`（子代理 frontmatter） |
| ZCode | `.zcode/agents/trellis-*.md` |
| Kimi Code | `.kimi-code/agents/trellis-*.md`（自定义子代理；相同提示也发布为 `.kimi-code/skills/trellis-*/SKILL.md`） |

GitHub Copilot 的代理与提示支持由 `.github/agents/`、`.github/prompts/` 和 `.github/skills/` 等目录共同提供；应检查用户项目中实际生成的文件。

Kilo、Antigravity 和 Devin 等主会话工作流平台可能没有 Trellis 子代理文件，通常依赖工作流与技能指导主会话。

## 两种上下文加载模式

### hook push（钩子推送）

平台钩子在代理启动前注入任务上下文。代理文件本身可更多关注职责与边界。

常见于支持代理钩子的平台。

### agent pull（代理主动读取）

代理文件要求代理启动后读取：

- `python3 ./.trellis/scripts/task.py current --source`
- `implement.jsonl` 或 `check.jsonl`
- JSONL 引用的规范与研究文件
- 当前任务的 `prd.md`
- `design.md`（如果存在）
- `implement.md`（如果存在）

此模式适用于钩子无法可靠改写子代理提示的平台。

## 本地变更场景

| 用户需求 | 修改位置 |
| --- | --- |
| 实施代理必须遵守额外限制 | 平台的 `trellis-implement` 代理文件。 |
| 检查代理必须运行项目特定命令 | `trellis-check` 代理文件，必要时也修改 `.trellis/spec/`。 |
| 研究代理必须输出固定格式 | `trellis-research` 代理文件。 |
| 代理无法读取任务上下文 | 代理前置指令或 `inject-subagent-context` 钩子。 |
| 添加项目特定代理 | 平台代理目录与相关工作流、命令或技能入口。 |

## 修改原则

1. **职责保持单一**。不要把 research、implement 与 check 的职责混在一个代理里。
2. **明确读取顺序**。代理必须知道先从活动任务开始，读取 JSONL 与规范上下文，再读取 `prd.md`、`design.md`（如果存在）、`implement.md`（如果存在）。
3. **明确写入边界**。research 通常只写 `research/`；implement 可写代码；check 可修复问题。
4. **多平台项目保持语义同步**。用户同时配置 Claude、Codex 和 Cursor 时，判断某个平台的代理变更是否也应应用到其他平台。

## 不要默认编辑上游模板

本地 AI 应默认修改用户项目内部的平台代理文件。只有用户明确希望将变更贡献给 Trellis 时，才讨论上游模板源码。
