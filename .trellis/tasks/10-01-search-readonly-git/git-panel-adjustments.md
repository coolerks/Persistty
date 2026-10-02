# Git 超时与面板调整（2026-10-02）

## 需求与边界

用户反馈 status 仍超时，并要求 VSCode 风格 Git 面板。当前 HTTP 在 10 秒截断，而工具默认预算为 15 秒；status 的私有快照复制被忽略工作文件。面板当前只有互斥 Tabs，历史详情脱离对应提交。

实际 owner：`internal/gitview/service.go` 与新增工作树快照模块负责安全读取与 ignore 裁决，`internal/toolrunner` 提供固定 check-ignore 命令；`internal/httpapi/router.go` 对已注册工具路由采用配置预算。Git 不直接读取原始目录、不遗漏已跟踪的忽略文件、不放宽安全或容量边界。

前端 owner 为 `web/src/features/git`：双区域默认 50/50，可拖动、分别折叠；真实父提交图；点击提交在其下展开文件；两种文件展示及现有 Material 图标。复用 shadcn Collapsible/ToggleGroup、已有 react-resizable-panels。保留只读比较、父选择、本地引用、冻结 HEAD 分页、缓冲区选择及请求取消。

不改 Git 写入/联网、编辑器保存、终端或部署；不引入持久正文缓存。局部拆分用既有比较及无轮询浏览器测试证明行为保持，新增合并/分页图、忽略目录中 tracked 文件与超过普通 HTTP 预算的工具路由测试。实际检查结果将在本文件补齐，真实 Debian/手机待办继续保留。

## 根因、实现与兼容性

1. HTTP middleware 固定 10 秒与工具默认 15 秒不一致；服务 `with` 内创建的短 context 也没有交给操作闭包。现根据真实注册 route 为工具请求采用配置预算，并把同一 context 传入操作及复验。普通文件请求预算不变；超过配置限制仍失败，不靠无期限延长掩盖开销。
2. 工作树复制发生在 Git ignore 裁决之前。现在先复制元数据和控制文件，按深度批量 check-ignore，剪枝 ignored/untracked 大目录；index 的 tracked 集与祖先强制保留。正文继续用既有批量安全读句柄复制，不改源目录 CLI 边界、link 文本、filter 拒绝或预算。
3. 历史 `--topo-order` 加活动父 lane 构造真实分叉/汇合图。两个区域默认平分且各自滚动、可拖动和收展；列表/树独立选择，提交文件展开在相应提交下面，merge 仍可切比较父。Material 文件/目录图标沿用原映射。本地引用移到标题按钮的独立只读 Dialog；比较全屏、行内/并排和 buffer 选项保留。
4. 初始历史→状态串行读取；状态取消/错误独立，历史动作取消慢状态且仍可用，取消后允许明确刷新变更。没有 fixed timer/retry，旧 effect continuation 与迟到结果不能写新项目。
5. shadcn 查找：本地 17 个现有组件、官方 CLI info/search/docs 与 Base UI Collapsible 的文件树组合；官方 `add collapsible toggle-group resizable` 生成三份 source，复用现有依赖，无 manifest/锁文件变化。提交关系图为业务数据的 SVG，基础交互不自造。

## 实际验证（本机）

| 检查 | 结果 |
| --- | --- |
| Go 全包 test/vet/race | 通过；追加断言后受影响 gitview/httpapi/toolrunner 再验 test/race，完整 vet 再验 |
| 超过 10000 条的忽略目录 | 通过；超字节预算的 ignored 文件不读取，tracked 忽略文件、嵌套否定及 info exclude 语义保留，完整路径/status 与自有源 Git 对照 |
| HTTP deadline | 实际注册 middleware 的 3/15/25 秒配置与普通 10 秒通过；操作闭包收到快照 deadline |
| 原 Git 回归 | unborn、改名、merge/根提交、冻结 HEAD 分页、缺对象、link/属性/元数据边界与源零写入均通过 |
| 前端 lint/typecheck/test/build | 通过；26 文件、142 项单测。构建验证字体与 37 项许可证，保留既有大 chunk 与 canvas 提示 |
| Chromium | 13 个不同用例通过：Git 专项 7 项（含真实 merge），原 W05 编辑恢复 6 项；局部重复运行单独输出，失败的新增测试登录等待已修正并复跑 |
| WebKit | Git 专项 7 项全部通过，含移动比较、合并/父切换、双收展、无轮询、慢状态展开 |
| 原仓库不变 | 真实 Halo 只读检查及合成 merge 检查的 HEAD/index/config hash 一致，比较不写工作树 |
| 文档/JSON/diff | 收尾检查见本轮日志；task.py 的 inline JSONL skipped 不作为产品测试证据 |

