# Go 终端桥接实验

## 1. 范围与触发条件
W02 D06 验证真实 Go/PTY/WS 与 Debian tmux/systemd 的基础链路，不注册产品路由。正式多观察端、controller generation、终止倒计时、历史快照/live 衔接和浏览器/TUI 验收仍由对应任务负责，不从实验单连接限制推导产品行为。

## 2. 签名
独立模块为 [bridgego](../../../tests/integration/debian/bridgego/README.md)，不改根 Go manifest。`bridgeprobe serve --tmux-bin <absolute> --socket <private> --session persistty_probe_<12hex> --token-file <private> --ready-file <new>`；client 使用 `--port`、`--token-file`、`--mode exercise|observe|hold`，hold 另有 ready 文件。远端复现归 [runner](../../../tests/integration/debian/bridge/README.md)，连接只安全解析私有 .env，不在命令中展开 SSH 身份。

## 3. 契约
服务仅监听 loopback 随机端口，token 为 owner-only 普通文件 0600，Bearer header 传输，不进入 argv/URL/ready/日志/证据。ready 文件 O_EXCL 创建 0600，父目录须 0700；仅含实验 pid/port，不能把文件存在当已收到 PTY 输出的证明。

实验 GET /attach 验 token 与精确 loopback Origin，最多单 active attach；WS binary 保留字节，文本仅为严格 resize/control。只通过 tmux -N 连接既有私有 server，WS 或服务 context 取消只关闭 PTY、终止并 Wait 精确 attach 子进程、回收 pump，绝无 kill-session/server 或任务自动重跑路径。输入只由明确 client exercise 发送一次；重连 observe/hold 不输入，计数不能增长。

输出队列和写入 deadline 必须有界，readlimit 与控制字段/重复键验证先于副作用。具体上限与实验回复见模块 README；有限资源预算和 trace 只是基础证据，不宣称任意负载下无泄漏。GET /stats 也需 Bearer，不开放未认证探测；Linux FD 采样不可在其他系统伪称执行。

PTY fd 经 Dup 后必须显式设置 CloseOnExec，避免后续子进程继承。回归须检查真实 PTY 的 fd 标志、未授权 stats 不泄露状态，以及慢客户端队列满后 1013 与 attach Wait 完成，不只测试返回码。

## 4. 验证与错误矩阵
| 场景 | 必需结果 |
| --- | --- |
| token/Origin 错误或额外 attach | upgrade 前拒绝，无新长期任务 |
| 控制非法/输入超限 | 关闭对应连接，不终止 pane |
| WS detach/cancel | attach 被 Wait，FD/goroutine 回到预算 |
| Go unit stop/restart/SIGKILL | server/pane 身份保持，逐样本心跳增长 |
| reconnect | 无自动输入，已执行输入计数不增 |
| socket 缺失 | 明确失败，不隐式启动 tmux |
| allocate 传输失败无 ROOT | 不宣称零资源，限定只读核查未知目录，不泛删 |

## 5. 正常、基础、错误用例
正常：一个 UTF-8 字符被分到两条 binary frame 后仍收集为原字节，实验 ACK 与一次输入计数一致。基础：observe 收到实际 PTY 输出再 detach。错误：服务重新启动时自动重发上次输入、重新创建同名任务，或把“连接成功”当原进程存活。

## 6. 所需测试
根模块与独立 bridgego 模块分别运行 test/race/vet；真实 httptest WS + PTY、JSON/字节边界及资源回归不可由 mock 代替。Debian 在新建私有 ROOT 中使用最终 binary 独立重跑，记录 binary SHA256、原 PID/start time/cgroup、逐样本心跳/输入计数和清理；证据先去敏，token/正文不入库。

服务和 runner 的测试、首轮/复核证据分别留记录，不以早期二进制的实测冒充最终版本。SIGKILL/本机退出可能跳过 runner finally，继续保持 [远端规范](remote-validation.md) 的清理边界；所有可能的已知资源精确核实，未知目录只读报告。

## 7. 错误与正确示例
错误：`disconnect -> kill-session`，或仅看 metadata/ready 就报告持久性通过。正确：`disconnect -> cancel bridge -> close PTY -> kill/Wait attach client`，再用独立采样确认原 pane 身份和任务推进。实验成功只放行已验证的基础机制，不能放行尚未验证的正式多端/历史/TUI 功能。
