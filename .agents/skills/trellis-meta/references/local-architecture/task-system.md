# 本地任务系统

Trellis 任务系统完全存储在用户项目的 `.trellis/tasks/` 下。每个任务是一个目录，包含需求、上下文、研究、状态与关系信息。

## 任务目录结构

```text
.trellis/tasks/
├── 04-28-example-task/
│   ├── task.json
│   ├── prd.md
│   ├── design.md
│   ├── implement.md
│   ├── implement.jsonl
│   ├── check.jsonl
│   └── research/
└── archive/
    └── 2026-04/
```

| 文件 | 用途 |
| --- | --- |
| `task.json` | 任务元数据：状态、负责人、优先级、分支、父子任务等字段。 |
| `prd.md` | 需求、约束与验收标准。轻量任务可只使用 PRD。 |
| `design.md` | 复杂任务的技术设计：边界、契约、数据流、兼容性与取舍。 |
| `implement.md` | 复杂任务的执行计划：有序检查清单、验证命令、审查关卡和回滚点。 |
| `implement.jsonl` | implement 代理必须先读取的规范与研究文件清单。 |
| `check.jsonl` | check 代理必须先读取的规范与研究文件清单。 |
| `research/` | 研究产物；复杂发现不应只保留在聊天中。 |

## `task.json`

`task.json` 记录任务状态与元数据。常见字段：

| 字段 | 含义 |
| --- | --- |
| `id` / `name` / `title` | 任务标识与标题。 |
| `status` | 状态，如 `planning`、`in_progress`、`review` 或 `completed`。 |
| `priority` | `P0`、`P1`、`P2`、`P3`。 |
| `creator` / `assignee` | 创建者与负责人。 |
| `package` | monorepo 中的目标包，可为空。 |
| `branch` / `base_branch` | 工作分支与 PR 目标分支。 |
| `children` / `parent` | 父子任务关系。 |
| `commit` / `pr_url` | 完成后的提交与 PR 信息。 |
| `meta` | 扩展字段。 |

## 父子任务树

父子任务关系用于组织工作。父任务在同一来源需求下聚合相关交付物；它不是依赖调度器，也不能替代子任务自己的规划产物。

请求包含多个可独立验证的交付物时，使用父任务。父任务负责：

- 来源需求与面向用户的范围。
- 子任务映射与职责边界。
- 跨子任务验收标准与最终集成审查。

对于可独立经历规划、实施、检查和归档的交付物，使用子任务。子任务有依赖时，将依赖写入该子任务的 `prd.md` / `implement.md`，不要依赖树位置来隐含顺序。

通过以下命令创建新子任务：

```bash
python3 ./.trellis/scripts/task.py create "<child title>" --description "<one-line summary>" --slug <child-slug> --parent <parent-dir>
```

通过以下命令关联或解除关联已有任务：

```bash
python3 ./.trellis/scripts/task.py add-subtask <parent-dir> <child-dir>
python3 ./.trellis/scripts/task.py remove-subtask <parent-dir> <child-dir>
```

父任务的 `children` 是历史列表。子任务归档后，Trellis 保留其名称，让 `[2/3 done]` 这类进度在已完成子任务移入 `archive/` 后仍有意义。

AI 不应将阶段编号当作任务状态。任务进度主要由 `status`、产物是否存在（`prd.md`、可选的 `design.md` / `implement.md`）、子代理模式是否配置 JSONL 上下文，以及 `workflow.md` 中的阶段说明决定。

## 活动任务

用户看到的是“当前任务”，但 Trellis 按会话存储活动任务状态。

```text
.trellis/.runtime/sessions/<context-key>.json
```

`task.py start` 将任务路径写入当前会话的运行时文件。`task.py current --source` 显示当前任务及其来源。不同 AI 窗口可指向不同任务而不相互覆盖。

如果平台或 Shell 环境没有稳定会话身份，`task.py start` 可能无法设置活动任务。AI 应读取错误、检查平台钩子与会话环境，不要退回共享全局指针。

## JSONL 上下文

`implement.jsonl` 与 `check.jsonl` 是子代理首先读取的上下文清单。它们不替代 `implement.md`；`implement.md` 是供人阅读的执行计划。

格式：

```jsonl
{"file": ".trellis/spec/cli/backend/index.md", "reason": "Backend conventions"}
{"file": ".trellis/tasks/04-28-example/research/api.md", "reason": "API research"}
```

规则：

- 包含规范与研究文件。
- 不要包含将要修改的代码文件。
- 不要把聊天中的临时结论作为唯一上下文。
- 读取器跳过没有 `file` 字段的行。旧 `{"_example": ...}` 占位行会被 `task.py validate` 拒绝，应删除。

## 常用命令

```bash
python3 ./.trellis/scripts/task.py create "<title>" --description "<one-line summary>" --slug <slug>
python3 ./.trellis/scripts/task.py start <task>
python3 ./.trellis/scripts/task.py current --source
python3 ./.trellis/scripts/task.py add-context <task> implement <file> <reason>
python3 ./.trellis/scripts/task.py validate <task>
python3 ./.trellis/scripts/task.py finish
python3 ./.trellis/scripts/task.py archive <task>
```

修改任务系统时，AI 应优先使用脚本命令维护结构。只有脚本无法满足需要时，才直接编辑 JSON 或 Markdown。

## 本地定制位置

| 需求 | 修改位置 |
| --- | --- |
| 修改默认任务模板 | `.trellis/scripts/common/task_store.py` 与任务创建指令。 |
| 修改状态语义 | `.trellis/workflow.md`、workflow-state 钩子逻辑与任务使用约定。 |
| 添加任务生命周期动作 | `.trellis/config.yaml` 中的 `hooks.after_*`。 |
| 修改上下文规则 | `.trellis/workflow.md` 的规划产物指引与相关平台代理、钩子指令。 |
| 修改归档策略 | `.trellis/scripts/common/task_store.py` / `task_utils.py`。 |

这些都是用户项目的本地文件。除非用户希望贡献上游，否则不要默认编辑 Trellis CLI 源码。
