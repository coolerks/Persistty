---
name: trellis-meta
description: "理解并定制用户项目中的本地 Trellis 架构。适用于修改 .trellis、平台 hooks、设置、代理、技能、命令、提示词、工作流、频道运行时（trellis channel）、.trellis/agents/ 中的内置运行时代理、可选工作流模板、基于 registry 的规范刷新、trellis init 生成的跨会话记忆（trellis mem），以及面向 AI 的内置技能（trellis-channel、trellis-session-insight、trellis-spec-bootstrap）和内置技能自动分派流程。"
---

# Trellis 本地架构与定制

本技能适用于已在项目中运行 `trellis init` 的本地 Trellis 用户。阅读后，AI 应理解该用户项目中的 Trellis 架构、运行模型与定制入口，再按用户请求修改生成的 `.trellis/` 和平台目录文件。

Trellis v0.6 在此前的工作流、持久化与平台模型上增加了三个架构面。第一，多代理协作运行时：`trellis channel` 通过项目范围的 JSONL 事件日志 `~/.trellis/channels/<project>/<channel>/events.jsonl` 协调多个 AI worker 进程，包含 worker OOM 防护、论坛/帖子频道、持久幂等键，以及内置 `.trellis/agents/{check,implement}.md` 运行时定义。第二，跨会话记忆：`trellis mem list | search | context | extract | projects` 读取磁盘上已有的 Claude Code、Codex 与 Pi Agent 原始 JSONL，按 `--phase brainstorm|implement|all` 切片，且不上传任何内容。第三，双包 npm 发布：`@mindfoldhq/trellis`（CLI）与 `@mindfoldhq/trellis-core`（SDK，含 `/channel`、`/task`、`/mem`、`/testing` 子路径）以同一版本同步发布。将这些与各平台集成文件一样，作为主要定制入口。

默认操作范围是用户项目的本地文件：

- `.trellis/`：工作流、配置、任务、规范、工作区、脚本、内置运行时代理和运行状态。
- 平台目录：`.claude/`、`.codex/`、`.cursor/`、`.opencode/`、`.kiro/`、`.gemini/`、`.qoder/`、`.codebuddy/`、`.github/`、`.factory/`、`.pi/`、`.reasonix/`、`.kilocode/`、`.agent/`、`.devin/`、`.kimi-code/` 等。Pi 在文件布局上还提供原生 `trellis_subagent` 工具，含 `single` / `parallel` / `chain` 分派模式、节流的进度卡与 `isTrellisAgent()` 验证。Reasonix 将工作流技能与子代理技能都存为 `.reasonix/skills/<name>/SKILL.md`；子代理技能的 frontmatter 含 `runAs: subagent`。Kimi Code 将工作流技能放在共享 `.agents/skills/` 层，将命令和代理提示词放为 `.kimi-code/skills/<name>/SKILL.md`，并将相同代理提示词安装为 `.kimi-code/agents/<name>.md` 下的自定义子代理。
- 共享技能层：`.agents/skills/`。
- 项目树外由用户持有的频道存储：`~/.trellis/channels/<project>/<channel>/events.jsonl`。
- 可通过 `trellis mem` 查询的原始平台对话日志：`~/.claude/projects/`、`~/.codex/sessions/` 与 `~/.pi/agent/sessions/`（v0.6 系列的 OpenCode 适配器处于降级状态）。

不要假设用户拥有 Trellis 源码仓库。不要默认修改全局 npm 安装目录或 `node_modules`；`@mindfoldhq/trellis` 与 `@mindfoldhq/trellis-core` 都作为发布包提供，每次发布共享同一版本与 git tag。

## 使用方式

1. 先读取 `references/local-architecture/overview.md`，建立本地 Trellis 系统模型。
2. 若请求涉及特定 AI 工具，读取 `references/platform-files/platform-map.md` 与相关平台文件说明。
3. 若请求涉及多代理分派或频道 worker，读取 `references/local-architecture/multi-agent-channel.md` 与内置 `.trellis/agents/` 文件。
4. 若用户希望改变行为，读取 `references/customize-local/overview.md`，再打开具体定制主题。
5. 编辑前读取用户项目中的实际文件，以本地内容为准。

## 引用

### 本地架构

