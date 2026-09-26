# 平台文件映射

本页按平台列出用户项目中常见的 Trellis 文件位置。实际项目是否存在某个平台目录，取决于用户运行了哪些 `trellis init --<platform>` 命令。

## 对照矩阵

| 平台 | CLI 参数 | 主目录 | 技能目录 | 代理目录 | 钩子与扩展 |
| --- | --- | --- | --- | --- | --- |
| Claude Code | `--claude` | `.claude/` | `.claude/skills/` | `.claude/agents/` | `.claude/hooks/` 与 `.claude/settings.json` |
| Cursor | `--cursor` | `.cursor/` | `.cursor/skills/` | `.cursor/agents/` | `.cursor/hooks.json` 与 `.cursor/hooks/` |
| OpenCode | `--opencode` | `.opencode/` | `.opencode/skills/` | `.opencode/agents/` | `.opencode/plugins/` |
| Codex | `--codex` | `.codex/` | `.agents/skills/` | `.codex/agents/` | `.codex/hooks/` 与 `.codex/hooks.json` |
| Kilo | `--kilo` | `.kilocode/` | `.kilocode/skills/` | 通常没有 | `.kilocode/workflows/` |
| Kiro | `--kiro` | `.kiro/` | `.kiro/skills/` | `.kiro/agents/` | `.kiro/hooks/` |
| Gemini CLI | `--gemini` | `.gemini/` | `.agents/skills/` | `.gemini/agents/` | `.gemini/settings.json` 与 `.gemini/hooks/` |
| Antigravity | `--antigravity` | `.agent/` | `.agent/skills/` | 通常没有 | `.agent/workflows/` |
| Devin | `--devin` | `.devin/` | `.devin/skills/` | 通常没有 | `.devin/workflows/` |
| Qoder | `--qoder` | `.qoder/` | `.qoder/skills/` | `.qoder/agents/` | `.qoder/hooks/` 与 `.qoder/settings.json` |
| CodeBuddy | `--codebuddy` | `.codebuddy/` | `.codebuddy/skills/` | `.codebuddy/agents/` | `.codebuddy/hooks/` 与 `.codebuddy/settings.json` |
| GitHub Copilot | `--copilot` | `.github/` | `.github/skills/` | `.github/agents/` | `.github/copilot/hooks/` 与提示文件 |
| Factory Droid | `--droid` | `.factory/` | `.factory/skills/` | `.factory/droids/` | `.factory/hooks/` 与设置文件 |
| DeepSeek Harness (dsh) | `--dsh` | `.dsh/` | `.agents/skills/`（共享）与 `.dsh/skills/`（入口技能） | 无（工作流技能在当前会话直接运行 implement/check） | 无（第 2 类主动读取；没有项目钩子或设置） |
| Pi Agent | `--pi` | `.pi/` | `.agents/skills/` | `.pi/agents/` | `.pi/extensions/trellis/`（原生 `trellis_subagent` 工具）与 `.pi/settings.json` |
| Trae IDE | `--trae` | `.trae/` | `.trae/skills/` | `.trae/agents/` | `.trae/hooks/` 与 `.trae/hooks.json` |
| Reasonix | `--reasonix` | `.reasonix/` | `.reasonix/skills/` | 无独立目录；子代理是 frontmatter 含 `runAs: subagent` 的技能 | 无 |
| ZCode | `--zcode` | `.zcode/` | `.zcode/skills/` | `.zcode/agents/` | `.zcode/hooks/` 与 `.zcode/config.json`（SessionStart、UserPromptSubmit、PreToolUse Agent/Task）；子代理使用钩子注入上下文 |
| Grok Build | `--grok` | `.grok/` | `.grok/skills/` | `.grok/agents/` | 主动读取的前置指令（无钩子；扁平命令 `.grok/commands/trellis-*.md`） |
| Kimi Code | `--kimi` | `.kimi-code/` | `.agents/skills/`（共享）与 `.kimi-code/skills/` | `.kimi-code/agents/`（自定义子代理；相同提示也以技能发布） | 无（主动读取的前置指令；没有项目钩子或设置） |
| Snow CLI | `--snow` | `.snow/` | `.snow/skills/` | `.snow/agents/`（自动发现；主要路径） | 第 1 类：自动注入、项目代理与 `beforeSubAgentStart`（`.snow/hooks/` 的 `session` / `user` / `subagent` 模式 → `additionalContext` JSON）；无旧式子代理 JSON；命令为 `.snow/commands/trellis-*.json` |

