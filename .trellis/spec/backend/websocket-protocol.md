# WebSocket 协议

## 当前实施边界
W01 不开放假 attach/events 端点；W04 已开放文件事件 `GET /api/v1/events?project_id=...&folder_id=...&project_version=...&path=...`，按展开目录建立受认证、严格 Origin 的连接，当前服务端发送 `{project_id,folder_id,revision,rescan:true,mode:"watching"|"polling"}`，客户端按事件重列；完整范围见 [W04 契约](workspace-files-contract.md)。下文 `workspace_id` watcher 帧和 subscribe/unsubscribe 为旧候选，非现有协议。[终端最新规则](terminal-lifecycle.md)覆盖单端占用与 confirm 立即关闭候选；terminal attach 仍属 W03。

D06 的隔离 loopback 探针已验证真实 WS/PTY 的基础认证、字节边界、取消及资源回收，见 [桥接验证](bridge-validation.md)。其临时 Bearer、单 attach 与实验回复不是产品 API；正式 Cookie/撤销、多观察端与 snapshot/live 协议仍待验证。

D07 的 [capture/attach 机制实验](history-validation.md) 记录了历史拼接的间隙遗漏反例；record 的限时帧记录不是产品 snapshot 协议。ready、resized 回复均不代表完整初屏/TUI 重绘结束，不得作为原子恢复截点。

D08 [序号/快照对照](snapshot-validation.md) 仅验证有限离线机制；序号连续不保存 parser pending 或全部终端状态。公共 serialize 仍有恢复差异，不据此冻结生产 snapshot/live 格式或宣布无损恢复。

## 1. 范围 / 触发
使用 github.com/coder/websocket。WS 是短期 transport，不拥有 Terminal job。认证/Origin/撤销规则引用 [安全契约](security-config.md)。不使用全局关闭 Origin 校验的选项。

## 2. 签名
Terminal attach 签名待 W03 实现。W04 文件事件使用 `GET /api/v1/events?project_id=<id>&folder_id=<id>&project_version=<version>&path=<relative>`；连接建立时服务端先发送一次 rescan，随后 watcher 或 15 秒轮询触发 rescan。当前文件事件不使用自定义 subprotocol，不能把旧候选写成实现要求。

## 3. 契约
Terminal 二进制帧双向为原始 PTY bytes：client input -> attach PTY；server output -> xterm.write(Uint8Array)。不能将任意 bytes 当 UTF-8 字符串丢数据。客户端不得在掉线时缓存并自动重放 keyboard input（避免重复执行命令）。
文本帧只用于 JSON 控制：

```json
{"type":"resize","cols":120,"rows":32}
```

```json
{"type":"ready","protocol":1,"terminal_id":"<id>","history_lines":10000}
```

```json
{"type":"error","code":"terminal_terminated","message":"终端已终止。"}
```

每次 attach 由 server 发送 ready，之后输出。历史先后顺序与 attach 初始屏幕的去重由 Spike 确定，必须更新此节后 UI 才能实现；ready.history_lines 为实际恢复上限，最大 restore_lines。协议不承诺跨连接 exactly-once output。
每个连接一个 reader + 串行 writer，输出队列有界，input 二进制帧初始上限 64KiB、控制 JSON 4KiB，resize rows/cols 为 1..1000 整数；超过容量主动关闭慢客户端，不 kill tmux。每个 session 初期只有一个可写客户端，第二个 upgrade 返回 409 terminal_attached。心跳/读写 deadline 清理假连接。
关闭：1000 正常 detach；1008 认证失效/协议错误；1009 超限；1011 内部错误；1013 背压/暂不可用。握手前用 HTTP error；升级后统一 control error+close。客户端有限次数指数退避加 jitter、有界上限；浏览器不能直接读取握手 HTTP status，失败后用 /auth/session probe 判断认证，probe 401/403 或已升级连接 1008 停止重试，未知原因仅有界重试后提示手动操作（详见 [客户端](../frontend/clients.md)）。每次重连重鉴权、重查状态、发送当前尺寸。
W04 watcher 事件只含 `project_id,folder_id,revision,rescan,mode`；`revision` 为本连接递增整数，新连接重新同步，不能当永久 journal。每条连接只监听 URL 指定的已展开目录；watcher 无法建立或溢出时以 `mode=polling` 兜底，客户端无论哪种模式都重新列举，不能靠事件猜磁盘事实。不自动对全树递归；上面的旧订阅帧/paths 机制未实现。

## 4. 验证与错误矩阵
| 条件 | 处理 |
| --- | --- |
| session/origin 无效 | 拒绝 upgrade |
| 不存在/terminated Terminal | 404 / 409 |
| cols=0/非整数、未知控制 type | 1008，不 resize |
| 超大 input/control | 1009，仅清理 bridge |
| 输出慢/队列满 | 1013，job 继续 |
| watcher overflow/断线 | rescan / reconnect 后重新查询 |

## 5. 优 / 基础 / 错误用例
优：重连重新鉴权、capture 有界历史、输出 Unicode 分块保持 bytes。基础：resize 同步 PTY。错误：在 unmount 发送 close-terminal 或把 input 写日志。

## 6. 必需测试
真实 WS/PTY 测试错误 Origin、过期连接撤销、二进制 Unicode/控制帧/大小/背压、resize、网络断开，断开后 job PID 不变。watcher 集成覆盖 mkdir/write/rename/remove、git switch、overflow、目录展开/折叠、连接退出释放 watcher；前端 E2E 操作 Terminal 外部改变 Explorer。

## 7. 错误与正确
错误：WS disconnect -> kill-session。正确：cancel attach bridge + 保留 tmux session；新连接重新 Attach。
