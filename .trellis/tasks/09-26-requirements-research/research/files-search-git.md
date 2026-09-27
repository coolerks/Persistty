# Research: 文件安全、编辑冲突、搜索替换与 Git 查看

- Query: 在 AI 与浏览器编辑器共同操作 Debian 工作目录时，如何定义文件访问、避免旧内容覆盖、遵守 .gitignore 的文本查找替换，以及 Git 提交记录和 diff 的可验证需求？
- Scope: mixed；用户明确需求 U06/U07/U11，仓库初始契约与官方能力核对。
- Date: 2026-09-26
- 状态：需求调研；不是实现设计定稿。本轮只读取仓库与公开官方资料，没有运行产品代码、Git 命令或远端实验，没有安装依赖。
- 需求证据见 [requirements-source.md](requirements-source.md)。初始规范不能替代用户确认，本文中的候选行为与实验都不代表已接受功能。

## Findings

### 1. 用户故事与当前证据

| 用户故事 | 来源 | 可验收结果候选 | 尚未定案 |
| --- | --- | --- | --- |
| 指定项目目录后，浏览文件树并编辑文本 | U06 | 目录与合法文件名正确展示；路径访问限定在批准目录 | 工作区根的选择方式；根内绝对符号链接；挂载目录 |
| AI 或终端修改文件后，浏览器获得变化且旧缓冲区不静默覆盖新内容 | U06 | 外部已完成修改使旧版本保存失败，本地未保存内容保留 | 持续竞争时可承诺的边界；关闭 dirty 标签行为 |
| 在项目中查找并替换文本，遵守 .gitignore，不扫描 node_modules | U11 | 忽略集合可解释；替换前看到变化；冲突文件零修改 | 隐藏文件、额外 ignore、已跟踪但命中 ignore 的文件；正则范围 |
| 看 Git 记录和差异，区分二进制 | U07 | 比较双方明确；新建、删除、重命名和二进制有合理结果 | diff 是未暂存、已暂存、HEAD 还是提交与父提交；worktree 支持 |

不从这些故事扩展到 Git 写操作、草稿恢复、全套 Explorer 写操作或自动保存；这些均需要独立范围确认。

### 2. 仓库文件与代码模式

| 文件与定位 | 内容及本专题用途 |
| --- | --- |
| `README.md:5` | 当前无可运行产品代码；因此以下是规范模式，不是已实现行为 |
| `.trellis/spec/backend/index.md:4` | 初始契约待后续实现验证 |
| `.trellis/spec/backend/filesystem-guidelines.md:4` | API 接受 workspace 相对路径，终端 shell 权限另算 |
| `.trellis/spec/backend/filesystem-guidelines.md:8` | hash/mtime/size 版本；不可仅依赖时间戳 |
| `.trellis/spec/backend/filesystem-guidelines.md:13` | root 句柄限定最终 I/O，拒绝 EvalSymlinks 后普通路径 I/O |
| `.trellis/spec/backend/filesystem-guidelines.md:14` | 链接、特殊文件、CLI 访问与安全快照边界 |
| `.trellis/spec/backend/filesystem-guidelines.md:18` | 同目录临时文件、版本复验、原子 rename、同步与权限 |
| `.trellis/spec/backend/filesystem-guidelines.md:20` | 原子 rename 不是外部 writer 的 CAS，存在提交窗口 |
| `.trellis/spec/backend/process-guidelines.md:16` | CLI cwd 不能限制真实读取；Landlock 或安全输入为候选 |
| `.trellis/spec/backend/transfer-search-git.md:17` | rg JSON、隐藏文件、ignore、强制排除 .git 与安全快照 |
| `.trellis/spec/backend/transfer-search-git.md:19` | preview/apply 共用语义，逐文件部分结果 |
| `.trellis/spec/backend/transfer-search-git.md:22` | Git 只读、配置抑制、worktree 元数据需批准 |
| `.trellis/spec/frontend/editor-terminal-lifecycle.md:5` | dirty buffer、保存期间输入、409 后 diff 与本地内容保护 |
| `.trellis/spec/frontend/editor-terminal-lifecycle.md:6` | clean tab 刷新、dirty tab 外部修改提示 |
| `.trellis/spec/backend/security-config.md:4` | File API 根限制不是 tmux shell 沙箱 |
| `.trellis/spec/guides/index.md:10` | 研究不得将假设当已验证故障 |

