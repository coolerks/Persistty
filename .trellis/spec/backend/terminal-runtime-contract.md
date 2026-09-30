# W03 终端运行时契约

## 1. 范围与触发条件

W03 [PRD](../../tasks/archive/2026-09/09-29-terminal-runtime/prd.md)、[设计](../../tasks/archive/2026-09/09-29-terminal-runtime/design.md) 与 [验收记录](../../tasks/archive/2026-09/09-29-terminal-runtime/check-report.md) 是本契约依据。2026-09-30 T01..T07 已通过；今后修改终端创建、恢复、控制、终止或部署归属时必须重验相关路径。Web 与 tmux server/pane 生命周期分离。tmux 会话 ID/精确 target 由服务端生成，客户端只提交项目/folder 身份与尺寸；不接受 shell 命令、socket 路径或任意 tmux 参数。

## 2. 签名

- HTTP：`GET/POST /api/v1/terminals`、`GET /api/v1/terminals/{id}`、`GET /api/v1/terminals/{id}/history`。
- 工作台 UI 调整新增：`PATCH /api/v1/terminals/{id}`、`POST /api/v1/terminals/termination-batches`、`GET /api/v1/terminals/termination-batches/{id}`；不新增迁移。以下新增契约覆盖旧的单目标内部实现，v2 对外形状仍保留。
- WS：`GET /api/v1/terminals/{id}/stream`；v2 帧样例见 [共享 fixture](../../../tests/contracts/terminal-runtime.json)。
- v3 自动设备属性应答：`{type:"device_attributes",kind:"primary"|"secondary"}` → `Viewer.DeviceAttributes(ctx,kind string) error`，只回答当前连接的只读 attach，不是键盘输入。
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

客户端文本 JSON：`{type:"takeover",generation}`、`{type:"resize",generation,cols,rows}`、`{type:"terminate",generation}`、`{type:"cancel_termination",request_id}`。仅 controller 可 resize/发起终止；observer 可显式接管，任一鉴权 viewer 可取消尚未到期的终止。接管递增 generation 并使旧端立即失权；服务端按连接发送 `{type:"control",role,generation}`，resize 向所有尺寸同步成功的 viewer 发送 `{type:"resized",cols,rows}`（v2/v3 形状不变）。未知字段/type、越界、超帧按 1008/1009；错误以 `{type:"error",code,message}` 脱敏报告。升级后认证到期/登出关闭 1008，断开不杀任务。

输入 owner、所有 read-only,ignore-size 输出 attach、浏览器 live renderer 必须使用同一 cols/rows。controller Resize 在 hub 锁内复验控制权/代次，先更新 owner PTY，成功后更新 hub 尺寸和各 viewer PTY，再发 resized；单个输出 PTY 失败仅取消该 viewer，不让其他端/任务终止。迟加入 viewer 在同一锁内按当前 hub 尺寸 attach（不是固定 80x24），ready 返回相同网格。observer 不能改变 pane；客户端跟随广播网格并在本地裁剪/留空。输出读线程经相同锁排队，不能将本次重绘先于对应 resized 入队。仅更新输入 attach 会让 tmux 在更大的输出 PTY 绘制边界 `─` 与 `·` 填充；禁止用正文过滤、fill-character 改配置或断开其他用户掩盖错配。

终止通知：`{type:"termination_pending",request_id,deadline}`、`{type:"termination_cancelled",request_id}`、`{type:"termination_executed",request_id,state}`。截止由服务端 UTC 时钟掌握；同 terminal 同时最多一个请求。仅 controller 的有效 generation 可发起，所有当前和新连接看到相同 deadline；任何有效 viewer 在截止前取消，取消成功后该 request ID 永不再执行。发起者断开/认证失效、控制转移或 Web 重启取消 pending。截止时在同一协调器串行裁决，复验发起者认证和精确 tmux target 后 kill-session，仅处理目标；未知结果查询 tmux，不自动重试输入/终止。

### 显示名称

创建省略 display_name 时，service 的 metadataMu 串行协调可靠 tmux 查询、受管理 session 对账、名称分配、创建与入库。每项目依次选择最小空号“终端”“终端1”“终端2”，上下区域共用；所有存续自定义同名也占号。隐藏、移动、倒计时不释放；真实已结束或改名后可复用，但不复用稳定 ID。查询失败不冒充空名单。手工重名仍允许，部署边界仍为单 Web 实例。

