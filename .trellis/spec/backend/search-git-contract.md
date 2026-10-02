# W06 搜索替换与只读 Git 契约

## 1. 范围与触发条件

W06 已接入多根搜索、选择后替换预览/应用、多仓库只读 Git 及 HEAD 到编辑缓冲区标记。实现归 `internal/search`、`internal/gitview`、`internal/toolrunner`；安全源读取归 `internal/files/snapshot*`，发布复用 `files.Save` 和 `Store.WithRegisteredFolder`。本契约覆盖旧[传输与搜索](transfer-search-git.md)中的候选签名；无数据库迁移。开发检查与真实 Debian/手机验收分别记录，延期验收不能标为通过。

## 2. 签名

全部位于 `/api/v1/projects/:id`，统一成功 envelope `{data,request_id}`。所有 route 后端鉴权；POST/DELETE 按现有 Origin/CSRF 校验。GET Git 需要正整数 `project_version`；所有标识符必须编码为单个 URL 段。请求 JSON 使用严格字段、重复字段与大小校验。

| Route 后缀 | 请求 | 响应 data |
| --- | --- | --- |
| `GET /file-names` | `project_version,query` | `{project_version,items:[{folder_id,path}],truncated}`；详见[名称搜索](file-name-search.md) |
| `POST /searches` | `project_version,folder_id,path,pattern,regex,case_sensitive,whole_word,include,exclude` | `id,project_version,files,truncated,skipped,expires_at` |
| `DELETE /searches/:searchId` | 无 body | 204，释放搜索快照 |
| `POST /replace-previews` | `project_version,search_id,selected_match_ids,replacement` | `id,project_version,files,results,state,expires_at` |
| `GET /replace-previews/:previewId` | 无参数；或 `file_id` | 摘要；或 `{id,path,original,modified}` |
| `POST /replace-previews/:previewId/apply` | `project_version,selected_file_ids,protected_file_ids` | 预览摘要与逐文件 results |
| `DELETE /replace-previews/:previewId` | 无 body | 204，取消未开始写入/释放非运行中正文 |
| `GET /repositories` | `project_version` | `{items,truncated}` |
| `GET /repositories/:repoId/status` | `project_version` | `{repo_id,head,branch,changes,total_paths}` |
| `GET /repositories/:repoId/refs` | `project_version` | `{items:[{name,commit_id}]}` |
| `GET /repositories/:repoId/log` | `project_version,offset=0,head=可选固定对象ID` | `{head,items,next_offset}` |
| `GET /repositories/:repoId/commits/:commitId` | `project_version,parent_id=可选父ID` | `{commit,parent_id,files,message,stats,github_url}` |
| `POST /repositories/:repoId/comparisons` | `project_version,path,kind,reference,commit_id,parent_id` | `{repo_id,path,old_path,original,modified,binary,baseline}` |
| `GET /git-baseline` | `project_version,folder_id,path` | `{state,repo_id,head,content,version}` |

`kind` 为 `head|staged|unstaged|reference|commit`，没有任意参数/revision/force 透传。与本地引用比较先从受限 refs 固定对象；提交/父提交为验证后的十六进制对象ID，父必须属于该提交；根提交原内容为空，merge 默认第一父。历史首屏返回 `head`，后续分页附同一 `head`，刷新才改基线；每页50项，`next_offset=-1` 表示结束，offset≤10000。

## 3. 请求、响应与资源契约

### 搜索与替换

`folder_id="",path=""` 是整个项目；选定文件夹后 `path` 可收窄目录。pattern 非空≤4096字节，不含真实 CR/LF/NUL；只支持 rg 默认正则引擎的单行模式，不启用 PCRE2。include/exclude 为≤32个、各≤256字节的 glob；前端用 `;` 分隔。include 与原忽略集合相交，不能重新纳入已忽略文件。

search file 为 `{id,folder_id,path,version,matches}`，version 沿用完整 `{mtime,size,etag,identity}`；match 为 `{id,line,column,end_column,preview}`，坐标为1-based UTF-16，包含 BOM、CRLF/裸 CR 与 emoji 的定位转换。先保留原字节快照，裸 CR 仅在等长的引擎输入中转为 LF，CRLF 使用 `--crlf`；替换依据原字节 offset 拼接，未替换的 BOM/混合换行/末尾换行保持。literal replacement 中美元符号按字面处理；regex 使用 `$1`、`${name}`、`$$`；rg JSON 缺少 replacement 能力报不可用，不能自行换另一引擎。skip 为 `{folder_id,path,reason}`。

