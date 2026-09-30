# 工作台当前 UI 证据

日期：2026-09-30。只读代码检查，未修改前端；已完成浏览器只读尺寸复现，见 [测量记录](browser-geometry.md)。

## 当前结构

| 位置 | 当前职责 | 用户标注 |
| --- | --- | --- |
| `web/src/app/App.tsx:39` | Shell 顶栏：品牌、主题、登出；所有路由共用 | 与项目标题行合并 |
| `web/src/features/workspaces/ProjectWorkbench.tsx:65` | 项目标题与编辑项目按钮，居中 | 靠左 |
| `web/src/features/terminal/TerminalWorkspace.tsx:59` | 标题、创建目录选择、新建、刷新、收起 | 删除独立行 |
| `web/src/features/terminal/TerminalWorkspace.tsx:75` | 会话标签、运行状态点、拖拽移到上方 | 保留标签 |
| `web/src/features/terminal/TerminalWorkspace.tsx:84` | 初始工作目录、状态、移到上方、删除 | 删除独立行 |
| `web/src/features/terminal/TerminalSession.tsx:259` | 连接/角色、接管、重试、历史、终止 | 删除独立行 |
| `web/src/features/workspaces/ProjectWorkbench.tsx:89` | 底部状态栏及侧栏/终端切换 | 必须保留在视口内 |

顶部 Shell 还服务登录、项目列表与全终端路由；本次顶栏合并限定项目工作台，其他路由保留既有功能，不扩大为全部页面重设计。

## 整页溢出排查边界

- `web/src/app/styles.css:76` 已有 workspace shell 的 `100dvh` 与 `overflow:hidden`，不能假定仅补这条规则即可修复。
- `web/src/app/styles.css:81`、`:87` 是工作台 flex/group 尺寸约束；`:155` 是 runtime host/mount 的 flex 与百分比高度；`:170` 是底部状态栏。
- runtime 使用稳定 DOM element 与 portal，见 `web/src/features/terminal/TerminalRuntime.tsx:60`、`:73`；未来布局调整必须保留实例和连接。
- 当前 `web/tests/e2e/terminal.spec.ts` 覆盖真实终端交互/接管等，但代码搜索未找到 document scrollHeight/clientHeight 的整页溢出断言。需补运行时几何验收，不能从源码或截图单独确定尺寸链根因。

## 已有能力与约束

- 删除工具栏不等于删除操作：创建目录选择对多根项目有意义；接管是既有明确授权动作；移动、终止、历史与手动重连仍须可达。
- 初始工作目录字段不是 shell 实际 cwd；提示中不得冒称实时 shell 所在目录。
- 控制权/连接状态来自 runtime，选中会话信息来自列表；迁移时需明确 terminal ID，不将全局状态栏误当所有终端的状态。
- 组件/状态迁移以 [前端终端契约](../../../spec/frontend/terminal-runtime-contract.md) 与 [组件规范](../../../spec/frontend/component-guidelines.md) 为准。
- 原始 UX 决策见 [UI 原型记录](../../09-26-requirements-research/research/ui-layout.md)；本任务仅按新标注调整，不擅自扩展为完整 IDE 或重新选择技术栈。

## 新增标签交互的协议差距

用户补充 [标签/目录菜单原型](ui-terminal-tabs-menu.png)，明确提出重命名、自动编号、三种接管入口、批量关闭及名称列表；属于本任务新增范围，不能把后端缺能力当成已实现。

