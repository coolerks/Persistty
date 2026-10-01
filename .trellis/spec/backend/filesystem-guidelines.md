# 文件路径、版本与原子保存

## 1. 范围 / 触发
所有 File/preview/upload/download/search/Git cwd/Terminal initial cwd 输入都不可信。W04 文件 API 接受 `project_id + folder_id + project_version +` 该 folder 下相对 POSIX path；每次检查当前关联、根身份与权限，不能接受客户端绝对文件目标。目录选择器的服务器绝对路径是独立入口，只用于验证并登记项目文件夹。已实现签名见 [W04 契约](workspace-files-contract.md)；shell 自身权限边界见 [认证](security-config.md)。

## 2. 签名
初始接口形状：`Open(ctx, workspaceID, path) (FileSnapshot, error)`、`Save(ctx, workspaceID, path, content, expectedVersion) (FileVersion, error)`。
`FileVersion={mtime:string,size:number,etag:string}`；mtime 为 UTC RFC3339Nano，size 为实际字节，etag 为 `sha256:<hex>` 强内容 hash。用于并发校验的 expected_version 必须三字段齐全；不依赖时间戳精度。创建文件用明确 create API 和不存在前置条件，不能把缺失版本当任意覆盖。

## 3. 契约
### 路径访问
配置 roots 用 Abs/Clean/EvalSymlinks 固定真实目录，逐个打开 root handle。请求拒绝 NUL、绝对路径、任何 `..` 段、非法 URL 编码；仅 decode 一次，路径不能经过多层 decode；`.` 仅用于目录根。含空格、Unicode、前导 '-' 的合法文件名保留。多 allowed roots 的 workspace 只能访问自己归属的 root。
词法检查采用 filepath.Rel 验证边界，不用 strings.HasPrefix。这些检查只用于输入和诊断，最后 I/O 必须通过 Debian 根句柄限定的 API（锁定 Go 版本后的 os.Root，或封装 openat2/dirfd）。禁止 EvalSymlinks 后普通 os.Open/WriteFile 作为安全方案。
安全父目录句柄覆盖创建、rename、remove、copy 和临时文件操作；禁止 fallback 到绝对路径 I/O。普通链接只允许读取最终目标仍在所属 root 内；写入先解析安全目标，不把“保存链接”误变成替换链接本身。目录树/ZIP 不 follow symlink；删除链接删除链接条目，不递归目标。拒绝 socket/FIFO/device 文件，避免阻塞和泄露。git/rg 进程不能仅靠 Dir 防路径竞态，事后复验不能撤销工具已发生的根外读取。工具的内容读取必须采用 [命令规范](process-guidelines.md) 中受限执行或安全输入策略；交付前还要安全根下复验和形成 snapshot，不能直接返回未验证 CLI preview。

### Go 根句柄实测边界
Go 1.26.8 在 Debian 13.4 的 [独立探针](../../../tests/integration/debian/files/README.md) 已验证根内相对链接、越界拒绝及有限 parent swap；仅是底层能力证据，不是 File API 验收。`os.Root` 绑定打开目录的身份：通过 `OpenRoot` 打开的内部目录被外部进程移出原树后，旧句柄仍可访问其内容。不能将句柄存活、Abs/EvalSymlinks 或一次检查当成请求时当前项目成员证明。W04 必须在注册/config version/根身份及提交裁决中明确移动后的处理，不以普通绝对路径 fallback 修补。

`os.Root` 不禁止跨挂载点、设备/FIFO 或 Linux `/proc` 魔术文件；须按产品契约单独验证与拒绝。W02 后续已有真实 [Landlock CLI 临时树实验](../../tasks/archive/2026-09/09-27-debian-spike/cli-report.md)，但动态工具的系统路径许可仍不能证明仅批准项目根可读，正式 CLI adapter 门禁未通过。有限竞态测试未发现越界，不能宣传任意竞争下的完整证明。

### 冲突和保存
Open 从同一打开句柄读取、hash、stat，若读取中发生变化就有界重试或报 conflict；不要把不同时间读取的 metadata/hash 拼成快照。
Save 在应用内按目标文件序列化；安全打开当前目标比较版本，不匹配 409 且零写入。创建同目录随机 O_EXCL 临时文件，写完整数据、恢复原始普通权限位（保留 executable、不要继承 setuid/setgid）、fsync/close。提交前再次校验目标 identity/版本，再用同一安全父目录原子 rename，fsync 父目录并返回新版本。任何失败清理自己的 temp；不能先 truncate 目标。上传提交/replace 复用此安全操作边界。
Overwrite 是用户查看当前版本后明确确认、携带当前 expected_version 的新保存；不是 force=true 永久绕过校验。再次外部修改仍 409。符号链接/父目录变动/目标被删除也需重新判断并拒绝不安全操作。
原子 rename 防半文件，不是任意外部 writer 的原子 compare-and-swap。Terminal/AI 不遵守应用锁；最终校验和 rename 之间仍可能竞态。必须在文件任务设计中说明不可消除的窗口，不宣传绝对并发保证；测试已发生外部修改必定 409，覆盖持续竞争时的限制并保留证据。若 task 无法满足硬性验收，应调整协议或设计并提请 review，不能静默降级。

