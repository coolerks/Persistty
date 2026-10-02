# UI 调整检查报告（第一轮历史）

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


## 第二轮最终范围与检查（2026-10-02）

第一轮已由用户提交 `6881b6c`，以上结果保留为历史。当前追加范围以 PRD 第二轮与[名称搜索契约](../../spec/backend/file-name-search.md)为准，包含项目页和后端 API；不再沿用“未修改 API/Go”的第一轮说明。

- 面板高亮铺满 4px 间隙、活动标签贴顶；三个侧栏入口统一 toggle，实际 onResize 同步 pressed，终端入口置底。
- Git 标题移除常驻底色，三类 Tabs 使用同一 shadcn 原语；目录使用完整路径 aria-label。实际侧栏收起会同步 GitPanel.visible 并执行原请求清理。
- 项目页统一 chrome/微框与紧凑条目，打开按钮 flex 分配剩余宽度，编辑/移除保留在框内；所有确认与真实资源边界保持。
- 项目名菜单提供设置/切换项目，主题图标复用 Select；名称搜索组合 Dialog/Input/Button，防抖/取消/版本与 folder 校验，结果仍由原编辑器打开。
- 搜索安全发现抽取为 shared discover：普通正文不读，ignore 控制安全复制至私有树，名称 DTO 有界；内容搜索原逻辑复用同一 owner，没有扩大 CLI 对源路径的权限。

| 检查 | 第二轮实际结果 |
| --- | --- |
| frontend lint/typecheck/test/build | 通过；31 文件 162 Vitest 测试；最终生产 build 含字体/许可验证，保留既有 canvas/chunk 提示 |
| go test ./... / go vet ./... | 通过；搜索、HTTP 新测实际执行，其余包含缓存；初次 sandbox 下 httptest 本机监听受限，使用经审核执行环境完成测试 |
| go test -race ./... | 通过；HTTP 含实际取消/生命周期测试，其余包含缓存；不表示远端 Debian 验收 |
| Chromium 相关完整回归 | 14 项通过，含真实隔离 Git 历史、merge、文件统计/卡片与源 hash 不变，以及编辑器恢复/草稿/终端几何 |
| WebKit 相关完整回归 | 14 项通过，包含同一真实 Git fixture 和触屏横竖屏 |
| 最终标题/可见性/项目页专项 | Chromium / WebKit 各 4 项通过 |
| DTO 与协议 | 共享 file_names fixture 同时通过 Go roundtrip 和 TS 严格解码；根外/额外字段/超限拒绝；HTTP 未认证/版本/非法 query/日志不泄露通过 |

浏览器专项修正：旧工作台测试点击 h1 清除 hover，如今该处是项目菜单，改为点击状态文本；真实 fixture 有两个 src 目录，以完整路径 aria-label 消除歧义。初次两个 Playwright 进程共享默认报告目录造成 trace 冲突，后续全部串行并使用独立输出目录。新名称搜索测试初始错误 CSS 定位器已按实际 editor-breadcrumb 修正；这些失败未记为通过，记录上表的是修正后完整成功运行。

视觉发现已修正：Dialog 的 translate 与 transform 叠加导致偏左，改为单一 translate 并加中心几何断言；项目打开按钮旧 width:100% 挤出右侧操作，改为 flex:1/min-width:0/width:auto 并补 1440/390 深浅主题的边界断言。原 model/runtime owner 不变，模拟触屏仍使用 textarea。

实际内置浏览器检查：真实名称查询 Terminal→原 Monaco 打开；编辑标签切换、Git 列表/树、主题 light/dark、项目菜单→编辑 Dialog→取消、项目页操作可达，真实终端已连接。截图位于忽略的 `.cache/modern-ui/round2/`；测试报告位于 `web/test-results/round2-*`。

未执行：Firefox、物理手机软键盘、Debian/systemd/部署。本次没有提权或系统配置改动，不改变 W05/W06/W07 未验收项；任务保持 in_progress，未提交/归档。


最终结算：每个浏览器 15 个不同专项用例通过（完整回归14项 + 新项目页1项），最终重复4项覆盖最后修改的可见性、标题菜单、Dialog位置及项目动作边界。横竖分隔线完整宽高另补断言并实际运行；lint/typecheck/build在项目页最终样式后通过。所有日志/截图/fixture保持忽略，无额外未识别工作树修改。


## 第三轮范围与检查（2026-10-02）

第二轮已由用户提交 `e7b44a1`；上方保留历史证据。本轮按新三张标注截图，细化 Git Tabs/刷新图标、2px 圆头、搜索关闭对齐，并重做欢迎页。文件名搜索加入 rg 不可用/能力失败时的同树 Go glob/ignore 兜底；`.gitignore` 在非 Git 目录同样生效。普通文件正文、根外链接和身份/版本安全边界不变。共享 discovery 同时修复控制文件超限不能误判扫描截断的问题。

| 检查 | 第三轮实际结果 |
| --- | --- |
| frontend lint/typecheck/test/build | 通过；31 文件/162 单测；build 含字体/许可证验证；既有 canvas/chunk 提示保留 |
| go test ./... / go vet ./... | 通过；新增独立无工具 fixture、工具能力失败、rg 对照、ignore 优先/嵌套/反选/字符类/转义、普通非 Git 目录、链接/截断/控制文件超限、畸形模式回归 |
| Chromium | 5 项通过：工作台3 + 欢迎页1 + 原工作台交互1；报告 round3-chromium-final |
| WebKit | 相同5项通过；报告 round3-webkit-warm |
| 真实内置浏览器 | 欢迎页→项目；AG查询→原 Monaco；Git 图标/标签与关闭对齐；控制台无 error/warn |

