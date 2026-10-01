# W06 技术设计

## 1. 变更边界与责任

当前差距是产品没有搜索/Git 服务和 UI；已有安全文件访问、强版本保存与共享编辑缓冲区。按 [PRD](prd.md) 实现，不重写这些 owner。

| 位置 | 职责与必要变更 |
| --- | --- |
| `internal/files` | 增加有界遍历/安全元数据读取与 snapshot 描述，复用 CopyVerified/ReadContent/安全句柄，不弱化原 API |
| `internal/toolrunner`（按需建立） | Git/rg 共享固定程序、白名单 env、有界进程输出、context/进程回收；只接收服务私有 staging/stdin |
| `internal/search`、`internal/gitview`（建立真实实现后使用） | 分别拥有搜索/预览/逐文件发布及仓库发现/快照/只读 CLI 解析，不依赖 Gin |
| `internal/config` 与示例配置 | 增加工具路径、搜索/仓库扫描与容量/时间/并发限制及严格合法范围 |
| `internal/httpapi`、`tests/contracts/search-git.json` | service 接线、验证输入、保护路由和统一错误；Go/TS 共用 fixture |
| `web/src/features/search`、`web/src/features/git` | 搜索选择/预览/结果，Git 仓库/状态/历史/比较，公共组件与 typed callbacks |
| `ProjectWorkbench`、`Explorer`、`EntryMenus`、`workspace-view` | 编排侧栏、文件夹搜索入口、定位和设备独立选择；不让新 feature import app 或复制文件副作用 |
| `editor-session`、`FileEditor`、`DesktopEditor`、`DesktopDiff` | 目标缓冲区替换保护/重验、只读比较、行定位和 HEAD decoration；不重建原编辑 model/undo |
| API client/decoder、样式和 owner spec | 协议校验、中文状态/语义色及准确交付状态 |

无 DB 正文存储、迁移、认证/终端协议或部署配置变更。若需要拆解 workbench 的局部协调，保留原 props/生命周期并先跑原有回归；不顺便重构其他功能。

## 2. 安全输入与私有 staging

采用 process-guidelines 策略 2。每请求先验证 project_version、folder 关联和注册根身份，再以现有安全 I/O 读取。staging 为服务拥有的独立 0700 目录，文件 0600，不位于项目根/public/download 可访问目录；不含真实路径链接、不使用 hardlink/reflink 到原目录。所有 staging 名称由服务随机生成，路径不来自用户。

共享 runner 接受固定白名单 Git/rg 程序和固定 operation，独立 argv；context 超时、stdout/stderr 字节上限、结果限制，取消 Wait 回收。stderr 不回显正文/原始配置。仅清理自身短命工具与 staging，不触及 tmux。缺工具/不支持必要行为时对应功能 503，既有编辑器/终端仍正常。

配置 `search` 与 `git` 分别锁定 binary/timeout/扫描容量及 snapshot limits；私有 staging 总预算单独计入磁盘/并发额度，文档明确与 transfer staging 的合计最大占用，不能宣称原上传 2 GiB 已包含新缓存。起始预算：同时工具请求 2 个；搜索最多 50000 条目录项、深度 32、2000 个结果文件/5000 个 matches、64 MiB 文本 snapshot；单文本沿用 8 MiB。Git 每次工作树+元数据 snapshot 不超过 512 MiB；总 staging 不超过 1 GiB，CLI 默认 15 秒。preview TTL 5 分钟，每 session 最多 4 个、全局内容缓存最多 128 MiB。数值为可调默认和测试上限，尚非性能验收结果。

安全扫描不 follow 目录 symlink，不打开 FIFO/device/socket，不穿越挂载或注册根变动。扫描变动需重验/报告，不能只从普通绝对路径读取后过滤结果。staging 中 Git 指针/配置及对象路径须全部受控，完整安全测试通过后才开放 API。

## 3. 搜索 snapshot 与语义