replacement≤4096字节，不能含NUL。preview file 为 `{id,folder_id,path,version,count,new_hash}`，正文仅通过其 `file_id` 比较接口读取；客户端不能发送新正文或 offset。预览 `state=ready|applying|completed|cancelled`。results 为 `{id,state,reason,version}`，`state=applied|conflict|error|skipped`；非成功 version=null，成功为新完整版本。reason 如 `not_selected,unsaved_input,cancelled,version_changed,save_failed`，不回显原始 OS/stderr。取消只停止尚未发布的文件；已成功的文件不回滚。

搜索/预览绑定认证 session 哈希和项目，生命周期5分钟，Web 重启丢失。每 session 最多4个未释放快照，全局128MiB额度同时计算 DTO 与私有正文；取消非运行中的预览清除正文，保留短期摘要用于立即查询/重放，后续新分配最多保留每 session 4个、全局128个取消摘要，可能返回410。Apply 第一次有效请求裁决一次，重复返回原结果，不再次执行 Save。TTL/额度不保证数据长期可用；明确新搜索/预览后410要求重新生成，不能自动强写。

每个搜索累计最多50000条源目录项、5000匹配、2000匹配文件；扫描文本默认64MiB，控制文件总量同样有界，单文本8MiB。默认排除 `node_modules,dist,.next,vendor`，可配置为空数组；`.git` 文件/目录永远强排除，隐藏代码仍可搜，二进制/符号链接/特殊文件不读取。遵守范围内 `.gitignore,.ignore,.rgignore,.git/info/exclude`；不读取父级/全局规则。命中/目录/扫描上限返回 `truncated`，不完整 ignore 控制文件和工具输出上限直接失败。

### Git 与工具隔离

源目录由注册 root identity 和 no-follow 目录/叶句柄读取，枚举/文件读取后复验；不允许通过“校验后 os.Open 绝对路径”重新打开。CLI 只读0700私有 staging；不能把 cwd 当文件沙箱，也不能通过输出过滤补救根外读取。Git metadata 源不写入；只复制 HEAD/index/refs/packed-refs/objects/shallow/白名单 info 与 sharedindex，逐文件版本复验。对象/元数据合计默认512MiB，元数据流式拷贝；状态及需要改名识别的比较复制 tracked 与未忽略的工作树候选，受同一上限约束，不静默遗漏 tracked 文件；历史/引用/提交详情/暂存与提交比较只复制元数据，baseline/本地引用与普通 HEAD/未暂存比较只补目标文件。仓库发现最多100个，重叠真实 `.git` identity 去重。

Git `.git` 文件/链接、commondir/gitdir、alternates/http-alternates、promisor、未知 extensions、外部 filter 配置或属性先不可用。外部工作树 symlink 只读取 link 文本、在 staging 作为普通文件配合 `core.symlinks=false`，不读取目标。config 用 `--no-includes --file` 显式解析后重写允许 core/objectformat 设置；忽略 include、helpers、hooks、fsmonitor、pager、diff driver。命令为固定参数列表，无 shell；环境不继承 Git/rg/凭证配置，设置 `GIT_NO_LAZY_FETCH=1,GIT_OPTIONAL_LOCKS=0,GIT_TERMINAL_PROMPT=0` 及禁全局/系统配置，禁止联网补对象。git 使用 no-pager、no-ext-diff、no-textconv；stdout32MiB/stderr64KiB，stderr不进入HTTP/日志。并发最多2个任务，默认15秒，取消杀死自己的进程组，退出清理自己 staging。

repo item 为 `{id,folder_id,path,name,state,reason}`，state 为 available/unavailable。ID 绑定项目版本和元数据真实身份；不接受客户端仓库绝对路径。`changes=[{path,old_path,index,worktree}]` 来自 porcelain v1 -z；`total_paths` 是 HEAD 到磁盘的实际不同路径加未跟踪文件。暂存后磁盘又恢复 HEAD 的文件仍可出现在 changes，但不出现在 total_paths。空/删除文件与缺失对象必须区分：通过 ls-tree/ls-files 确认缺路径才返回空内容，缺 blob/tree/commit 报失败。

