# WebSocket 协议

## 当前实施边界
W01 没有开放假 attach/events 端点；W04 已开放文件事件 `GET /api/v1/events?project_id=...&folder_id=...&project_version=...&path=...`，按展开目录建立受认证、严格 Origin 的连接，当前服务端发送 `{project_id,folder_id,revision,rescan:true,mode:"watching"|"polling"}`，客户端按事件重列；完整范围见 [W04 契约](workspace-files-contract.md)。W03 终端协议以 [运行时契约](terminal-runtime-contract.md) 为准；单端占用与 confirm 立即关闭候选均已废止。

D06 的隔离 loopback 探针已验证真实 WS/PTY 的基础认证、字节边界、取消及资源回收，见 [桥接验证](bridge-validation.md)。其临时 Bearer、单 attach 与实验回复不是产品 API；正式 Cookie/撤销、多观察端与历史分离由 W03 产品验收覆盖。

D07 的 [capture/attach 机制实验](history-validation.md) 记录了历史拼接的间隙遗漏反例；record 的限时帧记录不是产品 snapshot 协议。ready、resized 回复均不代表完整初屏/TUI 重绘结束，不得作为原子恢复截点。

D08 [序号/快照对照](snapshot-validation.md) 仅验证有限离线机制；序号连续不保存 parser pending 或全部终端状态。W03 [恢复实验](../../../tests/integration/debian/recovery/README.md) 选择普通只读 PTY attach 重绘活动画面和独立历史快照，不使用 serialize/control-mode/raw `%output` 拼接。正式接口以已通过产品集成验收的 [W03 运行时契约](terminal-runtime-contract.md) 为准。

## 1. 范围 / 触发
使用 github.com/coder/websocket。WS 是短期 transport，不拥有 Terminal job。认证/Origin/撤销规则引用 [安全契约](security-config.md)。不使用全局关闭 Origin 校验的选项。

## 2. 签名
Terminal WS v2 签名见 [W03 运行时契约](terminal-runtime-contract.md)。W04 文件事件使用 `GET /api/v1/events?project_id=<id>&folder_id=<id>&project_version=<version>&path=<relative>`；连接建立时服务端先发送一次 rescan，随后 watcher 或 15 秒轮询触发 rescan。当前文件事件不使用自定义 subprotocol，不能把旧候选写成实现要求。

## 3. 契约
Terminal 服务端二进制帧为原始只读 attach PTY bytes；客户端输入帧前 8 字节是大端 generation，后续 1..65536 字节是原始键盘/粘贴 bytes。不能将任意 bytes 当 UTF-8 字符串丢数据。客户端不得在掉线时缓存并自动重放 input（避免重复执行命令）。服务端 `ready`、控制转移、resize 和倒计时通知均为 JSON 文本，具体字段见 [W03 运行时契约](terminal-runtime-contract.md)。历史走独立 HTTP 快照并整体替换虚拟滚动视图，不进入活动 xterm。
每个连接一个 reader + 串行 writer，PTY 读取与慢 WS 写入用有界队列隔离，input 二进制帧上限 65544 字节、控制 JSON 4KiB，resize rows/cols 为 1..1000 整数；超过容量主动关闭慢客户端，只回收其 attach，不 kill tmux。心跳/读写 deadline 清理假连接。多个只读 viewer 可同时 attach；只有服务端当前 controller 可通过唯一可写控制 attach 输入/resize。
关闭：1000 正常 detach；1008 认证失效/协议错误；1009 超限；1011 内部错误；1013 背压/暂不可用。握手前用 HTTP error；升级后统一 control error+close。客户端有限次数指数退避加 jitter、有界上限；浏览器不能直接读取握手 HTTP status，失败后用 /auth/session probe 判断认证，probe 401/403 或已升级连接 1008 停止重试，未知原因仅有界重试后提示手动操作（详见 [客户端](../frontend/clients.md)）。每次重连重鉴权、重查状态、发送当前尺寸。
W04 watcher 事件只含 `project_id,folder_id,revision,rescan,mode`；`revision` 为本连接递增整数，新连接重新同步，不能当永久 journal。每条连接只监听 URL 指定的已展开目录；watcher 无法建立或溢出时以 `mode=polling` 兜底，客户端无论哪种模式都重新列举，不能靠事件猜磁盘事实。不自动对全树递归；上面的旧订阅帧/paths 机制未实现。

## 4. 验证与错误矩阵
| 条件 | 处理 |
| --- | --- |
| session/origin 无效 | 拒绝 upgrade |
| 不存在/terminated Terminal | 404 / 409 |
| cols=0/非整数、未知控制 type | 1008，不 resize |
| observer 输入/resize 或旧 generation | 拒绝操作，不写 PTY |
| 超大 input/control | 1009，仅清理该 viewer bridge |
| 输出慢/队列满 | 1013，仅回收该 viewer attach，job 继续 |
| watcher overflow/断线 | rescan / reconnect 后重新查询 |

## 5. 优 / 基础 / 错误用例
优：重连重新鉴权、独立 capture 有界历史、只读 attach 输出 Unicode 分块保持 bytes。基础：controller resize 同步可写 PTY。错误：在 unmount 发送终止或把 input 写日志。

## 6. 必需测试
真实 WS/PTY 测试错误 Origin、过期连接撤销、二进制 Unicode/控制帧/大小/背压、resize、网络断开，断开后 job PID 不变。watcher 集成覆盖 mkdir/write/rename/remove、git switch、overflow、目录展开/折叠、连接退出释放 watcher；前端 E2E 操作 Terminal 外部改变 Explorer。

## 7. 错误与正确
错误：WS disconnect -> kill-session。正确：cancel attach bridge + 保留 tmux session；新连接重新 Attach。