1. 在批准 folder 范围安全枚举，生成目录 skeleton/普通文件占位和项目范围内 ignore 文件；包含仅用于 ignore 的 `.git/info/exclude`，不交付 `.git` 内容。范围内嵌套 ignore 保留层级，项目外 parent/global config 不加载。
2. rg 默认 ignore 策略确定候选集合；`.git` 和默认依赖排除为强规则，include 仅与候选交集，exclude 再收窄。`--no-config`、`--hidden`、固定排除，不使用 `--no-ignore/--text/--follow`。强排除不可被用户 glob 顺序覆盖。
3. 对选中候选用安全句柄读取有效 UTF-8 非 NUL 普通文本、完整 Version；按真实 identity 去重。重叠根选稳定有效来源，保留 folder/path；不同根同名不能合并。
4. 将验证文本 snapshot 交 rg JSON 搜索，literal/regex、case_sensitive/whole_word 单行规则明确；不用 PCRE2/多行模式。禁隐式编码转换，用原字节 offsets 和同一版本验证 JSON 结果，中文/emoji 转 1-based UTF-16 column/end_column。UI preview 从 snapshot 截取。
5. response 绑定 session/project_version/query 与有界 search_id，携带 files/matches、完整版本、truncated 和 skipped 原因。无匹配退出 1 为成功空结果；无效 regex 400，其他错误不能伪装为空。

## 4. 替换预览和发布

search_id 不接收客户端 offset 或任意目标路径。preview 读取服务保存的搜索快照和 selected_match_ids，使用同一 rg 参数/engine 的 JSON replacement 得到每个原匹配展开文本；`${1}`/`$1`/`${name}`/`$$` 遵守 rg 规则，literal replacement 的 `$` 自动转义。按原 bytes ranges 一次性拼接，不用 stdout 全文件或 JS/Go 二次匹配。零宽匹配的稳定排序、重叠拒绝、expanded/new-size 上限需专项验证。

preview 保存随机 preview_id、session 绑定、project_version、folder/根身份、每文件 Version、原/新内容 hash、匹配选择与期限；列表仅摘要，按 file_id 请求有界原/新正文比较。修改选择/替换文本后旧预览失效，用户必须看当前预览再应用。

当前视图用 EditorScope 短期保护目标集合：识别 aliases/真实 identity，dirty、saving、conflict、failed、恢复暂停或版本不对齐均跳过并说明；保存调度状态与 generation 原样保留，不用 protect() 暂停全部项目。允许处理干净文件，但期间用户新输入仍保留，返回后不把新 generation 设为 saved。其他 tab 的输入仍靠本地草稿/事件冲突保护，不声称全浏览器协作锁。

apply 只允许本预览选定 file_ids，保护文件列表用于缩小范围；每文件通过 WithRegisteredFolder + files.Save 复验配置/根/完整版本，再原子发布。成功后重验受影响干净 buffer、目录/search/Git 缓存；dirty buffer 保留并显示外部冲突。一个文件只接受一次提交尝试，不盲重试。

预览应用状态记下各文件结果；断连后 GET 能查已知进度，重复 apply 只返回既有尝试结果，不重新写入。取消停止未提交部分，不回滚成功文件。TTL 清理不删除正在发布的结果；Web 重启使 search/preview 410，需重新预览。应用内配置修改与发布逐文件串行，整个批次允许部分成功，保留外部 writer 短窗口说明。

## 5. 仓库发现、metadata snapshot 与只读命令

每 folder 在有界安全扫描中识别普通 `.git` 目录及嵌套仓库，按工作树和元数据真实 identity 去重。最内层有效仓库拥有文件基线，repo_id 为服务签发的 opaque ID，绑定 project/config/scope；不能让客户端决定 gitdir/绝对 cwd。项目只覆盖父仓库内部子目录时不擅自探测父级整树；外部 `.git` 文件/commondir/alternates 未批准时报 repository_unavailable。

