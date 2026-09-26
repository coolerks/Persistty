# HTTP 与 WS 客户端

统一 fetch client 使用同源 /api/v1、credentials same-origin，写入附 session CSRF header，发送 JSON Content-Type；文件/chunk 使用各自类型，不能强设 multipart boundary。Cookie 由浏览器管理，不读 HttpOnly Cookie 或在 localStorage 保存密码/token。
先检查 HTTP status 再解码；204 不解析 JSON；下载/preview 使用 blob/stream，不套 JSON decoder。typed ApiError(code,status,requestId,details) 供 conflict/auth/rate-limit UI，网络错误和业务错误区分。401 保存 UI draft、进入登录、终止重试；429 尊重 Retry-After；409 不自动 overwrite。AbortError 不 toast 服务故障。

WS transport 统一拥有 socket、reader dispatch、close/retry timer/generation；feature 消费 typed control 或 raw bytes。协议来自 [backend WS](../backend/websocket-protocol.md)，不创建私有 close-terminal frame。连接状态 connecting/connected/reconnecting/disconnected 独立于 tmux status。以当前页面 origin 构造 ws/wss，生产不 token query，不任意连用户指定 WS URL。
重新 attach 获取 metadata 状态和尺寸，历史恢复策略必须等 Spike 定案。输出有界排队，xterm.write callback 协调渲染；高 bufferedAmount 时输入暂停反馈，不积累无界 input。掉线键盘输入不自动 replay。浏览器原生 WebSocket 不暴露握手 HTTP status：握手失败后通过同源 GET /auth/session 探测认证，401/403 停止并提示重新登录/拒绝访问；升级后 control error 或 close 1008 也停止重连。认证仍有效而失败原因未知时有限次数指数退避+jitter，耗尽后显示错误和手动重试，不能臆测 HTTP 状态或永久重连。具体尝试次数/退避上限由客户端任务锁定测试。dispose 取消所有重连；不能关闭服务器 session。
watcher rescan/断线后刷新目录，invalidate server snapshot；不直接以 event payload 替代 filesystem 事实。工作区切换丢弃旧连接事件。

测试共同协议 fixture、错误响应、abort、401/409/429、WS 握手状态不可见时的 session probe/1008/重试耗尽、网络恢复、obsolete generation、binary Unicode、resize 队列、timer/socket 回收、断连不 close-terminal。真实 WS 测试补齐 mocks 覆盖不了的时序。
