# 工作台 UI 布局调整设计

## 1. 状态与边界

本设计对应 [PRD](prd.md) UI01..UI10、AC01..AC15。用户已批准实施，当前 in_progress；新增 API 已接入，验证以 [验收记录](check-report.md) 为准。主会话直接研究、实现和检查，禁用 subagent。

交付是一个完整的工作台交互调整：布局、标签动作及支撑这些动作的元数据/批次协议互相依赖，按 [实施清单](implement.md) 分步集成，不另建可独立归档的伪交付物。登录、项目面板不重新设计；统一终端页保留原有路由/业务边界，共享运行时更新向后兼容，不新增全服务跨项目批量终止命令。

沿用 React/TypeScript/Vite/React Router/Tailwind/shadcn/ui/lucide/Zustand、Monaco、xterm 与 react-resizable-panels。不修改启动脚本或部署，不开发 W05 文件自动保存/草稿。

## 2. 界面与组件职责

### 顶部与视口

- `App.tsx` 的 Shell 在项目工作台路由不再占独立品牌行；复用已有品牌导航/主题/退出组件，由工作台单条顶栏编排。项目名称和编辑入口靠左，主题/退出靠右，窄屏收纳次要操作，不把业务注销副作用复制到新组件。
- 工作台为视口高度容器：单顶栏、剩余高度的主体和固定底部状态栏。主体内由 resizable panels 分配尺寸；滚动归 Explorer、编辑器、历史视图，不归 document。
- [尺寸证据](research/browser-geometry.md) 已复现 656px 视口对应 1013px document。优先修正 Explorer 滚动区/隐藏辅助节点的定位边界并做对照测量，再按证据补 flex 的 min-size/overflow 约束。不删除 sr-only 标签，不更改 Monaco/xterm 内部尺寸实现，不用全站 body overflow:hidden 隐藏缺陷。
- 手机保留单视图与终端快捷键栏；菜单和 Dialog 在视口内滚动，软键盘时输入可见。必要时按 visualViewport 生命周期调整手机工作台可用高度，必须清理监听；是否需要由真实浏览器测量决定，不影响桌面。

### 单条终端标签行

- 组成：每标签的未接管图标、名称、紧凑连接/运行状态、悬停/聚焦 X；最后一个标签后 +；行最右侧面板收起 X。标签 X 与面板 X 提示明确、作用不同。
- 多标签允许标签容器内部横向滚动，+ 紧随标签容器末尾并保持可达；面板 X 不随标签滚走。悬停 X 预留固定空间，不造成名称/宽度抖动。长名称截断但可获得完整提示，不横向撑大 document。
- 右键目标显式传 stable terminal ID，不依赖当前选中项。菜单包含接管、刷新终端、重命名、历史/返回实时、桌面上下移动、断线重试、关闭当前/其他/全部；不适用项禁用或隐藏并说明。
- 手机以点击“更多”显示相同业务菜单，除去上下移动；初始目录可在菜单会话信息中查看。单目录 + 直接创建，多目录 dropdown 展示关联目录并标记主目录；仅选取后创建，不把打开菜单当创建。
- 登录态异常/断线/命令失败使用已有 Alert 等组件明确展示；不添加替代旧工具栏的常驻一行。状态图标有中文提示/aria，不只靠颜色。
- 官方组件查找已完成，见 [记录](research/shadcn-components.md)。复用 ContextMenu/DropdownMenu/Dialog/Tooltip/Button/Field/Input/Toggle 等，实施时经 CLI 只添加缺少的 Tabs。不直接引入 Radix。TabsTrigger 内不嵌接管/关闭按钮，使用同级业务容器组合；Tabs 选择不能卸载稳定 runtime。

## 3. 前端运行时与动作归属

现有 `TerminalRuntimeProvider` 按 ID 缓存固定 element 并 portal 渲染真实会话。保留该模式，扩展 scope 中的 typed 状态订阅和动作登记，不新造第二个 socket 或第二套实时终端。

- Runtime entry 发布 connection、ready、role、viewer ID、generation、pending request 等必要状态；输出字节不进 Zustand 或通用 UI store。客户端状态只用于提示，服务器仍复验权限。
- `TerminalSessionView` 登记 takeover、retry、showHistory/returnLive、focus 等动作；标签/菜单/手机键盘共用动作。provider 更新完整显示元数据，修复当前 ensure 仅比较 state 导致 rename 过时的问题，不以显示名为 React key。
- `ensure`、宿主移动、资源刷新不主动发送 takeover。非当前标签需要参与关闭时，在用户明确确认后才准备必要的稳定 runtime/连接；等待有效 ready 后按同一显式意图获取控制，失败如实展示。不能通过预连接整个服务列表提前夺权。
- 接管命令发送后的布尔值不是成功：等待该连接的 ready/control controller 与有效 generation；发生 error、断线、超时、用户取消则结束等待，不自动补发或无限重试。无法确认的部分接管不声称回滚。
- 输入触发确认只识别意图。使用 xterm 公共事件/输入 textarea 的捕获阶段拦截观察端键盘、beforeinput、paste、IME、手机快捷按钮等；观察端保持禁止 stdin，绝不为了收到 onData 而提前开写。提示期间丢弃所有触发内容，不保留文本/字节。成功后恢复焦点，由新输入进入已有 sendInput，仍按 generation 与 ready 判断。
- 鼠标选择、复制、滚动、链接点击和纯修饰键不作为命令输入意图。断线不能假装接管可用，错误展示重试入口。输入内容不进入日志或确认文案。
- Dialog 由 scope 层统一编排且 portal 到 body，不随隐藏宿主消失；同 request ID 合并各成员通知为一个名称列表。多个独立批次分别登记，按可访问的对话框队列显示，不能因合并 UI 丢失取消入口。
- 截止和结果来自服务器；浏览器只显示剩余时间，不触发 kill。取消按钮发送请求并等待确认，关闭弹窗不能代替服务器取消。取消失败保留真实 pending 状态。确认终止后刷新真实状态，不能因本地计时到零提前删除入口。