PATCH body 严格为 `{expected_display_name,display_name}`，上限 8 KiB；两字段均为非空 UTF-8、无首尾空白、无 NUL/CR/LF、至多 200 字节。需要 Cookie/Origin/CSRF 和合法稳定 ID。SQL 参数化 CAS，返回 terminal DTO；旧名不符且当前也非请求新名为 409，同新名可幂等返回。此旧名 CAS 不检测 ABA，不是完整元数据版本机制。名称不要求 controller，不改变 PID/cwd/target。SQLite 为权威，固定 argv 更新 tmux `PERSISTTY_DISPLAY_NAME` 镜像失败不回滚或 kill，后续对账修复。

### 统一批次与 v3

POST body 严格为 `{members:[{terminal_id,viewer_id,generation}]}`，64 KiB、1..200 个唯一 terminal ID，viewer/terminal 为 32 位小写 hex，generation 为 1..2^53-1 的整数。Cookie/Origin/CSRF 必需；逐目标复验真实 session、存活 viewer、HTTP token 与 viewer token 相同、controller/generation、无 pending。全部通过才登记一个 timer/deadline；失败零新 timer/kill。上下范围由前端列明并冻结，后端绝不展开“全部”。

POST/GET data 同为 `{request_id,deadline,members:[{terminal_id,display_name}],state,results:[{terminal_id,state}]}`。batch state 为 pending/cancelled/executing/completed；未完成 results=[]，完成覆盖全部成员，逐项 running/terminated/unavailable。成员名称为创建批次时的快照；手工重名不改变身份。GET 需 Cookie；结果保留最多 256 条、结束后最长 5 分钟，满额只淘汰已结束项，否则拒绝新请求。重启后 GET 404，不重放 pending。失败 envelope 可含且仅含领域 details `{terminal_id}`，不含路径/输入/内部 stderr。

旧 WS terminate 复用单成员批次。同一成员最多属于一个 pending/executing 批次。任一有效查看端截止前 cancel，控制转移、发起连接断开/失效、hub 关闭均撤销受影响整批；独立批次不受影响。控制/批次协调锁先于 hub 锁，多 hub 按稳定 ID 排序；读取 map 可在协调器外先冻结参与者，但必须释放 hub 锁后再进入协调器，禁止 hub → coordinator。认证与 tmux I/O 不持有批次锁。

截止重新复验所有成员后串行声明 executing；之后不承诺撤销/回滚。锁外最多 4 worker、整体 8 秒 context，每目标最多一次精确 kill，随后查询真实状态，未知标 unavailable，不重试 kill。完成后解除成员占用并广播逐项结果，不能宣称多个 kill 原子成功。

stream 缺省/`?protocol=2` 保留 exact v2；显式 `?protocol=3` 才启用新帧，重复/空/其他 protocol 返回 400。v3 ready protocol=3、pending 额外 members；termination_pending 额外 members，termination_executed 额外 results，state 仍为该连接目标结果；metadata 为 `{type:"metadata",terminal_id,display_name}`。v2 不接收额外字段/metadata，仍能用原 cancel_termination 撤销整个 v3 批次。二进制输入/输出及 generation 不变。字段完整示例由共享 fixture 锁定。

### DA1/DA2 应答归属（2026-09-30）

只读 viewer attach 发出的 CSI c / CSI > c 查询由该 viewer 的前端 parser 接收；不能把 xterm 默认 onData 应答当键盘输入写到另一个 owner attach，否则 attach 握手时序变化可能让 shell 收到 `1;2c` / `0;276;0c`。

v3 新增严格文本帧 `{type:"device_attributes",kind:"primary"|"secondary"}`，共享 fixture 的 `device_attributes_primary_v3` / `device_attributes_secondary_v3` 锁定形状。无 generation、target 或 payload 字段；v2、未知 kind、null、重复/额外字段拒绝 1008。控制 JSON 仍至多 4 KiB，原二进制 input/controller/generation 校验保持。

服务端在 `Viewer.DeviceAttributes` 复验认证、protocol=3、未关闭 hub、存活 context 与 `h.viewers[v.ID] == v`，仅向 `v.attach.file` 写固定 `ESC[?1;2c` 或 `ESC[>0;276;0c`（锁定 xterm 6.0.0 的 DA1/DA2 原值）。observer 可应答自己的 read-only attach；不修改控制权、不写 owner、不接受任意字节/命令、无缓存重放，写 deadline 2 秒。失败不回退 owner 输入。仅处理已确认 DA1/DA2，不把该能力扩为任意终端响应代理。

## 4. 验证与错误矩阵

