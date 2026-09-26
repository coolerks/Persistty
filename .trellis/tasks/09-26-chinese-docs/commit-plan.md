# 提交方案（尚未执行）

本次文档修改和质量检查完成后，仍需一次用户确认才能提交。所有文件保持在工作区；不自动提交、归档或 push。

## 拟议提交

1. `chore: 初始化 Trellis 配置与现有项目规范`

   纳入本会话开始前已存在、此次未修改的初始化文件和 bootstrap 记录。下方逐项列出为“未识别修改”；用户确认包含后才会进入本批。

2. `docs: 中文化 Trellis 文档并补齐 Persistty 开发规范`

   纳入本次实际修改的 Markdown 和本任务规划/检查记录。先完成初始化批次，以保证文档依赖的脚本与其余规范在 Git 中存在。

## 本会话修改与新增文件（第 2 批）

- `.agents/skills/trellis-before-dev/SKILL.md`
- `.agents/skills/trellis-brainstorm/SKILL.md`
- `.agents/skills/trellis-break-loop/SKILL.md`
- `.agents/skills/trellis-channel/SKILL.md`
- `.agents/skills/trellis-channel/references/command-reference.md`
- `.agents/skills/trellis-channel/references/forum.md`
- `.agents/skills/trellis-channel/references/progress-debugging.md`
- `.agents/skills/trellis-channel/references/workers.md`
- `.agents/skills/trellis-channel/references/workflows.md`
- `.agents/skills/trellis-check/SKILL.md`
- `.agents/skills/trellis-continue/SKILL.md`
- `.agents/skills/trellis-finish-work/SKILL.md`
- `.agents/skills/trellis-meta/SKILL.md`
- `.agents/skills/trellis-meta/references/customize-local/add-project-local-conventions.md`
- `.agents/skills/trellis-meta/references/customize-local/change-agents.md`
- `.agents/skills/trellis-meta/references/customize-local/change-context-loading.md`
- `.agents/skills/trellis-meta/references/customize-local/change-hooks.md`
- `.agents/skills/trellis-meta/references/customize-local/change-skills-or-commands.md`
- `.agents/skills/trellis-meta/references/customize-local/change-spec-structure.md`
- `.agents/skills/trellis-meta/references/customize-local/change-task-lifecycle.md`
- `.agents/skills/trellis-meta/references/customize-local/change-workflow.md`
- `.agents/skills/trellis-meta/references/customize-local/overview.md`
- `.agents/skills/trellis-meta/references/local-architecture/bundled-skills.md`
- `.agents/skills/trellis-meta/references/local-architecture/context-injection.md`
- `.agents/skills/trellis-meta/references/local-architecture/generated-files.md`
- `.agents/skills/trellis-meta/references/local-architecture/multi-agent-channel.md`
- `.agents/skills/trellis-meta/references/local-architecture/overview.md`
- `.agents/skills/trellis-meta/references/local-architecture/spec-system.md`
- `.agents/skills/trellis-meta/references/local-architecture/task-system.md`
- `.agents/skills/trellis-meta/references/local-architecture/workflow.md`
- `.agents/skills/trellis-meta/references/local-architecture/workspace-memory.md`
- `.agents/skills/trellis-meta/references/platform-files/agents.md`
- `.agents/skills/trellis-meta/references/platform-files/hooks-and-settings.md`
- `.agents/skills/trellis-meta/references/platform-files/overview.md`
- `.agents/skills/trellis-meta/references/platform-files/platform-map.md`
- `.agents/skills/trellis-meta/references/platform-files/skills-and-commands.md`
- `.agents/skills/trellis-session-insight/SKILL.md`
- `.agents/skills/trellis-session-insight/references/cli-quick-reference.md`
- `.agents/skills/trellis-session-insight/references/triggering-patterns.md`
- `.agents/skills/trellis-spec-bootstrap/SKILL.md`
- `.agents/skills/trellis-spec-bootstrap/references/mcp-setup.md`
- `.agents/skills/trellis-spec-bootstrap/references/repository-analysis.md`
- `.agents/skills/trellis-spec-bootstrap/references/spec-task-planning.md`
- `.agents/skills/trellis-spec-bootstrap/references/spec-writing.md`
- `.agents/skills/trellis-start/SKILL.md`
- `.agents/skills/trellis-update-spec/SKILL.md`
- `.trellis/agents/check.md`
- `.trellis/agents/implement.md`
- `.trellis/spec/backend/database-guidelines.md`
- `.trellis/spec/backend/filesystem-guidelines.md`
- `.trellis/spec/backend/http-api.md`
- `.trellis/spec/backend/process-guidelines.md`
- `.trellis/spec/backend/quality-guidelines.md`
- `.trellis/spec/backend/security-config.md`
- `.trellis/spec/backend/terminal-lifecycle.md`
- `.trellis/spec/backend/transfer-search-git.md`
- `.trellis/spec/backend/websocket-protocol.md`
- `.trellis/spec/frontend/component-guidelines.md`
- `.trellis/spec/frontend/editor-terminal-lifecycle.md`
- `.trellis/spec/frontend/state-management.md`
- `.trellis/spec/guides/index.md`
- `.trellis/tasks/00-bootstrap-guidelines/check-report.md`
- `.trellis/tasks/09-26-chinese-docs/changed-files.json`
- `.trellis/tasks/09-26-chinese-docs/check-report.md`
- `.trellis/tasks/09-26-chinese-docs/check.jsonl`
- `.trellis/tasks/09-26-chinese-docs/commit-plan.md`
- `.trellis/tasks/09-26-chinese-docs/design.md`
- `.trellis/tasks/09-26-chinese-docs/implement.jsonl`
- `.trellis/tasks/09-26-chinese-docs/implement.md`
- `.trellis/tasks/09-26-chinese-docs/prd.md`
- `.trellis/tasks/09-26-chinese-docs/research/localization-contract.md`
- `.trellis/tasks/09-26-chinese-docs/task.json`
- `.trellis/workflow.md`
- `.trellis/workspace/index.md`
- `.trellis/workspace/moyok/index.md`
- `.trellis/workspace/moyok/journal-1.md`
- `AGENTS.md`
- `README.md`

