---
name: trellis-start
description: "通过读取 .trellis/ 中的工作流指南、开发者身份、git 状态、活动任务和项目规范，初始化 AI 开发会话。对新请求分类，并路由到需求讨论、直接编辑或任务工作流。适用于开始编码会话、恢复工作、启动新任务或重新建立项目上下文。"
---

# 开始会话

初始化由 Trellis 管理的开发会话。此平台没有会话开始 hook，因此按以下步骤手动加载等效的精简上下文。

---

## 步骤 1：当前状态
开发者身份、git 状态、当前任务、活动任务和日志位置。

```bash
python3 ./.trellis/scripts/get_context.py
```

若输出包含以 `Trellis update available:` 开头的行，在总结会话上下文时逐字复制完整行。不要缩短操作命令提示。

## 步骤 2：工作流概览
精简的阶段索引、请求分类规则、规划产物契约和步骤详情命令。

```bash
python3 ./.trellis/scripts/get_context.py --mode phase
```

完整指南位于 `.trellis/workflow.md`（按需读取）。

## 步骤 3：规范索引
发现包与规范层，再读取每个相关索引文件。

```bash
python3 ./.trellis/scripts/get_context.py --mode packages
cat .trellis/spec/guides/index.md
cat .trellis/spec/<package>/<layer>/index.md   # 对每个相关层执行
```

索引列出真正开始编码时应读取的具体规范文档。

## 步骤 4：确定下一步
步骤 1 已提供当前任务及其状态。检查任务目录：

- **活动任务状态为 `planning`，且没有 `prd.md`** → 阶段 1.1。加载 `trellis-brainstorm` 技能。
- **活动任务状态为 `planning`，且存在 `prd.md`** → 保持在阶段 1。轻量任务可以只有 PRD；复杂任务需要 `design.md` 与 `implement.md`。运行 `task.py start` 前先加载相关阶段 1 步骤详情。
- **活动任务状态为 `in_progress`** → 阶段 2 的步骤 2.1。加载步骤详情：
  ```bash
  python3 ./.trellis/scripts/get_context.py --mode phase --step 2.1 --platform codex
  ```
- **没有活动任务** → 先分类。简单对话或小任务，只询问本轮是否创建 Trellis 任务。复杂工作，询问是否允许创建 Trellis 任务并进入规划。若用户不同意，本会话跳过 Trellis。

---

## 技能路由速查

| 用户意图 | 技能 |
|---|---|
| 新功能或需求不明确 | `trellis-brainstorm` |
| 即将编写代码 | `trellis-before-dev` |
| 编码完成或质量检查 | `trellis-check` |
| 卡住或多次修复同一缺陷 | `trellis-break-loop` |
| 学到值得记录的知识 | `trellis-update-spec` |

完整规则与防止自我辩解的对照表位于 `.trellis/workflow.md`。
