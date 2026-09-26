# 本地工作区记忆系统

`.trellis/workspace/` 存储跨会话记忆，帮助 AI 与人在不同窗口、不同日期间理解之前发生的工作。

## 目录结构

```text
.trellis/workspace/
├── index.md
└── <developer>/
    ├── index.md
    ├── journal-1.md
    └── journal-2.md
```

| 文件 | 用途 |
| --- | --- |
| `.trellis/.developer` | 当前开发者身份。 |
| `.trellis/workspace/index.md` | 全局工作区概览。 |
| `.trellis/workspace/<developer>/index.md` | 开发者的会话索引。 |
| `.trellis/workspace/<developer>/journal-N.md` | 会话日志。 |

## 开发者身份

首次运行：

```bash
python3 ./.trellis/scripts/init_developer.py <name>
```

这会创建 `.trellis/.developer` 与对应工作区目录。AI 不应随意修改开发者身份；身份错误时，先确认当前项目由谁使用。

## 日志

`journal-N.md` 记录每个会话已完成或部分完成的工作。默认每份日志约容纳 2000 行，之后轮换到下一份文件。

记录会话的常用命令：

```bash
python3 ./.trellis/scripts/add_session.py \
  --title "Session title" \
  --summary "What changed" \
  --commit "abc1234"
```

没有提交的规划或审阅工作也可通过 `--no-commit` 或空提交值记录。

## 工作区记忆与任务的关系

| 系统 | 存储内容 |
| --- | --- |
| `.trellis/tasks/` | 特定任务的需求、设计、研究与状态。 |
| `.trellis/workspace/` | 跨任务、跨会话的工作记录。 |
| `.trellis/spec/` | 沉淀为长期约定的工程知识。 |

只对当前任务有用的信息，放入任务目录。
描述当前会话发生了什么的信息，放入工作区日志。
未来每次编码都应遵循的信息，放入规范。

## 本地定制位置

| 需求 | 修改位置 |
| --- | --- |
| 修改日志最大行数 | `.trellis/config.yaml` 中的 `max_journal_lines`。 |
| 修改会话自动提交消息 | `.trellis/config.yaml` 中的 `session_commit_message`。 |
| 修改会话内容格式 | `.trellis/scripts/add_session.py`。 |
| 修改工作区在上下文中的显示方式 | `.trellis/scripts/common/session_context.py`。 |

## AI 使用规则

AI 不应把工作区视为唯一权威来源。恢复任务时，先读取当前任务，再用工作区补充背景。任务完成后，在工作区记录重要过程说明；产生长期规则时，更新规范。
