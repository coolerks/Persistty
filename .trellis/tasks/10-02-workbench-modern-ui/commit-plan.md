# 第二轮提交方案

第一轮 `6881b6c` 已由用户提交。本轮拟议单一完整提交：`feat：细化工作台 UI 与文件名搜索`。

该变更统一用户截图要求的工作台/项目页控件，并包含名称搜索的服务、DTO、浏览器与单测、owner 规范和任务验收记录；拆开后顶栏入口会缺少协议，故归为同一可审查单元。

本会话修改文件：

- `.trellis/spec/backend/file-name-search.md`
- `.trellis/spec/backend/index.md`
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
- `internal/httpapi/search_git.go`
- `internal/httpapi/search_git_test.go`
- `internal/search/discovery.go`
- `internal/search/file_names.go`
- `internal/search/file_names_test.go`
- `internal/search/service.go`
- `tests/fixtures/search-git.json`
- `web/src/app/App.tsx`
- `web/src/app/styles.css`
- `web/src/features/git/GitFiles.tsx`
- `web/src/features/settings/ThemeSelect.tsx`
- `web/src/features/workspaces/FileQuickOpen.test.tsx`
- `web/src/features/workspaces/FileQuickOpen.tsx`
- `web/src/features/workspaces/ProjectWorkbench.tsx`
- `web/src/features/workspaces/ProjectsPage.tsx`
- `web/src/lib/api/search-git-client.ts`
- `web/src/lib/api/search-git-decoder.test.ts`
- `web/src/lib/api/search-git-decoder.ts`
- `web/tests/e2e/git-hover.spec.ts`
- `web/tests/e2e/git-layout.spec.ts`
- `web/tests/e2e/projects-modern-ui.spec.ts`
- `web/tests/e2e/workbench-interactions.spec.ts`
- `web/tests/e2e/workbench-modern-ui.spec.ts`

未识别的未提交文件：无。临时服务/数据库/tmux、截图、测试 traces、编译产物与日志全部忽略，不纳入提交。

按 [.trellis/workflow.md](../../workflow.md) 阶段 3.4 的“一次展示方案，一次确认”，等待用户回复 ok/行后再执行 git add/commit；不 amend、不 push，不自动归档其他活动任务。