安全复制元数据白名单：HEAD、refs/packed-refs、必要 index/sharedindex、shallow、普通本地 objects/pack 与必要 info exclude/attributes；逐条拒绝 symlink/special file/越界对象路径，记录指纹并在交付前重验 HEAD/ref/index/元数据变化。原 config 在 staging 中用固定 `git config --file ... --no-includes` 读取仓库格式和内建必要设置，再生成白名单 config；禁止继承原始 include/core.worktree/core.gitdir/helper/pager/fsmonitor/filter/remote 等配置。支持的 repository 格式通过能力探测和真实 fixture 锁定，未知扩展或必须使用外部过滤器准确报不可用。

工作文件按安全句柄复制原字节/可执行位，Git 不从真实目录遍历。tracked symlink 用安全 readlink 读取链接文本并配合 `core.symlinks=false` 表示为私有普通 snapshot，须验证 status/diff 等价；不能让 staging 链接实际指向根外。Git 内部 submodule/外部对象指针不得触发递归访问；无法准确表示时报不可用，不能形成伪 clean。Git snapshot 达限或缺对象时返回限制/不可用，不把缺文件当真实删除。

子进程使用白名单 env（禁系统/global 配置、GIT_OPTIONAL_LOCKS=0、GIT_NO_LAZY_FETCH=1、禁 prompt/pager，literal pathspec）。命令固定关闭 ext-diff/textconv、hooks/fsmonitor、自动 gc/维护和签名显示；不保留 remote/credential/filter 配置。tool 不接触 source，因此所有意外 index 更新也只发生在自有 snapshot；原 HEAD/index/config/工作文件前后 hash 对照仍为必需测试。

status 用 porcelain v1 -z，区分 index/worktree/untracked、解析 rename 两路径；默认总变更 HEAD → 磁盘。refs 与 log/detail 使用 NUL/明确格式及有界 pagination，验证对象 ID，不透传任意 revision/pathspec option。branch/tag 必须来自本次 refs allowlist 并 peel 为 commit ID；历史属于已验证仓库，工作文件/比较限定关联范围。

root commit 用对应 object format 的空树，merge 默认第一父并标明，可选择已列出的其他父；无 HEAD/缺对象/二进制只展示准确元信息。引用比较为固定 commit → 当前磁盘；打开编辑文件可选择 commit → 当前 buffer 并标注未保存，不切分支，不生成 Git 对象或自动取对象。

## 6. API 形状与错误

均在 `/api/v1` protected 下，POST/DELETE 沿用 Origin/CSRF。以下是本任务待实施契约，当前不存在这些 API。

| 路由 | 输入/返回重点 |
| --- | --- |
| `POST /projects/:id/searches` | project_version、可选 folder_id/path、pattern/regex/case_sensitive/whole_word、include/exclude → search_id、files/matches、truncated/skipped/expires_at |
| `DELETE /projects/:id/searches/:searchId` | session/project 绑定；取消及释放快照 |
| `POST /projects/:id/replace-previews` | project_version、search_id、selected_match_ids、replacement → preview_id、files/hash/count、expires_at |
| `GET /projects/:id/replace-previews/:previewId` | 摘要与应用结果；可带 file_id 获取受限原/新内容 |
| `POST /projects/:id/replace-previews/:previewId/apply` | project_version、selected_file_ids、protected_file_ids → 逐文件 applied/conflict/error/skipped 与新 Version |
| `DELETE /projects/:id/replace-previews/:previewId` | 明确取消；保留已提交结果，不反向写文件 |
| `GET /projects/:id/repositories` | project_version → 有效 repo 列表、scope、head、unavailable/truncated |
| `GET /projects/:id/repositories/:repoId/{status,refs,log}` | project_version、snapshot_id/分页 → 状态/本地 refs/有界历史 |
| `GET /projects/:id/repositories/:repoId/commits/:commitId` | project_version、可选已验证 parent_id → 提交详情/有界文件变化 |
| `POST /projects/:id/repositories/:repoId/comparisons` | project_version、file scope、kind(head/staged/unstaged/reference/commit)、允许 ref 或 commit/parent → readonly original/modified/version/object ids |
| `GET /projects/:id/git-baseline` | project_version、folder_id/path → 所属 repo/head/blob/text 或明确 untracked/no_repository/unavailable |

