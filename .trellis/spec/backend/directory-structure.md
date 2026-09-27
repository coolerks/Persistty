# Go 目录与职责

## 初始目录约定
foundation已按实际需求建立cmd/persistty及config/auth/storage/httpapi；其余目录由对应工作包按需建立，不提前创建空package。

```text
cmd/persistty/       命令入口（serve、password）
internal/config/    配置读取、默认值、验证
internal/httpapi/   Gin router、middleware、DTO、handler
internal/auth/      密码校验、server-side session
internal/terminal/  tmux 元数据协调、attach bridge
internal/workspace/ 工作区和安全文件访问
internal/files/     文件版本、编辑、预览、watcher
internal/search/    rg 搜索、替换预览
internal/transfer/  上传状态、校验、下载
internal/gitview/   只读 Git 数据
internal/storage/   SQLite repositories、migrations/*.sql
web/                React/Vite 应用
tests/integration/  真实进程与文件集成测试
tests/e2e/          Playwright
deploy/             Debian systemd/Nginx 示例
```

## 包与调用方向
Gin handler 只做认证结果读取、decode/validate、调用 service、响应映射。业务类型和错误放在拥有功能的 package；不让 service 接收 *gin.Context。repository 不依赖 HTTP；domain 不依赖 Gin 或 UI。跨包接口定义在使用方，仅针对需要替换的数据库/进程/时钟边界，不给每个函数建立接口。

示例为初始形状：`Create(ctx context.Context, input CreateTerminal) (Terminal, error)`。handler 使用 `c.Request.Context()`；service 协调 tmux adapter 与 metadata repository。禁止 handler 内拼 tmux 命令和 INSERT SQL。避免 utils/common 万能包、全局 DB、import cycle。

## Gin 路由
使用 gin.New() 显式注册恢复/请求 ID/脱敏日志。`/api/v1` 下认证除登录外强制覆盖全部路由；WebSocket upgrade 前执行相同 session 和 Origin 校验。静态资源允许未登录加载；受保护 preview/download 不放 public/static。默认 127.0.0.1；关闭 debug response，明确 body/time/并发限制。路由完整表随 owning task 写入 [HTTP 契约](http-api.md)。

## 测试与审查
同包 *_test.go 表驱动测试；跨边界断言 public 行为。httptest 验证完整 router 的保护矩阵，不能只测试 handler 绕过 middleware。
