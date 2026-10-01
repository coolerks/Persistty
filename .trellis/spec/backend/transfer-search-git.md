# 传输、搜索替换与 Git

## 1. 范围 / 触发
这些功能共用 [安全文件访问](filesystem-guidelines.md)、[命令执行](process-guidelines.md)、[认证](security-config.md)，不能各自实现更弱的路径验证。传输沿用 W04；搜索/替换/Git 的已实施 API、DTO、配置和隔离边界以 [W06 契约](search-git-contract.md)为准，本文相关签名为历史初始设计。

## 2. 签名
上传状态机 `create -> put chunk(index,sha256) -> status -> complete(total_sha256) -> committed`；可 cancel/expire，所有 upload ID 操作仍鉴权并绑定 workspace/path。search 返回 search_id、files 的 version、match_id/file/line/column/preview；replace 先生成 preview/diff，再 apply(search_id, selected_match_ids, expected_versions)。Git 仅 branch/status/working diff/staged diff/log/commit detail。

## 3. 契约
### 上传与下载
以 [配置](security-config.md) 的 threshold/chunk_size/max_file_size 为准。临时状态/bitmap/hash 与不完整数据在服务私有目录，0700/0600；记录持久状态以支持 Web restart 后查询已收到 chunks。固定 chunk index/offset、检查总大小/最后 chunk 长度、重复相同 chunk 幂等、不同内容拒绝。chunk hash 和整文件 SHA-256 由后端验证，不信任客户端声明。complete 有界流式读校验，目标目录同 filesystem staging，安全 atomic rename 后才标记 committed；跨 filesystem 不直接 rename，改用目标同目录暂存。complete 重试返回同一结果；DB 与 filesystem 间崩溃必须可恢复辨认，不把 partial 目标当完成。
磁盘空间/并发/累计 quota/闲置 TTL 在上传任务锁定；不能只有限单文件 20GB 导致磁盘耗尽。TTL 回收仅自身 staging。小文件同样先验证再发布。W04 目录上传通过逐项受验证的 `create_directory` 操作与文件上传实现；拖拽目录遍历可显式建立空目录，文件选择器的 `webkitRelativePath` 不保证暴露空目录。每个相对路径由后端验证，不信任浏览器提供的路径。
W04 上传冲突仅提供 skip/replace，不提供 keep_both。replace 使用弹窗初次确认时捕获的目标 `expected_version`，发布前复验；确认后再变更返回 409 并重新提示确认。批量上传不变成无条件覆盖开关，具体签名见 [W04 契约](workspace-files-contract.md)。
文件下载流式、nosniff、attachment 文件名安全编码 Unicode；ZIP archive entry 为相对 path、无 ../、保留空目录、大文件流式、有界内存。symlink 默认跳过并报告（不 follow，不导出根外目标）；不得递归打开 device/FIFO。断开取消流并释放 fd；目录压缩不是磁盘一致快照，变动错误不能输出伪造成功。SVG 默认 source；safe preview 用独立 sandbox 无脚本/同源权限、禁外部资源，不以内联 HTML 信任其内容。PDF 只读隔离；binary metadata/有界 hex/download，不能 text/replace。

### 搜索与替换
通过 [命令工具访问边界](process-guidelines.md) 限定 rg 的真实读取，不能仅过滤结果路径。候选内容交付前由安全 snapshot 重算 preview/hash；使用 rg JSON 流，不解析普通冒号分隔文本；args 固定 `--json --hidden --glob !.git/**` 加受限 flags，不启用 --no-ignore、--text 或 --follow；遵守 .gitignore、Git ignore、.ignore，不误忽略 .github/.env.example。cwd 为验证 workspace；ignore 二进制；include/exclude glob 长度/数量有界，不能取消强制 .git 排除。结果和运行时间有界且显示 truncated。
case_sensitive/whole_word/regex 明确映射，literal 与 regex 分开；错误 regex 返回 invalid_pattern，rg code 1 为空结果，其他失败不可当空结果。line 为 1-based；rg 的 byte offset 不能直接当 Monaco UTF-16 column，shared DTO column 固定 1-based UTF-16，使用正确文本转换并测试中文/emoji。
Search 对候选内容形成稳定快照/hash；无法将 matches 与同一版本对齐时重查/报 conflict。Replace 必须 Search -> Preview -> Diff -> 选择 -> Apply；每文件 apply 前复验版本，冲突文件零修改，返回每文件 applied/conflict/error，其他已成功文件不可伪称全事务回滚。替换文本/regex capture 的规则与 preview/apply 共用函数，UTF-8/换行保持一致。不以 rg 输出 offset 无版本直接写文件。

### Git
系统 git 独立 args，禁 shell 字符串；按 [工具访问边界](process-guidelines.md) 限定读取范围，不能只校验 cwd。status 用 porcelain v1 -z 等机器格式，diff 使用 no-ext-diff/no-textconv/no-pager，log/commit 用明确格式与有界 pagination；处理 rename/空格/Unicode/untracked/deleted/staged。工作区内 .git 文件指向 worktree 外部元数据是合法候选但须按安全 task 明确批准 metadata 访问边界，不把 allowed roots 当 shell sandbox。禁任意 git option/revision 透传，revision 解析为已验证 commit ID，`--end-of-options` 能力依锁定版本验证。UI 不提供 commit/push/pull/checkout/reset/rebase/discard。二进制 diff 只 metadata/download。

## 4. 验证与错误矩阵
chunk 错 hash/非法 offset：400；目标版本变更：409；超大小/磁盘 quota：413 或明确 insufficient_storage；失效 upload/search snapshot：410 expired；tool 不可用：503；非法 regex：400 invalid_pattern。错误 code 在 owning task 与 HTTP 表对齐。

## 5. 优 / 基础 / 错误用例
优：上传网络断开后只补缺 chunks，最终 hash 等于原始文件；Search 后 Terminal 修改一个结果，该文件 replace conflict。基础：Unicode 空文件夹 ZIP。错误：逐 chunk 直接写最终目标、Search 后直接无 preview 修改全部、忽略 git/rg 非正常退出。

## 6. 必需测试
上传：单/多/文件夹/拖拽空目录/Unicode/大文件/hash/skip 与 replace/中断恢复/Web restart/重复 chunk/错误 hash/超额/确认后再变更。未实现 keep_both 的测试不作为 W04 门禁。
下载：真实 unzip 验路径/内容/hash/空目录/symlink/大文件流式内存界限。
搜索：src、公有目录/.github/.env.example 可搜；node_modules/dist/.next 按 ignore 排除、.git 强排除、binary/symlink 不搜；emoji 定位；preview 与 apply 同输出；外部改动 conflict。
Git：临时真实 repo 与 CLI 对照 clean/modified/staged/untracked/deleted/branch/log/diff/rename，恶意 path/option/config 不执行外部命令。

## 7. 错误与正确
错误：shell 拼 `rg <query> <path>` 或上传成功仅信 Content-Length。正确：独立 args + parser/limits + 安全路径 + 后端整文件 hash + 原子发布。
