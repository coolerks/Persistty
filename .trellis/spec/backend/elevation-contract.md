# W07 单文件提权保存契约

## 1. 范围与触发条件

2026-10-02 已接入源码、持久请求和确认 UI，默认关闭；真实 Debian sudo/PAM/root/systemd/Nginx 验收尚未执行，不能写“提权已可用”。普通 PUT 真实 EACCES/EPERM 才映射 `403 permission_denied`，其他 forbidden 不获得新权限。仅已有、普通账户可读取的 UTF-8 文本一次保存；无提权读/新建/删除/重命名/批量操作。安装产物见 [部署审查稿](../../../deploy/elevation/README.md)。

实现 owner 为 `internal/elevation`（协调/进程/协议/策略/账本）、`internal/files/privileged*`（安全文件提交）、`internal/storage/elevation_requests.go`（请求 CAS/发布裁决）、`internal/httpapi/elevation.go`（鉴权与输入）。普通文件 Save 保持原协议；非 Linux 高权限入口不可用。

## 2. 签名（API / 命令 / DB）

所有路径在 `/api/v1` 下并经 session；POST/DELETE 还经 Origin/CSRF。

| 方法与路径 | 请求 / 响应 |
| --- | --- |
| POST `/projects/:id/folders/:folderId/elevation-requests` | `{project_version,path,expected_version,content}` → Prepared |
| POST `/elevation-requests/:requestId/execute` | `{password,content}` → Result |
| GET `/elevation-requests/:requestId` | 无正文 → Result，原会话绑定 |
| DELETE `/elevation-requests/:requestId` | 无正文 → Result，接受前取消 |

`Prepared={id,target_id,target_path,content_hash,expires_at,state:"prepared"}`。`Result={id,state,code,version}`，state 为 prepared/executing/applied/rejected/cancelled/expired/indeterminate；仅 applied 的 version 非 null，仅失败态 code 非 null，其他两字段为 null。Version 仍为 `{mtime,size,etag,identity}`，UTC RFC3339Nano、实际 UTF-8 字节、SHA256、设备/inode；共同 DTO 见 [fixture](../../../tests/contracts/elevation.json)。

命令 `persistty-elevatord` 和 `persistty-file-helper` 不接受操作数。broker 仅从 systemd 的 LISTEN_FDS=1/LISTEN_PID=当前 PID 接收 FD3 socket；helper 仅 root UID/eUID，经 SUDO_UID/GID 检查批准调用者。固定 sudo argv：`/usr/bin/sudo -k -S -p '' -u root -C 5 -- /usr/local/libexec/persistty-file-helper`，不得添加用户 argv/shell/env；stdin 认证、FD3 控制、FD4 原始正文分别使用匿名 pipe。

迁移 `0005_elevation_requests.sql`：id PK、session_hash、payload_json、expires_at、state、code/version_json、created_at。payload 仅 Grant 元数据：id、session_hash、root（RegisteredFolder 的现有 Go 字段名）、path、target_id、expected_version、content_hash、issued_at/expires_at。无正文/密码/raw token；prepared→executing 持久 CAS 在认证之前，任何已执行尝试不恢复 prepared。启动将 executing→indeterminate；不恢复执行队列。

## 3. 请求、配置与提交契约

`elevation.enabled` 默认 false；开启须提供规范绝对 `socket_path`（≤100 字节）。Web 保持非 root/NNP；独立非 root broker 才有命令专用 sudo 例外。socket 父目录 root-owned、不准 group/other 写，无 symlink；socket 0600，通过 SO_PEERCRED 双向核对开发 UID。非 Linux、未配置或依赖缺失明确不可用。

nonce 为 32 随机字节的 64 位小写 hex；session_hash 为现有 HashToken。Grant 绑定项目/文件夹/配置版本、root dev/inode、相对 path、target_id、完整旧 Version、正文 SHA256 和 TTL≤60 秒（不超过登录剩余寿命）。正文≤8MiB、UTF-8、无 NUL；密码 1～1024 字节、UTF-8、无 NUL/CR/LF。HTTP 两个 JSON 接口上限为 `6*8MiB+8192`，拒绝重复/未知字段；Unix 长度前缀 JSON≤64KiB，且拒绝大小写别名字段、重复键、未知字段、过深结构，原始正文独立有界。

prepare/execute 合计每会话 10 次/分钟、全局 60 次/分钟，最多 1024 个活跃限流 bucket。Web 和 broker 各最多两个在途，不设无界执行队列。SQLite 请求总量512/会话64，终态超过24h可清理，executing/indeterminate 保留用于核对。执行预算30秒，只有该 HTTP execute 路由35秒；最终发布≤5秒且不得越过授权到期。regular 文件 hash 也有8MiB+1硬限额与 context；不可中断的内核 I/O 不承诺绝对 wall-clock 终止。

策略固定 `/etc/persistty-elevation/policy.json`，严格 root-owned/no-follow祖先，schema=1、caller_uid>0、caller_gid≥0、ledger_path、protected_paths、1～128精确targets。拒绝重名/重复目标、非规范绝对路径和内建基础设施/账本/protected_paths。自定义 Web/DB/二进制/前端/socket 安装位置必须加入 protected_paths，不能只依赖默认目录。目标父链同样 root-owned、无 group/other write 或 symlink。目标必须已有 regular nlink=1、无 setuid/setgid/sticky；依赖 Linux openat2/NO_XDEV 安全根，不作不安全 fallback。