失败处理：移除独立终端路由后，App 单测仍期待旧终端页面，更新为旧地址返回欢迎页/零终端读取，并保留 401 登录检查。Go 新超限回归发现共享 discovery 将 ignore 读取超限误判 truncated，已修正并通过搜索测试。首次 WebKit 与 build/Vitest/race 同时运行，编辑器加载5秒与全测试30秒超时；停止重负载后，保留原界面断言、整项允许60秒串行重跑5项成功。首次完整 race 有3项既有认证测试 login timeout（无 data race 报告），串行重跑结果待下节结算，不将失败记为通过。

范围限制：本机模拟不表示 Firefox、真实手机软键盘、Debian/systemd 或部署验收；没有修改提权执行、安全配置或其他任务状态。截图/日志/fixture均忽略，任务保持 in_progress，未自动提交/归档。


完整 `go test -race -p 1 ./...` 串行重跑通过（httpapi 72.924s，其余含缓存），不再出现认证 timeout。额外 rg 对照识别了 POSIX 形式字符类与 globset 的差异，修正 native matcher 中嵌套 `[` 的语义后普通搜索专项通过；最终 race 结算见末尾。首次并发失败不计为通过。


最终结算：Names 最终源码通过全量普通 test/vet、`go test -race -p 1 ./...`（round3-go-race-final，HTTP 64.455s，搜索3.515s，其余含缓存）；最后只补精确名称路由预算及其无状态测试，普通全量 test/vet 与路由专项 race 重新验证。前端31/162、Chromium/WebKit各5项均通过。Markdown本地链接与diff清洁检查通过。第三轮改动25个文件，暂无未识别修改，不提交、不部署、不归档。


## 第四轮截图调整（2026-10-02）

第三轮已由用户提交 `978097d`。本轮只处理最新五张标注图：折叠终端分隔占位、状态栏居中、指定提示/ID移除、Git数量及加载布局、紧凑搜索面板和全文缺工具兜底。

| 检查 | 第四轮实际结果 |
| --- | --- |
| frontend lint/typecheck/test/build | 通过；31 文件/162 Vitest 测试；build 含字体/许可验证，既有 canvas/chunk 提示保留。新增 E2E 后再次 lint/typecheck 通过。 |
| go test ./... / go vet ./... | 通过；缺 rg 的安全发现/glob/UTF-16/BOM/混合换行/捕获替换/应用、二进制与外链、取消/结果/字节限额、新 rg 能力失败专项实际执行。 |
| go test -race -p 1 ./... | 通过，HTTP 82.056s、search4.044s，其余包含缓存，日志 round4-go-race。随后只优化native坐标增量计算与长行preview分配、补能力失败/全词预算测试；最终search普通/race与全量vet再次通过，日志 round4-search-race-final。 |
| Chromium | 7项通过：工作台4（新增慢加载/数量/clean留空）+欢迎页1+原交互1+真实无rg搜索1，round4-chromium-final。 |
| WebKit | 相同7项通过，round4-webkit；包含390×844与844×390触屏/浅深色、原标签与树交互。 |
| 真实内置浏览器 | 故意缺rg隔离后端的全文结果/原Monaco、HEAD无ID、搜索紧凑选项、Git真实数量与branch、浅深主题；控制台error/warn为空。静态截图 .cache/modern-ui/round4。 |

几何证据：底部Panel实际0时editor/侧栏底边差≤1px；状态文字Range与footer中心差≤2px；恢复后separator恢复4px与2px圆头；慢status响应前后基线Tabs y差≤1px。truncated=true名称响应保留结果但不出现“结果已截断”提示。Git Badge 2→暂存1→clean0，用实际响应和当前基线统计，不隐藏错误。

失败与修正：第一次前端类型检查发现exactOptionalPropertyTypes的count属性及FileTypeIcon props写错，已修正后通过全部门禁。初次 Chromium 在刚expand后立即读伪元素，读取到尚未提交的collapsed样式auto；增加等待实际data-collapsed=false，产品收展实际正常，保留全部4px几何断言，最终7项全部通过。新增rg对照中a*遇到旧rg parser在Unicode空匹配处能力错误；该场景通过搜索逐文件native fallback处理并固定File.Native（不进DTO），不将错误记为通过；常见literal/全词/^/$/捕获表达式继续真实对照。

规范一致性检查发现第三轮提交中的file-name-search.md被误写为HTTP测试源码；恢复e7b44a1的规范正文，并同步第三轮工具兜底/HTTP预算与第四轮提示规则。明确保留HTTP测试原文件，没有删除测试。源码与规范均使用中文说明，机器字段/协议不变。Go RE2兜底支持范围见search-git-contract，不声称覆盖rg全部regex扩展；原rg快照在预览工具丢失时要求重新搜索，不隐式改变捕获语义。

未执行：Firefox、物理手机软键盘、Debian/systemd/部署；本机浏览器和模拟触屏不替代这些验收。W05/W06/W07状态不改，任务保持in_progress；不自动提交/归档。临时fixture/服务仅归本轮，完成后清理自己的资源，旧用户预览保留。

最终补充：内置浏览器截图采集时观察到表单临时重置，独立Chromium/WebKit以clock推进16秒跨过项目后台刷新后，条件与结果保留专项均通过，未在产品代码加入推测修复。最终浅深截图使用该真实无rg专项的稳定产物（round4-search-retention-*），复制到.cache/modern-ui/round4，保留完整页面无编辑。最终search race 3.856s通过（含真实rg Unicode空匹配兜底/预览零写入）；新增E2E后的lint通过。8个修改文档本地链接及task JSON、git diff --check通过；最终30个修改/新增文件均已识别，未提交/部署/归档。