模式评价：强版本、分离本地缓冲区与服务器内容、逐文件应用都契合 U06/U11；但无源码，不能宣称可用。现有 spec 包含的默认容量、草稿和 Git 只读细节仍属于初始约定。

### 3. 文件根安全与 Go 版本

文档事实：[Go os.Root](https://pkg.go.dev/os#Root) 从 Go 1.24 提供；`Root.Rename`、`Readlink`、`ReadFile` 等从 1.25 提供。它阻止路径逃出 root，跟随不越界的相对符号链接；绝对符号链接被拒绝，即使目标文本指向根内。它不阻止跨文件系统、bind mount、设备或 `/proc`。Unix 上 root 跟随最初打开的目录身份，即目录移动后句柄仍指向该目录。资料页面显示 go1.27.1，仅用于核对 API；没有据此选定项目 toolchain。

[Go 官方安全说明](https://go.dev/blog/osroot)解释了先检查符号链接再普通打开的 TOCTOU 风险。设计推论：词法验证解决用户输入错误，句柄限定解决真实打开的竞态，两者不能互相替代；对 fsnotify、CLI 和临时文件也不能只验证字符串。

需明确的语义：

- 如果项目目录被外部重命名，继续绑定原目录，还是暂停并要求重新选择？root 句柄安全与用户认为的“路径”未必一致。
- 根内相对链接可读写是候选；根内绝对链接需要拒绝或独立实现安全解析，不能为了兼容回退为普通 `os.Open`。
- 根路径下合法挂载目录与硬链接并不等于纯路径越界。`os.Root` 不追踪内容最初来自哪个目录；有根外别名的 inode 仍可从根内路径读取。若期望内容来源隔离，必须另定义部署边界，不能用 root 的名称暗示保证。
- 初始 spec 的“写入安全目标且保留链接本身”不是简单 `Root.Rename(temp, linkPath)`；后者替换目录条目。应在安全句柄内辨认并锁定目标语义。

候选验收：拒绝 ../、多次解码、根外相对链接、绝对链接与特殊文件；根内合法空格/中文/前导短横线保留。交换父目录或链接时不能访问根外 sentinel；不能通过事后过滤输出假装未读取。

### 4. CLI 的读取范围及 Landlock 前提

文档事实：[Linux Landlock 官方文档](https://docs.kernel.org/userspace-api/landlock.html)要求内核支持、编译启用 `CONFIG_SECURITY_LANDLOCK`、启动启用对应 LSM，并运行时探测 ABI。ABI 1/2 不能拒绝文件 truncate，ABI 3 增加该权限。规则对线程生效并由子进程继承；已打开的 fd 不会自动失去原权限。`no_new_privs`/权限配置、未处理的访问类别与特殊文件系统必须单独考虑。

设计推论：需要一个受限子进程入口在 exec git/rg 前设置规则，避免在多线程 Go 主服务中临时限制一个线程后假定所有线程已受限。不能默认目标 Debian 有新 ABI；若承诺“只读”，只限制 WRITE_FILE 在旧 ABI 不够。工具进程只继承必要 fd；工作目录、metadata 与二进制/动态库的许可分别列出。允许库目录读取意味着 CLI 能读取那里，故此边界是“项目内容限定加明确运行时许可”，不是“系统上只可读取 workspace”。

候选策略比较：

| 策略 | 对用户语义的影响 | 必须验证 |
| --- | --- | --- |
| Landlock 下真实 CLI | 更容易保留 Git/rg 原生语义，但有 Debian 内核与 fd/配置约束 | ABI、只读、根外读取、helper、metadata、真实动态链接运行 |
| root 发现文件，安全 snapshot 供搜索引擎 | 可限定内容读取；引擎不遍历真实目录 | 完整 ignore 语义、编码、query 与 preview/apply 相同；stdin 不会自动保留目录忽略语义 |
| Git 使用重建的 staging repo | 有可能降低真实目录访问，但不应直接采用 | index、属性、worktree、rename、对象等语义是否与真实 CLI 一致；成本较高，当前未证明 |

Landlock 为现有 spec 候选而非已确定依赖。目标不支持时不能不告知就使用裸 CLI。网络与 helper 限制还需配置/执行边界，不能单凭文件读取限制声称离线。所有实验均未执行。

### 5. Git worktree 元数据与只读的实际含义

[Git repository layout](https://git-scm.com/docs/gitrepository-layout)说明 `.git` 可以是指向外部 gitdir 的文件；对象库还可通过 alternates 借用其他对象目录。[Git worktree](https://git-scm.com/docs/git-worktree)说明每个 worktree 的私有 metadata 与公共目录分离。推论：不能以 `.git` 必须是目录判断仓库，也不能把 gitfile 的任意外部地址自动授权。

候选流程语义：工作区首次识别仓库后，登记确切 gitdir/common-dir；根外对象 alternates、submodule metadata、配置 include 另行明确支持或拒绝。拒绝的仓库展示“当前访问边界不支持”，不要返回“没有 Git”。普通文件 API 根仍不扩大为整个主仓库/HOME。

只读不只是 UI 隐藏写按钮：

- [git-status](https://git-scm.com/docs/git-status#_background_refresh)默认会写回 index 的 stat 缓存；建议候选使用 `--no-optional-locks`，并在限制环境验证无 metadata 写入。
- [git-config](https://git-scm.com/docs/git-config#Documentation/git-config.txt-corefsmonitor)中的 fsmonitor 可触发 helper，include 可访问别处配置；对旧版本 `core.fsmonitor=false` 的解释也有差别。不能只禁 external diff 便声称没有外部程序。
- [Git 主命令文档](https://git-scm.com/docs/git)提供 `--no-lazy-fetch`/`GIT_NO_LAZY_FETCH` 禁按需取对象、literal pathspec 避免文件名被解读为 pathspec 表达式。需按锁定 Git 版本核对；partial clone 缺对象应报不可用，不能联网补齐。

参数列表、`--`、受限 revision 分别解决 shell/flag/revision 注入；`--` 后的 Git pathspec 仍不是天然字面文件名。`.gitignore` 不是安全边界；CLI 配置、外部 diff/textconv、pager、fsmonitor、继承环境与 Git 属性全部属于工具实验矩阵。

### 6. 版本校验、原子保存与链接语义

现有版本契约是 hash+mtime+size，适合发现已完成的外部改动。相同大小/相同时间但内容不同必须冲突；内容相同但 inode 被替换也可能代表目标身份变化，应在设计中定义。强 hash 并不证明读取时获得单一时刻快照：外部程序可通过同一个 inode 边写边读。读取前后复验与有界重试可降低风险；在任意非合作 writer 下不能据此证明全局原子快照。

[Go Rename](https://pkg.go.dev/os#Rename)是路径替换操作，不携带 expected hash。设计推论：最终校验后外部 writer 写入，再 rename，仍可能覆盖；应用 mutex 只协调 Persistty 自身。fsync 解决持久性步骤，不能让“检查+rename”变成 CAS。操作系统文件锁也需要外部工具遵守，不能假定 Codex 或所有编辑器合作。

| 文件状态 | 可选保存语义 | 用户可见影响 |
| --- | --- | --- |
| 普通文本文件 | 同目录 temp 完整写入，安全 rename 发布 | 避免半写内容；inode 变化，旧打开 fd 仍见旧对象 |
| 相对符号链接到根内文件 | 安全解析目标再替换目标；或首版只读 | 应保持链接条目，不把链接变成普通文件 |
| 多个硬链接 | 拒绝/只读并解释；或替换当前名字并明确断开关联 | temp rename 只换一个名字，另一个名字继续见旧 inode；原地写则影响全部别名且损害原子发布 |
| executable、ACL/xattr/所有者特殊 | 明确需要保留的 metadata | 初始 spec 只保留普通 mode 位，不能声称完整元数据无损 |
| 文件被删除/父目录移动/权限变化 | 重新读取并冲突或报不可用 | 不应自动重建、改路径或静默降级 |

“旧内容不覆盖新内容”有两种候选验收层级：

1. 已在提交前观察到的外部修改必须阻止，冲突时磁盘不变且本地输入不丢；持续竞争窗口明确告知。这是当前候选设计能覆盖的范围。
2. 任意时刻的任意外部 writer 都绝对不会被覆盖。这要求所有 writer 合作或改变工作流；当前讨论/规范未提供能满足该保证的机制，不能先承诺再实现。

协议事实补充：[RFC 9110 §13.1.1](https://www.rfc-editor.org/rfc/rfc9110.html#name-if-match)定义标准 `If-Match`，使用强 ETag 比较；条件失败禁止执行请求，标准失败状态为 `412 Precondition Failed`，已确定请求此前成功的情况可按标准返回成功。[§15.5.10](https://www.rfc-editor.org/rfc/rfc9110.html#name-409-conflict)的 409 则表达与资源当前状态冲突。原讨论将 `If-Match` 与 409 混写，不能作为已定案标准协议复制。仓库 `.trellis/spec/backend/http-api.md:9`/`:29` 使用 JSON `expected_version` 与 409，可独立作为业务协议成立；JSON 字段 `etag` 也不自动等于具备双引号语法的 HTTP `ETag` header。候选为沿用业务版本+409，或设计标准 ETag/If-Match+412；前后端统一处理并测试，不能一边宣传遵循标准 If-Match，一边忽略其语义。此选择不消除磁盘外部 writer 的提交竞态。

### 7. dirty editor 与批量替换

仓库的本地 generation 检查很重要：点击保存后继续输入，旧保存响应只推进已保存基线，不能把新输入设为 clean。外部改动时 clean buffer 可刷新；dirty buffer 应保留并提示比较、重载或基于新版本重新提交。watcher 只是失效通知，[fsnotify](https://github.com/fsnotify/fsnotify#faq)不自动递归监控子目录，移动后也不是自动持续监控；事件丢失/断线后的复验仍需服务器版本。

批量替换与已打开 dirty buffer 的冲突有三种候选行为，需澄清：

- 跳过 dirty 文件并显示理由，其他文件正常替换。
- 对 dirty buffer 做本地替换，保存仍按原文件版本走正常冲突保护；不能把磁盘 search 的 offset 用在 dirty 文本上。
- 要求先处理 dirty 文件后再提交整批；成本是用户需往返操作。

建议验收按文件表达，不默认跨文件事务：A 成功，B 在 preview 后被 AI 修改，C 写入失败，则结果精确为 applied/conflict/error；B 内容保留；不显示“全部成功”，也不声称 A 已回滚。中途取消只阻止未提交文件，已提交结果可查；通信超时不等于失败，需要能核对实际落盘。对已成功文件自动重试可能再次替换，应以 preview/version/操作结果识别重复。

单文件多个 match 应先在同一快照生成完整新内容，再一次发布；不能逐 match 写盘造成局部文件变化。所有替换都需保持未涉及的原始字节，并保留适用换行/BOM规则。

### 8. ignore、文本分类与正则一致性

文档事实：[ripgrep GUIDE](https://github.com/BurntSushi/ripgrep/blob/master/GUIDE.md)说明隐藏文件默认不搜索、忽略规则存在优先级，并可自动转码 UTF-16 BOM；它可搜索非 UTF-8 原始字节。不能据此把所有 rg 命中文件都当可安全编辑文本。

[ripgrep flag 源码](https://raw.githubusercontent.com/BurntSushi/ripgrep/master/crates/core/flags/defs.rs)确认：`--no-config` 阻止配置注入；二进制检测主要是 NUL 启发式；`--replace` 只改输出；`-g` 可覆盖 ignore 且后给的 glob 优先。推论：把用户 include glob 直接传 `-g '*.ts'` 可能纳入 node_modules，违反 U11。应先得到不可被 include 放宽的允许集合再求交集，或采用等价可验证机制；追加 `.git` 排除仅解决 `.git`，没有解决 node_modules 的优先级。

还需区分 Git 与搜索的 ignore：[gitignore](https://git-scm.com/docs/gitignore)不影响已跟踪文件；rg 是基于遍历过滤，不等同 Git 跟踪状态。`.ignore`/`.rgignore` 的重新纳入规则是否允许覆盖 `.gitignore`、父目录/全局 ignore 是否参与、非 Git 目录中的 `.gitignore` 是否生效，都需要决定。“node_modules 不搜”究竟是依赖项目 ignore 还是额外硬排除，也不能默认为已定案。

尤其是没有 `.gitignore` 或其中没有 `node_modules/` 的项目：rg 不会因目录名意味着“依赖”就自动忽略它。U11 中“避免 node_modules 被查找替换”的意图清楚，但是否设置产品默认依赖目录排除、允许用户修改该默认值，仍是产品语义。验收必须覆盖缺失 ignore 的项目，不能只在恰有正确 `.gitignore` 的样本上通过。

文本候选语义：首版仅有效 UTF-8，BOM/CRLF/LF/无末尾换行明确保存，其他编码只读或拒绝替换；或者完整编码识别与无损回编码。前者范围更小，后者需要额外实现。NUL-free 不等于文本；非法 UTF-8 不能用 replacement character 静默解码再保存。混合换行是保留原始字节还是统一规范也需决定。

[grep-printer JSON 协议](https://docs.rs/grep-printer/latest/grep_printer/struct.JSON.html)提供 text/base64 数据和字节 offset；字节定位不等于 Monaco UTF-16 列。转换必须基于相同快照；UTF-16 自动转码得到的 offset 更不能用于原文件切片。emoji、组合字符、中文应验定位和替换。

[regex 官方 crate 文档](https://docs.rs/regex/latest/regex/)说明默认语法不支持全部 PCRE 特性。推论：rg Rust regex、Go regexp 与 Monaco/JavaScript 搜索的 word boundary、capture、空匹配和替换串未必一致；不能默认用三个引擎处理同一 query。候选为后端同一匹配/替换引擎从 snapshot 产生服务端 preview，apply 提交完全相同的新字节；前端只展示。使用 Rust/PCRE2/受限 Go 子集属于技术选择，须以样本一致性证明，不在此定稿。

候选验收包括 literal `$`、反斜线、捕获组、空匹配、whole word、大小写、CRLF 锚点和选中 match；预览 hash 与实际新文件 hash 相同。过期 preview、结果截断、权限失败不能当成空结果；大结果与长行需有限资源和明确提示，限值待真实项目规模确认。

### 9. Git diff 比较对象

[Git diff 官方文档](https://git-scm.com/docs/git-diff)区分工作树、index 与 commit。需求 U07 的“diff”没有指定比较双方，候选可用下表澄清：

| 展示名称候选 | 左侧 -> 右侧 | 命令语义参考 |
| --- | --- | --- |
| 未暂存更改 | index -> 磁盘 working tree | `git diff` |
| 已暂存更改 | HEAD -> index | `git diff --cached` |
| 当前总更改 | HEAD -> 磁盘 working tree | `git diff HEAD` |
| 提交详情 | 指定 parent -> 指定 commit | `git diff <parent> <commit>` |

未保存 Monaco buffer 不属于上述磁盘 Git diff；若展示它，需要单独标明“编辑器未保存”。同一文件可同时已暂存和未暂存，不能合并成一个模糊 modified。未跟踪文件不出现在普通 `git diff` 中，可展示新文件全文；二进制不强行生成文本 diff。根提交无 parent、merge 多 parent、未解决冲突、rename old/new、无 HEAD 的新仓库都有独立状态，首版支持深度应明确。

差异不保证跨多个命令的全仓库单时刻快照。log 中提交 ID 通常稳定，但工作树与 index 仍可被终端改变。页面刷新可展示观测时间与比较对象；提交对象不可读/缺失时提示，不能转为与任意当前 HEAD 比较。两个 commit 之间比较与 merge-base 三点比较不同，未被用户明确要求则不扩为完整比较工具。

### 10. 待用户决定（调研结束后再问）

1. 项目根如何选择；是否支持根内绝对链接、挂载目录和 Git worktree？实际开发是否使用这些布局？
2. 外部并发保护是否接受“已检测修改拒绝保存，持续竞争存在明确窗口”，还是硬性要求任意 writer 的绝对防覆盖？
3. 根内符号链接及硬链接如何编辑，是否要求 ACL/xattr 完整保留？
4. dirty 文件关闭、外部修改、批量替换碰到 dirty buffer 时选哪种行为？本地草稿持久化另行决定。
5. 隐藏文件、已跟踪 ignore 文件、全局/父目录 ignore、`.ignore`/`.rgignore` 的优先级；node_modules 是否无条件排除？
6. 正则、跨行、capture、whole word 是否首版需要；UTF-8 之外编码与混合换行如何处理？
7. Git 哪几种 diff 是日常需要，merge/root commit、submodule、partial clone 支持到什么程度？
8. 搜索/替换规模、超限反馈、逐文件部分成功与取消后的结果展示。

### 11. 待实验验证（可执行实验建议，均未执行）

实验在独立临时目录/专门 Spike 中进行；不操作用户真实项目、不连接远端。本轮无产品代码，以下命令为未来执行入口而非“已通过”。

| 实验 | 执行方式与样本 | 通过/失败判据 |
| --- | --- | --- |
| E-F01 环境能力 | 在目标 Debian 记录 `uname -r`、`go version`、`rg --version`、`git --version`；检查启用 LSM；用小探针调用 `landlock_create_ruleset(NULL,0,LANDLOCK_CREATE_RULESET_VERSION)` | 实际 ABI/配置可见；ENOSYS/EOPNOTSUPP 被明确区分；macOS 结果不能替代 |
| E-F02 root 读写 | Go 临时 harness 建 root/外部 sentinel；创建根内相对、根外、绝对、循环链接；另线程持续交换父目录/链接 | 从未打开根外 sentinel；根内绝对链接行为与选定语义一致；特殊文件不阻塞 |
| E-F03 CLI 隔离 | 相同样本通过受限 rg/Git；利用 trace/审计观察实际 open；根外 sentinel 唯一 marker；交换目录，试继承 fd 和 `/proc/self/fd` | 不能只验证 API 输出；根外实际读取须拒绝；不需要的 inherited fd 已关闭 |
| E-F04 Landlock 只读 | 受限 launcher 执行写/rename/truncate 小探针；Git status/diff/log；对 runtime 库最小许可 | 所有声明禁止操作确实拒绝；ABI不足拒绝能力，无裸 CLI fallback |
| E-F05 保存竞争 | barrier 分别停在 read/stat/hash、最终校验、rename 前；外部 writer 原地写/rename、保持 mtime/size、删除或重建；每次记录 inode/hash | 已检测变更 409 零提交；无半文件；明确记录最终窗口反例，不能把它记为绝对 CAS 通过 |
| E-F06 metadata/链接保存 | executable、hardlink 两名字、相对 symlink、ACL/xattr 样本，保存后 stat/readlink/读另一名字 | 与所选保留/拒绝规则相同，不静默断开链接或丢 metadata |
| E-F07 ignore 真值 | 样本含 node_modules、.github、.env、嵌套 .gitignore、!例外、.ignore 重纳入、tracked-but-ignored、include '*.ts'；比较允许集合 | include 不能扩大硬性 ignore；各例外符合定案；不将当前 spec 的 flags 当证据 |
| E-F08 编码/regex | UTF-8/BOM、UTF-16 BOM、非法 UTF-8、CRLF/LF/混合、无末尾 LF、emoji；字面 $、组名、空匹配、word boundary；保存 preview 字节再 apply | 选中 match 精确；preview/apply hash 一致；不支持输入明确拒绝，不损坏原字节 |
| E-F09 dirty 与部分成功 | A 普通、B dirty、C preview 后外改、D 写入失败；保存请求期间继续输入、重连、apply 中途取消 | 本地输入不丢；每文件结果准确；失败文件不写；已成功不伪称回滚；重复不会二次替换 |
| E-F10 Git 查看 | 实验 harness 创建普通 repo/worktree，制造 staged+unstaged 同文件、untracked、rename、root/merge commit、二进制、缺对象；CLI 对照四类 diff | 比较双方标注正确；worktree private/common metadata 准确授权；缺对象不联网 |
| E-F11 Git 配置副作用 | 临时 repo 配置 ext diff/textconv/fsmonitor/config include/alternates；helper 仅写 marker；读取前后对 metadata hash，禁止远端 | helper marker 不出现；无可选 index 写入；根外 include/对象访问按批准规则拒绝；无 lazy fetch |

## 外部参考与版本

- [Go os](https://pkg.go.dev/os)：浏览返回页面为 go1.27.1；研究采用 API `added in` 信息确定 1.24/1.25 下限，没有锁定项目版本。
- [Traversal-resistant file APIs](https://go.dev/blog/osroot)：Go 官方原理说明，不覆盖全部本项目写入协议。
- [Landlock](https://docs.kernel.org/userspace-api/landlock.html)：在线内核文档包含新 ABI；不能据此推断用户 Debian ABI。
- [ripgrep GUIDE](https://github.com/BurntSushi/ripgrep/blob/master/GUIDE.md)、[flags 源码](https://raw.githubusercontent.com/BurntSushi/ripgrep/master/crates/core/flags/defs.rs)：master 是移动目标；后续须锁 release 并用对应 `rg --help`/实际行为复验。
- [grep-printer JSON](https://docs.rs/grep-printer/latest/grep_printer/struct.JSON.html)、[regex](https://docs.rs/regex/latest/regex/)：上游作者维护 crate 文档；latest 不等于最终系统 rg 版本。
- [Git diff](https://git-scm.com/docs/git-diff)、[Git ignore](https://git-scm.com/docs/gitignore)、[repository layout](https://git-scm.com/docs/gitrepository-layout)、[worktree](https://git-scm.com/docs/git-worktree)、[Git 主命令](https://git-scm.com/docs/git)、[status](https://git-scm.com/docs/git-status)、[config](https://git-scm.com/docs/git-config)：官方在线文档；目标 Debian Git 版本未知，能力需按安装版本复核。
- [fsnotify](https://github.com/fsnotify/fsnotify)：官方 FAQ；仓库尚未选定模块版本。
- [RFC 9110](https://www.rfc-editor.org/rfc/rfc9110.html)：2022 年 HTTP Semantics 标准，If-Match/ETag/409/412 区别；未据此修改已有 API。

## Related Specs

主要为 `backend/filesystem-guidelines.md`、`backend/process-guidelines.md`、`backend/transfer-search-git.md`、`backend/security-config.md` 与 `frontend/editor-terminal-lifecycle.md`；定位见第 2 节。以下是研究发现的需后续设计/规范复核点，并非本轮修改建议执行：

- 当前“根内链接”说明未区分绝对链接被 os.Root 拒绝；挂载/硬链接与目录身份迁移边界还需表达。
- 当前 rg include/exclude flags 契约若直接放行正向 glob，可能覆盖 ignore；`.ignore` 更高优先级也与用户要求存在待澄清张力。
- Landlock 是未验证候选；ABI、只读 truncate、线程入口、fd、运行时许可和 helper 约束需实验。
- Git 只读契约还需 optional locks、fsmonitor 版本、pathspec literal、lazy fetch 和 worktree/alternates 授权矩阵。
- 文件原子保存的硬链接、ACL/xattr、dirty 批量替换、通信失败后的重复提交语义尚未定案。

## Caveats / Not Found

- 无产品代码、真实测试结果、Go/npm manifest，也没有用户 Debian/浏览器具体版本；全部架构推论均待设计与实验。
- 本专题未研究上传/ZIP/图片隔离的完整流程，交由对应专题；不因本文出现文件保存就扩首版传输功能。
- 没有访问远端、安装依赖、运行 Git 操作、创建实现文件或修改已有规范。实验列表只是后续验证入口。
- 不能承诺任意外部 writer 下原子 CAS、全仓库一致快照、inode 来源隔离或完整 metadata 无损；官方能力与当前候选机制不支持直接作这些结论。
- 尝试读取 Open Group 在线 rename/link 页面失败，未以其为证据；rename/link 结论采用 Go 官方 API 与明确标注的文件系统语义推论。
