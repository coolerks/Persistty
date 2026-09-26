# 技能、命令、提示与工作流

技能与命令是用户和 Trellis 交互的文本入口。不同平台使用不同名称，但核心目的一致：用户表达某种意图时，告诉 AI 如何进入 Trellis 流程。

## 概念差异

| 类型 | 触发方式 | 适用场景 |
| --- | --- | --- |
| skill（技能） | AI 自动匹配或用户显式提及 | 长期能力、工作流规则与修改指南。 |
| command（命令） | 用户显式调用 | continue、finish-work 等明确操作入口。 |
| prompt（提示） | 用户显式调用或平台选择 | 与命令类似，但使用平台提示格式。 |
| workflow（工作流） | 用户显式选择或平台自动匹配 | 没有子代理或钩子时指导主会话。 |

Trellis 工作流技能通常共享一套语义：brainstorm、before-dev、check、update-spec、break-loop。`trellis-meta` 等多文件内置技能使用分层引用。

## 常见路径

| 平台 | 常见入口 |
| --- | --- |
| Claude Code | `.claude/skills/`、`.claude/commands/` |
| Cursor | `.cursor/skills/`、`.cursor/commands/` |
| OpenCode | `.opencode/skills/`、`.opencode/commands/` |
| Codex | `.agents/skills/`、`.codex/skills/` |
| Kilo | `.kilocode/skills/`、`.kilocode/workflows/` |
| Kiro | `.kiro/skills/` |
| Gemini CLI | `.agents/skills/`、`.gemini/commands/` |
| Antigravity | `.agent/skills/`、`.agent/workflows/` |
| Devin | `.devin/skills/`、`.devin/workflows/` |
| Qoder | `.qoder/skills/`、`.qoder/commands/` |
| CodeBuddy | `.codebuddy/skills/`、`.codebuddy/commands/` |
| GitHub Copilot | `.github/skills/`、`.github/prompts/` |
| Factory Droid | `.factory/skills/`、`.factory/commands/` |
| Pi Agent | `.agents/skills/` |
| Reasonix | `.reasonix/skills/` |
| ZCode | `.zcode/skills/`、`.zcode/commands/` |
| Kimi Code | `.agents/skills/`、`.kimi-code/skills/`（命令以 `/skill:trellis-*` 技能交付） |

用户项目中，以 init 实际生成的文件为准。

## 技能结构

常见技能是一个目录：

```text
trellis-meta/
├── SKILL.md
└── references/
```

`SKILL.md` 应告诉 AI：

- 何时使用此技能。
- 针对当前任务先读取哪份引用文档。
- 哪些事情不能做。

引用文档承载较长说明，入口文件不需要包含全部内容。

## 命令、提示与工作流结构

命令、提示与工作流通常是单文件，内容应包括：

- 何时使用。
- 需要读取哪些 `.trellis/` 文件。
- 需要运行哪些脚本。
- 完成后如何报告。

它们不应存储任务状态；任务状态属于 `.trellis/tasks/` 和 `.trellis/.runtime/`。

## 本地变更场景

| 用户需求 | 修改位置 |
| --- | --- |
| 修改 AI 自动触发规则 | 对应技能 frontmatter 中的 description。 |
| 修改用户命令行为 | 对应命令、提示或工作流文件。 |
| 添加项目本地技能 | 平台技能目录，或共享 `.agents/skills/`。 |
| 让多个平台共享某项能力 | 在各平台技能目录写入等价技能，或在支持的平台上使用 `.agents/skills/` 共享层。 |
| 修改 finish 或 continue 入口 | 平台命令、提示与工作流。 |

## 修改原则

1. **入口文件简短，较长内容放入引用文档**。这对 `trellis-meta` 这类多文件技能尤其重要。
2. **触发描述要具体**。过宽可能误触发，过窄可能无法触发。
3. **跨平台保持相同语义一致**。文件格式可不同，行为说明应一致。
4. **项目特定能力放入本地技能**。不要将团队私有流程写入公开的 `trellis-meta`。

用户只希望本地 AI 多了解一条项目规则时，通常应创建项目本地技能或更新 `.trellis/spec/`，而不是修改 Trellis 内置工作流技能。
