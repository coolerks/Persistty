# 本地上下文注入系统

Trellis 上下文注入的目标是让 AI 在合适的时间读取合适的文件，而不是依赖模型记忆。在用户项目中，注入由 `.trellis/` 脚本与平台钩子、代理和技能共同实现。

## 注入的上下文类型

| 类型 | 来源 | 用途 |
| --- | --- | --- |
| 会话上下文 | `.trellis/scripts/get_context.py` | 当前开发者、Git 状态、活动任务、活动任务列表、日志和包。 |
| 工作流上下文 | `.trellis/workflow.md` | 当前 Trellis 流程和下一步行动。 |
| 规范上下文 | `.trellis/spec/` 与任务 JSONL | 实施与检查期间必须遵循的规范。 |
| 任务上下文 | `.trellis/tasks/<task>/prd.md`、`design.md`、`implement.md`、`research/` | 当前任务需求、设计、执行计划和研究。 |
| 平台上下文 | 平台钩子、设置与代理 | 让不同 AI 工具通过各自机制读取上述文件。 |

## session-start（会话启动）

支持 session-start 的平台，在会话启动、清空、压缩或类似事件发生时注入 Trellis 概览。注入内容通常包括：

- 工作流摘要。
- 当前任务状态。
- 活动任务。
- 规范索引路径。
- 开发者身份与 Git 状态。

用户认为 AI 在新会话中不了解当前任务时，先检查平台的 session-start 钩子或等价机制是否已安装并运行。

## workflow-state（工作流状态）

workflow-state 是在每次用户交互时注入的轻量提示。根据当前任务状态，从 `.trellis/workflow.md` 选择一个块，如 `no_task`、`planning`、`in_progress` 或 `completed`。

用户希望修改“给定状态下 AI 接下来应该做什么”时，先编辑 `.trellis/workflow.md` 中对应状态块。

## 子代理上下文

implement 与 check 代理需要任务上下文。Trellis 有两种加载模式：

1. **hook push（钩子推送）**：平台钩子在代理启动前注入 JSONL 引用的文件，以及 `prd.md`、`design.md`（如果存在）、`implement.md`（如果存在）。
2. **agent pull（代理主动读取）**：代理定义要求代理在启动后读取活动任务、JSONL 上下文和任务产物。

两种模式中，任务目录下的 JSONL 都是规范与研究上下文清单。任务产物单独按以下顺序读取：`prd.md` → `design.md`（如果存在）→ `implement.md`（如果存在）。

## JSONL 读取规则

`implement.jsonl` 和 `check.jsonl` 每行包含一个 JSON 对象：

```jsonl
{"file": ".trellis/spec/backend/index.md", "reason": "Backend rules"}
```

读取器应跳过没有 `file` 字段的行（例如旧 `_example` 占位行）。配置 JSONL 时，AI 应只包含规范与研究文件，不要预登记将要修改的代码文件。

## 活动任务与上下文键

活动任务状态位于 `.trellis/.runtime/sessions/`，按会话隔离。钩子尝试从平台事件、环境变量、对话记录路径或 `TRELLIS_CONTEXT_ID` 解析上下文键。

Shell 命令看不到相同上下文键时，`task.py current --source` 可能报告无活动任务。此时应检查平台是否将会话身份传入 Shell，而不是手动编写全局当前任务文件。

## 本地定制位置

| 需求 | 修改位置 |
| --- | --- |
| 修改 session-start 注入内容 | 平台的 `session-start` 钩子或插件文件。 |
| 修改每轮 workflow-state 规则 | `.trellis/workflow.md` 中的 `[workflow-state:STATUS]` 块。平台 workflow-state 钩子逐字解析这些块，不嵌入回退文本。 |
| 修改子代理读取上下文的方式 | 平台代理定义、`inject-subagent-context` 钩子或代理前置指令。 |
| 修改 JSONL 校验与显示 | `.trellis/scripts/common/task_context.py`。 |
| 修改活动任务解析 | `.trellis/scripts/common/active_task.py`。 |

修改上下文注入后，验证两点：新会话能看到正确任务，子代理能看到正确的任务产物、规范与研究。