- `references/local-architecture/overview.md`：分层本地 Trellis 架构（工作流 / 持久化 / 平台 / 频道运行时）及定制原则。
- `references/local-architecture/generated-files.md`：`trellis init` 生成的文件及其定制边界，包括 `.trellis/agents/`。
- `references/local-architecture/workflow.md`：`.trellis/workflow.md` 中的阶段、路由、workflow-state 块与可选工作流模板（`native`、`tdd`、`channel-driven-subagent-dispatch`、市场模板）。
- `references/local-architecture/task-system.md`：任务目录、活动任务、JSONL 上下文、父子任务树与任务运行时。
- `references/local-architecture/spec-system.md`：如何组织和注入 `.trellis/spec/`，以及如何从 `registry.spec` 来源刷新。
- `references/local-architecture/workspace-memory.md`：`.trellis/workspace/` 日志、`trellis mem` 跨会话回忆与 `@mindfoldhq/trellis-core/mem` SDK。
- `references/local-architecture/context-injection.md`：hooks、子代理前置上下文与频道运行时 worker 收件箱路由。
- `references/local-architecture/multi-agent-channel.md`：`trellis channel` 子命令、项目范围事件存储、论坛/帖子频道、worker OOM 防护、持久幂等性与内置 `.trellis/agents/` 运行时代理。
- `references/local-architecture/bundled-skills.md`：自动分派的内置技能（`trellis-meta`、`trellis-spec-bootstrap`、`trellis-session-insight`），以及 `getBundledSkillTemplates()` 如何将其交付到各平台技能根目录。

### 平台文件

- `references/platform-files/overview.md`：共享 `.trellis/` 文件与平台目录的关系，以及四种平台集成模式（hook 驱动、代理前置上下文、主会话工作流、频道运行时）。
- `references/platform-files/platform-map.md`：全部受支持平台的技能、代理、hooks 和扩展目录及路径，包括 Reasonix 与 Pi 原生 `trellis_subagent` 扩展。
- `references/platform-files/hooks-and-settings.md`：设置与配置文件、hooks、插件和扩展如何连接 Trellis；包含 `channel.worker_guard.*` 与 `codex.dispatch_mode`。
- `references/platform-files/agents.md`：各平台 `trellis-research` / `trellis-implement` / `trellis-check` 子代理文件，以及频道运行时的内置 `.trellis/agents/{check,implement}.md`。
- `references/platform-files/skills-and-commands.md`：技能、命令、提示词和工作流的区别及修改方法。

### 本地定制

- `references/customize-local/overview.md`：为用户请求选择正确的本地定制入口。
- `references/customize-local/change-workflow.md`：修改阶段、路由、下一步动作、workflow-state 与所选工作流模板。
- `references/customize-local/change-task-lifecycle.md`：修改任务创建、状态、归档行为、父子链接、归档 slug 冲突处理与生命周期 hooks。
- `references/customize-local/change-context-loading.md`：修改任务、规范、日志、hook 上下文、频道收件箱消息与 `trellis mem` 回忆的加载方式。
- `references/customize-local/change-hooks.md`：修改平台 hooks、设置、任务生命周期 hooks（`hooks.after_*`）与 shell 会话桥接。
- `references/customize-local/change-agents.md`：修改平台子代理、内置频道运行时代理与 Codex `dispatch_mode` 切换下的 research、implement 和 check 行为。
- `references/customize-local/change-skills-or-commands.md`：添加或修改本地技能、命令、提示词与工作流；包含上游内置技能自动分派。
- `references/customize-local/change-spec-structure.md`：调整 `.trellis/spec/` 下的项目规范结构，包括基于 registry 的来源。
- `references/customize-local/add-project-local-conventions.md`：将团队规则写入项目本地规范或本地技能。

## 当前规则