| 条件 | 结果与副作用 |
| --- | --- |
| 无 Cookie/认证过期 | 握手前 401；升级后 1008，撤销 attach，不杀 pane |
| 缺失/错误 Origin、写请求 CSRF 无效 | 403，不创建或连接 |
| project/folder/version 过时 | 409/领域错误，不创建；项目外 cwd 不扩大文件 API |
| tmux server 缺失或命令故障 | 503，不隐式启动，不假装空列表 |
| 已结束终端请求 attach | 409，不复活 target |
| observer 输入/resize 或旧 generation | `control_denied`/`stale_generation`，零 PTY 写入 |
| v3 DA1/DA2 自动应答 | 固定字节只进当前 viewer read-only attach，零 owner/pane 输入，不改变 controller |
| DA 非法 kind/原始 payload/额外字段/v2 | 1008，不写 PTY |
| DA 认证撤销/连接取消/已关闭 hub | 拒绝，不重放、不回退键盘通道 |
| 重复终止/取消已结束请求 | `termination_pending`/`termination_missing`，无重复 kill |
| 未知控制字段/type、无效尺寸 | 1008；超帧 1009，不执行动作 |
| 输出队列满/写超时 | 1013/断开，只回收自身 attach，pane 继续 |
| PATCH 旧名冲突/无效名称 | 409/400，不覆盖新元数据；镜像失败不杀进程 |
| batch 伪 viewer、旧 generation、已有 pending | 409 control_denied/stale_generation/termination_pending，details 指明目标，整批零新 timer |
| batch 成员已结束/不可观测 | 409 conflict/503 unavailable；不跳过失败成员 |
| batch result 不存在/重启后已丢弃 | 404，不自动重发终止 |

## 5. 正常 / 基础 / 错误用例

正常：A 控制、B/C 观察；B 接管后 A 旧代输入被拒。B 发起终止，C 取消，同 request 永不执行。基础：按主 folder 创建，关页后以同 ID 恢复当前画面和独立历史。错误：把 socket 故障当空列表、重放断线输入、因移除项目或 DB 写失败 kill 真实任务。

## 6. 所需测试

共享 JSON/WS fixture 与 Go/TS decoder 测试需覆盖三端控制、旧 generation 二进制帧、观察端 resize、背压、重连、Web restart、倒计时取消竞态、项目解绑。真实 Debian/systemd 必须证明原 pane PID/start/cgroup 保持、TUI/普通历史分离、split UTF-8/OSC、resize、慢端回收及只清理自身资源。已通过的 40 项探针检查与 9 条 Playwright 路径见验收记录，mock 不能替代此门禁。

新增名称并发/CAS/空号/镜像失败测试、共享 v2/v3 fixture、批次 A/B/C 与独立 D/E 撤销、校验失败零 timer/kill、锁外 kill 与无重试、HTTP 写保护与严格解码，均须运行 race。浏览器新增接管零重放、跨端改名同 DOM/零 WS、上下目标隔离和 v2 observer 整批取消。最新证据与未验证边界见 [UI 调整验收](../../tasks/archive/2026-09/09-30-workbench-ui-layout/check-report.md)，不能将单测记为所有真实设备验收。

设备应答回归：HTTP decoder 用共享 fixture 验 v3 可达/v2 拒绝、null/重复/任意 payload 拒绝；`device_attributes_test.go` 用独立 pipes 验 controller/observer 各自应答与 owner 零多余输入、失效/非法状态零写。`device_attributes_integration_test.go` 自建私有 tmux socket/raw-input Python pane，真实观察两端 DA 查询并应答，pane 仅收到显式键盘字节；缺 tmux/Python 显式 skip，不能计为真实验收。测试仅清理自身 server/session/root，运行 race。该本机证据不代替 Debian/systemd 持久性验收。

## 7. 错误与正确示例

错误：Web 创建默认 tmux server；历史 capture 后直接 append raw attach；关闭 WS 调 `kill-session`；失败后自动重发输入。正确：外部独立 server + `-N`、只读 attach 重绘当前画面、独立整体替换历史、断开只 detach，终止只在服务器有效 deadline 到期后精确裁决。

错误：只读 attach 的 DA 应答经二进制 Input 写到 owner。正确：公开 parser 截获查询 → 固定 kind 文本帧 → 当前 viewer 的只读 attach；普通输入和粘贴保持原样，不用正则删除正文中的数字/ANSI。

尺寸同步补充门禁：`device_attributes_integration_test.go` 的自有真实 tmux/PTY fixture 验 140x12→100x30→160x10 各 attach ioctl 网格与真实 pane 相等、pane PID 保持、迟加入 observer 继承当前网格、observer resize 拒绝、全端收到 resized、pane 输入仍仅显式 x。浏览器观察端跟随 ready/resized 且容器缩放不改网格。该本机集成不代替 Debian/systemd 或真实触控板手感验收。
