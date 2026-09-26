# 修改本地工作流

用户希望修改 Trellis 阶段、下一步提示、是否创建任务、是否使用子代理，或何时检查与收尾时，应先编辑 `.trellis/workflow.md`。

## 先读取这些文件

1. `.trellis/workflow.md`
2. 当前平台的入口文件，例如技能、命令、提示或工作流
3. 当前任务的 `task.json` 和 `prd.md`

## 常见需求与修改位置

| 需求 | 修改位置 |
| --- | --- |
| 修改阶段名称或顺序 | `Phase Index` 与对应的 Phase 章节。 |
| 修改无活动任务时是否创建任务 | `[workflow-state:no_task]` 状态块。 |
| 修改规划期间的下一步 | 第 1 阶段与 `[workflow-state:planning]`。 |
| 修改 in_progress 期间是否要求代理 | 第 2 阶段与 `[workflow-state:in_progress]`。 |
| 修改完成后的收尾 | 第 3 阶段与 `[workflow-state:completed]`。 |
| 修改用户意图触发哪个技能 | 技能路由表（`Skill Routing`）。 |

## 修改步骤

1. 在 `.trellis/workflow.md` 中找到相关章节。
2. 修改规则时保留明确的触发条件与下一步行动。
3. 添加或重命名技能、代理时，同步平台目录中的对应文件。
4. 工作流状态变更只需编辑 `.trellis/workflow.md` 中的 `[workflow-state:STATUS]` 块。钩子只负责解析，会读取块中的内容。起始与结束标签的 STATUS 字符串必须一致（`[workflow-state:foo]…[/workflow-state:foo]`）；STATUS 不匹配的标签对会被静默丢弃。
5. 让 AI 重新读取 `.trellis/workflow.md`，不要继续使用旧对话中的规则。

## 示例：放宽任务创建要求

修改何时可跳过任务创建时，通常编辑 `[workflow-state:no_task]`：

```md
[workflow-state:no_task]
如果答案只是一次回复的解释，不修改文件，也不需要研究，则无需任务。
[/workflow-state:no_task]
```

如果正式的第 1 阶段流程也需要改变，同步第 1 阶段章节。

## 示例：某个平台不使用子代理

如果用户只希望某个平台避免使用子代理，先确认该平台是否在工作流中有独立分组。然后修改该平台组的第 2 阶段路由，不要删除所有平台的 `trellis-implement` / `trellis-check` 指令。

## `/trellis:continue` 路由表

`/trellis:continue` 通过判断接下来加载哪个阶段步骤来恢复任务。决策结合 `task.json.status` 与任务目录中的产物是否存在。该映射固定在命令自身中；添加自定义状态的定制版本必须同时扩展 workflow.md 标签块与此表。

| `status` | 产物状态 | 恢复位置 |
| --- | --- | --- |
| `planning` | 缺少 `prd.md` | 第 1.1 步（加载 `trellis-brainstorm`） |
| `planning` | 轻量任务，`prd.md` 已完成 | 请求启动审阅，然后运行 `task.py start` |
| `planning` | 复杂任务缺少 `design.md` 或 `implement.md` | 补齐缺少的规划产物 |
| `planning` | 复杂任务已有 `prd.md`、`design.md` 和 `implement.md` | 请求启动审阅，然后运行 `task.py start` |
| `in_progress` | 对话历史中没有实施记录 | 第 2.1 步（`trellis-implement`） |
| `in_progress` | 实施完成，尚未运行 `trellis-check` | 第 2.2 步（`trellis-check`） |
| `in_progress` | 检查通过 | 第 3.3 步（更新规范）→ 3.4（提交） |
| `completed` | 任务仍在活动任务树中 | 第 3.5 步（运行 `/trellis:finish-work` 归档） |

添加自定义状态（例如 `in-review`）时，在 `.trellis/workflow.md` 中添加 `[workflow-state:in-review]` 块以显示逐轮提示，并扩展此路由表。通常通过编辑 `/trellis:continue` 命令文件（`.{platform}/commands/trellis/continue.md` 或等价文件）添加一行，决定从哪里恢复。没有路由项时，`/trellis:continue` 会落入默认分支，用户无法进入预期步骤。

## 注意事项

`.trellis/workflow.md` 是项目本地工作流，而非不可变模板。用户可按团队习惯调整。编辑后，平台入口文件仍可能包含旧说明，也应检查。
