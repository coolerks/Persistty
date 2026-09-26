# 修改本地上下文加载

上下文加载决定 AI 何时读取工作流、任务、规范、研究、工作区和 Git 状态。用户说“AI 不知道当前任务”“代理没有读取规范”或“上下文太多／太少”时，读取本页。

## 先读取这些文件

1. `.trellis/workflow.md`
2. `.trellis/scripts/get_context.py`
3. `.trellis/scripts/common/session_context.py`
4. `.trellis/scripts/common/task_context.py`
5. `.trellis/scripts/common/active_task.py`
6. 当前平台的钩子或代理文件
7. 当前任务的 `implement.jsonl` / `check.jsonl`

## 上下文来源

| 来源 | 用途 |
| --- | --- |
| `.trellis/workflow.md` | 工作流与下一步行动提示。 |
| `.trellis/tasks/<task>/prd.md` | 当前任务需求。 |
| `.trellis/tasks/<task>/design.md` | 复杂任务的技术设计。 |
| `.trellis/tasks/<task>/implement.md` | 复杂任务的执行计划。 |
| `.trellis/tasks/<task>/implement.jsonl` | 实施前要读取的规范与研究。 |
| `.trellis/tasks/<task>/check.jsonl` | 检查时要读取的规范与研究。 |
| `.trellis/spec/` | 项目规范。 |
| `.trellis/workspace/` | 会话记录。 |
| Git 状态 | 当前工作区变更。 |

## 常见需求与修改位置

| 需求 | 修改位置 |
| --- | --- |
| 新会话中注入更多／更少信息 | `session_context.py` 或平台的 `session-start` 钩子。 |
| 修改每次用户输入时的提示 | `.trellis/workflow.md` 中的 `[workflow-state:STATUS]` 块。`inject-workflow-state` 钩子只负责解析，逐字读取该块。 |
| 代理没有读取规范 | 任务 JSONL、代理前置指令、`inject-subagent-context` 钩子。 |
| 活动任务丢失 | `active_task.py` 与平台会话身份的传递。 |
| 修改 JSONL 校验规则 | `task_context.py`。 |

## JSONL 规则

`implement.jsonl` / `check.jsonl` 是关键的上下文加载接口：

```jsonl
{"file": ".trellis/spec/backend/index.md", "reason": "Backend conventions"}
{"file": ".trellis/tasks/04-28-x/research/api.md", "reason": "API research"}
```

只包含规范与研究文件。不要把将要修改的代码文件放入这些清单；代理会在实施时自行读取代码文件。

## 修改会话上下文

如果用户希望每个新会话都能看到更多项目状态，应编辑：

- `.trellis/scripts/common/session_context.py`
- 对应平台的 `session-start` 钩子

上下文不能无限增长。应优先注入索引和路径，让 AI 按需读取详细文件。

## 修改子代理上下文

先判断平台使用哪种模式：

- hook push（钩子推送）：编辑 `inject-subagent-context` 钩子。
- agent pull（代理主动读取）：编辑对应 `trellis-implement` / `trellis-check` 代理文件中的读取步骤。

两种模式都要确保代理最终读取：

1. 活动任务
2. 对应 JSONL
3. JSONL 引用的规范与研究
4. `prd.md`
5. `design.md`（如果存在）
6. `implement.md`（如果存在）

## 排查顺序

```bash
python3 ./.trellis/scripts/task.py current --source
python3 ./.trellis/scripts/task.py list-context <task>
python3 ./.trellis/scripts/task.py validate <task>
python3 ./.trellis/scripts/get_context.py --mode packages
```

编辑钩子或代理之前，先确认任务与 JSONL 正确。
