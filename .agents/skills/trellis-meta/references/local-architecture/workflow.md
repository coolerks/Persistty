# 本地工作流系统

`.trellis/workflow.md` 是用户项目内部 Trellis 工作流的权威来源。AI 不需要 Trellis 源码即可理解当前项目应如何推进任务；读取此文件即可。

## 文件职责

`.trellis/workflow.md` 有三项职责：

1. **解释工作流阶段**：规划（Plan）、执行（Execute）、收尾（Finish）。
2. **定义技能路由**：用户表达某种意图时，AI 应使用哪个技能或代理。
3. **提供 workflow-state 提示块**：钩子可将当前状态的提示块注入对话。

## 当前阶段模型

```text
Phase 1: Plan    -> 澄清构建内容，产出 prd.md 与必要研究
Phase 2: Execute -> 按 PRD 与规范实施，然后检查
Phase 3: Finish  -> 最终验证、沉淀经验并收尾
```

每个阶段包含编号步骤，如 `1.3 Configure context`（配置上下文）。这些编号不是 `task.json` 中的运行时字段，而是供 AI 与人阅读的工作流结构。

## 技能路由

`workflow.md` 按平台能力区分路由：

- 支持子代理的平台：实施默认分派 `trellis-implement`，检查分派 `trellis-check`。
- 不支持子代理的平台：主会话读取 `trellis-before-dev` 等技能，然后直接执行。

修改本地 AI 行为时，先更新 `workflow.md` 的路由说明，再检查对应平台的技能、命令或代理文件是否需要同步。

## 工作流状态提示块

`workflow.md` 底部可包含如下状态块：

```text
[workflow-state:no_task]
...
[/workflow-state:no_task]
```

钩子根据当前任务状态选择正确的块并注入对话。常见状态包括：

| 状态 | 含义 |
| --- | --- |
| `no_task` | 当前会话没有活动任务。 |
| `planning` | 任务仍处于需求、研究或上下文配置阶段。 |
| `in_progress` | 任务已进入实施与检查。 |
| `completed` | 任务已完成，等待收尾或归档。 |

用户希望修改“没有任务时是否创建任务”“何时可跳过创建任务”或“是否必须使用子代理”等策略时，应编辑这些状态块与其上方的路由表。

## 本地修改模式

常见变更：

| 目标 | 修改位置 |
| --- | --- |
| 添加阶段 | 更新 Phase Index、阶段正文、路由和状态块。 |
| 修改任务创建策略 | 更新 `no_task` 状态块与第 1 阶段说明。 |
| 修改默认实施与检查路径 | 更新第 2 阶段与技能路由。 |
| 修改收尾流程 | 更新第 3 阶段与 `finish-work` 相关说明。注意当前分工：第 3.4 步为 AI 驱动的代码提交（分批、用户确认），第 3.5 步为 `/finish-work`（归档与记录会话）。工作区有未提交变更时，`/finish-work` 拒绝执行。 |
| 修改平台差异 | 更新按平台分组的路由说明。 |

编辑后，让 AI 重新读取 `.trellis/workflow.md`；不要假设旧对话中的流程仍然有效。

## 与平台文件的关系

`workflow.md` 是本地工作流的语义中心，但每个平台也可有自己的入口文件：

- 技能，如 `trellis-brainstorm` 和 `trellis-check`。
- 命令、提示与工作流，如 continue 和 finish-work。
- 钩子，如 session-start 或 workflow-state 注入。

只修改 `workflow.md` 时，平台入口文件仍可能保留旧说明。用户希望改变“AI 实际做什么”时，也应检查相关平台目录。
