# W03 终端运行时契约

## 1. 范围与触发条件

W03 [PRD](../../tasks/09-29-terminal-runtime/prd.md)、[设计](../../tasks/09-29-terminal-runtime/design.md) 与 [验收记录](../../tasks/09-29-terminal-runtime/check-report.md) 是本契约依据。2026-09-30 T01..T07 已通过；今后修改终端创建、恢复、控制、终止或部署归属时必须重验相关路径。Web 与 tmux server/pane 生命周期分离。tmux 会话 ID/精确 target 由服务端生成，客户端只提交项目/folder 身份与尺寸；不接受 shell 命令、socket 路径或任意 tmux 参数。

## 2. 签名

- HTTP：`GET/POST /api/v1/terminals`、`GET /api/v1/terminals/{id}`、`GET /api/v1/terminals/{id}/history`。
- WS：`GET /api/v1/terminals/{id}/stream`；v2 帧样例见 [共享 fixture](../../../tests/contracts/terminal-runtime.json)。
- DB：复用既有 `terminals` 表，由 repository 保存稳定 ID/target/cwd；W03 不需要新增迁移，不改写历史迁移。
- CLI：固定 argv 调用 `tmux -N -S <socket>`，精确 target `=persistty_<id>:0.0`；禁止用户控制参数或补偿 kill。

## 3. 契约

### 元数据与配置

SQLite 只保存终端稳定 ID、不可猜的 `persistty_<随机ID>` tmux session、显示名、可空 project ID、创建时 cwd 和创建时间。状态由 `tmux -N -S <私有socket>` 实时观测：session 存在为 `running`，明确不存在为 `terminated`，server/socket/命令故障为 `unavailable`。项目删除靠外键 `SET NULL`，不销毁真实进程；旧 metadata 不自动复活。创建时按项目版本锁定主或指定 folder，复验注册 root 的 dev/inode，启动 tmux 成功后写入 SQLite。若 DB 写失败保留 tmux session，按受管理命名重新发现，不能补偿 kill。tmux server 缺失时 503，Web 不隐式启动。

终端配置：私有 socket 路径、明确绝对路径的 tmux/shell、history limit 默认 5000 行、restore limit 不大于 history limit、终止倒计时默认 10 秒且允许 1..120 秒。配置启动时校验路径形状/不重合，终端操作时 probe 依赖/server，故障返回 unavailable，不能把配置加载成功当作 tmux 已可用。窗口尺寸 1..1000 且不超过 PTY `uint16` 范围。history 与输出队列按字节额外设限，超限显式报错或截旧留新，不无限分配。独立同 UID tmux systemd unit 预启动 server；Web unit 不拥有 server socket 生命周期，也没有 `PartOf/BindsTo` 传播。

### HTTP

- `GET /api/v1/terminals`：`{items:[{id,display_name,project_id,working_directory,state}]}`，列表真实查询 tmux；缺依赖不伪装空列表。
- `POST /api/v1/terminals`：`{project_id,project_version,folder_id?,display_name?,cols?,rows?}`；不填 `folder_id` 使用该版本主 folder，尺寸省略/0 默认 80×24，最终有效范围 1..1000。需要 Cookie、Origin、CSRF。返回同一 terminal DTO。
- `GET /api/v1/terminals/{id}`：返回同一 terminal DTO，状态来自当前 tmux；不存在的稳定 ID 返回 404。
- `GET /api/v1/terminals/{id}/history`：仅当前普通历史的有界快照，返回 `{content_base64,history_size,returned_lines,alternate_on,cols,rows,truncated}`。使用 `capture-pane -p -e -S - -E -1`；响应只是某一时点的历史，不是实时输出帧或跨请求稳定游标。客户端在虚拟滚动视图**整体替换**快照，不与活动 xterm 字节拼接；滚动中的已有快照保持，用户刷新时替换。活动 xterm `scrollback:0`，不重复旧行。

### WebSocket v2

`GET /api/v1/terminals/{id}/stream` 必须先验证 Cookie、精确 Origin 与真实 tmux 状态。每连接启动一个 `read-only,ignore-size` PTY attach；tmux 重绘该连接当前画面。另有一个 Web 生命周期内的可写控制 attach 接收授权输入和 resize，输出持续排空。Web 停止/连接断开只取消并强制回收对应 attach 客户端，不 kill session/server/pane。每连接输出队列、写 deadline 和 PTY/JSON 帧均有界；慢端 1013 断开，不能把读取 PTY 绑在慢 WS Write 上。

