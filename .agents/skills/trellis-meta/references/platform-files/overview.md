# 平台文件概览

Trellis 将相同的本地架构连接到不同 AI 工具。`.trellis/` 存储共享运行时，平台目录存储适配文件，定义各 AI 工具如何进入 Trellis。

本地 AI 修改 Trellis 时，应先区分两类文件：

- **共享文件**：`.trellis/workflow.md`、`.trellis/tasks/`、`.trellis/spec/`、`.trellis/scripts/`。
- **平台文件**：`.claude/`、`.snow/`、`.codex/`、`.cursor/`、`.opencode/`、`.kiro/`、`.gemini/`、`.qoder/`、`.codebuddy/`、`.github/`、`.factory/`、`.pi/`、`.trae/`、`.kilocode/`、`.agent/`、`.devin/`、`.reasonix/`、`.zcode/`、`.kimi-code/` 等目录。

平台文件不存储业务状态。它们让对应 AI 工具读取 Trellis 状态、调用 Trellis 脚本并加载 Trellis 技能、代理与钩子。

## 平台文件类别

| 类别 | 常见路径 | 用途 |
| --- | --- | --- |
| 设置与配置 | `.claude/settings.json`、`.codex/hooks.json`、`.qoder/settings.json`、`.trae/hooks.json` | 注册钩子、插件、扩展或平台行为。 |
| 钩子、插件与扩展 | `.claude/hooks/`、`.opencode/plugins/`、`.pi/extensions/` | 在会话启动、用户输入、代理启动、Shell 执行等事件中注入上下文。 |
| 代理 | `.claude/agents/`、`.codex/agents/`、`.kiro/agents/`、`.zcode/agents/` | 定义 `trellis-research`、`trellis-implement` 和 `trellis-check`。 |
| 技能 | `.claude/skills/`、`.agents/skills/`、`.qoder/skills/`、`.zcode/skills/` | 自动触发或按需读取的能力说明。 |
| 命令、提示与工作流 | `.cursor/commands/`、`.github/prompts/`、`.devin/workflows/`、`.zcode/commands/` | 用户显式调用的入口。 |

## 三种平台集成模式

### 1. 钩子或扩展驱动

这些平台能在特定事件触发脚本或插件，主动向 AI 注入 Trellis 上下文。

常见能力：

- session-start 注入 `.trellis/` 概览。
- 每次用户交互提供 workflow-state 提示。
- 子代理启动时注入 PRD、规范与研究。
- Shell 命令继承会话身份。

修改“AI 何时知道哪些信息”时，先检查钩子、插件、扩展与设置。

### 2. 代理前置指令或主动读取

部分平台无法可靠地让钩子改写子代理提示，因此代理文件自己要求代理启动后读取活动任务、PRD 和 JSONL 上下文。

修改子代理加载上下文的方式时，检查代理文件自身。

### 3. 主会话工作流

部分平台不具备 Trellis 子代理或钩子能力，依赖工作流、技能与命令指导主会话 AI 读取文件、运行脚本并推进任务。

修改行为时，检查平台工作流、技能、命令和 `.trellis/workflow.md`。

## 本地修改顺序

用户要求定制某个平台行为时，AI 应按以下顺序检查文件：

1. 读取 `.trellis/workflow.md`，确认共享流程。
2. 读取目标平台的设置与配置，了解注册了哪些钩子、代理、技能与命令。
3. 读取目标平台的代理、技能、命令与钩子。
4. 修改最接近用户需求的本地文件。
5. 变更影响共享流程时，同步 `.trellis/workflow.md` 或 `.trellis/spec/`。

不要只修改平台文件而忘记共享工作流，也不要只修改 `.trellis/workflow.md` 而忘记平台入口可能仍保留旧说明。
