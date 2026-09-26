---
name: trellis-continue
description: "恢复当前任务。加载工作流阶段索引，判断应从哪个阶段和步骤继续，再通过 get_context.py --mode phase 获取步骤详情。适用于返回进行中的任务、需要确定下一步工作时。"
---

# 继续当前任务

恢复当前任务，从 `.trellis/workflow.md` 中正确的阶段和步骤继续。

---

## 步骤 1：加载当前上下文

```bash
python3 ./.trellis/scripts/get_context.py
```

确认当前任务、git 状态与最近提交。

## 步骤 2：加载阶段索引

```bash
python3 ./.trellis/scripts/get_context.py --mode phase
```

显示阶段索引（规划 / 执行 / 收尾）及路由和技能映射。

## 步骤 3：判断当前进度

`get_context.py` 显示活动任务的 `status` 字段。根据 `status` 与产物是否存在进行路由。此命令帮助用户免于记忆 Trellis 流程，但它本身不构成实施批准。

- `status=planning`，且没有 `prd.md` → **1.1**（加载 `trellis-brainstorm`）
- `status=planning`，且只有 `prd.md` → 判断任务是轻量还是复杂。轻量任务可以进入 **1.4** 评审；复杂任务返回 **1.1**，补充 `design.md` 与 `implement.md`。
- `status=planning`，复杂任务产物齐全，但子代理 jsonl 尚未整理（为空，或只有旧版 `_example` 占位行）→ **1.3**
- `status=planning`，所需产物齐全，且所需 jsonl 已整理或使用 inline 模式 → **1.4**（请求启动评审；仅在用户确认后运行 `task.py start`）
- `status=in_progress`，实施尚未开始 → **2.1**
- `status=in_progress`，实施完成但未检查 → **2.2**
- `status=in_progress`，检查通过 → **3.3**（更新规范）→ **3.4**（提交）
- `status=completed`（少见，通常立即归档）→ 归档流程

阶段规则（完整详情位于 `.trellis/workflow.md`）：

1. 阶段内**按顺序**执行步骤，不能跳过 `[required]` 步骤
2. 若所需输出已经存在，`[once]` 步骤视为完成。只有轻量任务才可以仅有 `prd.md`；复杂任务还需要 `design.md` 与 `implement.md`。
3. 若新发现要求调整，可以返回更早的阶段

## 步骤 4：加载具体步骤

确定恢复位置后：

```bash
python3 ./.trellis/scripts/get_context.py --mode phase --step <X.X> --platform codex
```

遵循加载的指令。每个 `[required]` 步骤完成后，进入下一步。

---

## 参考

完整工作流与详细阶段步骤位于 `.trellis/workflow.md`。此命令只是入口，规范性指南以该文件为准。