Git snapshot_id 与缓存按 repo_id、配置版本、元数据指纹、对象ID隔离；刷新允许重新生成 snapshot，不将过期状态拼到新 HEAD 上。轮询/focus/online/文件事件触发失效，有效期内也校验引用；旧响应丢弃。

400 invalid_request/invalid_pattern；403 forbidden；404 not_found；409 conflict/configuration_changed；410 expired；413 limit_exceeded；428 version_required；429 capacity_exceeded；503 tool_unavailable/repository_unavailable；timeout 为明确非成功。不能回显原始 stderr。fixture 同步 Go DTO、TS decoder、client 与错误文案。

## 7. 前端数据流与生命周期

ProjectWorkbench 编排 Explorer/Search/Git 侧栏，桌面快捷键 Ctrl/Cmd+Shift+F/H；目录右键沿同一搜索意图传 folder/path。手机增搜索/Git 单内容视图及返回编辑器入口，不引入并排列；设备视图 schema 兼容迁移并保持原桌面组/终端记录。

SearchPanel 使用现有 shadcn Input/Toggle/Select/Button/Alert/Empty/ContextMenu，选择控件需官方 Checkbox；请求 AbortController + request generation，保留查询/选择但不持久化服务 search_id 或正文。结果打开调用既有 open buffer，并单次 selection 请求定位对应行列；文件版本已变先复验，不能在新内容错误定位或替换输入。

ReplacePreview/Git comparison 使用同一只读比较展示薄封装，桌面懒加载 DesktopDiff，手机原/新文本分步查看；关闭释放比较 model/worker，保持编辑原 model。搜索替换按钮须明确预览与应用，失败/部分结果可见，默认取消焦点，不添加永久 force。

GitPanel 负责选择 repo/refs/history，不控制文件所属仓库。FileEditor 获取对应 HEAD baseline；DesktopEditor 仅新增 decoration collection 和有界行 diff 计算，监听 buffer generation/head 对象变化、丢弃迟到结果，主题切换使用语义色不重建 model。无仓库/未跟踪状态用文字，不把空字符串冒充 HEAD。保存只更新磁盘，不清 HEAD 修改条。页面卸载取消读请求和释放比较资源，不对真实 terminal runtime 发命令。

## 8. 开发验证、延期验收与回滚

必须先通过 snapshot 读取边界及 CLI 对照的本地自动化阻塞关卡，再开放产品 API。后端目录/错误/质量/认证/HTTP/文件/命令、前端组件/状态/类型/质量/clients/hooks/生命周期与跨层/复用指南在实施前完整加载。具体检查见 implement.md。

统一实机验收按用户明确指令恢复；安全机制的本地集成证明是开发门禁，不能记为 Debian/手机已验收。若环境缺能力，不为赶进度裸运行 CLI。配置/接口回滚仅关闭 Search/Git 并清理自有 staging，不触碰真实仓库、文件、终端或 W05 草稿。schema 升级有兼容读取，不丢设备独立记录。

## 9. 实施收敛（2026-10-01）

用户“开始实施”后已激活任务。最终协议以 [后端 owner](../../spec/backend/search-git-contract.md) 与 [前端 owner](../../spec/frontend/search-git-workbench.md)为准，前述表格保留为规划资料。实际 search/preview 使用 `id`，Git repo列表仅身份/范围/可用状态；每请求独立私有snapshot，不提供持久snapshot_id，历史分页携带首屏head冻结对象遍历。取消预览释放正文并保留有界摘要；额度计算DTO和正文。行比较采用有界LCS及超限区间标记。实机验收未执行。
