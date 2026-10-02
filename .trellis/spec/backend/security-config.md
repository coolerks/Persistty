# 配置、认证与安全边界

## 1. 范围 / 触发
单密码用户拥有服务 UID 的 shell 权限，不实现 username/register/OAuth/RBAC。用户注册的项目/folder身份限制File API，不能限制tmux shell的cd、命令、git hooks或本机用户权限；不得把Web IDE宣传为沙箱。用户已批准多文件夹项目，废止固定allowed_roots/default_root配置。

## 2. 签名
CLI：`persistty serve --config /etc/persistty/config.yaml`；`persistty password` 从 TTY 隐藏输入、确认后仅输出 Argon2id PHC hash，不接受密码命令行参数。配置错误在 listen 前失败。
HTTP 登录 `POST /api/v1/auth/login` body `{"password":"..."}`；登出 `POST /api/v1/auth/logout`；状态 `GET /api/v1/auth/session`。成功只返回 session 状态和 CSRF token；session secret 只在 Cookie。

## 3. 契约
下方为 W01 基础配置片段；完整可解析字段以 [配置模板](../../../deploy/config.example.yaml)为准。W04 传输字段已实现，W03 终端字段为 `socket_path/tmux_binary/shell/history_lines/restore_lines/history_bytes/termination_seconds`：默认 `/usr/bin/tmux`、服务 UID 的登录 shell（检测失败回退 `/bin/sh`）、5000 行、5000 行、8 MiB、10 秒；socket 默认在数据库父目录。history 为 100..50000 行，restore 为 100..history，history_bytes 为 1..64 MiB，倒计时为 1..120 秒，路径必须规范绝对且不能互相/与存储重合。启动校验配置路径形状，终端操作时验证依赖/server 可用；缺 server 不从 Web 隐式启动，见 [W03 契约](terminal-runtime-contract.md)。其余已批准默认值为文本8MiB、预览16MiB、单文件上传256MiB、批次1GiB、暂存2GiB。

```yaml
server:
  listen: 127.0.0.1:8080
  public_origin: http://10.66.66.1
  mode: vpn_http
  trusted_proxies: [127.0.0.1]
auth:
  password_hash: '$argon2id$...'
  session_ttl: 168h
storage:
  path: /var/lib/persistty/metadata.db
```

PHC须由password CLI实际生成，10.66.66.1必须替换成真实VPN地址。旧workspace字段及旧默认数值已废止。W01拒绝未知/重复键、多个YAML文档、配置超过64KiB、无效hash/origin、非规范绝对DB路径、非owner或group/other可读配置；TTL为1分钟到30天，默认7天。不隐式建HOME项目。三种模式的Go监听均限loopback；tls要求HTTPS public_origin，development还要求loopback HTTP origin，vpn_http是批准的WireGuard内HTTP生产特例，由Nginx绑定VPN地址，不自动降低模式。VPN隔离是部署实测前提，不以字段证明公网隔离通过。服务拒绝root运行，trusted_proxies仅明确loopback IP。
Argon2id 基线 m=65536 KiB、t=3、p=1、随机 16-byte salt、32-byte key；密码 CLI benchmark 目标 Debian 后可提高，验证 hash 参数有上下界以防 DoS。比较用恒定时间，支持版本校验，密码输入有合理字节上限并在 task 中固化。
tls Cookie为`__Host-persistty_session`（Secure）；显式vpn_http/development为`persistty_session`（不可伪造Secure属性）。均HttpOnly、SameSite=Strict、Path=/、无Domain；32-byte CSPRNG secret，每次登录生成独立会话，其他设备不失效。DB只保存token hash、expiry与CSRF绑定信息。绝对TTL到期/登出仅撤销对应认证/attach，不kill Terminal；应用密码变更撤销全部登录会话而不杀任务。
生产同源、禁宽松 CORS。所有写请求（含登录）检查可信 public_origin；已登录写请求另带 `X-CSRF-Token` 绑定 session。WS 验 session 与严格 Origin scheme/host/port；缺失/null Origin 拒绝，开发 origin 显式白名单。WS URL 不放 token；session 过期/登出需撤销存量连接。只信任明确配置的 loopback Nginx proxy，不能根据任意 X-Forwarded-* 判断真实来源或 HTTPS。
登录速率限制按来源 + 全局、包括失败/成功路径，避免仅按伪造 XFF；Argon2 并发有界。启动配置/DB权限至少 owner-only，日志遵循 [脱敏](logging-guidelines.md)。

## 4. 验证与错误矩阵
| 条件 | 结果 |
| --- | --- |
| 错误密码 | 401，统一消息 |
| 多次尝试/并发超限 | 429，不分配无界 hash worker |
| 无 session 的 API/WS/download/preview | 401，不接触资源 |
| 错误 Origin / CSRF | 403，无副作用 |
| TLS origin 但 Cookie 被降级 | 启动/部署检查失败 |
| config 无 hash/非法模式/配置权限不安全 | 启动失败 |

## 5. 优 / 基础 / 错误用例
优：登录生成独立secret，其他设备会话仍有效；退出只撤销当前会话，Terminal仍运行。基础：密码校验/到期。错误：只隐藏前端UI、允许任意Origin、将password写入SQLite明文。

## 6. 必需测试
配置解析/单位/非法值、Argon PHC malformed/边界/校验、Cookie flags、固定 session 攻击、登出与过期、完整受保护路由矩阵、CSRF/Origin/XFF 伪造、并发限流、日志秘密扫描。HTTP session 状态匿名行为归 [HTTP](http-api.md)。

## 7. 错误与正确
错误：`CheckOrigin = true` 或 Cookie Secure 由任意 header 控制。正确：public_origin 严格匹配 + 显式可信代理 + 生产固定安全属性。

W06 新增 `search`/`git` 固定程序路径及有界配置，默认值和范围见 [搜索/Git 契约](search-git-contract.md)。`exclude_directories: []` 可清空搜索依赖排除，不解除 `.git` 强排除。

W07 默认关闭的 elevation 配置、独立 broker/socket/PAM 边界及系统安装门禁见 [单文件提权契约](elevation-contract.md)。Web NNP 与非 root 边界保持。
