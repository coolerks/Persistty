# 执行计划

1. 读取两段引用聊天、现有 spec、Trellis 工作流和本地解析脚本；记录已覆盖与遗漏需求。
2. 保存本次所有目标 Markdown 原文、机器标记、代码和引用清单，用于翻译完整性检查。
3. 按目录并行翻译技能、引用文档；主会话翻译 AGENTS/workflow/工作区/代理说明，并补充归属 spec。
4. 安装暂存译稿；核对文件集合、技能 frontmatter、workflow 提取、工作区定位、Markdown 链接和 JSON 例子。
5. 分派 Trellis check 执行全范围审查，修正遗漏和语义漂移；检查上下文清单与空白。
6. 将验证结果、未执行的产品测试和后续边界写入 check-report。向用户展示结果与提交方案，不自动提交此前未识别的初始化文件。

## 验证
- python3 .trellis/scripts/task.py validate 09-26-chinese-docs
- python3 .trellis/scripts/get_context.py --mode phase --platform codex
- 各阶段 1.0..3.5 的实际步骤提取及 workflow-state 块检查
- 本地文件链接/锚点、代码围栏、技能 name、机器标签对照
- JSON 围栏解析与逐文件尾随空白/末尾换行检查
- git diff --check；新增未跟踪文件单独检查

本仓库无产品源码，Go/React/Playwright/systemd 产品测试不适用；文档检查不替代 Terminal Spike 或安全运行验收。

## 当前结果

2026-09-26：中文化与规范补充已完成，68 份原有 Markdown 内容更新，其他既有中文文档保留。46 份技能及引用译稿已安装；阶段提取、状态块、实际 hook 输出、技能/代理元数据、链接、JSON 与空白检查通过。全范围 Trellis check 审查及最终复验通过，结果见 check-report.md。

本任务仅完成文档工作；Go/React/Playwright/systemd 产品验收不适用。阶段 3.3 的经验已写入 guides/index.md 与本任务研究记录。阶段 3.4 的具体文件和两批提交方案见 commit-plan.md，尚未提交/归档。此前初始化文件单列，须用户确认包含；既有 bootstrap 任务保持原状态。
