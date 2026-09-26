# 配置、认证与安全边界

## 1. 范围 / 触发
单密码用户拥有服务 UID 的 shell 权限，不实现 username/register/OAuth/RBAC。allowed_roots 限制 File API，不能限制 tmux shell 的 cd、命令、git hooks 或本机用户权限；不得把 Web IDE 宣传为沙箱。

## 2. 签名
CLI：`persistty serve --config /etc/persistty/config.yaml`；`persistty password` 从 TTY 隐藏输入、确认后仅输出 Argon2id PHC hash，不接受密码命令行参数。配置错误在 listen 前失败。
HTTP 登录 `POST /api/v1/auth/login` body `{"password":"..."}`；登出 `POST /api/v1/auth/logout`；状态 `GET /api/v1/auth/session`。成功只返回 session 状态和 CSRF token；session secret 只在 Cookie。

## 3. 契约
初始配置键与默认值（尺寸单位明确定义为 MiB/GiB，解析支持示例中的 MB/GB 并按二进制倍数处理）：

```yaml
server:
  listen: 127.0.0.1:8080
  public_origin: https://ide.example.com
  development: false
auth:
  password_hash: '$argon2id$...'
  session_ttl: 24h
workspace:
  allowed_roots: [/home/user/projects]
  default_root: /home/user/projects
terminal:
  history_lines: 50000
  restore_lines: 10000
files:
  max_edit_size: 10MB
  max_preview_size: 50MB
upload:
  max_file_size: 20GB
  chunk_threshold: 32MB
  chunk_size: 8MB
```

这里 example.com/PHC 示例是字段形状，不是可直接运行的配置。部署任务提供有效安装示例。拒绝未知键、空 roots、无效 hash/origin、负大小、阈值矛盾，default_root 必须真实存在且位于 allowed roots。密码/根目录无不安全 fallback。生产仅支持 HTTPS public origin；本地 HTTP 仅显式 development，不能自动降低 Cookie 安全。
Argon2id 基线 m=65536 KiB、t=3、p=1、随机 16-byte salt、32-byte key；密码 CLI benchmark 目标 Debian 后可提高，验证 hash 参数有上下界以防 DoS。比较用恒定时间，支持版本校验，密码输入有合理字节上限并在 task 中固化。
Cookie `__Host-persistty_session`：Secure、HttpOnly、SameSite=Strict、Path=/，无 Domain。32-byte CSPRNG session secret，每次登录轮换；DB 仅保存 token hash、expiry、CSRF 绑定信息。绝对 TTL 到期/登出撤销；session fixation、登录并发、过期可测试。登出只撤销认证/attach，不 kill Terminal。
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
| config 无 hash/越界 default root | 启动失败 |

## 5. 优 / 基础 / 错误用例
优：重登录新 secret，旧 secret 失效，Terminal 仍运行。基础：密码校验/到期。错误：只隐藏前端 UI、允许任意 Origin、将 password 写入 SQLite 明文。

## 6. 必需测试
配置解析/单位/非法值、Argon PHC malformed/边界/校验、Cookie flags、固定 session 攻击、登出与过期、完整受保护路由矩阵、CSRF/Origin/XFF 伪造、并发限流、日志秘密扫描。HTTP session 状态匿名行为归 [HTTP](http-api.md)。

## 7. 错误与正确
错误：`CheckOrigin = true` 或 Cookie Secure 由任意 header 控制。正确：public_origin 严格匹配 + 显式可信代理 + 生产固定安全属性。
