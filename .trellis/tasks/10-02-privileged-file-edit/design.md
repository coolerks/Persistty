# W07 单文件提权保存设计

状态：用户已批准实施，产品 API 与源码已接入，服务尚未安装。实际签名与边界以 [W07 owner 契约](../../spec/backend/elevation-contract.md) 为准。对应 [PRD](prd.md) E01～E09；证据见 [研究](research.md)。

## 1. 实际变更边界

规划时的缺口是权限失败后没有一次性保存入口，且现有Save不能代表高权限元数据保持。实现归属：files负责安全打开/版本/原子提交；新增elevation负责请求状态与Unix协议；storage负责元数据及会话/项目发布裁决；httpapi负责校验与映射；editor-session负责冻结输入/结果合并。不能在UI、通用HTTP403或Save里加force开关。

新增 `cmd/persistty-elevatord` 非rootbroker和 `cmd/persistty-file-helper` Linux root短命执行器；`internal/elevation` 只处理固定协议。不把Web主程序作为sudo白名单命令，不给broker任意命令执行接口，不增加CGO/PAM库。仅使用实际manifest已有Go/x/sys与前端依赖。

## 2. 权限、配置与部署

Web新配置 `elevation.enabled` 默认为false、`socket_path`为规范绝对路径。开启后只连接Unix socket；关闭/非Linux/依赖不可用给明确状态，不影响普通文件/终端。broker与helper各自读取root-owned、禁止symlink/非root可写祖先/文件的严格配置；Web用户不能改权威允许列表。

broker以现有开发UID运行，独立systemd socket/service，Unix socket只允许该UID访问，并用SO_PEERCRED核对。它的NoNewPrivileges必须为false以运行sudo；ProtectSystem=strict与ReadWritePaths仅给精确允许目标父目录及root账本目录，禁止整个文件系统放开。Web unit保留NoNewPrivileges=true。broker不在tmux cgroup内，停止/升级不得操作tmux。

helper安装路径固定为 `/usr/local/libexec/persistty-file-helper`，不带自由参数，从FD3读取协议、FD4读取正文，stdin仅用于sudo密码且helper不读取。调用固定 `/usr/bin/sudo -k -S -p '' -u root -C 5 -- /usr/local/libexec/persistty-file-helper`。设置最小env，禁-E/SETENV，broker不继承用户sudo/Git/editor变量。root-owned命令级sudoers限定该helper及无参数、PASSWD/NOSETENV、closefrom_override、timestamp_timeout=0、passwd_tries=1；只针对本命令关闭输入/输出正文记录。模板用虚构developer，不安装或推断现有sudo策略。

root策略含 schema、调用 UID/GID、账本路径、protected_paths、精确 targets（target_id/target_path）；8MiB/60秒为代码固定上限，不由策略扩大。无目录通配。policy/broker/helper/账本/认证与sudo策略自身不得作为可修改目标，启动校验拒绝自覆盖。target必须已有、普通文件且nlink=1；祖先/root/parent按no-follow句柄复验，跨挂载沿用已有NO_XDEV边界。目标正文须当前UID可读取；提权不可读文件是范围外。

口令仅在弹窗短时字段→HTTPS或已批准VPN内HTTP→Web→Unix私有socket→broker→sudo匿名stdin pipe传递，不在控制FD/正文FD/argv/env中。关闭请求body记录/落盘；helper和broker禁core dump，stderr有界且不透传。系统认证多轮/OTP/策略不可用统一给可恢复失败，不伪报特定密码错误。

## 3. 用户流程与HTTP契约

路径相对于 `/api/v1`，都经session；POST/DELETE还验证Origin/CSRF。严格拒绝未知/重复JSON键，共用 `tests/contracts/elevation.json` fixture。ID服务器随机生成，无客户端绝对路径或force字段。