commit 为 `{id,parents,author,date,subject}`，根 parents=[]，不能返回null。HEAD/暂存/未暂存及历史提交比较识别 rename，old_path 非空时原边读取原路径，不把纯改名伪装成新增正文。文本比较单边≤8MiB；binary=true 时正文为空。baseline `state=tracked|untracked|binary|unavailable|no_repository`，version 可空；按真实绝对所属路径选最内层仓库，即使文件从另一重叠文件夹打开也正确。baseline 不用于写回缓冲区。

提交详情 `message` 为完整 UTF-8 多行消息，最多 64KiB，保留正文换行；读取 `%B` 时只去掉 Git format 添加的一条额外 LF，不使用日志 subject 代替正文。`stats=[{path,old_path,status,additions,deletions}]` 与 `files` 的顺序和路径一一对应，最多 5000 项；status 为 A/M/D/T/R/C，未改名 old_path 为空。增删行为非负安全整数，二进制的两个计数同时为 null，0 表示真实零行。统计基线与 `parent_id` 一致，根提交按空树，合并默认第一父或显式已验证父。

固定 `diff-tree --raw --numstat --no-abbrev -M -z -r` 在同一私有元数据快照生成状态与计数，NUL 解析保留文件名的 tab/LF；重命名两边路径匹配统计，历史比较使用该 old_path。`github_url` 为空或 canonical `https://github.com/owner/repo/commit/<对象ID>`；只从显式本地 config 的 remote.origin.url 提取 github.com 的 HTTPS/SSH/scp Git 地址，拒绝 lookalike host、额外路径、query/fragment 和异常端口，去掉凭证后生成公共网页地址。原始 URL/凭证不返回、不记录、不写入可执行私有 config；不调用 remote/fetch 或 GitHub API，没有新增配置/环境变量。

Git每次读请求生成独立快照，不提供持久 snapshot_id；一个响应内部由同一个 staging 和元数据复验保证一致，不承诺整个项目原子截点。状态、引用、历史的连续请求之间可能发生外部修改，不能拼成全仓库原子事务。历史分页和比较响应使用固定对象标明各自基线。

### 配置

`search: {binary: rg,timeout_seconds: 15,max_entries: 50000,max_results: 5000,max_bytes: 67108864,exclude_directories: [...]}`；`git: {binary: git,max_bytes: 536870912}`。binary 为程序名或绝对路径，不能附参数；timeout 1..120，entries 1..50000，results 1..5000，search bytes 1MiB..64MiB，git bytes 1MiB..512MiB。tool-staging 位于数据库同级目录，父目录受原存储配置约束；仅清理自有 `snapshot-*`。

## 4. 验证与错误矩阵

| 情形 | 结果 |
| --- | --- |
| 未认证 / Origin或CSRF错误 | 401 / 403，包含GET、比较和预览 |
| 非法字段/模式/版本缺失 | 400 invalid_request或invalid_pattern / 428 version_required |
| 配置/文件身份变化 | 409 conflict；Apply为逐文件conflict |
| session不匹配、过期、已释放正文 | 410 expired，不泄露是否存在 |
| 结果/扫描截断 | 200 truncated=true；控制文件/输出/文件上限 413 limit_exceeded或too_large |
| 活跃工具/缓存容量满 | 429 capacity_exceeded、Retry-After:5 |
| 工具缺失/必要能力缺失 | 503 tool_unavailable |
| 不安全元数据/缺对象 | 503 repository_unavailable或tool_unavailable，不伪造空树/clean |
| 超时/请求取消 | 503 timeout / 中止请求；Apply已开始时保留逐文件状态 |

## 5. 正常、基础与错误用例

正常：搜索 emoji 后UTF-16列定位原 Monaco model；选择匹配后预览、应用一次，原 HEAD/index/config hash保持。基础：无仓库或未跟踪显示文字，根提交比较空原文。错误：预览后外部修改，后端 conflict 或前端发现基线变化而跳过，外部正文与本地输入分别保留；缺对象不返回“文件不存在”。

