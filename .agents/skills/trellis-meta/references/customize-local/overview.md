# 本地定制概览

本目录面向在用户项目中工作的本地 AI：Trellis 已通过 npm 安装，并且已执行 `trellis init`。AI 应修改项目内部生成的 `.trellis/` 和平台目录，而不是 Trellis CLI 上游源码。

## 先确定用户实际想修改什么

| 用户表述 | 先读取 |
| --- | --- |
| “修改 Trellis 流程、阶段或下一步提示” | `change-workflow.md` |
| “修改任务创建、状态、归档或钩子” | `change-task-lifecycle.md` |
| “AI 没有读取上下文，或要修改注入内容” | `change-context-loading.md` |
| “某个平台钩子行为不符合预期” | `change-hooks.md` |
| “修改 implement/check/research 代理行为” | `change-agents.md` |
| “添加技能、命令、工作流或提示” | `change-skills-or-commands.md` |
| “调整项目规范结构” | `change-spec-structure.md` |
| “添加团队约定与本地说明” | `add-project-local-conventions.md` |

## 通用操作顺序

1. **确认平台与目录**：检查实际存在的目录，例如 `.claude/`、`.codex/`、`.cursor/`、`.zcode/`。
2. **确认当前活动任务**：运行 `python3 ./.trellis/scripts/task.py current --source`。
3. **读取本地权威来源**：优先使用 `.trellis/workflow.md`、`.trellis/config.yaml` 和相关平台文件。
4. **限定修改范围**：只编辑与用户请求相关的文件。
5. **同步语义**：共享流程变化时检查平台入口是否也需要修改；平台入口变化时检查 `.trellis/workflow.md` 是否仍一致。

## 本地文件优先级

| 层 | 文件 |
| --- | --- |
| 工作流 | `.trellis/workflow.md` |
| 项目配置 | `.trellis/config.yaml` |
| 任务材料 | `.trellis/tasks/<task>/` |
| 项目规范 | `.trellis/spec/` |
| 运行时脚本 | `.trellis/scripts/` |
| 平台集成 | `.claude/`、`.codex/`、`.cursor/`、`.opencode/`、`.zcode/` 等目录 |
| 共享技能 | `.agents/skills/` |

## 默认不要做的事

- 不要编辑全局 npm 安装目录。
- 不要编辑 `node_modules/@mindfoldhq/trellis`。
- 不要假设用户拥有 Trellis GitHub 仓库。
- 不要用默认模板覆盖用户已经修改的本地文件。
- 不要把团队项目规则放入公开的 `trellis-meta`；项目规则属于 `.trellis/spec/` 或本地技能。

## 何时检查上游源码

只有用户明确表达以下目标之一时，才切换到上游源码视角：

- “我要向 Trellis 提交 PR”
- “我要修改 npm 包发布内容”
- “我要 fork Trellis”
- “我要修改 `trellis init/update` 的生成逻辑”

否则，默认修改用户项目内部的本地 Trellis 文件。
