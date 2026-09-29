# W04 技术设计审查稿

## 边界与已有实现

W01 的 Gin 认证/CSRF、SQLite 0001/0002、`Project{version,main_folder_id,folders}` DTO、React Router 项目页和严格 decoder 已存在；只能增量扩展，不改已应用迁移 checksum。W02 的 `os.Root` 与 Landlock 只提供隔离实验依据。W04 直接实现项目配置、文件树/普通操作/传输；编辑缓冲区、自动保存与草稿 UI 属于 W05，但 W04 的文件版本/冲突接口必须可供其直接消费。搜索/Git 仍等 W06，绝不把 D09 的宽系统白名单 launcher 用于产品请求。

## 服务端目录选择器与项目配置

目录选择器从服务用户 HOME 开始，提供路径输入、上级导航和仅目录列表；`GET /api/v1/directories?path=<absolute>&cursor=...` 对认证用户按真实服务 UID 做有界只读列举，返回规范路径、parent、可进入目录及 `next_cursor`。不读文件正文，不使用浏览器设备的 File System Access API。根 `/` 可导航，访问失败明确返回 403/404；输入与列表项的最终项目注册都重新验证普通目录、读/遍历权限与真实 `dev/ino`。UI 参考用户截图的路径栏、目录行及确认动作，服务器路径与上传本机路径分别标明。

`POST /api/v1/projects` 接受 `{name,folder_paths:[absolute...],main_index}`；服务端生成不可复用 project/folder ID、规范真实目录路径与根身份，事务保存并返回已有 `Project` DTO。`PATCH /api/v1/projects/:id` 接受 `{expected_version,name,add_paths,remove_folder_ids,main_folder_id?,main_added_index?}`，同一事务完成改名、增删关联与主文件夹替换，若主 ID 指向新增目录则使用 index。每次成功版本 +1；输入为空/重复根/失效目录/零 folder 或无效主 folder 拒绝。`DELETE /api/v1/projects/:id` 带 `expected_version`，仅删项目/关联元数据；已有 terminal FK SET NULL，不调用真实文件/进程操作。`GET /projects` 与详情仍从真实 SQLite 读取。W04 新迁移增加 folder 的 root device/inode（十进制文本）、可选目录可用性观测不写入 DB；注册路径 canon 与 identity 每次请求重新核对，身份变化返回 `folder_unavailable`，不自动改绑同路径新目录。

项目内部配置写与文件副作用采用进程内按 project ID 串行裁决；不在浏览器弹窗、整段上传或 ZIP 构建期间持锁，只在最后接受/发布边界重新比对 config version 与关联。跨进程多实例不在首版部署模型中，若未来多 Web 进程须改成数据库/分布式序列化。数据库事务不包住文件流；真实磁盘与 SQLite 不能原子回滚，结果必须可重查。项目删除后的上传 staging 仅按 TTL 清理，不发布到失效关联。

## 安全文件访问与版本

所有常规文件请求包含 `project_id,folder_id,relative_path,project_version`。输入拒绝绝对路径、NUL、任何 `..` 段、双重 URL decode 与非 UTF-8 可表示名称；读取 DB 当前关联与根 path/dev/ino，重新打开并比较根身份。Linux 正式路径解析使用根 dirfd 约束的 `openat2`（`RESOLVE_BENEATH|RESOLVE_NO_MAGICLINKS|RESOLVE_NO_XDEV`），父目录以安全句柄操作；若目标内核不支持则拒绝文件 API 而非回退裸路径。macOS 单元测试可用 `os.Root` 等效有限夹具，但 Debian 集成负责 Linux 权限/竞态验证。选定的 root 本身可在挂载点，内部跨挂载/魔术文件拒绝。普通最终 symlink 仅在仍位于根内且类型安全时按产品约定读取；写操作对链接/硬链接保守拒绝，不直接截断目标。任何被移出注册位置的根/父句柄，在接受副作用前重新核对其当前项目成员身份；外部恶意并发移动的剩余窗口写入风险如实报告，不能称绝对 CAS。

文件 version 为 UTC mtime、size、原字节 SHA-256 etag 与内部身份 token；请求携带完整 expected version，服务器重开同一安全目标复验身份/强 hash。文本只接受有效 UTF-8（BOM 保留），二进制按内容判别，不按后缀决定；可编辑上限默认 8MiB，超限/二进制仍可下载。`GET /api/v1/projects/:id/folders/:folderId/content?path=` 返回内容、version、媒体判定；`PUT` 带 project_version/expected_version/content。保存先同安全父目录创建 0600 O_EXCL temp，完整写入、恢复普通权限位、fsync/close，发布前复验项目关联与目标版本，使用无覆盖/明确替换语义的 dirfd rename、fsync 父目录，失败只清理自身 temp。服务内按目标身份串行；非协作外部 writer 在最后复验与 rename 之间的窗口不能消除，测试和错误说明保留。

`GET /api/v1/projects/:id/folders/:folderId/entries?path=&cursor=&limit=` 以稳定有界页列举目录，返回 `items`（name/kind/size/mtime/identity/可操作状态）、`next_cursor`、当前 project_version；无损处理 Unicode/空目录，未支持的非 UTF-8 名称明确数量/错误，不静默变成另一个名称。cursor 带目录身份/mtime/位置，变化返回 `rescan_required`；客户端重取首屏。大目录不截断称全部。

