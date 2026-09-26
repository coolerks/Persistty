# 错误模型

## 领域与边界
内部错误用 errors.Is/As 与 `%w` 包装；例如 files.ErrConflict、workspace.ErrOutsideRoot。错误携带可用的安全业务 code，HTTP 层统一映射；service 不返回状态码，也不暴露 SQL、绝对服务器路径或 stderr。对客户端 message 用简体中文，code 用稳定英文。

```go
// 形状示例：内部保留因果链，HTTP 只返回脱敏 code/message。
return fmt.Errorf("保存文件版本校验: %w", ErrConflict)
```

| 情况 | HTTP / code | 行为 |
| --- | --- | --- |
| decode/参数非法 | 400 invalid_request | 不执行副作用 |
| session 无效/过期 | 401 unauthenticated | UI 登录、关闭 attach |
| Origin/CSRF/越界 | 403 forbidden | 不升级、不执行 I/O |
| 资源不存在 | 404 not_found | 不自动创建 |
| 文件/替换/上传版本冲突 | 409 conflict | 保留磁盘当前内容 |
| 缺少写入版本 | 428 version_required | 不保存 |
| body/文件超限 | 413 too_large | 清理本次临时资源 |
| 登录/请求限流 | 429 rate_limited | 返回 Retry-After |
| 依赖暂不可用/DB busy | 503 unavailable | 不伪装空列表 |
| 未知内部错误 | 500 internal_error | 一个脱敏错误日志 |

success/error JSON 形状归 [HTTP](http-api.md)。WS 升级后按 [协议](websocket-protocol.md) 发 error，不再写 HTTP JSON。客户端取消不记成服务故障；已开始流式下载后失败只能终止流，不能返回伪造成功或追加 JSON。

## 正反例与测试
错误：catch 所有错误返回 200 或“文件不存在”。正确：区分 ENOENT、EACCES、依赖不可达并映射。测试同一错误的包装后仍可匹配；panic recovery 不泄露栈；409 不改变目标；传输中断不留下完成文件。
