# W06 功能交付与开发检查

## 状态与授权

2026-10-01 用户回复“开始实施”后，主会话以单代理 inline 完成 W06。功能已接入，必要本地开发检查通过；任务保持 `in_progress`、`acceptance=deferred`，等待用户安排统一实机验收。W05 的延期安排保留。本轮没有提交、推送、归档、连接 Debian 或修改正式运行数据。

## 已交付功能

- 多根搜索：项目/文件夹/目录范围、字面量/大小写/全词/单行正则、include/exclude、忽略规则、重叠真实文件去重、UTF-16 搜索结果定位；提供截断/跳过/失败提示和取消。
- 安全替换：选择匹配、按文件确认、桌面只读差异/手机前后文本、同引擎捕获组和原字节拼接；复用配置裁决与原子保存，逐文件报告成功/冲突/失败/跳过；取消剩余项、状态查询和首次尝试幂等，保护目标缓冲区及处理期间的新输入。
- 只读 Git：多仓库/嵌套仓库发现，HEAD 总变更及暂存/未暂存状态，本地分支/标签、固定 HEAD 分页历史、根/merge 提交详情、磁盘或明确选择的编辑器快照比较；改名比较读取原路径。
- 编辑器 HEAD 行标记：按文件实际所属最内层仓库取得 baseline，标记输入相对 HEAD 的新增/修改/删除；保留原 model、undo、自动保存和草稿机制。
- 工作台入口：桌面 Search/Git 活动栏、目录菜单、查找/替换/Git 快捷键；手机单内容导航和有界只读文本；原 schema v2 兼容恢复。

安全输入采用注册目录句柄与私有 staging。Git/rg 固定参数、最小环境、容量与超时/进程组取消；不在源目录运行 CLI，不读取链接目标，不执行仓库 helper/filter，不联网补对象。Darwin 目录元数据改为 `fstatat`；Linux 快照叶文件使用受限 `openat2`。无数据库迁移或新增产品依赖，官方 shadcn Checkbox 使用现有依赖。

实际接口、上限、错误及生命周期以[后端 owner](../../spec/backend/search-git-contract.md)和[前端 owner](../../spec/frontend/search-git-workbench.md)为准；设计稿第9节记录实施收敛。

## 实际开发检查

| 检查 | 结果与范围 |
| --- | --- |
| `go test ./...` | 通过，包括真实本机 Git/rg 临时目录测试及 HTTP 会话/CSRF 链路 |
| `go test -race ./...` | 通过，覆盖快照、缓存、替换、runner 和现有并发组件 |
| `go vet ./...` | 通过 |
| `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/persistty` | 通过；只证明 Linux 编译，不等于 Debian 运行验收 |
| 前端 `lint` / `typecheck` | 通过 |
| `test -- --maxWorkers=2` | 23 个测试文件、134 项测试通过 |
| `build` | 通过；字体校验和37项许可证保留通过，已有大 chunk 提示仍可见 |
| Chromium 开发回归 | 10项通过：W06真实链路2项、editor-recovery 6项、workbench-interactions 1项、terminal-geometry 1项；取消保护收尾修改另复跑W06两项 |
| agent-browser | 本地登录页快照与截图检查通过，未出现 Vite 错误覆盖层，检查后关闭浏览器 |
| 文档与工作树 | Markdown 本地链接、任务/fixture JSON、忽略规则、Go格式及 `git diff --check` 检查通过 |

新增 Go 测试覆盖安全句柄/父目录替换/特殊文件、CLI 环境/容量/取消、ignore/include/Unicode/混合换行/零宽/截断/重叠根、版本与配置冲突、首次 apply 重放、释放额度、Git 状态/根及 merge/删除/缺对象/恶意配置/alternates/filter/重叠归属、改名和固定 HEAD 分页。共享 JSON fixture 在 Go 和严格 TypeScript decoder 中验证。缓冲区测试覆盖目标保护、暂停/版本不符、新输入保留与干净缓冲区刷新；行标记测试覆盖插入/修改/删除和有界回退。

浏览器真实链路在自有临时项目中验证：搜索忽略规则与 emoji 列定位、预览差异、外部修改后冲突或前端跳过且不覆盖、重新预览成功、状态查询、HEAD 行标记、只读 Git/编辑器快照、历史及手机 viewport；确认源 HEAD/index hash 不变、无 pageerror。第二项在 apply 传输等待期间明确取消，确认零写入。现有终端几何和工作台回归使用原有 mocks；测试 fixture 不提供真实 tmux，终端入口的不可用响应不代表终端验收。

## 本地复现

测试工具、缓存及 fixture 均在本机隔离目录中。先运行 `go run ./scripts/w06-local-fixture`，它创建自有临时 repo/数据库并打印项目ID与根路径；测试密码是代码内的虚构值 `local-w06-fixture`。服务监听 `127.0.0.1:8089`，Origin 为 `http://127.0.0.1:5179`。

在 `web/` 运行 `PERSISTTY_DEV_API_TARGET=http://127.0.0.1:8089 npm run dev -- --port 5179 --strictPort`，设置以下测试变量后运行 Playwright：

```sh
export PERSISTTY_E2E_BASE_URL=http://127.0.0.1:5179
export PERSISTTY_E2E_PASSWORD=local-w06-fixture
export PERSISTTY_E2E_PROJECT='W06 本地开发检查'
export PERSISTTY_E2E_W06_PROJECT='<fixture 打印的项目ID>'
export PERSISTTY_E2E_W06_ROOT='<fixture 打印的 repo 根路径>'
npx playwright test tests/e2e/w06-local.spec.ts tests/e2e/editor-recovery.spec.ts tests/e2e/workbench-interactions.spec.ts tests/e2e/terminal-geometry.spec.ts
```

W06测试只允许显式指定 `/private/tmp/persistty-w06-browser-*` fixture；未指定会跳过，不能记为通过。构建会重生成静态资源，应在浏览器回归前完成。关闭自有 fixture 进程时清理该临时树；不关闭其他服务或用户进程。

## 统一验收待办与已知限制

- W05：真实手机软键盘/输入法/旋转/前后台、触控板及完整图片/字体平台矩阵，仍延期。
- W06：真实 Debian Git/rg 与文件安全边界、真实浏览器和手机交互/性能、三主题及完整设备矩阵，等待用户明确安排。
- W08：正式部署、systemd/Nginx、安全入口和完整首版矩阵，保持原工作包边界。

本轮 Chromium 手机 viewport 不等于真实手机；Linux 编译和仓库中名含 debian 的本机测试不等于 Debian/systemd 实测。测试框架仍有 jsdom canvas 与 NO_COLOR/FORCE_COLOR 提示，生产构建有大 chunk 提示，未通过抑制警告掩盖。

搜索只支持单行 rg 默认引擎，快照/历史/文本大小有界。Git worktree `.git` 文件、根外元数据、alternates/promisor、外部 filter 或未知扩展先明确不可用；大仓库成本需后续实机性能验收。连续 Git API 各有独立快照，不承诺全项目原子截点。取消不回滚已发布文件；其他浏览器未保存输入、非协作 writer 的最后检查/rename 短窗口继续遵守原 owner 边界。未加入 Git 写操作、提权、LSP/AI 或正式部署。
