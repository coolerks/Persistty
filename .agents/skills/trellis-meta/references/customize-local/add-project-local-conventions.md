# 添加项目本地约定

用户往往不需要修改 Trellis 的运行机制，而是需要让本地 AI 理解团队约定。此时应优先使用 `.trellis/spec/` 或项目本地技能，而不是修改 `trellis-meta`。

## 内容放在哪里

| 内容类型 | 位置 |
| --- | --- |
| 代码必须遵循的规则 | `.trellis/spec/<layer>/` |
| 跨层思考方法 | `.trellis/spec/guides/` |
| 项目特定流程所需的 AI 能力 | 平台本地技能 |
| 一次性任务材料 | `.trellis/tasks/<task>/` |
| 会话总结 | `.trellis/workspace/<developer>/journal-N.md` |

## 创建项目本地技能

如果用户希望 AI 了解“这个项目如何定制 Trellis”，可创建本地技能：

```text
.claude/skills/trellis-local/
└── SKILL.md
```

示例：

```md
---
name: trellis-local
description: "本仓库的项目本地 Trellis 定制。修改本项目的 Trellis 工作流、钩子、本地代理或团队特定约定时使用。"
---

# Trellis 本地定制

## 本地范围

此技能只记录本仓库的 Trellis 定制。

## 自定义工作流规则

- ...

## 本地钩子变更

- ...

## 本地代理变更

- ...
```

多平台项目应在其他平台技能目录中放置等价版本；支持共享层的平台也可使用 `.agents/skills/`。

## 写入 `.trellis/spec/`

如果内容是编码约定，就写入规范。例如：

```text
.trellis/spec/backend/error-handling.md
.trellis/spec/frontend/components.md
.trellis/spec/guides/cross-platform-thinking-guide.md
```

写入后更新对应的 `index.md`，让 AI 能从入口找到新规则。

## 让当前任务使用新约定

编写规范后，将它加入当前任务上下文：

```bash
python3 ./.trellis/scripts/task.py add-context <task> implement ".trellis/spec/backend/error-handling.md" "Error handling conventions"
python3 ./.trellis/scripts/task.py add-context <task> check ".trellis/spec/backend/error-handling.md" "Review error handling"
```

## 不要在 `trellis-meta` 中保存项目私有规则

`trellis-meta` 是用于理解 Trellis 架构和本地定制入口的公开技能。项目私有内容应放在：

- `.trellis/spec/`
- 项目本地技能
- 当前任务
- 工作区日志

这样可避免未来更新 Trellis 内置 `trellis-meta` 时覆盖团队自己的约定。
