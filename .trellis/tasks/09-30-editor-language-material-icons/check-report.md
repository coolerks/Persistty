# 完整代码高亮与 Material 图标验收

日期：2026-09-30。用户批准最终规划并要求开始执行；主会话单代理完成实现及 trellis-check 自审，无 subagent。保留未提交修改，任务保持 in_progress，未提交/归档。

本报告记录高亮/图标阶段的验收状态。用户随后提出的关闭已结束标签、文件树右键/展开定位见[交互补充](interaction-followup.md)；最新终端历史排版/滚轮与 tab 悬停滚动条修复、**108 条单测、3 个浏览器用例**及 Go test/vet/race 结果见[滚动修复补充](scrolling-followup.md)。下文“未改终端实现”等边界仅指高亮/图标阶段。

后续设备属性数字误入 shell 已修复，最终累计 **109 条前端单测、5 个浏览器用例、Go test/vet/race 与真实隔离 tmux/PTY**结果见[设备应答补充](device-attributes-followup.md)。新增 v3 固定枚举应答帧，需前后端共同更新；不操作用户既有进程或自动提交。

触控板连续位移/横向白屏及黑线/句点的最新修复和 **110 条前端单测、5 个浏览器用例、Go test/vet/race、真实 tmux/PTY 网格与录制回放**见[触控板与字符网格补充](trackpad-followup.md)。后续验收以该补充与已更新 owner 规范为准；历史阶段的 scrollLines/observer 本地 fit 方案已替换，不属于最终实现。

## 结果与边界

- [语言子任务](../09-30-monaco-language-coverage/prd.md)：从固定 Monaco 0.57.0 的真实注册 AST 生成完整 91 模式，替换后缀白名单。桌面支持自动与全模式选择，包含六个 FreeMarker 变体和三种无后缀 SQL 方言；选择仅暂存视图，按项目/文件夹/路径隔离，同文件分组共享，最后视图关闭/删除清理、重命名迁移，刷新恢复自动。
- [图标子任务](../09-30-material-file-icons/prd.md)：material-icon-theme 5.38.1 默认完整 manifest 与 1251 SVG，自托管。文件树/根/标签/目录选择器复用 resolver，exact basename、最长复合后缀、扩展、语言、fallback，大小写归一；上游 patterns/clones/目录开闭/light 别名保留。手动语言选择不改变路径图标。
- model 保留和 scope/最后标签释放单独管理；选择语言/主题不重建 URI/model，诊断和 LSP 禁用。当前仍为只读快照，手机基础文本视图。未改 API、数据库、保存/草稿、终端行为或字体。

## 验证矩阵

| 检查 | 实际结果 |
| --- | --- |
| npm lint / typecheck / test / build | 通过；18 个单测文件、100 条测试，真实构建成功 |
| Go go test ./... / go vet ./... | 通过（部分测试缓存，未改 Go 源码）；未修改并发/终端 bridge，无新增 race 要求 |
| 完整语言真实 Chromium | 91 ID 全部已注册；90 个非 plaintext 模式逐项用样例创建真实 model、等待并断言语法 token；plaintext 保留基础文本 |
| 引擎视图真实 Chromium | SQL/FreeMarker 切换、自动恢复、标签往返、主题后 model ID/正文/光标/滚动保持；本地图片可解码，无外部请求/pageerror |
| 真实工作台 dev / production | 同一专项在 Vite dev 与实际 dist preview 均通过；树/标签相同、yaml/yml、d.ts/ts、package.json、go.mod、Dockerfile/.env、目录展开、双组模式同步、light/dark/system 媒体变化正确；本地图片完整解码，无正文写请求/新终端 WS/外部请求/pageerror |
| 四种视口 | 1440×656、1024×540、390×844、844×390：document 无额外宽高溢出；手机仍基础 textarea，无语言选择入口 |
| 全量静态资源 | 1251 SVG 共 1,031,090 bytes；源与 dist 每项 SHA-256一致，assets.json 完整；完整 MIT 原文同时在图标目录及 third-party-licenses.txt |
| 生成稳定性 | 重生成字节和 mtime 不变；两次并发生成期间 106 轮读取 JSON/SVG，零缺失/半写输出 |
| 文档/忽略/diff | 本轮规划与 frontend 规范本地链接、围栏、忽略规则、git diff --check 通过；生成目录忽略、manifest/锁文件/源信息/LICENSE/脚本受版本管理 |

