# 修改本地任务生命周期

任务生命周期包括创建、启动、上下文配置、结束、归档、父子任务和生命周期钩子。默认定制目标是 `.trellis/tasks/`、`.trellis/config.yaml` 和 `.trellis/scripts/`。

## 先读取这些文件

1. `.trellis/workflow.md`
2. `.trellis/config.yaml`
3. `.trellis/scripts/task.py`
4. `.trellis/scripts/common/task_store.py`
5. `.trellis/scripts/common/task_utils.py`
6. 当前任务的 `.trellis/tasks/<task>/task.json`

## 常见需求与修改位置

| 需求 | 修改位置 |
| --- | --- |
| 任务创建后自动同步外部系统 | `.trellis/config.yaml` 中的 `hooks.after_create`。 |
| 任务启动后自动更新状态 | `.trellis/config.yaml` 中的 `hooks.after_start`。 |
| 任务结束后运行脚本 | `.trellis/config.yaml` 中的 `hooks.after_finish`。 |
| 归档后清理外部资源 | `.trellis/config.yaml` 中的 `hooks.after_archive`。 |
| 修改默认任务字段 | `.trellis/scripts/common/task_store.py`。 |
| 修改任务解析与搜索 | `.trellis/scripts/common/task_utils.py`。 |
| 修改活动任务行为 | `.trellis/scripts/common/active_task.py`。 |

## 生命周期钩子

`.trellis/config.yaml` 支持：

```yaml
hooks:
  after_create:
    - "python3 .trellis/scripts/hooks/my_sync.py create"
  after_start:
    - "python3 .trellis/scripts/hooks/my_sync.py start"
  after_finish:
    - "python3 .trellis/scripts/hooks/my_sync.py finish"
  after_archive:
    - "python3 .trellis/scripts/hooks/my_sync.py archive"
```

钩子命令接收 `TASK_JSON_PATH` 环境变量，指向当前任务的 `task.json`。钩子失败通常应给出警告，但不阻断主要任务操作。

## 修改任务字段

用户希望添加项目本地字段时，应优先放在 `task.json` 的 `meta` 下，避免破坏现有脚本对标准字段的假设。

示例（以下是 JSON 对象中的字段片段）：

```text
"meta": {
  "linearIssue": "ENG-123",
  "risk": "high"
}
```

如果确实需要修改标准字段，应检查所有读取 `task.json` 的本地脚本。

## 修改活动任务

活动任务是存储在 `.trellis/.runtime/sessions/` 中的会话级状态。不要退回全局 `.current-task` 模型。用户希望修改活动任务行为时，应编辑：

- `.trellis/scripts/common/active_task.py`
- 平台钩子或 Shell 会话桥接
- `.trellis/workflow.md` 中的活动任务说明

### `task.py create` 设置活动指针

`.trellis/scripts/common/task_store.py` 中的 `cmd_create` 在写入新任务目录后，会尽力调用 `set_active_task`。其行为如下：

- 调用 Shell 带有会话身份时（`TRELLIS_CONTEXT_ID` 环境变量，或 `resolve_context_key` 能识别的任意平台会话环境变量，见 `active_task.py:_ENV_SESSION_KEYS`），`.trellis/.runtime/sessions/<context_key>.json` 中的会话指针被重写为新任务。任务的 `status=planning`，下一次 `UserPromptSubmit` 就会触发 `[workflow-state:planning]`。
- 无法获得会话身份时（在 AI 会话外直接调用 CLI，或平台未将身份传给 Shell），仍创建任务目录并写入 `status=planning`，但不改变活动指针。用户回到 AI 会话后，可通过 `task.py start <dir>` 关联任务。

这样，`[workflow-state:planning]` 就成为 `task.py create` 后头脑风暴与 JSONL 策划期间的实时状态提示。R7 之前的行为会让提示一直停留在 `no_task`，直到执行 `task.py start`，因此规划块实际上无法触发。

如果自行修改 `task.py` 添加新的创建路径（例如绕过 `cmd_create` 的外部导入），应检查该路径是否也调用 `set_active_task`。缺少该调用，新建任务不会显示为活动任务。完整状态写入表见 `.trellis/spec/cli/backend/workflow-state-contract.md`（这是上游 CLI 仓库的规范路径，用户项目不一定包含该文件）。

## 修改步骤

1. 通过 `python3 ./.trellis/scripts/task.py current --source` 确认当前任务。
2. 读取当前任务的 `task.json`，确认状态与字段。
3. 配置需求优先编辑 `.trellis/config.yaml`。
4. 脚本行为需求再编辑 `.trellis/scripts/`。
5. 如果 AI 流程改变，同步 `.trellis/workflow.md`。

## 禁止事项

- 不要直接编辑 `.trellis/.runtime/sessions/` 来“修复”业务状态。
- 不要将项目私有字段硬编码到脚本中；优先使用 `meta`。
- 不要默认要求用户 fork Trellis CLI。