1. 普通PUT仅在真实EACCES/EPERM保存失败时返回稳定 `permission_denied`（403）；其他forbidden保持原义。前端只对此错误提供显式提权入口，其他UI仍按错误处理。服务不能仅用403推断目标有资格。
2. `POST /projects/:id/folders/:folderId/elevation-requests`：`{project_version,path,expected_version,content}`。调用普通安全读取与root-owned策略查询验证目标，服务hash原始UTF-8正文，生成nonce/TTL。响应 `{id,target_id,target_path,content_hash,expires_at,state:"prepared"}`；绝对路径只在已认证允许目标的确认界面显示，不加入通用错误。正文仍留前端捕获快照，不保存到DB。
3. `POST /elevation-requests/:requestId/execute`：`{password,content}`。服务复算hash与冻结请求比对，原子prepared→executing后才调用broker。password最大1024字节、拒绝NUL/CR/LF，JSON body沿用 `6*(8MiB)+8192`；正文有效UTF-8且无NUL。系统失败需要新请求和重新输入密码。成功/失败返回同一状态DTO。
4. `GET /elevation-requests/:requestId`：`{id,state,code,version}`。`state=prepared|executing|applied|rejected|cancelled|expired|indeterminate`；version仅applied非null，code仅失败非null，其余null。原session绑定，匿名401，其他session404，无密码/正文/原口令回显。
5. `DELETE /elevation-requests/:requestId`：接受前取消；prepared原子cancelled零执行；executing发取消/断开commit通道；已进入committing则只返回实际/待定状态。重复取消不触发再执行。

通用错误400 invalid_request、401 unauthenticated、403 forbidden/permission_denied、404 not_found、409 conflict/request_consumed、410 expired、413 too_large、428 version_required、429 rate_limited、503 elevation_unavailable。执行中系统认证/策略失败以rejected + authorization_failed脱敏，不能把sudo stderr暴露或用系统认证401触发应用退出。支持出错后只查状态；execute无自动重试。

创建/执行限流按session+全局，broker最多2个在途且无无界队列。请求prepare TTL60秒，执行30秒，上游仅该execute路由HTTP预算35秒；其他工具预算保持。Unix控制JSON每帧≤64KiB、正文≤8MiB、stdout协议≤64KiB、stderr≤64KiB但只保留类别，不落正文日志。任何端超时都关闭自身FD并Wait；roothelper有独立deadline，不能仅依赖非root方kill到root子进程。

## 4. 一次授权、提交裁决与崩溃

Web持久请求记录绑定token hash、project/folder/version、root dev/inode、相对path、target_id、expected Version、content_hash、issued_at/expires_at、state/code/result Version；不保存token/password/content。新增连续SQL迁移，无长期DB事务等待进程。

execute CAS先持久消费一次尝试；包括错误密码、正文不一致执行尝试、取消/超时都不能恢复prepared。服务器重启prepared可过期、executing先indeterminate并核对账本；绝不恢复执行队列或重放密码。roothelper验证sudo提供的调用UID、root配置和全部目标参数，在有界root-owned持久账本中独占消费nonce，fsync后才准备写入；另有per-target锁防两个helper同时发布。账本只保存绑定摘要、expires、阶段及结果Version，不存正文、原token或密码。达到账本配额拒绝新请求；只在原nonce严格过期后安全回收，持久时间高水位拒绝时钟回退后的不安全重放。

两阶段执行：

1. Web消费请求→broker执行sudo认证。此阶段不持项目锁；配置更新/注销照常。
2. helper消费nonce并验证policy/文件/正文，准备同目录私有temp、保留元数据并fsync，回READY；此时尚未rename。
3. Web收到READY后在短发布裁决中复验session未撤销/到期、配置版本、root身份及允许目标；通过才向FD3发送COMMIT。发布、注销和项目变更采用统一锁顺序；现有DeleteSession要参与该狭窄裁决，防入口查验后注销穿透。提交等待默认≤5秒，过期未接受则取消，不能让PAM持有projectMu。
4. helper再验deadline、父/叶身份、Version/安全元数据，再rename+dir fsync，记录result并回APPLIED。COMMIT前断连/取消零rename；COMMIT被接受后按实际落盘记录，不承诺网络取消回滚。