真实浏览器为三个独立用例：两条 `web/tests/e2e/editor-assets.spec.ts` 引擎 fixture 用例，一条 `editor-workbench.spec.ts` 真实工作台用例；工作台另在生产 preview 复验，不把重复运行称为新场景。

## 重现入口

```sh
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run test
npm --prefix web run build
go test ./...
go vet ./...
```

真实浏览器用当前 `web/playwright.config.ts`，需显式 `PERSISTTY_E2E_BASE_URL`、`PERSISTTY_E2E_PASSWORD`、`PERSISTTY_E2E_PROJECT`。开发实例上执行：

```sh
npm --prefix web run test:e2e -- tests/e2e/editor-assets.spec.ts
```

完整工作台用例额外提供 `PERSISTTY_E2E_EDITOR_ROOT`，必须为合成 fixture 的规范绝对目录，包含 config.yaml/yml、query.sql、view.ftl、config.toml、a.d.ts/a.ts、package.json、go.mod、Dockerfile、.env、src/node_modules/.git；不指向用户项目。使用隔离 API/database，不禁认证限流，测试仅创建自有项目元数据，不写正文。开发或生产 preview 上执行：

```sh
npm --prefix web run test:e2e -- tests/e2e/editor-workbench.spec.ts
```

未提供额外根目录时该工作台用例显式跳过，不能计为验收；本轮提供专属本机临时根、合成密码/数据库，全部运行，无跳过。原始 Playwright screenshots/trace 在忽略的 web/test-results，不包含真实用户文件正文。

## 自审与修复

- 仅增加 workspaces owner 的语言/图标/模型管理与资源生成；复用已有 shadcn Select、Button/Tabs 和官方 Base UI 文档，无新交互原语、无类型绕过或调试全局进入产品。window.editorAssets 仅在 tests fixture，生产应用不打包该 fixture。
- 推断、临时状态、资源解析的消费者已搜索并统一；纯图标/语言 metadata 不导入 Monaco。树中所有节点共用有效主题 observer，保持 runtime 控制与文件安全边界。
- 实测发现重复生成先清空目录会有资源暂时缺失，改为字节比较+原子替换，最后才清理过期 SVG；并发读取复验。无需改需求或扩大模块边界。
- 初轮测试查询的 exact 参数类型问题、HTML 诊断选项 API 差异、未知 plaintext 的上游 document 图标预期已修正，最终检查通过。
- 规范已同步 frontend editor-terminal-lifecycle、theme-assets、state-management 与索引。许可 generator 显式包含 devDependency 分发资产，不漏入运行依赖扫描。

## 体积与验证限制

主 index bundle 从 1001.76 kB / gzip 299.73 kB 增至 1468.85 kB / gzip 364.57 kB，来自完整内置映射与轻量语言元数据；DesktopEditor 从 1312.18/gzip331.84 kB 到 1311.91/gzip331.63 kB，editor.api 2737.15/gzip705.25 kB 不变，语法仍动态 chunks。既有大于 500 kB 构建提示保留，不调整阈值掩盖。

jsdom canvas getContext 未实现提示仍存在，真实着色由 Chromium 验证；未因此安装无关 canvas 依赖。手机验证是 Chromium 视口模拟，未声称真机或所有浏览器引擎验收。本轮专属本机服务没有 tmux，终端面板正常显示 unavailable；未使用用户终端、未重验 Debian/systemd 持久性，不把零新 WS 等同于活跃终端完整运行时验收。W03 历史验收保持有效，本次未改终端实现。

## 清理与当前状态

本轮专属 5175/18981 测试端口已关闭，随机 fixture 根目录、临时数据库/配置/合成文件及 tmp/editor-assets-setup 已精确清理；用户的服务/文件/终端未处理。保留代码、规划和报告供审阅。未经用户要求不提交、不归档。

最新末行遮挡与黑色底边修正及本轮验证见[终端几何补充](terminal-geometry-followup.md)，其几何/背景结论补充此前 tmux 字符网格录制的覆盖边界。