- 创建默认名固定为“终端”：`internal/terminal/service.go:39`；允许创建时提供 display_name，但没有自动编号。现有 SQLite 插入在 `internal/storage/terminals.go:32`，稳定 terminal ID 与显示名独立。
- 路由 `internal/httpapi/router.go:139`..`:143` 仅终端列表、创建、详情、历史与 stream，没有终端 rename 或批量关闭端点。
- runtime 当前 `ensure` 只在 state 改变时替换已缓存 DTO：`web/src/features/terminal/TerminalRuntime.tsx:29`。新增重命名需同步显示元数据而不销毁 element/WS；tmux 中的发现元数据也要与 SQLite 权威更新保持一致，不通过改进程身份实现改显示名。
- `web/src/features/terminal/TerminalSession.tsx:78`、`:132` 禁止观察端 stdin，`:117` 处理 onData；仅修改报错文本不足以支持输入触发确认，需研究键盘/粘贴/手机输入意图捕获且绝不绕过服务端控制权。
- `web/src/lib/ws/terminal.ts:140`..`:146` 只接受单终端 takeover/resize/terminate/cancel_termination。`internal/terminal/termination.go:19` 复验当前 viewer/generation；控制转移、断线或认证失效取消单目标 pending，任一有效查看端可取消。
- 对话框 `web/src/features/terminal/TerminalSession.tsx:285` 泛称“此终端”，没有名称列表。批量关闭不能同时弹出 N 个旧 Dialog 就称为已完成用户要求的统一列表/取消体验。
- 既有不重放约束在 [终端协议](../../../spec/backend/terminal-runtime-contract.md)、[W03 PRD](../../archive/2026-09/09-29-terminal-runtime/prd.md)；用户明确的是确认成功后可正常输入，没有明确授权补发提示前的回车、快捷键或粘贴命令。

批量目标权限的 UX 已确认：明确确认批量接管、全部取得控制权后才能统一计时，失败不启动该批次。`web/src/lib/ws/terminal.ts:140` 的 sendControl 返回值只表示本地可以发送，不是服务器接管成功；需等待 control/ready 的有效角色和 generation。

输入确认处理已确定为不保留/不补发：只捕获输入意图触发确认，不形成待发送队列。接管成功回到终端重新输入；取消、失权、失败、重连均不得补发首次按键/粘贴。

批量范围已由用户明确为上下各自独立，下方菜单不处理上方，上方菜单不处理下方；终端菜单不终止文件标签。当前下方列表 `web/src/features/terminal/TerminalWorkspace.tsx:30` 会排除上方 terminal ID；上方标签由 `web/src/features/workspaces/ProjectWorkbench.tsx:34`、`:106` 单独编排。实施时按确认的区域建立并列明目标集合，不能把服务端列表所有项目作为“全部关闭”。

批次取消已确认：A/B/C 同批时 B 被接管，A/B/C 全部取消；其他独立批次 D/E 不受影响。现有 `internal/terminal/runtime.go:307` 在目标控制转移时取消该单会话 pending；`internal/terminal/termination.go:37` 由任一有效 viewer 取消该单会话，尚无“撤销整个批次”协调。统一截止前取消的状态裁决必须由服务器保证，不能以关闭前端 Dialog 代替取消真实计时器。实际进程终止不可回滚，设计不能把截止后的 tmux 逐目标执行宣称为可回滚的进程事务。

编号当前无独立序号字段：`internal/storage/migrations/0002_resource_ids.sql:10` 的 terminals 表只有稳定身份、显示名、项目关联、cwd 与创建时间。用户已选择按项目独立、上下共享、复用最小空号，并举例下次叫“终端1”。服务端需协调分配，不能由浏览器根据本端上下标签计数。现有 terminated 元数据保留用于历史；名称空位与元数据/稳定身份是否仍存在必须分离，不能为了复用编号删除旧记录或复活旧 ID。迁移方式由 design 决定。

“刷新终端”的现有证据：`web/src/features/terminal/TerminalWorkspace.tsx:22` 与 `web/src/lib/api/use-resource.ts:24` 的 refresh 只重新 load 列表；`web/src/features/terminal/TerminalSession.tsx:190` 的 retry 才会 dispose 浏览器 socket 并重新 connect（不重启 tmux）。用户已明确保留前者，仅列表/状态刷新，不刷新画面/WS。手动重连与历史刷新不得混入该菜单动作。

手机当前使用 `web/src/features/workspaces/ProjectWorkbench.tsx:66` 的单内容视图，终端复用 TerminalWorkspace，快捷键栏由 `web/src/app/styles.css:183` 显示。用户已确认同步精简，采用“更多”菜单承接桌面右键/双击/hover 操作，同时保留单内容与 Ctrl/Alt/方向等快捷操作；不能把桌面 hover 入口直接当手机可用。

用户已确认其他入口收纳：历史/移动/断线重试进菜单，面板最右侧 X 仅收起，目录提示与状态图标替代常驻行，断线错误明确显示。无阻塞性产品问题；必要协议和 owner 变更见本任务 design/implement。当前没有修改产品源码或现有权威 spec。