- `.trellis/workflow.md` 是本地工作流的唯一准则；初始内容在 `trellis init` 时从工作流模板（内置 `native`、`tdd`、`channel-driven-subagent-dispatch` 或市场模板）选择，并可通过 `trellis workflow --template <id>` 重新选择。若活动模板引用的 `.trellis/agents/<name>.md` 缺失，会输出指向 `trellis update` 的非阻塞 stderr 警告。
- `.trellis/config.yaml` 是项目级 Trellis 配置入口。包括任务生命周期 hooks（`hooks.after_create` / `after_start` / `after_finish` / `after_archive`）、日志形式（`session_commit_message` / `max_journal_lines` / `session_auto_commit`）、频道 worker 防护（`channel.worker_guard.idle_timeout` / `max_live_workers`）、Codex 分派模式（`codex.dispatch_mode: inline | sub-agent`），以及规范 registry 配置块（`registry.spec.source` + `registry.spec.template`）。
- `.trellis/spec/` 存放用户的项目编码约定与设计约束。配置 `registry.spec` 后，通过 `trellis update` 刷新文件；本地修改会在 `.trellis/.template-hashes.json` 中表现为“用户已修改”的冲突。
- `.trellis/tasks/` 存放任务 PRD、设计笔记、实施计划、研究文件与 JSONL 上下文。任务构成父子树：`task.py create --parent <slug>`、`task.py add-subtask <parent> <child>`、`task.py remove-subtask <parent> <child>` 与 `task.py list-context <task>`。`task.py create` 会拒绝已存在于 `.trellis/tasks/archive/**` 中的 slug。
- `.trellis/workspace/` 存放**主动撰写**的开发者日志。原始跨会话对话**不**存于此；它位于磁盘上的 `~/.claude/projects/`、`~/.codex/sessions/` 与 `~/.pi/agent/sessions/`，通过 `trellis mem search|extract|context` 恢复。内置 `trellis-session-insight` 技能说明何时使用 `mem`。
- `.trellis/agents/{check,implement}.md` 是内置且与平台无关的频道运行时代理定义，由 `trellis channel spawn --agent <name>` 加载。它们可编辑；`trellis update` 会补齐缺失文件。编辑各平台的 `trellis-implement.md` / `trellis-check.md` **不会**改变频道运行时 worker 的行为。
- `~/.trellis/channels/<project>/<channel>/events.jsonl` 是每个项目、每个频道的运行时事件日志。由用户持有，序号分配使用文件锁，支持持久 `idempotencyKey`；永远不放在 `.trellis/` 下。
- 内置多文件技能（`trellis-meta`、`trellis-spec-bootstrap`、`trellis-session-insight`、`trellis-channel`）通过 `packages/cli/src/templates/common/index.ts` 中的 `getBundledSkillTemplates()` 自动分派到各平台技能根目录。上游在 `packages/cli/src/templates/common/bundled-skills/` 下添加新目录后，下次 `trellis update` 会将它交付给各平台。
- 平台设置或配置文件决定实际运行哪些 hooks、代理、技能、命令、提示词与工作流。Reasonix 没有设置文件，其行为编码在技能 frontmatter 中。
- `.trellis/.template-hashes.json` 与 `.trellis/.runtime/` 是管理或运行状态文件，编辑前确认必要性。

## 禁止做法

- 不要把 Trellis 上游源码视为本地定制的默认目标。
- 不要为项目需求修改全局 npm 安装目录、`node_modules/@mindfoldhq/trellis` 或 `node_modules/@mindfoldhq/trellis-core`；两包同步发布。
- 不要用默认模板覆盖用户已修改的本地文件；先检查 `.trellis/.template-hashes.json`，优先生成 `.new` 旁文件，避免破坏性覆盖。
- 不要将团队私有项目规则写入公开内置技能（`trellis-meta`、`trellis-spec-bootstrap`、`trellis-session-insight`、`trellis-channel`）；项目规则应放在 `.trellis/spec/`、项目本地技能、当前任务或工作区日志中，因为 `trellis update` 会覆盖内置技能目录内的内容。
- 不要手动编辑 `~/.trellis/channels/<project>/<channel>/events.jsonl`；序号在文件锁下分配，可安全重放的写入通过 `trellis channel` CLI 或 `@mindfoldhq/trellis-core/channel` SDK 完成。
- 若目标是改变频道运行时 worker 行为，不要编辑 `.claude/agents/trellis-implement.md` 或其他平台子代理文件；应编辑 `.trellis/agents/<name>.md`。
- 不要将已移除或从未发布的机制描述为当前 Trellis 行为；声明某配置项存在前，核对本地 `.trellis/config.yaml` 与已安装 CLI 的 `trellis --help`。