两阶段：认证不持 projectMu → helper 消费持久 nonce/per-target flock → prepare temp/fsync/READY → Web 在统一 projectMu 复验有效 session 和当前 RegisteredFolder，attempt 锁裁决取消并发送 COMMIT → helper 复验 deadline、最新完整策略与可信父链、旧文件/元数据和 temp Version，再 rename/dir fsync/账本结果。DeleteSession 与配置变更参与同一裁决；先撤销/更新则零发布，COMMIT 已接受后查询实际结果，不承诺取消回滚。

保留 UID/GID/普通模式、user.*、POSIX access ACL、SELinux 标签；属性总量64KiB，其他属性或无法完整复制时拒绝。temp 原始正文 hash 与快照一致，提交前再次验证 temp，Close 仅移除自己的未发布 temp。提交复验与 rename 间仍有非协作 writer 的短竞态，不能宣传绝对 CAS。rename 后 sync/Version/账本/传输失败为 indeterminate；查询只从账本核对，不用正文 hash 相同推断本次一定成功。

root 账本目录 root:开发组0750、记录 root:开发组0640，nonce 原子 exclusive/fsync 消费；每目标 flock，持久时间高水位拒绝时钟回退。最多10000个nonce、枚举上限10200，原nonce到期+1分钟后才可清理。broker 只读精确 nonce/Grant binding；其终态才能收敛 Web executing/indeterminate，prepared/executing 账本响应不能使请求恢复可执行。错误系统口令在 helper 启动之前已由 Web CAS 消费；同 UID 直接访问 broker/DB 属 OS 账户权限边界，不称 shell 沙箱。

密码/正文不进入日志、argv/env、SQLite、nonce记录或sudo stderr响应。JSON临时字节和 pipe slices尽量清理，writer goroutine 必须 join 后调用者才能清理；不承诺托管字符串绝对擦除。helper/broker禁core，有界stderr丢弃；代理日志/临时正文策略与 sudo有效规则必须实机验收。

## 4. 验证与错误矩阵

| 条件 | 响应 / 状态 |
| --- | --- |
| 匿名 / 登录撤销 | 401；提交复验拒绝，系统认证失败不使用应用401 |
| Origin/CSRF/策略拒绝 | 403 forbidden；普通保存权限失败单独 permission_denied |
| 其他会话或未知请求 | 404 not_found |
| 输入/JSON/路径无效 | 400 invalid_request；版本缺失428 |
| 旧项目/文件 Version | 409 conflict，或执行 Result rejected/conflict |
| 重复 execute | 409 request_consumed，无新进程/发布 |
| 新请求过期 / 已消费过期尝试 | 410 expired / Result expired |
| 限流 / 请求存储配额 | 429 rate_limited / 503 elevation_unavailable |
| 未配置、非Linux、socket/依赖不可用 | 503 elevation_unavailable（prepare），execute 已消费时保守未知 |
| sudo认证/入口未成功 | Result rejected/authorization_failed，脱敏，不登出应用 |
| 断线、rename后故障、恢复中的请求 | indeterminate/outcome_unknown，仅查询、禁止重放 |

## 5. 正常 / 基础 / 错误用例

正常：普通保存 permission_denied → 用户确认精确路径和冻结正文 → 输入系统密码 → 一次 applied；执行中的新输入仍 dirty/paused。基础：默认禁用时给不可用原因，输入/草稿继续保留。错误：关闭弹窗就删除正文、用缓存授权自动保存新内容，或用状态查询恢复 prepared。

## 6. 所需测试与当前验证边界

`internal/elevation/*_test.go` 覆盖严格帧/策略/共享fixture、会话绑定、一次消费、并发重复/取消、注销/项目移除裁决、重启与状态不可恢复prepared、独立FD/argv-env/超时与回收、Unix限额/peer拒绝/shutdown join。HTTP测试验证全部保护矩阵、日志合成秘密与应用session保持；前端见 [W07工作台](../frontend/elevation-workbench.md)。

Linux-only `internal/files/privileged_linux_test.go` 覆盖普通文件阶段零写入、元数据/user xattr、精确字节、取消/版本/模式/temp篡改、链接/FIFO拒绝；Mac不能执行，交叉编译不能代替通过。root权限、ACL/SELinux、账本与真实PAM/systemd/Nginx故障矩阵须 [Debian专项](../../../tests/integration/debian/w07/README.md)。检查报告保留所有未执行项，不完成/归档任务。

## 7. 错误与正确示例

错误：`exec.Command("sh", "-c", "sudo ..." + path)`，或把整个Web设root/关闭其NNP。正确：固定sudo参数、root-owned精确策略、独立非rootbroker、单文件helper。

错误：丢失响应后重新 POST execute 或把记录改回prepared。正确：保留捕获输入和request_id，只GET状态；无法核实则普通复读并由用户比较，重新授权必须是新请求。
