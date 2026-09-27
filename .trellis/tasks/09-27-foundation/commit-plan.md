# W01 拟议提交

## 顺序与消息
1. `实现 W01 基础认证服务与项目工作台`

单个完整变更单元：Go 配置/认证/SQLite/只读元数据 API、React 认证路由与主题、协议测试、依赖锁与许可证、部署示例、同步规范与父子任务实施记录。暂不包含归档或开发者会话日志；得到用户确认后才 stage/commit，不 amend、不 push。

## 文件清单
- `.trellis/spec/backend/database-guidelines.md`
- `.trellis/spec/backend/directory-structure.md`
- `.trellis/spec/backend/http-api.md`
- `.trellis/spec/backend/index.md`
- `.trellis/spec/backend/quality-guidelines.md`
- `.trellis/spec/backend/security-config.md`
- `.trellis/spec/backend/terminal-lifecycle.md`
- `.trellis/spec/backend/websocket-protocol.md`
- `.trellis/spec/frontend/component-guidelines.md`
- `.trellis/spec/frontend/directory-structure.md`
- `.trellis/spec/frontend/editor-terminal-lifecycle.md`
- `.trellis/spec/frontend/hook-guidelines.md`
- `.trellis/spec/frontend/index.md`
- `.trellis/spec/frontend/quality-guidelines.md`
- `.trellis/spec/frontend/state-management.md`
- `.trellis/spec/frontend/theme-assets.md`
- `.trellis/tasks/09-26-requirements-research/check-report.md`
- `.trellis/tasks/09-26-requirements-research/design.md`
- `.trellis/tasks/09-26-requirements-research/implement.md`
- `.trellis/tasks/09-26-requirements-research/prd.md`
- `.trellis/tasks/09-26-requirements-research/task.json`
- `README.md`
- `.trellis/spec/backend/foundation-contract.md`
- `.trellis/tasks/09-27-foundation/backend-report.md`
- `.trellis/tasks/09-27-foundation/browser-report.md`
- `.trellis/tasks/09-27-foundation/check-report.md`
- `.trellis/tasks/09-27-foundation/check.jsonl`
- `.trellis/tasks/09-27-foundation/design.md`
- `.trellis/tasks/09-27-foundation/frontend-report.md`
- `.trellis/tasks/09-27-foundation/implement.jsonl`
- `.trellis/tasks/09-27-foundation/implement.md`
- `.trellis/tasks/09-27-foundation/prd.md`
- `.trellis/tasks/09-27-foundation/task.json`
- `cmd/persistty/main.go`
- `cmd/persistty/main_test.go`
- `deploy/README.md`
- `deploy/config.example.yaml`
- `deploy/nginx/persistty.conf`
- `deploy/systemd/persistty.service`
- `go.mod`
- `go.sum`
- `internal/auth/auth_test.go`
- `internal/auth/password.go`
- `internal/auth/service.go`
- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/httpapi/json.go`
- `internal/httpapi/router.go`
- `internal/httpapi/router_test.go`
- `internal/storage/migrations/0001_metadata.sql`
- `internal/storage/migrations/0002_resource_ids.sql`
- `internal/storage/storage.go`
- `internal/storage/storage_test.go`
- `tests/contracts/foundation.json`
- `web/components.json`
- `web/eslint.config.js`
- `web/go.mod`
- `web/index.html`
- `web/licenses/react-remove-scroll-bar-2.3.8.txt`
- `web/licenses/shadcn-ui.txt`
- `web/package-lock.json`
- `web/package.json`
- `web/public/third-party-licenses.txt`
- `web/scripts/collect-licenses.mjs`
- `web/src/app/App.test.tsx`
- `web/src/app/App.tsx`
- `web/src/app/layout-model.test.ts`
- `web/src/app/layout-model.ts`
- `web/src/app/styles.css`
- `web/src/components/Feedback.tsx`
- `web/src/components/ui/alert.tsx`
- `web/src/components/ui/badge.tsx`
- `web/src/components/ui/button.tsx`
- `web/src/components/ui/dialog.tsx`
- `web/src/components/ui/empty.tsx`
- `web/src/components/ui/field.tsx`
- `web/src/components/ui/input.tsx`
- `web/src/components/ui/label.tsx`
- `web/src/components/ui/select.tsx`
- `web/src/components/ui/separator.tsx`
- `web/src/components/ui/skeleton.tsx`
- `web/src/components/ui/tooltip.tsx`
- `web/src/features/auth/AuthProvider.test.tsx`
- `web/src/features/auth/AuthProvider.tsx`
- `web/src/features/auth/LoginPage.tsx`
- `web/src/features/auth/auth-context.ts`
- `web/src/features/auth/return-path.ts`
- `web/src/features/settings/ThemeSelect.test.tsx`
- `web/src/features/settings/ThemeSelect.tsx`
- `web/src/features/settings/theme.ts`
- `web/src/features/terminal/TerminalsPage.tsx`
- `web/src/features/workspaces/ProjectsPage.tsx`
- `web/src/lib/api/client.test.ts`
- `web/src/lib/api/client.ts`
- `web/src/lib/api/decoder.test.ts`
- `web/src/lib/api/decoder.ts`
- `web/src/lib/api/use-resource.ts`
- `web/src/lib/utils.ts`
- `web/src/main.tsx`
- `web/src/test/setup.ts`
- `web/src/vite-env.d.ts`
- `web/tsconfig.json`
- `web/vite.config.ts`
- `.trellis/tasks/09-27-foundation/commit-plan.md`

## 未识别文件
无。上述路径均为本次 W01 会话及其实施/检查代理的修改，未纳入用户既有提交、忽略的本地配置/数据库、构建物或 node_modules。确认时如工作树出现新修改，重新核对，不自动扩大提交范围。

## 验证与剩余范围
Go test/race/vet 和前端 lint/typecheck/test/build 通过，前端 29 项测试；本地浏览器真实认证、404、桌面/手机尺寸与主题验证见 browser-report.md。任务上下文及 Markdown 本地链接检查通过。真实 Debian/Nginx/systemd/WireGuard、PTY/文件/提权等未验收，不以本批提交代表完整首版完成。

回复“ok”/“行”批准本批提交，或“我自己来”/“manual”取消代提交。
