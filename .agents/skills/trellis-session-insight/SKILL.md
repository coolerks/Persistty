---
name: trellis-session-insight
description: "通过 trellis mem CLI 检索过去的 AI 对话。适用于用户问“上次怎么解的”“之前讨论过吗”“关于 X 当时怎么决定的”“提醒我这次任务做了什么”“想起一段对话”，或开始与历史工作重叠的需求讨论、调试熟悉的缺陷、跨会话继续任务、进行收尾回顾时。返回原始历史对话；根据当下情况决定更新规范、追加任务笔记、在回答中引用，或只理解吸收。"
---

# Trellis 会话洞察

本技能教 AI **如何调用 `trellis mem`**（项目跨会话记忆的原始素材来源），以及**何时适合使用它**。

它刻意定位为**能力技能，而非工作流**。没有固定输出文件、必需回写步骤，也没有“每次 finish-work 后必须运行”的规则。如何使用 `mem` 返回的内容，由当前对话中的判断决定。此技能让 AI 知道该能力存在，并能自行判断。

## `trellis mem` 是什么

这是一个本地 CLI，索引用户过去的 Claude Code、Codex、Devin CLI、Grok、OpenCode、Pi Agent 与 ZCode 对话日志，支持列举、搜索、按 Trellis 任务边界切片，以及导出清理后的对话。Claude 与 Codex 使用 `~/.claude/projects/` 和 `~/.codex/sessions/`。Devin CLI（Cognition 终端代理，并非 `trellis init --devin` 的桌面版本）使用 `~/.local/share/devin/cli/sessions.db`。Grok 使用 `~/.grok/sessions/`。OpenCode 使用 `~/.local/share/opencode/opencode.db`（无依赖 SQLite 读取器）。Pi 使用默认或由环境变量配置的会话根目录、全局 `~/.pi/agent/settings.json` 与项目范围内的 `.pi/settings.json`；相对 `sessionDir` 值以配置文件目录为基准解析。项目本地 Pi 配置需要通过当前 cwd 或 `--cwd` 做项目范围查找。ZCode 使用 `~/.zcode/cli/db/db.sqlite`。

`mem` 不上传任何内容，所有读取均在本地进行。

## 何时使用

判断标准是：“资深同事是否会先问‘我们之前是不是讨论过这个？’”。以下是具体场景：

- **重复需求讨论的风险。** 新任务涉及用户此前做过的领域，希望在重新询问前确认是否已做过决策。
- **熟悉缺陷的调试。** 当前缺陷模式似乎曾被用户报告或修复。获取相关历史会话可节省一整轮调试。
- **跨会话继续。** 用户间隔一段时间后回来，只说“上次做到哪了”或“继续上次的”，没有具体说明。
- **决策检索。** 用户提到“关于 X 当时的决定”，但决定在旧讨论里，而非任何 `prd.md` 或 `spec/` 文件中。
- **收尾回顾。** 用户明确要求回顾本任务的决策、困难或意外发现，而非每次 finish-work 都强制执行。
- **识别历史工作模式。** 用户问“我是否一直在 X 上犯同样的错”或“我每次都踩这个坑吗”，跨会话搜索可以回答。

若这些场景都不适用，不要调用 `mem`。它是工具，不是仪式。

## 何时不应使用

- 相关上下文已在当前轮、`prd.md`、`design.md`、最近 `git log` 或已打开文件中。`mem` 用于当前无法直接获取的内容。
- 用户询问代码事实，而非过去对话的事实。`git log -p`、`grep` 或直接读取文件更快、更权威。
- 你是子代理（`trellis-implement` / `trellis-check`），且分派提示已包含整理后的 `implement.jsonl` / `check.jsonl` 上下文。额外调用 `mem` 通常只会增加干扰。
- 用户明确说“别翻历史，直接回答我的问题”。

## 如何使用 `mem` 返回的内容

将输出视为**原始素材**，而非交付物。获取后，根据当前对话决定：

- **在回复中引用**：若某段历史交流回答了当前问题，引用 session-id 与阶段，让用户可以核对。
- **更新 `<task>/prd.md` 或 `<task>/design.md`**：若 `mem` 找到本应记录但遗漏的关键决策，先向用户展示拟议修改。
- **追加任务本地笔记**（例如 `<task>/notes.md`，或扩展现有文件）：若发现属于当前任务记录，但不适合写入 PRD。
- **更新 `.trellis/spec/`**：若发现是有助于未来任务的项目级约定或陷阱，运行 `trellis-update-spec`；`session-insight` 的职责止于发现。
- **只理解吸收**：后续几轮更准确地回答，不写入任何文件。一次性回忆场景中，这往往合适。

Trellis 不规定唯一记录位置。强迫每次回忆写入固定文件，会让文件充满噪声。根据情况决定。

## 如何调用

完整 CLI 参考位于 `references/cli-quick-reference.md`。大多数情况使用以下命令之一：

```bash
# 搜索内容提及关键词的会话（默认项目范围；
# 添加 --global 可搜索本机所有项目）。
trellis mem search "<keyword>"

# 导出一个会话的对话，可选按阶段或关键词过滤。
trellis mem extract <session-id> --phase brainstorm
trellis mem extract <session-id> --grep "<keyword>"

# 深入查看会话：前 N 个命中轮次及周围上下文。
trellis mem context <session-id> --turns 3 --around 2

# 尚不知道会话 id 时，先列举再过滤。
trellis mem list --cwd <project-path>
trellis mem projects   # → 列出活动项目 cwd，再缩小范围
```

阶段切片（`--phase brainstorm|implement|all`）在 `task.py create` 和 `task.py start` 的边界处切分会话。回顾当前任务时，`--phase brainstorm` 恢复规划讨论，`--phase implement` 恢复执行过程。默认 `all`。

## 触发模式

`references/triggering-patterns.md` 列出更多应让你想到使用 `mem` 的用户表达示例（已统一译为中文）。培养判断时可参考。

## 范围外事项

- `mem` 不编辑代码或更新文件。是否回写由当下判断决定。
- `mem` 只读平台 JSONL 存储，不推送或同步到远程。
- 本技能不替代 `trellis-update-spec`（用于将发现提升为项目级指南），也不替代平台原生的任务或规范工作流。
