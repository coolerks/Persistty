# 搜索修复与统一验收提交方案

前四轮6881b6c/e7b44a1/978097d/acfc15b已由用户提交。本会话不自动提交、推送或归档。建议一个完整提交：`fix：优化搜索性能并补齐工作台验收回归`。

安全单遍扫描、ignore剪枝、已编译literal与有界批量regex修复搜索重复开销；文件名请求依赖稳定folder ID，跨15秒项目轮询仍保持，查询/版本/根/关闭才取消。纳入安全/性能/跨轮询回归、当前UI可访问名称与真实终端测试迁移、W05/W06/W07统一验收记录及规范。实际门禁与保留失败见[检查报告](check-report.md)和[统一验收报告](overnight-acceptance.md)，未运行不记通过。

本会话修改文件（含新增与删除；旧discovery迁至测试对照，共53项）：

- `.trellis/spec/backend/file-name-search.md`
- `.trellis/spec/backend/filesystem-guidelines.md`
- `.trellis/spec/backend/search-git-contract.md`
- `.trellis/spec/frontend/hook-guidelines.md`
- `.trellis/spec/frontend/quality-guidelines.md`
- `.trellis/tasks/09-30-workbench-editor-recovery/acceptance-report.md`
- `.trellis/tasks/09-30-workbench-editor-recovery/implement.md`
- `.trellis/tasks/09-30-workbench-editor-recovery/prd.md`
- `.trellis/tasks/09-30-workbench-editor-recovery/task.json`
- `.trellis/tasks/10-01-search-readonly-git/acceptance-report.md`
- `.trellis/tasks/10-01-search-readonly-git/implement.md`
- `.trellis/tasks/10-01-search-readonly-git/prd.md`
- `.trellis/tasks/10-01-search-readonly-git/task.json`
- `.trellis/tasks/10-02-privileged-file-edit/check-report.md`
- `.trellis/tasks/10-02-privileged-file-edit/implement.md`
- `.trellis/tasks/10-02-privileged-file-edit/prd.md`
- `.trellis/tasks/10-02-privileged-file-edit/task.json`
- `.trellis/tasks/10-02-workbench-modern-ui/check-report.md`
- `.trellis/tasks/10-02-workbench-modern-ui/commit-plan.md`
- `.trellis/tasks/10-02-workbench-modern-ui/design.md`
- `.trellis/tasks/10-02-workbench-modern-ui/implement.md`
- `.trellis/tasks/10-02-workbench-modern-ui/overnight-acceptance.md`
- `.trellis/tasks/10-02-workbench-modern-ui/prd.md`
- `.trellis/tasks/10-02-workbench-modern-ui/task.json`
- `AGENTS.md`
- `internal/files/snapshot.go`
- `internal/files/snapshot_test.go`
- `internal/search/batch_engine.go`
- `internal/search/candidates.go`
- `internal/search/discovery.go`
- `internal/search/discovery_reference_test.go`
- `internal/search/engine.go`
- `internal/search/file_names.go`
- `internal/search/file_names_test.go`
- `internal/search/filter_paths.go`
- `internal/search/native_engine.go`
- `internal/search/native_names.go`
- `internal/search/native_names_reference_test.go`
- `internal/search/performance_test.go`
- `internal/search/scan_performance_test.go`
- `internal/search/service.go`
- `web/src/features/workspaces/FileQuickOpen.test.tsx`
- `web/src/features/workspaces/FileQuickOpen.tsx`
- `web/tests/e2e/editor-assets.spec.ts`
- `web/tests/e2e/editor-recovery.spec.ts`
- `web/tests/e2e/editor-workbench.spec.ts`
- `web/tests/e2e/git-layout.spec.ts`
- `web/tests/e2e/git-requests.spec.ts`
- `web/tests/e2e/screenshot-adjustments.spec.ts`
- `web/tests/e2e/terminal.spec.ts`
- `web/tests/e2e/w06-local.spec.ts`
- `web/tests/e2e/workbench-interactions.spec.ts`
- `web/tests/e2e/workbench-modern-ui.spec.ts`

未识别改动：无。数据库副本、虚构测试配置、截图、trace、日志及构建产物保留在忽略目录；不纳入用户真实文件或认证数据。

按[项目工作流](../../workflow.md)阶段3.4“一次展示方案，一次确认”，后续获得提交授权才执行git add/commit；本轮用户要求直接验收与记录，因此不中途等待提交确认。
