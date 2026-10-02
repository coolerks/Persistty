# UI 调整检查报告

日期：2026-10-02。实施和本机视觉验收已完成，任务保持 `in_progress`，待提交确认及后续归档。W05/W06/W07 的状态和未验收项保持原记录。

## 结果与范围

实际产品修改为 `web/src/app/styles.css` 和 `web/src/features/workspaces/ProjectWorkbench.tsx`；新增 `web/tests/e2e/workbench-modern-ui.spec.ts`。使用工作台专用浅深色 token、1px/8px 主面板、4px 间隔与条目圆角、柔和标签选中态和侧栏 pressed 状态；局部裁剪收起面板的微边框，底部 xterm 与面板共享背景。既有 shadcn 原语、Material/lucide、Nerd Mono 继续复用。

未修改 API、认证、提权、文件写入、持久化 schema、依赖清单或终端/runtime/model 生命周期。

## 检查记录

| 检查 | 实际结果 |
| --- | --- |
| `npm run lint` | 通过，含新增 Playwright 测试 |
| `npm run typecheck` | 通过 |
| `npm test` | 30 个文件、159 个测试通过；jsdom 的既有 canvas 提示不影响测试结果 |
| `npm run build` | 通过，浏览器专项结束后串行复验；字体/许可/类型检查及 Vite 构建；保留既有大 chunk 提示 |
| `go test ./...` | 通过，含既有缓存结果；本轮未改 Go |
| `go vet ./...` | 通过 |
| Chromium 专项 | workbench-modern-ui / workbench-interactions / editor-recovery / terminal-geometry 共 10 项通过，最终样式复跑 |
| WebKit 专项 | 相同 10 项全部实际通过：9 项首轮通过，长 workbench-interactions 首轮在 30 秒总预算末尾超时，保持原断言以 60 秒总预算独立复跑，16.6 秒通过。初次缺匹配 WebKit 二进制，补齐官方 v2359 后执行，不把缺依赖记为通过 |
| 内置浏览器 | 实际隔离后端、Monaco 与 tmux，文件展开/标签切换、终端输出、三主题与 390px 窄屏检查；最终 error/warn 列表为空 |
| 视觉 QA | [根目录报告](../../../design-qa.md)；归一化全图和标签/文件树/终端聚焦组合比较通过 |
| Markdown 本地链接 | 217 条路径检查无缺失；任务 JSON 解析通过 |
| 忽略规则 / diff | 临时后端、数据库/日志、截图、构建与生成图标位于忽略路径，`git diff --check` 通过 |

## 验收映射

- AC1：内置浏览器 1440×845 浅色/深色截图，以及组合比较证据；微框、间隔、选中态可见。
- AC2：390×844/844×390 的触屏专项深浅主题、document 几何及 textarea 正文通过；内置浏览器真实主题选择也正常。
- AC3：分隔器键盘和鼠标调整、终端收起真实面板 0px、重新展开、标签横向滚动、固定动作、多组恢复、灰点、模型 undo、草稿和 xterm 几何通过。
- AC4：上述门禁已执行；浏览器报告不代替真实系统验收。

## 证据与复现

截图与 traces 按项目忽略规则保存于 `web/test-results/modern-ui-*`；实际浏览器截图为 `modern-ui-visual/desktop-light.png`、`desktop-dark.png`、`mobile-light.png`、`mobile-dark.png`，组合证据为 `comparison-full.png` 和 `comparison-details.png`。

本机隔离预览为 `http://127.0.0.1:5178/`，后端 8098，专属 `/private/tmp/persistty-modern-ui-*` 的测试项目/数据库/tmux socket；仅预览 fixture 使用临时密码，未改用户开发配置或真实项目文件。预览保持运行供本轮查看。

针对现有 fixture 的浏览器复现：在 web/ 明确提供隔离 `PERSISTTY_E2E_BASE_URL`、fixture 密码与项目名，用 `npm run test:e2e -- tests/e2e/workbench-modern-ui.spec.ts tests/e2e/workbench-interactions.spec.ts tests/e2e/editor-recovery.spec.ts tests/e2e/terminal-geometry.spec.ts --output=<独立忽略目录>`；WebKit 加 `--browser=webkit`。这四个专项的 HTTP/WS 由测试隔离，不操作用户资源。

## 未执行与限制

本轮未执行 Firefox、真实手机软键盘、Debian/systemd 或正式部署。没有新增后端并发行为，不额外执行 Go race。任务暂不提交/归档，等待工作流提交方案确认；不将其他活动任务记为完成。