性能测量根为用户已提供的 Halo 本机目录；产品 Git 命令只在私有 staging 执行，源目录没有 Git 写操作。一次完整复测：发现 0.172 秒、status 2.432 秒、baseline 0.326 秒、HEAD 比较 0.431 秒、历史 0.471 秒、详情 0.346 秒、提交比较 0.473 秒。此结果是本机 service 复测，不是用户最新 opaque repo ID 的已认证 HTTP 延迟，也不代表 Debian 实测；最新报错请求无法仅凭 ID 推断其目录和大小。

Browser 截图位于被忽略的 `web/test-results/git-panel-chromium/`、`git-merge-chromium/`、`git-panel-webkit/`；临时后端/前端只使用 8089/5179 和自有 `/private/tmp/persistty-w06-browser-*`。本轮不连接 Debian、不提交/推送/归档，W05/W06 保留进行中及原真实设备待办。后端源码修改需重启实际后端进程才生效，刷新 Git 重新发现当前 repo ID。

## 生效与收尾

确认原 `.cache/dev/run-*/persistty` 构建时间早于本轮后端源码，且管理 socket 的 ready URL 为 `http://127.0.0.1:5173`。使用项目 `./scripts/dev.sh restart` 恢复原参数重启已管理的本机实例，实际返回前后端就绪、tmux 任务保留；新运行构建时间覆盖最新后端源码。浏览器刷新后重新发现新服务签发的 repo ID。没有连接或重启正式部署。

21 个本地 Markdown 链接与任务 JSON 检查通过，`git diff --check` 通过。新增 Git 测试二进制 Linux/amd64 交叉编译通过，只是编译证据。专属 8089/5179 测试进程已退出，`/private/tmp/persistty-w06-browser-411756268` 已删除；用户 5173/8080 开发实例继续运行。代码未提交，原实机验收待办不变。

## 追加：历史滚动加载与提交范围（2026-10-02）

用户要求移除“加载更多提交”按钮，改为滚动到底自动加载；随后要求父提交选择移到右键菜单，提交文件向内 3px 并增加左边框。

- `GitHistory` 只订阅历史区域自己的滚动，距底部 24px 内请求下一页；沿用首屏 HEAD/offset，追加已有记录，末页停止。同步请求锁和 pending 避免慢请求期间重复发出；失败不连续重试，滚离底部后重新滚回可重试。卸载释放监听，隐藏区域不读取；加载提示移到列表底部。
- 合并提交复用已有 shadcn ContextMenu/RadioGroup，菜单显示父序号/短 ID 和当前勾选，选择后 `closeOnClick` 关闭菜单。回调显式传 commit/parent，收起状态下也直接加载所选父；作者/时间保留，详情不再常驻选择框。本轮查询 CLI `docs context-menu` 与[官方组合文档](https://ui.shadcn.com/docs/components/base/context-menu)，没有新增依赖或自制基础菜单。
- 列表与文件树共用左边界容器：margin-left 3px、1px 主题边框贯穿本条提交文件范围；上方当前变更区保留原样式。

最终 lint/typecheck/test/build 全部通过，27 文件、146 项单测。新增单测覆盖触底/请求去重/失败重试/隐藏/末页/卸载，以及右键选父和收起时选择默认父。Chromium 和 WebKit 各 3 项真实后端用例通过：双区域/图标/收展，merge 右键父切换→对应文件树→只读比较，105 提交自动分页（50→100→105）。第二页以浏览器请求屏障模拟慢请求，继续调用真实后端，断言只发一次；两次后续请求均带冻结 HEAD，105 条无重复/缺失，末页无新请求。合成 merge 源 HEAD/index/config hash 及分页源 HEAD 均保持不变。

初次 Chromium 使用临时浏览器路径时缺可执行文件，未执行；切换本机已安装 Chromium 后通过。右键单测发现 RadioItem 默认选择后不关闭，已显式使用 closeOnClick 并复验。最终截图在 `web/test-results/git-history-chromium/` 与 `git-history-webkit/`，专属测试根为 `/private/tmp/persistty-w06-browser-895442550`；这些运行产物不入版本管理。本批仅修改前端与记录，不重跑无新增改动的 Go 检查；上节 Go 结果为此前实际运行证据，真实设备验收状态不变。

追加批次收尾：20 个本地 Markdown 链接、任务 JSON 与 `git diff --check` 通过。专属 8089/5179 进程已退出，测试根确认已删除；会话日志 Session 10 已记录，未提交、推送或归档。
