# 第三轮提交方案

前两轮 `6881b6c`/`e7b44a1` 已由用户提交。本轮拟议单一完整提交：`feat：重做项目欢迎页并增强文件名搜索`。

按三张标注截图细化 Git 标签/刷新图标、圆头分隔线和搜索关闭对齐，重做欢迎页并移除独立终端导航；名称查询在 rg 不可用时使用同安全树 Go glob/ignore，保留配额/版本/取消，并统一 HTTP 工具预算。测试、规范与报告一并纳入。

本会话修改文件：

- `.trellis/spec/backend/file-name-search.md`
- `.trellis/spec/backend/search-git-contract.md`
- `.trellis/spec/frontend/search-git-workbench.md`
- `.trellis/spec/frontend/theme-assets.md`
- `.trellis/tasks/10-02-workbench-modern-ui/check-report.md`
- `.trellis/tasks/10-02-workbench-modern-ui/commit-plan.md`
- `.trellis/tasks/10-02-workbench-modern-ui/design.md`
- `.trellis/tasks/10-02-workbench-modern-ui/implement.md`
- `.trellis/tasks/10-02-workbench-modern-ui/prd.md`
- `.trellis/tasks/10-02-workbench-modern-ui/task.json`
- `design-qa.md`
- `internal/httpapi/router.go`
- `internal/httpapi/tool_deadline_test.go`
- `internal/search/discovery.go`
- `internal/search/file_names.go`
- `internal/search/file_names_test.go`
- `web/src/app/App.test.tsx`
- `web/src/app/App.tsx`
- `web/src/app/styles.css`
- `web/src/features/git/GitPanel.tsx`
- `web/src/features/workspaces/FileQuickOpen.tsx`
- `web/src/features/workspaces/ProjectsPage.tsx`
- `web/tests/e2e/projects-modern-ui.spec.ts`
- `web/tests/e2e/workbench-modern-ui.spec.ts`
- `internal/search/native_names.go`

未识别改动：无。临时服务、fixture、截图、日志和测试产物全部忽略。

按[项目工作流](../../workflow.md)阶段3.4“一次展示方案，一次确认”，用户确认后才 git add/commit；不 amend、不 push，不归档其他活动任务。