## 此前已有的未识别文件（第 1 批，须确认包含）

这些文件此次没有编辑，不能因父目录未跟踪而自动合并进本次文档提交。

- `.codex/agents/trellis-check.toml`
- `.codex/agents/trellis-implement.toml`
- `.codex/agents/trellis-research.toml`
- `.codex/config.toml`
- `.codex/hooks.json`
- `.codex/hooks/inject-subagent-context.py`
- `.codex/hooks/inject-workflow-state.py`
- `.codex/hooks/session-start.py`
- `.gitattributes`
- `.trellis/.gitignore`
- `.trellis/.template-hashes.json`
- `.trellis/.version`
- `.trellis/config.yaml`
- `.trellis/scripts/__init__.py`
- `.trellis/scripts/add_session.py`
- `.trellis/scripts/common/__init__.py`
- `.trellis/scripts/common/active_task.py`
- `.trellis/scripts/common/cli_adapter.py`
- `.trellis/scripts/common/config.py`
- `.trellis/scripts/common/developer.py`
- `.trellis/scripts/common/git.py`
- `.trellis/scripts/common/git_context.py`
- `.trellis/scripts/common/io.py`
- `.trellis/scripts/common/log.py`
- `.trellis/scripts/common/packages_context.py`
- `.trellis/scripts/common/paths.py`
- `.trellis/scripts/common/safe_commit.py`
- `.trellis/scripts/common/session_context.py`
- `.trellis/scripts/common/task_context.py`
- `.trellis/scripts/common/task_queue.py`
- `.trellis/scripts/common/task_store.py`
- `.trellis/scripts/common/task_utils.py`
- `.trellis/scripts/common/tasks.py`
- `.trellis/scripts/common/trellis_config.py`
- `.trellis/scripts/common/types.py`
- `.trellis/scripts/common/workflow_phase.py`
- `.trellis/scripts/get_context.py`
- `.trellis/scripts/get_developer.py`
- `.trellis/scripts/hooks/linear_sync.py`
- `.trellis/scripts/init_developer.py`
- `.trellis/scripts/task.py`
- `.trellis/spec/backend/directory-structure.md`
- `.trellis/spec/backend/error-handling.md`
- `.trellis/spec/backend/index.md`
- `.trellis/spec/backend/logging-guidelines.md`
- `.trellis/spec/frontend/clients.md`
- `.trellis/spec/frontend/directory-structure.md`
- `.trellis/spec/frontend/hook-guidelines.md`
- `.trellis/spec/frontend/index.md`
- `.trellis/spec/frontend/quality-guidelines.md`
- `.trellis/spec/frontend/theme-assets.md`
- `.trellis/spec/frontend/type-safety.md`
- `.trellis/spec/guides/code-reuse-thinking-guide.md`
- `.trellis/spec/guides/cross-layer-thinking-guide.md`
- `.trellis/tasks/00-bootstrap-guidelines/check.jsonl`
- `.trellis/tasks/00-bootstrap-guidelines/design.md`
- `.trellis/tasks/00-bootstrap-guidelines/implement.jsonl`
- `.trellis/tasks/00-bootstrap-guidelines/implement.md`
- `.trellis/tasks/00-bootstrap-guidelines/prd.md`
- `.trellis/tasks/00-bootstrap-guidelines/research/initial-contracts.md`
- `.trellis/tasks/00-bootstrap-guidelines/task.json`

## 确认方式

同意两批与上述文件范围后执行；若选择自行提交，则保持工作区供手动处理。不会 amend 或 push；归档和会话日志按后续 finish 流程独立记录。