root账本状态由broker以只读方式查询：helper写入root-owned记录，目录/记录只给服务组只读，broker核对owner/mode/nonce/绑定hash后返回必要元数据，Web再次检查请求所属session。路径按固定账本及受限nonce构造，不能请求任意文件。若rename已发生而dir fsync/账本更新或结果传输失败，状态indeterminate，禁止重新执行；当前UID普通复读用于人工确认，不从hash相同推断一定由本请求保存。

## 5. 文件安全与元数据

roothelper把target_id映射到root策略精确文件，验证与Web注册root/相对path恰好对应；客户端提供的root信息不能扩大允许列表。policy加载/目标打开均使用句柄访问而非检查后绝对路径重开。安全打开/Version/原子提交内核留在files，新增窄的高权限保存入口由helper调用；所有者/模式/xattr策略在该入口明确，普通Save行为由既有回归保护。

读取目标强Version（identity/mtime/size/SHA256），同时捕获UID/GID/mode与xattr/ACL/安全标签摘要；设置有界属性总量64KiB。拒绝特殊权限位、不支持的安全属性或无法完整读取/复制的元数据。temp初始0600，写原始正文→fchown→恢复mode及ACL/xattr/SELinux标签→复验元数据→fsync；提交前还复验原安全元数据未变。复制失败则删除自己的temp，保留原目标。不能把root新临时文件默认owner当作原owner，也不能静默丢ACL。

父目录重新打开/identity一致、leaf no-follow/nlink/type及完整Version复验后renameat；只清理自己的随机temp。提交后返回新identity/Version，处理同文件多组/重叠根。受限文件系统、不支持openat2或安全标签保持失败时给不可用，不弱化访问路径。

## 6. 前端保存与凭据生命周期

权限失败在FileEditor既有失败区域显示“提权保存”；点击先冻结content/generation/expected/scope epoch，并暂停该buffer普通autosave，创建请求确认目标。Dialog用既有shadcn Dialog/Input/Field/Button/Collapsible，默认取消焦点，系统密码input不进入Zustand/IDB/draft，关闭/提交清理字段与引用，不声称所有托管副本擦除。

同一buffer只允许一个普通/提权写入在途，多组共用；编辑期间新输入允许保留但不修改已冻结请求。成功base只对应授权content与新Version，等待draftQueue再按提交generation清草稿，新输入仍dirty且保持paused，需要下一次明确普通保存或新提权；不自动重新授权。失败/取消/过期保存原buffer/model/undo/草稿，网络未知显示查询结果操作，不自动重送execute。配置变化、注销、移动/关闭失去有效scope时取消未接受请求并丢弃迟到响应，保护输入。

## 7. 兼容、发布与回滚

默认disabled，W01/W04/W05/W06及终端协议不增加权限。普通权限失败新增具体code是兼容扩展，测试锁定其他403语义。schema迁移只加实际请求表，保留既有数据；回滚二进制需核对schema版本，不能删表强行降级。

增加broker socket/service、sudoers/root策略、Nginx execute路径内存buffer与proxy_request_buffering配置示例，以及安装/检查/禁用步骤。只写示例不应用；真实启用需指定目标UID、安装路径、允许文件、现有sudo/PAM有效配置、精确systemd修改与隔离验证授权。先禁Web提权新请求→停止broker并核对在途状态→回退helper/策略，不影响tmux或真实文件，不自动恢复系统文件内容。

## 8. 验证边界

本机真实文件/SQLite/HTTP/Unixsocket/合成密码/进程协议、前端行为与Chromium/WebKit可验证产品协调；Linux交叉编译验证构建。真实root所有者/ACL/SELinux、sudo/PAM、NNP隔离、systemd/Nginx及root进程取消须Debian隔离专项，未授权/未执行保留未验收。W05/W06原统一验收仍按本次用户要求延期，不能被W07测试替代。