### Explorer 变更操作的任务门禁

新建文件/目录、重命名、移动、复制、创建副本和删除均复用安全根/父目录句柄。源与目标分别验证，禁止仅验证源路径；目录操作不得进入自身子目录，链接操作按前述规则，不递归根外目标。目标已存在时默认拒绝且原目标不变，覆盖必须是独立明确操作并携带当前版本；不能把复制/移动隐式变为覆盖。剪切属于客户端待移动意图，直到后端成功移动前不能删源。

Explorer 任务在实施前于 [HTTP 契约](http-api.md) 固化各端点签名、源/目标字段、文件与目录前置条件、删除确认、多文件部分失败及跨文件系统移动语义。目录变更不能套用正文 hash 声称整个目录具有原子版本；无可靠目录一致性保证时必须说明边界。大目录复制/删除有取消、深度、数量和执行时间限制，失败不伪称全成功。

测试源/目标分别越界、目标已存在零覆盖、目录移入自身、剪切失败保留源、symlink 删除不删目标、跨文件系统失败和批量部分失败结果；前端对应操作清单见 [组件规范](../frontend/component-guidelines.md)。

## 4. 验证与错误矩阵
| 输入/状态 | 结果 |
| --- | --- |
| ../、绝对路径、根外/魔术链接 | 403 或 invalid_request，不做越界 I/O |
| 安全根内链接读取 | 返回真实目标快照 |
| 文件不存在 | 404；不创建 |
| expected_version 缺失 | 428 |
| hash/mtime/size/identity 改变 | 409，磁盘当前内容保留 |
| binary 或超编辑限制 | 415 unsupported_media_type / 413 |
| temp 写入/同步/rename 失败 | 非成功响应，原文件完整 |

## 5. 优 / 基础 / 错误用例
优：Monaco 读旧版本，Terminal echo new，保存旧内容返回 409 且 new 留存。基础：保存 0755 脚本后权限仍 0755。错误：只比较 mtime、只做字符串前缀验证、失败重试改成无条件 overwrite。

## 6. 必需测试
路径 traversal/双编码/相似前缀、根内/根外/断链/循环 symlink、新目标父目录 symlink、在操作间切换链接/移动父目录、非法 special files；文件删除/相同大小同 mtime 内容变化/读中写入/应用并发保存/故障注入/可执行位；atomic rename 后返回 hash 一致。真实隔离 filesystem 集成不能用 mock 代替。

## 7. 错误与正确
错误：`EvalSymlinks -> HasPrefix -> os.WriteFile`。正确：词法校验 + root/安全父目录 handle + 版本复验 + 同目录原子提交。

W06 的 SnapshotEntries/CopySnapshot/WalkSnapshot 使用 no-follow 父目录/叶句柄，读取后复验根、父与文件身份；非 Linux 的安全读取不得在检查后用绝对路径重新 os.Open。CLI 只接受私有快照，详见 [W06 契约](search-git-contract.md)。
Darwin 目录项元数据同样用 `fstatat(dirfd, name, AT_SYMLINK_NOFOLLOW)`，不能依赖 `dir.Name()` 重新 `Lstat`；Linux snapshot 叶文件继续使用 openat2 的 NO_XDEV/BENEATH/NO_SYMLINKS，与原安全目录契约一致。

## 批量安全快照与 Darwin 目录打开（2026-10-01）

`WalkSnapshotWithCopy` 在当前访问回调提供限生命期 SnapshotCopy，复用已枚举的 no-follow 父句柄。叶文件强内容版本、两次内容一致性、普通模式与叶identity复验保持；回调/子目录结束后重新打开目录，复验当前注册根/目录身份和目录项快照。不能保存copy回调供异步或遍历返回后使用；取消、深度32、全局entries和单目录10000上限仍生效。`WalkSnapshot` 复用该walker并同样在遍历后复验。

Darwin `openMutationParent` 通过逐组件 `openat(O_DIRECTORY|O_NOFOLLOW)`，在打开前后核对注册root device/inode；每个后续操作仍复验parent/leaf。避免每个深路径前缀重复调用os.Root.Lstat导致二次遍历。普通可跟随根内链接的openConstrained独立保持；Linux继续openat2的BENEATH/NO_XDEV/NO_SYMLINKS。两平台编译和本机竞态/链接哨兵回归不能代替真实Debian验收。
