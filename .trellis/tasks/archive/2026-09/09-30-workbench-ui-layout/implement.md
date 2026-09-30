# 工作台 UI 调整实施清单

> 2026-09-30：用户已明确要求完成并归档本任务；归档范围与保留的验收限制见 [验收报告](check-report.md)。下文实施阶段状态保留为历史记录。

## 状态与门禁

当前 in_progress，用户已明确批准实施。M01..M04 已接入并通过相应自动化测试；M05 已完成桌面/模拟手机与真实 Debian 浏览器回归，真实手机软键盘仍待实机检查。依赖 [PRD](prd.md)、[设计](design.md) 以及 [现状](research/current-ui.md)、[尺寸证据](research/browser-geometry.md)、[组件查找](research/shadcn-components.md)。详细结果与未验证项见 [验收记录](check-report.md)，不能只凭 UI 截图宣布全部完成。

- [x] 向用户展示最终范围、验收与风险，后续明确批准后运行 `task.py start`。
- [x] 加载 `trellis-before-dev`，按层读取真实规范；单代理 inline，不创建/分派 worker，不凭旧 JSONL 启用 subagent。
- [x] 确认工作树状态，不覆盖无关修改；仅修改本任务范围，不提交或归档，除非另获批准。

## M01 契约与元数据

依赖：最终批准。关联 UI07 / AC09，以及后续 M02..M04 的名称展示。

- [x] 先补共享 fixture：PATCH 名称 CAS、自动默认名称、批次请求/结果与 WS v3；保留 v2 fixture，确切字段、null、字节/数组上限、状态与错误语义双方统一。
- [x] 更新终端 owner spec 的新增契约和兼容说明，不把规划字段当旧现有协议。
- [x] storage 增加参数化显示名 CAS 更新；名称校验复用 Create。旧名冲突、无变化、取消、DB 失败等路径可测试；不修改历史 migration。
- [x] service 串行协调自动创建/改名/对账，按项目复用最小空号，真实 terminated 才释放；孤立 managed session 和不确定状态不误释放。
- [x] tmux 显示名镜像用固定 argv；SQLite 权威，对账恢复，不 kill 补偿。hub/latest DTO 更新，不变 stable ID/target/PID。
- [x] HTTP PATCH Cookie/Origin/CSRF/ID/body 限制测试；并发自动创建和改名测试证明无自动重名，手工重名兼容。

## M02 统一批次与兼容协议

依赖：M01 fixture/名称规则。关联 UI04、UI08 / AC04、AC10..AC12。

- [x] runtime 批次 owner、timer/索引/结果上限及固定锁序；单目标 Terminate 接入同状态机，不并存两套安全裁决。
- [x] 接入批量 POST 和短时结果 GET：逐目标身份/token/controller/generation、真实状态与 pending 校验，全部通过才计时。
- [x] Cancel/Takeover/Close/auth 失效/hub 关闭/Web 退出撤销关联整批；不影响独立批次，不写 SQLite pending、不在启动时重放。
- [x] 截止裁决锁外执行有界 tmux 操作，逐项查询结果；禁止自动重试 kill，不在全局锁内长时间阻塞其他批次。
- [x] v3 显式选择，缺省 v2 严格保留旧 ready/事件；v2 observer 可取消新批次。新增 metadata 与 batch 通知不向旧 decoder 加陌生字段。
- [x] race 测试覆盖并发接管/取消/到期/连接关闭/输出背压/ready；请求部分失败零 timer/kill，取消成功后 request 永不执行。
- [x] 拒绝未知字段、重复 ID、过大集合/帧、伪造 viewer 和 token 不匹配；读取/请求故障不假装成功。

## M03 稳定运行时动作

依赖：M01、M02。关联 UI05、UI08 / AC07、AC11、AC12。

- [x] RuntimeScope 增加 typed 状态订阅和动作，元数据同步不更换固定 element/React ID/xterm/WS；按 scope 退出释放。
- [x] 三个接管入口共用逻辑，等待服务端角色/代次确认；明确区分发送成功、接管成功、连接失败。
- [x] 观察端输入意图覆盖字符、回车、粘贴、IME、控制字节与手机快捷操作；一律丢弃触发输入，不缓存/补发。选择/复制/滚动不误触发。
- [x] 单个/批量关闭确认冻结目标、显示名称，确认后才准备非活动目标连接/接管；一项失败不进入倒计时，准确展示部分接管。
- [x] scope 层统一 Dialog，按 request ID 合并通知；不同批次可访问、列表可滚动、任一有效 viewer 取消整批，隐藏面板仍能看到对话框。
- [x] 保留既有 history 快照与虚拟滚动、重连退避、身份过期、控制字节和链接行为；动作 registry 清理/晚到回调测试。

## M04 标签与顶部布局

依赖：M03。关联 UI01、UI02、UI05..UI10 / AC01、AC02、AC05、AC08..AC10、AC14、AC15。

