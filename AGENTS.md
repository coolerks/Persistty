<!-- TRELLIS:START -->
# Trellis 项目指引

本指引适用于在本项目中工作的 AI 助手。

本项目使用 Trellis 管理开发流程。工作所需的知识位于 `.trellis/`：

- `.trellis/workflow.md` — 开发阶段、创建任务的时机和技能路由。
- `.trellis/spec/` — 按包和层组织的开发规范；修改某一层的代码前必须阅读对应规范。
- `.trellis/workspace/` — 各开发者的日志和会话记录。
- `.trellis/tasks/` — 活动及归档任务，包括需求文档（PRD）、研究资料和 JSONL 上下文。

若当前平台提供 Trellis 命令（如 `/trellis:finish-work`、`/trellis:continue`），优先使用命令执行流程。并非所有平台都提供全部命令。

使用 Codex 或其他支持代理的工具时，可在以下位置查找项目级辅助资源：

- `.agents/skills/` — 可复用的 Trellis 技能。
- `.codex/agents/` — 可选的自定义子代理。

此区块由 Trellis 管理。区块外的修改会保留；区块内的修改可能在后续 `trellis update` 时被覆盖。

<!-- TRELLIS:END -->

## 项目语言与文档兼容性

项目文档、开发规范和用户界面说明使用简体中文。文件名、引用路径、代码标识符、协议字段、命令和机器解析标记保持原有形式。具体文档维护规则见 [.trellis/spec/guides/index.md](.trellis/spec/guides/index.md)。
