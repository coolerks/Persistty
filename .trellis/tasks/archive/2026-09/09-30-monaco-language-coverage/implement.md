# Monaco 实施清单

> 2026-09-30：用户已明确要求完成并归档本任务；归档范围与保留的验收限制见 [验收报告](check-report.md)。下文实施阶段状态保留为历史记录。

## 开始前

1. 用户后续明确批准父任务最新最终规划后，按 inline 路由激活本子任务；使用 `trellis-before-dev` 读取 PRD/design/本文及 frontend 目录、组件、状态、类型、质量、生命周期、主题规范和复用指南。不得分派子代理。
2. 再核对 DesktopEditor/ProjectWorkbench/workspace-view 的真实 model 与标签关闭/移动行为；查阅既有 select 和官方文档。不改变计划的产品范围。

## 实施顺序

1. 建生成 metadata 的脚本及构建/开发入口，生成完整 registry snapshot；添加严格失败及版本校验，保持引擎懒加载。
2. 新增 typed 路径/首行推断 adapter；移除 DesktopEditor 白名单，复用运行时语言 loader/worker。
3. 在现有视图状态加入临时语言覆盖、关闭/移动清理；在桌面内容栏加入既有 shadcn Select 的自动/全语言入口。
4. 验证 model 语言切换不重建，不开放写入或诊断/LSP；控制窄栏与长变体名称。
5. 新增推断行为、视图隔离、选择和切回自动测试；在真实浏览器验证全部语言加载/合法样例 token，并测试主题、左右分组、视图稳定及手机回归。

## 验证与审查

- 在 web/ 使用 npm 执行 lint、typecheck、test、build；记录前后主包/动态语法 chunks 与 worker 本地地址。
- 真实浏览器用例放在现有 `web/tests/e2e/`，实际配置为 `web/playwright.config.ts`（不是尚不存在的根 tests/e2e）。使用显式隔离实例地址、测试密码、项目名，仅创建/清理自身 fixture，不操作用户真实项目。
- 逐项运行 AC-L01..05；浏览器验收使用引擎真实 token/DOM，不以 mocked Monaco 代替着色。
- `trellis-check` 自查；按项目门禁运行 go test ./...、go vet ./... 作回归，若无并发/终端改动不额外做 race。
- 执行 Markdown 本地链接检查、忽略规则与 git diff --check。若依赖/环境缺失，记录明确未验收项。
- 实施完成将完整语言及选择/生命周期契约更新到 frontend owner spec，交父任务做图标集成审查。未经用户要求不提交或归档。

## 回滚点

语言选择前可单独验证推断 adapter；最终回滚涉及 DesktopEditor、ProjectWorkbench、workspace-view、新 adapter/生成数据与 scripts，保留用户无关修改及父任务资料。无服务端迁移。