## 6. 所需测试与断言

- `internal/files/snapshot_test.go`：traversal、叶/父symlink、FIFO、父目录替换、取消/容量，根外哨兵未读取。
- `internal/toolrunner/runner_test.go`：私有权限、工具缺失、环境去继承、容量、取消进程组与清理。
- search临时真实rg：ignore/include、多根去重、BOM/CR/CRLF/emoji、捕获组/选择/原字节、结果截断、配置与文件冲突、取消和幂等、释放额度。
- git临时真实仓库：staged/unstaged/HEAD总变化、根/merge父选择、删除、特殊名称、最内层/重叠根、缺对象/恶意配置/alternates/filter，原HEAD/index保持。
- 共享 `tests/fixtures/search-git.json` 同时由Go DTO roundtrip与严格TS decoder验证；HTTP session/Origin/CSRF及日志不含正文。
- `commit-details_test.go` 对照真实根提交/修改/删除/新增/纯改名/二进制/多行长消息，保留含 tab/LF 的路径、历史改名原边及源 HEAD/index/config；解析拒绝不匹配或不完整 NUL 记录。GitHub 地址测试覆盖 HTTPS/SSH/scp、凭证不回显及伪造 host/端口/路径拒绝；超过消息/文件/工具输出配额明确失败，不能回传截断成功或伪装零统计。
- 本机真实后端浏览器链路与编辑器保护/工作台回归；真实Debian工具、手机软键盘和完整设备矩阵按用户明确指令后续验收。

## 7. 错误与正确示例

错误：直接 `git -C 用户路径 diff`，或用rg offset配新读取版本执行替换；取消后重新POST形成第二次写入。正确：安全句柄→私有快照→固定CLI参数→与原版本绑定的预览→目标buffer保护→WithRegisteredFolder+Save复验→保留首次逐文件结果。


## 无提交仓库与快照目录（2026-10-01）

元数据snapshot必须创建白名单目录本身及有界子目录，包含空的objects/refs/heads；只复制叶文件会让 `git init` 无commit、无index仓库失去Git仓库身份。无HEAD返回真实branch、`head=""`、未跟踪/已暂存新增项；历史与refs为空，HEAD/暂存比较以空原内容为基线。禁止通过隐式commit/index或伪造对象“修复”源仓库。

仓库扫描或snapshot遇到files.ErrUnsupported归Git领域ErrUnavailable，HTTP为503 repository_unavailable，提示无法安全读取的文件/元数据/对象；不能显示文件编辑的“此文件类型暂不支持”，也不能伪装空仓库。非法UTF-8文件名测试为Linux专属，macOS/APFS不能创建该fixture；未实际运行不算通过。

回归 `TestUnbornRepositoryWithoutIndex` 从真实init开始，无index时状态/空历史/空refs/比较准确且源无新增index；stage后新增状态和空树比较正确、原index字节不变。既有缺对象/恶意配置/链接边界测试继续必需。

## Git 请求成本与安全快照范围（2026-10-01）

- 不以放宽并发/超时解决重复快照。Repositories 仍重新发现仓库；Service 仅保存最多1000项 opaque ID 到已发现位置的映射，满时清空；find 每次复验项目/文件夹版本、注册 root 和 `.git` identity，响应后再复验。映射不缓存状态、历史、正文或授权，未知ID回到有界发现，不能按客户端绝对路径执行Git。
- Baseline 只检查请求文件的祖先目录；对所有包含该真实路径的注册根分别检查，选最内层仓库，保留重叠根语义。已删除的祖先可向上继续；链接/不安全父目录不能跳过安全失败。
- 每请求独立0700 staging、白名单元数据/对象拷贝与完整版本复验不变。对象读取无须复制无关工作树。HEAD/未暂存目标在原边存在时只取该文件，直接按原路径比较；新增/潜在改名目标仍用 tracked 与未忽略的完整候选识别改名。过滤器配置仍拒绝，普通配置只解析一次并以验证过的枚举值生成私有安全config。
- 目标工作树路径不能含任意大小写的 `.git` 组件，不能覆盖 staging 元数据；完整工作树遍历跳过真实 `.git`，遇到大小写别名拒绝读取。单文件快照同样计算剩余字节预算；链接仍只复制 link 文本。
- metadata 复验流式写向 io.Discard，不能为丢弃的对象内容分配整包正文内存。源 HEAD/index/config 保持不变，不将源仓库交给 CLI，不建立跨请求正文/对象缓存，不使用源硬链接。
- 回归：`snapshot_scope_test.go` 用超过快照预算的无关大文件证明历史/详情/baseline/普通比较可用，而status不能静默省略文件；验证旧ID不能接受替换后的 `.git`、不可用仓库保留领域错误、保留元数据路径被拒绝。`performance_test.go` 以显式 PERSISTTY_GIT_PERF_ROOT 启用真实本机只读复测，所有CLI仍只执行于私有快照，断言源 HEAD/index/config hash 不变；耗时日志不含正文。

