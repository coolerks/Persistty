# W05 实施清单

用户已于 2026-09-30 明确授权创建并实施此前展示的 W05 范围；PRD/design/本文齐全后激活，主会话实施/检查，不分派子代理。

- [x] M01 保存 API/decoder、共享缓冲区、版本/generation 调度、原始文本换行、IndexedDB 配额/修订/清理。
- [x] M02 desktop/mobile 可编辑、保存状态/快捷键、草稿/冲突/只读 diff/导出，恢复零隐式 PUT。
- [x] M03 文件事件、rename/move/delete 与项目配置保护、重叠关联复验、失效项目草稿导出。
- [x] M04 inspect/preview、SVG/大小/像素、安全鉴权及共享 fixture。
- [x] M05 左右多组/标签排序、终端位置/组、Explorer/设备/尺寸/折叠恢复，零终端生命周期副作用。
- [x] M06 实际 Nerd Font NL、family/hash/license/self-hosted 与尺寸复测。
- [x] M07 trellis-check、owner spec、Go test/vet/race、npm lint/typecheck/test/build、真实浏览器专项、文档链接/忽略/diff check。

## 变更边界

workspaces 的 ProjectWorkbench/DesktopEditor/view/model/Explorer/ExplorerActions/ProjectEditor/ProjectsPage 分别承担视图、buffer、文件和配置协调；新 buffer/draft/diff/preview 模块按职责建立。API client/decoder 与 backend files/httpapi 扩展有界预览并复用原保存。TerminalWorkspace 只扩展本地分组/位置恢复，原 runtime 连接 owner 保持。字体静态资源/许可为本包交付。无 SQL/认证协议/任意进程接口修改。

## 规范与验证

读取 frontend 目录/组件/状态/类型/质量/clients/hooks/lifecycle/theme-assets，backend 目录/错误/质量/filesystem/workspace-files/http-api/security-config，guides 跨层/复用。运行 gofmt 修改 Go、go test ./...、go vet ./...、go test -race ./...；锁定 npm 执行 lint/typecheck/test/build。Playwright 自有合成 fixture/隔离服务，回归 editor-assets/workbench/terminal-geometry 和新增 W05；依真实 Debian 安全文件行为需求复用已批准私有 probe，不改用户服务。真机手机未运行如实保留。

不自动 commit/push/archive；若出现产品范围实质差异，明确差异，不自行删减验收。


## 实施记录

M01～M06 完成代码与对应自动化；M07 最终 Chromium 9 项、前端 128 项单测、Go test/vet/race 与生产构建全部通过，文档链接/忽略/分发/diff 检查通过。新发现的迟到读取倒退强版本、diff worker 清理顺序、字体缓存尺寸和干净标签关闭缓存问题均已原地修复，并加入测试/owner 契约。见 [检查报告](check-report.md)。真机软键盘与完整图片/平台矩阵仍为最终验收事项；任务不自动完成或归档。

## 截图反馈实施批次

- [x] U01 标签脏圆点、隐藏常驻保存状态、文件操作固定在标签栏右侧。
- [x] U02 活动文件语言选择移到单一底部状态栏；自动检测仅显示实际语言，手动选择保持原 model。
- [x] U03 空历史保留 live 画面及明确空态；连续惯性/底部返回/短历史和双角色回归。
- [x] U04 lint/typecheck/test/build、Go test/vet/race、文档/diff 及专项浏览器验收；真实 Debian 隔离专项。

变更边界：FileEditor/ProjectWorkbench 负责编辑 UI 与活动组，LanguageSelect/新增 EditorFileActions/EditorLanguageStatus 复用现有 shadcn 原语；styles.css 只调整对应区域。TerminalSession 只处理历史读取与展示边界，不改变 WS/终止/接管协议或真实进程生命周期。真实资源验证继续使用专属 harness，不操作截图中的用户项目或终端。

截图批次 U01～U04 完成；最终 128 项单测、18 项 Chromium 与 2 项真实 Debian 专项通过，隔离资源全部清理。实施/失败修正/未验收边界见 [截图反馈检查报告](feedback-check-report.md)。本轮修改未提交；W05 仍进行中。