## 4. 名称元数据与自动分配

### 权威与接口

SQLite display_name 是已有记录的显示名权威，tmux session 名仍为随机精确 target。新增 `PATCH /api/v1/terminals/{id}`，候选固定 body：

```json
{"expected_display_name":"终端1","display_name":"构建"}
```

返回既有 terminal DTO，不添加必须的新 DTO 字段。Cookie/Origin/CSRF、稳定 ID 和 JSON 严格校验沿用现有 handler。名称非空、合法 UTF-8、不含 NUL/CR/LF、无首尾空白、至多 200 UTF-8 字节。用预期旧名称做 SQL compare-and-set，过时返回 409；本方案不是可检测 ABA 的完整元数据版本系统，不扩大为通用版本迁移。重复请求在目标已是请求新名称时可返回当前 DTO，不覆写不同的新名称。

名称修改是显示元数据操作，不要求取得 shell 控制权，也不改变 cwd/PID/session ID。允许既有手工重名，自动命名避开所有存续的完全同名显示名。已结束历史仍保留，不以重新编号复活其 ID。

### 协调与恢复

- service 增加窄范围元数据协调锁，串行化 Create 的自动分配、Rename 和 managed-session 对账。分配前取得可靠 tmux 名单并恢复尚未入库的 managed session 元数据，查询失败返回 unavailable，不把故障当空号。
- 按 project ID 查存续名称：先“终端”，再最小空号“终端1”“终端2”……；上下区域/browser layout 不参与分配。已结束不占号，pending/隐藏/移动不释放。项目删除后的 orphan 不迁入其他项目的编号集合。
- 自动名称选取与 tmux 创建/SQLite 插入处于同一 service 协调范围，但不在 SQL 事务中执行 tmux。保留 WithRegisteredFolder 的版本/dev/inode 校验。服务部署仍是单 Web 实例，不新增多实例共享 socket 竞争模型。
- Create 若 tmux 成功而 DB 失败，继续保留并通过对账找回；孤立 managed session 的名字也占用候选，不能误重用。进程本身由 tmux 实际存在性判断。
- Rename 先以 CAS 更新 SQLite，再用固定 argv 更新 session 的 `PERSISTTY_DISPLAY_NAME` 镜像；不执行 shell，不操作 pane 输入或重启。镜像失败不回滚已提交的新名、不 kill 会话；后续对账以 SQLite 修复现有 session 镜像。缺失 DB 的 session 才从镜像恢复。
- 向新协议查看端广播 metadata 名称更新并同步 hub/provider DTO；后续 ready/list 刷新使用同一最新名称。旧协议客户端可通过原列表刷新读取改名，不向旧 exact decoder 推送陌生帧。
- 本方案不需要为计数器建表或修改旧 SQL 迁移。若实现验证发现确需 schema 变化，追加新迁移并重新审查实质变更，不改写已应用迁移。

## 5. 批量终止协调与协议

### 请求与安全边界

新增 `POST /api/v1/terminals/termination-batches`，由新前端完成显式接管确认后调用。body 仅包括去重后的成员 `terminal_id/viewer_id/generation`，不接收 tmux target、display_name、socket 或命令。成员 1..200，沿用 HTTP 有界 JSON body，逐层拒绝未知/重复字段、无效 ID/代次/空集合/重复目标。

- 需要 Cookie/Origin/CSRF。每一 viewer 必须是存活的对应 hub 连接，token 与该 HTTP 发起者认证匹配，当前为 controller 且 generation 相符；不能以同 UID 身份替代控制权验证。
- 开始前按真实 tmux 状态确认全部目标仍存续，任一不可用、已结束、pending 或失权就拒绝整批，零新增 timer/kill；列明失败对象，不跳过目标部分关闭。
- 上下区域目标由工作台 scope 显式冻结。后端按列出的稳定 ID 处理，不能重新展开成“全部终端”；终端所属项目配置被移除仍不推断应关闭进程。用户确认后的新终端不自动加入该批次。
- 返回统一 request ID、UTC deadline、服务器生成的成员 ID/名称快照。名称只是展示；重名可通过目录提示/短 ID 区分，真正终止仅靠精确身份。pending 时改名不会替换成员或扩大集合。
- 成功响应丢失不自动重发终止；从成员 WS ready/pending 恢复同 request。短时结果记录有界保留，必要时通过鉴权 GET 同批次 ID 查询裁决结果；不持久化 pending 到 SQLite，不在 Web 重启后恢复计时。