## 文件操作与确认

`POST /api/v1/projects/:id/file-operations` 是固定 discriminated union：create_file/create_directory/rename/copy/move/delete，每个携带 project_version、源/目标 folder_id+relative_path、已存在源的 expected version/identity；不接受自由 shell option。项目配置锁内分别验证源/目标关联，安全父句柄操作；目标默认必须不存在。rename/move 同根在 Linux 使用 `renameat2(RENAME_NOREPLACE)`，跨根先受控复制到目标同目录 temp、校验、提交后删除源；任何一步失败返回部分状态而非声称原子。目录复制/删除迭代有深度、项数、时间和取消上限；检测目录移入自身/子孙。删除前先 `POST .../delete-preview` 返回短时、绑定 session/project/folder/path/身份/版本的确认 token 与目标明细；真正 delete 复验确认，目标换代 409，取消时前端不发真正请求。批次返回每项 `applied/conflict/error`，成功项不假装回滚。

## 上传、下载、ZIP

新增私有 staging 配置与上限（初值：单文件 256MiB、批次 1GiB、全局 2GiB、块 4MiB、上传 TTL 24h、ZIP TTL 1h；实际字段和范围在实现时以配置测试锁定）。SQLite 新迁移保存 upload ID、目标项目/关联/config version、期望目标版本、源大小/hash、bitmap/各块 hash、状态/完成结果与过期时间；块数据在 0700 私有目录，0600 普通文件，启动后核对 DB 与文件并恢复或标错误。`POST /uploads` 建立会话，`PUT /uploads/:id/chunks/:index` 以 octet-stream + hash 传输，`GET /uploads/:id` 查缺块，`POST /uploads/:id/complete` 整体 SHA-256 验证后才发布；重复相同块幂等，内容不同冲突。发布前与项目配置串行、根/目标版本复验：同 hash 返回 skipped，不同目标必须携带确认时的 expected version 才 replace，之后变化仍 409。目录上传由浏览器逐文件及显式空目录 manifest 组成，逐项结果呈现；手机只展示文件上传。

原字节下载用 `GET /.../download`，鉴权/no-store/nosniff/安全 Content-Disposition，目标打开后限额流式传送，下载过程变化/中断准确终止。目录 ZIP 使用 `POST /archives` 创建有界后台任务，安全枚举并流式写入本服务私有 temp，保留空目录、不跟 symlink/特殊文件；构建和校验全部成功后原子标 ready，`GET /archives/:id/download` 仅下载 ready 产物。取消、过期、重启清理仅自身资源；跨文件变化检测失败标 failed，不把已发 200 当 ZIP 成功。ZIP 不承诺全树同一时点快照。

## 事件与前端

使用 `fsnotify` 监听当前展开目录（非递归、引用计数），认证 `/api/v1/events` WS 只发送 `project_id/folder_id/revision/rescan` 失效提示；不在事件中信任/回显任意绝对路径。事件缓冲上限、rename/overflow/关联变更置 `rescan=true`，WS 重连总是重新读取项目、展开目录及打开文件版本。若 watcher 不支持某些文件系统，提供有界可见轮询兜底并标状态；不宣称事件必达。[fsnotify 官方说明](https://github.com/fsnotify/fsnotify)确认目录监听非递归且网络/FUSE 类文件系统可能无事件。

前端沿用现有 API client/decoder、React Router 与 shadcn/ui；项目面板增加创建/编辑/移除，工作台显示多根树与工具栏，目录选择器可输入/浏览。Explorer 上下文菜单与键盘可完成全部操作，桌面拖拽只映射 move，手机显示单列菜单；不能将项目配置移除映射为磁盘删除。确认与错误逐项显示，跨项目配置变更或事件到来重查而不清除 W05 将来拥有的编辑输入。W04 可用基础只读文本/二进制信息和下载入口，不提前实现 Monaco/自动保存。

## 发布与风险

迁移 0003+ 只增不改旧版本，数据库升级失败拒绝启动且保留旧数据；文件 API 默认无特权、无任意命令，产品二进制在 Debian 若缺必要内核安全能力则 fail closed。升级前备份需要 SQLite/WAL 一致性，不复制单独 `.db`。失败回滚 Web/DB 兼容版本，不通过 Git 回滚真实用户文件；上传/ZIP 临时资源由明确 ID/TTL 清理。W04 通过不解除 W03、W05、W06、W07 门禁，也不对用户真实目录运行破坏性测试。

## 2026-09-28 前端范围补充

用户在 W04 实施过程中指出技术栈与已给原型不一致。W04 工作台增加原型的活动栏、多根树、左右编辑分组、底部可收起面板和手机单视图；UI 基础控件由 shadcn CLI 按 `base-nova` 生成，布局用 react-resizable-panels，跨视图标签状态用 Zustand。桌面只读内容入口使用按需加载的 Monaco，手机使用普通只读文本框；这是界面骨架，不把 W05 保存、草稿、diff 或 W03 xterm/tmux 运行时提前验收。终端区域必须明确呈现尚未接入，不可用占位按钮伪装可执行命令。