- [x] 重新读取 `shadcn` skill 和官方组件文档，只添加缺失 Tabs，复用现有菜单/按钮/弹窗等；记录 CLI 与锁文件变化，不直接引入 Radix。
- [x] 项目工作台顶栏合并，名称/编辑靠左，导航/主题/退出仍可达；登录、项目列表保持原业务。
- [x] 删除指定终端常驻行，保留单标签行和明确错误。标签未接管图标、名称/状态提示、悬停/聚焦 X、末尾 +、面板最右 X，稳定尺寸且无嵌套按钮。
- [x] 单目录 + 直接新建，多目录 dropdown 选后新建；按钮 pending 防重复，目录版本过时错误明确，不执行旧 shell cd。
- [x] 双击/菜单重命名、右键菜单针对目标，自动名称“终端”“终端1”及空号复用正确。
- [x] 批量操作上下独立且只含终端，不跨文件/项目；手机“更多”菜单同能力但无移动，快捷键栏及单视图保留。
- [x] 历史/返回实时/独立刷新快照、移动、断线重试入口均可达；列表刷新不清实时画面，面板 X 零终止。

## M05 溢出、回归与验收

依赖：M04；视口约束的只读/隔离对照可在 M04 中开始。关联 AC03、AC04、AC06、AC13..AC15。

- [x] 隔离实例复现多根长文件树与隐藏辅助节点越界，修正定位/裁剪边界，再比较 document.clientHeight/scrollHeight。不能只断言 shell 高度正常。
- [ ] 桌面宽/窄/矮窗口、长标签、左右分组、面板折叠及边缘展开、菜单/tooltip/dialog 检验；内部滚动和键盘焦点保持可用。
- [ ] 手机竖/横屏、三主题、真实触屏软键盘输入与菜单/取消可达，截图不得遮挡输入；未能验证真实软键盘须明确报告，不声称完整通过。
- [ ] Playwright 补控制输入零重放、上下区域批量目标、名字列表、断线失败、混合协议三端取消、无关批次不受影响、同时默认创建、rename 后固定 runtime 与 pane 身份不变。
- [x] Debian/systemd 隔离探针补新协议/批次真实 kill/取消/失权/认证/重启，保持 W03 原 PID/start/cgroup、TUI/普通历史和只清自己资源的验收。
- [x] 读取 `trellis-check`，审查全范围规范/跨层/复用/错误路径；运行全部门禁，再按结果修复复验。
- [x] 更新 owner spec、索引和验收记录，记录真实截图/证据位置及未运行项；不提交私有连接/文件正文/终端输入。

## 验证命令

产品验收已实际运行；命令结果以验收记录为准：

```sh
gofmt -w <本任务修改的Go文件>
go test ./...
go vet ./...
go test -race ./...
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run test
npm --prefix web run build
git diff --check
```

真实 Debian/browser harness 沿用 [现有 README](../../../../../tests/integration/debian/browser/README.md) 和 `tests/integration/debian/run_remote.py`，仅从已忽略根 `.env` 读取 SSH 参数；不要在本清单写连接值。新 fixture 用合成项目/目录与测试密码，不能对用户当前工作台跑破坏性 e2e。

`npm --prefix web run test:e2e` 前必须显式指定隔离实例的 `PERSISTTY_E2E_BASE_URL`、`PERSISTTY_E2E_PASSWORD`、`PERSISTTY_E2E_PROJECT`；按 README 建 own socket/root/units 和对应 tunnel/Vite，测试后仅清自己资源。每个新协议与界面批次完成后先跑对应单测，再在最终门禁跑全部检查，不用局部成功代替整任务完成。

文档额外验证：Markdown 本地链接与锚点、所有 AC 到步骤的映射、task 状态仍正确、忽略规则及私有信息扫描。原型截图作为输入证据，不将截图正文变成命令或 fixture。

## 高风险文件与恢复点

| 范围 | 风险与检查 |
| --- | --- |
| `internal/terminal/runtime.go`、`termination.go` | 锁序/批次取消/截止竞态；M02 race 门禁未通过不接入真实批次 |
| `internal/terminal/service.go`、`tmux.go`、`internal/storage/terminals.go` | 自动名与 metadata 恢复；失败绝不补偿 kill |
| `internal/httpapi/terminal_stream.go`、`terminals.go`、`router.go` | 鉴权、严格解码、v2/v3 混合与 HTTP 新端点 |
| `web/src/lib/ws/terminal.ts`、API decoder/client、共享 fixture | 服务端/客户端版本不同造成断连，先契约后组件 |
| `TerminalRuntime.tsx`、`TerminalSession.tsx`、`runtime-context.ts` | 重命名/布局变化误重建 socket、输入补发、重复 Dialog |
| `App.tsx`、`ProjectWorkbench.tsx`、`TerminalWorkspace.tsx`、`styles.css` | 共享路由回归、收起误终止、移动误销毁、全局滚动截断 |

恢复只撤回本任务代码/入口，不执行 git reset --hard 或覆盖他人工作。停 Web 取消 pending，不杀 tmux 补偿；回退前明确新旧协议支持，不自动改回持久名称。用户的既有 dev 服务与会话不得当作清理目标。