服务端第一帧为 JSON `ready`：`{type:"ready",protocol:2,terminal_id,viewer_id,role:"controller"|"observer",generation,cols,rows,pending_termination:null|{request_id,deadline}}`。无人控制时首个有效查看端自动取得控制；额外查看端为 observer。后续服务端二进制帧是**原始 PTY 输出 bytes**。客户端二进制输入帧是 8 字节大端无符号 `generation` 前缀加 1..65536 字节原始输入；每帧在串行控制裁决里核对连接身份、当前 generation、认证与 controller 权限，旧帧不写 PTY。不得缓存重放结果不确定的输入。

客户端文本 JSON：`{type:"takeover",generation}`、`{type:"resize",generation,cols,rows}`、`{type:"terminate",generation}`、`{type:"cancel_termination",request_id}`。仅 controller 可 resize/发起终止；observer 可显式接管，任一鉴权 viewer 可取消尚未到期的终止。接管递增 generation 并使旧端立即失权；服务端按连接发送 `{type:"control",role,generation}`，resize 回 `{type:"resized",cols,rows}`。未知字段/type、越界、超帧按 1008/1009；错误以 `{type:"error",code,message}` 脱敏报告。升级后认证到期/登出关闭 1008，断开不杀任务。

终止通知：`{type:"termination_pending",request_id,deadline}`、`{type:"termination_cancelled",request_id}`、`{type:"termination_executed",request_id,state}`。截止由服务端 UTC 时钟掌握；同 terminal 同时最多一个请求。仅 controller 的有效 generation 可发起，所有当前和新连接看到相同 deadline；任何有效 viewer 在截止前取消，取消成功后该 request ID 永不再执行。发起者断开/认证失效、控制转移或 Web 重启取消 pending。截止时在同一协调器串行裁决，复验发起者认证和精确 tmux target 后 kill-session，仅处理目标；未知结果查询 tmux，不自动重试输入/终止。

## 4. 验证与错误矩阵

| 条件 | 结果与副作用 |
| --- | --- |
| 无 Cookie/认证过期 | 握手前 401；升级后 1008，撤销 attach，不杀 pane |
| 缺失/错误 Origin、写请求 CSRF 无效 | 403，不创建或连接 |
| project/folder/version 过时 | 409/领域错误，不创建；项目外 cwd 不扩大文件 API |
| tmux server 缺失或命令故障 | 503，不隐式启动，不假装空列表 |
| 已结束终端请求 attach | 409，不复活 target |
| observer 输入/resize 或旧 generation | `control_denied`/`stale_generation`，零 PTY 写入 |
| 重复终止/取消已结束请求 | `termination_pending`/`termination_missing`，无重复 kill |
| 未知控制字段/type、无效尺寸 | 1008；超帧 1009，不执行动作 |
| 输出队列满/写超时 | 1013/断开，只回收自身 attach，pane 继续 |

## 5. 正常 / 基础 / 错误用例

正常：A 控制、B/C 观察；B 接管后 A 旧代输入被拒。B 发起终止，C 取消，同 request 永不执行。基础：按主 folder 创建，关页后以同 ID 恢复当前画面和独立历史。错误：把 socket 故障当空列表、重放断线输入、因移除项目或 DB 写失败 kill 真实任务。

## 6. 所需测试

共享 JSON/WS fixture 与 Go/TS decoder 测试需覆盖三端控制、旧 generation 二进制帧、观察端 resize、背压、重连、Web restart、倒计时取消竞态、项目解绑。真实 Debian/systemd 必须证明原 pane PID/start/cgroup 保持、TUI/普通历史分离、split UTF-8/OSC、resize、慢端回收及只清理自身资源。已通过的 40 项探针检查与 9 条 Playwright 路径见验收记录，mock 不能替代此门禁。

## 7. 错误与正确示例

错误：Web 创建默认 tmux server；历史 capture 后直接 append raw attach；关闭 WS 调 `kill-session`；失败后自动重发输入。正确：外部独立 server + `-N`、只读 attach 重绘当前画面、独立整体替换历史、断开只 detach，终止只在服务器有效 deadline 到期后精确裁决。
