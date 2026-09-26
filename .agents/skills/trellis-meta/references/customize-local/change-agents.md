# 修改本地代理

用户希望修改 `trellis-research`、`trellis-implement` 或 `trellis-check` 的行为时，应编辑用户项目中的平台代理文件。

## 先读取这些文件

1. 目标平台的代理目录
2. `.trellis/workflow.md` 的第 2 阶段与研究路由
3. 当前任务的 `prd.md`
4. 当前任务的 `implement.jsonl` / `check.jsonl`
5. 相关钩子或代理前置指令

## 常见路径

| 平台 | 路径 |
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

以用户项目中的实际路径为准。

## 常见需求

| 需求 | 应编辑哪个代理 |
| --- | --- |
| 研究必须写入文件，不能只在聊天中回复 | `trellis-research` |
| 实施前必须读取某些本地规范 | `trellis-implement` 与 `implement.jsonl` 配置规则 |
| 检查时必须运行指定命令 | `trellis-check` |
| 代理不得修改某些目录 | 对应代理的写入边界指令 |
| 代理输出必须使用固定格式 | 对应代理的最终回复与报告指令 |

## 修改原则

1. **保留角色边界**：research 调研并持久化结果；implement 编写实现；check 审查并修复。
2. **不要将项目规范硬编码到代理中**：长期规范属于 `.trellis/spec/`，代理负责读取它们。
3. **明确读取顺序**：活动任务 → PRD → 信息 → JSONL → 规范与研究。
4. **明确写入边界**：哪些目录可以写入，哪些不可以。
5. **跨平台同步**：用户配置了多个平台时，判断是只修改当前平台，还是修改所有平台的代理。

## 代理主动读取的平台

如果代理文件包含“启动后读取任务与上下文”的前置指令，编辑时不要移除这些步骤。否则代理将只依赖聊天上下文工作，绕过 Trellis 的核心机制。

## 钩子推送的平台

如果上下文由钩子注入，代理文件仍应保留职责边界。不要因为钩子会注入上下文，就从代理中移除 PRD 与规范要求。