## 状态快照剪枝与请求预算（2026-10-02）

状态及改名识别所需工作树快照先在私有 staging 用 `ls-files --cached -z` 固定 tracked 路径及祖先集合，再广度枚举，每层复制安全的 `.gitignore/.gitattributes` 控制文件、批量 `check-ignore -z --stdin` 裁决。忽略的非 tracked 目录在枚举内部之前剪枝；tracked 路径与其祖先不得因 ignore 被丢弃，嵌套规则、否定规则、info/exclude 继续遵守。被忽略的属性文件仍是有效控制文件，外部 filter 保持不可用。stdin 每层至多 4MiB；目录项/深度/私有快照容量维持原边界。

候选集合确定后复用 `WalkSnapshotWithCopy` 按安全目录句柄复制正文，仅进入候选目录；不在源目录调用 Git，不引入源对象 hardlink 或持久正文缓存。`check-ignore` 本身不接受 pathspec magic，固定命令使用 `--no-literal-pathspecs`，路径经 NUL stdin 提供，不启用 glob 或任意参数。

HTTP 根据已注册 route pattern 为 search/Git 路由使用 `ToolTimeout()`，普通请求仍 10 秒；默认工具操作 15 秒、配置最多 120 秒。服务的同一有界 context 贯穿快照、命令和交付复验，不能让闭包重新使用无期限父 context。超预算仍明确 timeout，不增加自动重试。历史 `git log --topo-order` 保证后代先于父节点，分页继续固定首屏 HEAD。


### 剪枝与预算的可执行检查

- 范围：GET status、需要全树的 comparisons 和所有注册 search/Git 请求；API 签名及 envelope 不变，无新增配置或环境变量。
- 签名：`copyWorktree(ctx, snap, size, limit)`；`Runner.GitIgnored(ctx, dir, input)` 只接受私有 staging 的路径列表，返回 NUL 匹配；exit 1 为空匹配成功。
- 契约：忽略候选不能消耗正文预算，tracked 忽略文件必须仍被读取；同一配置预算覆盖 HTTP/service/子进程与元数据复验。命令及配额见上节。
- 错误矩阵：源变化→409；非忽略/跟踪正文超预算→413；外部 filter/元数据指针→503 unavailable；配置预算耗尽→503 timeout；取消→中止并清理自己的 staging。
- 用例：正常为 ignored 依赖目录剪枝；基础为 clean/未跟踪/无 HEAD；错误为忽略目录内 tracked 文件被错误丢弃或存在 filter 却报告 clean。
- 必需测试：`worktree_test.go` 使用超过单目录 10000 条限制的忽略树、超过正文预算的忽略文件、tracked 修改与嵌套否定规则，完整对照合成仓库 CLI；`tool_deadline_test.go` 通过实际注册 middleware 验 3/15/25 秒配置与普通文件 10 秒，相似路径不能命中工具预算；原改名/缺对象/安全根与源零写入测试继续通过。
- 反例：只提高 HTTP timeout 或硬编码排除 node_modules；正例：Git 原生忽略裁决先剪枝，同时用 index 跟踪集保留真实变更，并统一有界 context。


共享 discovery 的 `files.ErrTooLarge` 必须区分目录项/深度遍历限额与 ignore 控制文件读取限额：前者 truncated，后者原样返回 413，不能交付未完整过滤的候选。名称请求的工具兜底和 no-require-git 只在名称 owner 启用，全文搜索保留原 rg 行为，详见[名称搜索契约](file-name-search.md)。
