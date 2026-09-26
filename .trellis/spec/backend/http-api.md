# HTTP 与共享概念（跨层权威）

## 1. 范围 / 触发
所有 HTTP API 使用 /api/v1；新端点必须在 owning task 固化完整 request/response/error fixture，并由前后端契约测试共用。以下是初始公共契约，未创建实现。前端不自行定义不同版本。

## 2. 签名
- POST /auth/login；POST /auth/logout；GET /auth/session。
- GET/POST /terminals；POST /terminals/{id}/close；PATCH /terminals/{id} 仅元数据。
- GET /files/content?workspace_id=...&path=...；PUT /files/content body 含 workspace_id/path/content/expected_version。
- GET /terminals/{id}/attach 是 WS upgrade；GET /events 是受认证 watcher WS。
全部路径相对于 /api/v1；其他 Explorer/search/upload/git/settings 端点在对应任务设计中补齐，不自由增加任意执行命令 API。

## 3. 契约
JSON 字段 snake_case、ID 为服务器生成 opaque string、时间 UTC RFC3339Nano；null/缺省含义必须声明。size/chunk offset 限定 JS safe integer，最大 20GB 不跨此界。Unix 文件名可能非 UTF-8，v0.1 API 对无法编码的名字明确 unsupported，不能 lossy 转换后操作错误目标。

```json
{"data":{"content":"hello\n","version":{"mtime":"2026-09-26T00:00:00Z","size":6,"etag":"sha256:<hex>"}},"request_id":"<id>"}
```

```json
{"error":{"code":"conflict","message":"文件已在外部修改。","details":{"current_version":{"mtime":"2026-09-26T00:00:01Z","size":4,"etag":"sha256:<hex>"}}},"request_id":"<id>"}
```

以上 `<hex>/<id>` 仅示例形状。list 用 data.items + 有界 limit/cursor；超过上限不能静默截断却声称全部。成功默认 200、create 201；仅无正文操作使用 204。下载/preview 返回正确 Content-Type/Content-Disposition，不套 JSON。binary 不能进入 text endpoint；etag/hash 以 [文件契约](filesystem-guidelines.md) 为准。
terminal 状态 running/terminated/unavailable 是 tmux 当前观测，connected/disconnected 是浏览器 attach 状态，不能合并。running 包含 alive shell，不保证某个前台程序仍活着。metadata 与 observation 分离。关闭 UI tab 的本地命令叫隐藏/分离，明确销毁才叫 Close Terminal。
匿名 GET /auth/session 返回 401；登录成功/已登录 session endpoint 返回 data.authenticated=true、expires_at、csrf_token。session secret、password 与 password_hash 不在 JSON；CSRF token 为绑定 session 的写入凭据，允许在已认证响应中返回，但禁止记录日志或写入持久 UI store。客户端 auth failure 保留 draft 并停止重连风暴，重登录重新获取 Terminal 列表。

## 4. 验证与错误矩阵
[错误模型](error-handling.md) 为状态码权威；请求拒绝未知字段/重复 JSON key 策略在 decoder 测试锁定，body 上限在 decode 前生效。缺 version 428；被外部修改 409；preview 不支持 415。内部路径/SQL/stderr 不能出现在 details。

## 5. 优 / 基础 / 错误用例
优：409 typed ApiError 驱动 Diff/Reload/Overwrite。基础：空 list 是 items=[]。错误：tmux 查询失败返回空 list、把 filesystem 路径当 DOM HTML、客户端传任意 absolute path。

## 6. 必需测试
Go httptest 与 TS decoder 使用共同 fixture；覆盖 Unicode/空值/边界/无认证/冲突/超限/错误类型/未知字段。每个 endpoint 注册级别检查 auth；下载和 WS 握手也在矩阵内。

## 7. 错误与正确
错误：组件 `await response.json() as File`，忽略状态。正确：统一 api client 检查 status、decode unknown、返回 domain DTO 或 typed ApiError。