### 单一批次状态机

运行时建立小型批次 owner，单个终止也作为单成员批次复用裁决，保留现有 WS terminate 入口。状态为 pending → cancelled，或 pending → executing → completed；每 terminal 同时最多属于一个 pending/executing 批次，一个批次只有一个 timer/deadline。

- 创建时一次验证全部成员并登记索引、冻结发起连接与代次，再广播；不向某一成员先发布可执行 pending 后再校验其他成员。
- 任一有效成员查看端在截止前取消，撤销其 request ID 整批；控制转移、发起成员断连/认证失效、hub 关闭也撤销关联整批。不影响别的 request ID。
- 当前和后加入的成员查看端均收到同批 deadline/成员列表；隐藏/上下移动不断 WS、不撤销。任一成员有效 observer 可以取消整批，不要求持有所有成员的控制权。
- 截止时重新验证全部发起者/连接/代次/认证，串行决定整批进入执行还是整批撤销。取消与截止竞争在同 owner 内裁决；取消已成功的 request 永不执行。到期后不能承诺已执行的进程可回滚。
- 外部 Kill/SessionExists 在裁决锁外执行，使用固定小并发和整体有界 context，不能在全局协调锁中等待 N 个 tmux 命令阻塞无关批次撤销。每成员只尝试一次精确 kill，然后查询真实状态；未确定结果标 unavailable，不自动重试 kill、不称整批原子销毁。
- 全部结果展示逐成员 state/error；只有确认 terminated 才按成功处理。取消或运行中/不可用结果保留入口与可见错误。
- 批次结果/成员索引/事件大小均设有界上限，沿用背压断连接规则；完成后清理 timer/成员占用，短时结果只保留少量记录供重复取消与结果查询。Web 退出丢弃 pending，不执行补偿终止。

### 锁与生命周期

现状 `runtime.go` 存在 runtime map 与 hub mutex；`termination.go` 在 hub 锁内执行 kill。新增批次不得在已有 hub 锁内反向获取协调锁。

明确锁序：控制/批次协调锁 → 必要时 runtime map 锁 → 按稳定 ID 排序的 hub 锁。Connect/Close/Takeover/pending 生命周期和 timer 裁决统一遵守；输出和 Input 可按现有 hub 锁使用，但不得回调批次 owner 形成逆序。所有外部认证与 tmux I/O 放在协调锁外，进入裁决时重新核对内存前提。广播使用冻结 DTO 快照，不能读写共享 pending 指针造成 race。实施先补并发测试，再替换旧单目标计时路径。

### 新旧 WS 兼容

现有 v2 Go/TS 解码严格拒绝陌生字段，不能把批次成员直接塞进旧事件。新增前端 stream 显式选择 `?protocol=3`；缺省继续 v2，拒绝未知协议版本。二进制输入/输出和 generation 安全边界不变。

- v3 ready 的 pending 包含 request_id/deadline/members；pending/cancelled/executed 使用同批次身份，executed 提供逐成员结果；新增 metadata 事件仅含 terminal ID/显示名。确切 JSON fixture 在实现第一步锁定，Go/TS 共用。
- v2 保留既有 ready 与事件字段。批次成员上的 v2 查看端收到同 request/deadline 的旧式 pending，以旧 cancel_termination 请求仍能撤销整批；executed 只带该连接目标的 state，不推送 metadata 或新字段。
- 服务端内部按 batch ID 去重；v3 客户端按 ID 合并多个成员通知，取消一次即可，不将多次事件误当新批次。名字列表有界，JSON 控制与输出排队限额补测试。
- 单个/批量的 HTTP/WS 错误按既有 envelope/code 映射，新增错误明确区分角色过时、已有 pending、目标不可用与不存在；不得吞错误或将 JSON 解码失败静默忽略。

## 6. 验证与发布

关键门禁见 [实施清单](implement.md)。除组件单测外，必须有真实浏览器工作台几何/截图、混合 v2/v3 三端取消、并发名称分配，以及 Debian/systemd 原 PID 保留/精确终止实测。仅 mock 不足以验收。

发布时先后端支持 v2/v3，再前端选择 v3；旧缓存客户端不触发协议错误。不修改旧迁移，不触碰用户生产 tmux 资源；任何探针都用独立 root/socket/units。根 `.env` 仅给现有远端验证 helper 读取，报告只含脱敏结果。

回滚前若还有 pending，先通过有效取消确认撤销或停止自有 Web 使其丢弃；不得补偿 kill 或批量删除元数据。恢复旧前端仍可 v2 查看/取消，旧后端不支持新增批次/rename，禁用新入口再回退，不宣称旧服务可继续提供新功能。名称更新是真实持久元数据，不自动改回。此前既有终端输入、history、资源宿主移动和 Web 重启测试必须全部回归。