## 能力分组

### Trellis 子代理支持

这些平台通常具有 `trellis-research`、`trellis-implement` 和 `trellis-check` 文件：

- Claude Code
- Cursor
- OpenCode
- Codex
- Kiro
- Gemini CLI
- Qoder
- CodeBuddy
- GitHub Copilot
- Factory Droid
- Pi Agent
- Trae IDE
- Reasonix（作为 frontmatter 含 `runAs: subagent` 的技能交付，位于 `.reasonix/skills/`，没有独立 `agents/` 目录）
- ZCode
- Grok Build（`.grok/agents/`；通过带 `subagent_type` 的 `spawn_subagent` 分派）
- Kimi Code（`.kimi-code/agents/`；相同提示也作为技能交付到 `.kimi-code/skills/`）
- Snow CLI（`.snow/agents/`；自动发现项目代理与第 1 类钩子）

修改实施、检查或研究行为时，先查找对应平台代理文件。

### 原生 Trellis 子代理工具

部分平台暴露宿主运行时能理解的原生工具。模型像调用其他工具一样调用它，宿主渲染进度卡片、按 `.<platform>/agents/` 校验代理名称，并执行分派模式约束。

- Pi Agent：`trellis_subagent` 工具，定义于 `.pi/extensions/trellis/index.ts`。支持 `single` / `parallel` / `chain` 分派模式，并发出实时 `trellis-subagent-progress` 事件。

这些平台需要修改子代理分派行为时，应编辑扩展文件，而不是代理 Markdown。代理 Markdown 定义职责，宿主扩展负责分派、校验与进度渲染。

### 主会话工作流平台

这些平台更多依赖工作流与技能指导主会话：

- Kilo
- Antigravity
- Devin

修改行为时，先检查工作流与技能。不要假设存在 Trellis 子代理。

### 共享 `.agents/skills/`

Codex、Gemini CLI、Pi Agent、Kimi Code 和 DeepSeek Harness（dsh）写入共享 `.agents/skills/` 层。支持 agentskills.io 的部分工具也能读取此目录。用户希望多个兼容工具共享同一技能时，优先考虑 `.agents/skills/`，但不要假设所有平台都会读取。ZCode 将 Trellis 管理的技能保存在 `.zcode/skills/`。

## 修改平台文件的决策规则

1. 用户指定平台时，只修改该平台目录，除非共享工作流或规范也必须改变。
2. 用户说“所有平台都应这样做”时，逐平台同步等价入口，不要只修改一个目录。
3. 用户只说“我的 AI”时，检查项目中实际存在的配置目录，推断当前 AI 平台。
4. 用户希望添加项目规则时，优先使用 `.trellis/spec/` 或项目本地技能。
5. 用户希望改变 Trellis 行为时，编辑 `.trellis/workflow.md` 与平台钩子、代理、技能、命令。

## 路径不一致时

平台生态会变化，用户项目也可能已定制。此表与本地文件不一致时，以用户项目实际设置与配置为准：

- 检查设置注册了哪个钩子。
- 检查命令、提示或工作流指向哪个脚本。
- 根据代理文件当前写明的读取规则判断行为。

不要仅因自定义文件未列在路径表中就删除它。

### `.omp/` — Oh My Pi（OMP）

由扩展支持的平台。OMP 原生提供方自动发现所有子目录。

```text
.omp/
├── commands/          # 斜杠命令（扁平 .md 文件）
├── skills/            # 自动触发技能（每个目录一个 SKILL.md）
├── agents/            # 代理定义（.md 文件）
└── extensions/
    └── trellis/
        └── index.ts   # Trellis 扩展（上下文注入）
```

无 `settings.json`：OMP 自动扫描 `.omp/` 子目录。
无 Python 钩子：等价钩子行为位于 TypeScript 扩展中。
