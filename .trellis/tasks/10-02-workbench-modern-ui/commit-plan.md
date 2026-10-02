# 第四轮提交方案

前三轮6881b6c/e7b44a1/978097d已由用户提交。本轮拟议单一完整提交：`feat：细化工作台布局并增强全文搜索兜底`。

修复终端收起底边和状态栏居中，按截图移除指定提示/对象ID、显示Git变更数量并稳定加载布局，统一紧凑搜索交互；全文搜索复用安全占位树和Go regexp/glob兜底，保持原字节、版本与替换确认。纳入测试、规范及检查报告，修复名称规范误写内容。

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
- `internal/search/discovery.go`
- `internal/search/engine.go`
- `internal/search/file_names.go`
- `internal/search/filter_paths.go`
- `internal/search/native_engine.go`
- `internal/search/native_engine_test.go`
- `internal/search/service.go`
- `internal/search/types.go`
- `web/src/app/styles.css`
- `web/src/features/git/GitPanel.tsx`
- `web/src/features/git/GitSections.tsx`
- `web/src/features/search/SearchPanel.tsx`
- `web/src/features/workspaces/FileEditor.tsx`
- `web/src/features/workspaces/FileQuickOpen.tsx`
- `web/src/features/workspaces/ProjectWorkbench.tsx`
- `web/tests/e2e/git-requests.spec.ts`
- `web/tests/e2e/search-fallback.spec.ts`
- `web/tests/e2e/w06-local.spec.ts`
- `web/tests/e2e/workbench-modern-ui.spec.ts`

未识别改动：无；依赖、产物、fixture/截图/日志保持忽略。

按[项目工作流](../../workflow.md)阶段3.4“一次展示方案，一次确认”，用户确认后才git add/commit；不amend、不push，不归档其他任务。
