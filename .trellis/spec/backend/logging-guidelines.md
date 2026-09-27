# 日志与审计

使用标准库 log/slog；生产结构化 JSON 到 stdout，由 journalctl 收集。每个请求关联服务生成的 request_id，客户端传入值必须校验长度字符；handler 边界统一记录一次错误。

| 级别 | Persistty 事件 |
| --- | --- |
| debug | 可选的计时/有界 watcher 计数，无文件内容 |
| info | 启动/停止、迁移版本、Terminal create/explicit close |
| warn | 登录限流、非法 Origin、越界拒绝、依赖不可用 |
| error | 非预期数据库/保存/bridge 失败 |

允许字段：event、request_id、status、duration_ms、terminal_id、workspace_id、error_code、字节计数。路径仅在必要时记录受限相对标识；文件名/URL/branch 也可能包含秘密，不直接全量记录。

禁止记录 password、password_hash、Cookie、session token、CSRF token、请求体、键盘 input、Terminal output、完整命令参数/环境变量、URL 查询 token。Gin 默认 logger/recovery 必须确认不输出敏感 header。认证失败只记原因类别，不能写密码。
运维/实验同样禁止记录实际 SSH 用户名、目标 IP、端口及可识别的 HOME/cgroup 身份。只记录环境变量键名和脱敏状态；连接入口及可提交证据遵守 [远端验证规范](remote-validation.md)，不能用“调研记录”绕过保密规则。

形状示例：`logger.WarnContext(ctx, "请求被拒绝", "event", "path_rejected", "error_code", "forbidden")`。错误：记录 `c.Request.Header` 或 websocket payload；正确：仅记录白名单字段。测试使用捕获日志注入假密码/token/input，断言任何级别均不包含；禁止用真实秘密作为测试 fixture。
