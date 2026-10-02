# 提交方案

拟议单一工作提交：`feat：更新工作台微圆角与细框线 UI`。

本会话改动全部属于同一可审查单元：
- `web/src/app/styles.css`
- `web/src/features/workspaces/ProjectWorkbench.tsx`
- `web/tests/e2e/workbench-modern-ui.spec.ts`
- `.trellis/spec/frontend/theme-assets.md`
- `design-qa.md`
- 本任务 `prd.md`、`design.md`、`implement.md`、`check-report.md`、`commit-plan.md`、`task.json`

未识别的未提交文件：无。临时预览后端、截图、日志和构建产物不纳入提交。

按 `.trellis/workflow.md` 阶段 3.4，展示方案后等待一次用户确认；确认后才提交，不 amend，不 push。归档和日志留在工作提交之后，其他任务不纳入本批。
