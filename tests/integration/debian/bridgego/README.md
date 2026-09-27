# 隔离 Go / PTY / WS 探针

此独立模块仅用于 W02，不注册产品路由。Go 1.26.8、`creack/pty v1.1.24`、`coder/websocket v1.8.15` 固定于 manifest 与校验锁文件；版本依据[PTY 官方包资料](https://pkg.go.dev/github.com/creack/pty@v1.1.24)与[WS 官方包资料](https://pkg.go.dev/github.com/coder/websocket@v1.8.15)，不宣称它们是最新版本。第三方通知来自这些实际模块的 LICENSE/ LICENSE.txt，完整[通知](third-party-notices.txt)嵌入二进制，可运行 `bridgeprobe licenses` 查看。

## 入口与私有文件

```text
bridgeprobe serve --tmux-bin <绝对路径> --socket <私有socket> --session persistty_probe_<12hex> --token-file <私有token文件> --ready-file <未存在文件>
bridgeprobe client --port <loopback端口> --token-file <私有token文件> --mode exercise|observe|hold --duration 30s
bridgeprobe client --port <loopback端口> --token-file <私有token文件> --mode record --duration 1s --cols 80 --rows 24
```

`hold` 额外必需 `--ready-file`。token 文件为当前用户拥有、非符号链接的普通文件，权限必须为 0600，内容为 64 个小写 ASCII hex 字符，可带一个末尾换行。token 只通过 Bearer header 传输，不进入 argv、URL、ready、stdout 或报告。输入错误及外部错误只输出固定类别，不输出原始参数或输入。连接身份读取及 SSH/SCP 仅由上层 [runner](../run_remote.py) 管理，探针不读取 `.env` 或 SSH 环境。

服务仅监听 `127.0.0.1:0`；ready 文件使用 O_EXCL 新建 0600，JSON 为 `{port,pid}`，父目录由 runner 保证 0700。hold ready 为 `{pid,attached:true}`。duration 默认 30s，必须大于零且不超过 1min。ready 表示 attach 子进程已创建，并不代替收到实际 PTY 输出的断言。

## 实验协议与资源边界

- GET `/attach`：Bearer 认证与 Origin 必须精确为 `http://127.0.0.1`；错误分别 401/403。最多一个 active attach，其余 409，此限制是实验边界，不是产品多观察端方案。
- `tmux -N -S <socket> attach-session -t =<session>` 仅连接既有 server；没有创建或终止 session/server 的路径。连接停止只关闭 PTY、精确 kill attach 子进程并 Wait，等待读写 pump 退出。
- 初始文本 `{"type":"ready"}`；输入/输出均为 WS binary，任何字节都不转换为 UTF-8 字符串。文本仅接受严格三字段 resize，cols/rows 整数 1..1000，拒绝重复/未知/缺失字段与尾随 JSON。成功返回 `{"type":"resized","cols":91,"rows":27}`。
- binary 消息上限 64KiB，控制帧上限 4KiB；无效控制关闭 1008，超限 1009。输出最多排队 32 个 1024-byte 块及 8 条尺寸应答，队列满关闭 1013；每次 socket/PTY 写入 deadline 2s。WS Read 绑定服务及连接取消，当前没有产品级认证撤销、心跳或长期 idle 探测。
- GET `/stats` 必须 Bearer 认证，返回 active/started/reaped/fd_count/goroutines/pid；Linux fd 来源 `/proc/self/fd`，非 Linux 缺少该接口时值 -1，不假称 FD 检验已执行。

`exercise` 只发送一次固定实验输入 `BRIDGE_INPUT_雪\n`，两个 binary 消息在 UTF-8 字符内分割；runner 的 raw-mode Python pane 收到完整行后将输入计数加一，分块输出 `BRIDGE_ACK_雪`。随后错误尺寸、超限与五次 detach/reconnect 不发键盘输入。observe/hold 不发 binary input；observe 仅确认收到实际输出再 detach。exercise 比较同一服务的 FD/goroutine 基线，允许分别 +2/+4 的运输层预算，并断言所有 attach 已 Wait；有限循环及预算不是任意负载无泄漏证明。

## 本地检查与限制

在本目录分别运行 `go test ./...`、`go test -race ./...`、`go vet ./...`；根 module 的测试不会覆盖此模块。Linux 二进制以 `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build` 交叉构建到忽略目录 `.cache`。本地测试使用真实 PTY 与 httptest WS、固定 cat 子进程，只测试 bridge；真实 tmux/systemd 续存由 Debian runner 独立执行。

PTY 的 Dup fd 设置 close-on-exec 并以非阻塞 fd 包装，避免后续 exec 继承该资源，同时使取消可打断读写。回归额外核对 fd 标志、未授权 stats 不泄露 PID，以及真实无限输出遇到不读客户端时关闭 1013 并回收 attach；该有限测试不替代实际慢浏览器或完整生产流控验收。

D07 `record` 仅用于固定合成输出：不发送 keyboard binary input，发一次尺寸控制，最多 10s、256 KiB、4096 binary 帧，要求收到尺寸应答与实际输出；退出后等待 attach 回收。stdout 是 base64 帧、字节数、尺寸与 stats，只允许上层 runner 私有暂存，不允许将原始帧入库或用于记录用户终端。[历史实验](../history/README.md) 使用锁定 xterm headless 解析；这不是历史 snapshot 协议。

尚不覆盖浏览器/xterm/TUI、历史快照和 live 衔接、多 controller/终止倒计时、生产认证及反向代理；有界队列和写 deadline 的实现与基础回归不等于已完成真实慢浏览器负载验收。不得以本模块通过宣布 W02 或完整终端产品验收通过。
