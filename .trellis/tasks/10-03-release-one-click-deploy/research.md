# 仓库及官方依据

仓库事实：Go/CLI/config归cmd/persistty/main.go、internal/config/config.go和go.mod；前端版本/锁归web/package.json及package-lock.json；Nginx、独立tmux和Web unit归deploy/；认证健康判定可复用GET /api/v1/auth/session的401/unauthenticated（不是新增公开health资源）；SQLite迁移高版本拒绝，不能无条件代码回滚。

官方文档已核对：

- [GitHub Release API](https://docs.github.com/en/rest/releases/releases)：latest/按tag及发布资产，权限contents:write；latest不包括草稿/预发布，发布渠道需明确。
- [Actions push触发](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#push)：main与标签触发取决于用户更新节奏。
- [Nginx WebSocket](https://nginx.org/en/docs/http/websocket.html)：hop-by-hop Upgrade/Connection必须显式代理，默认读超时不足长会话。
- [Nginx buffering](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_request_buffering)：请求/响应缓冲与落盘边界按路径配置。

实施时从官方仓库复核Actions实际版本/commit SHA，不凭搜索结果混用v6/v7。systemd官网本轮工具访问返回内部错误，实施可用Debian/本机man替代核对并保留限制。
