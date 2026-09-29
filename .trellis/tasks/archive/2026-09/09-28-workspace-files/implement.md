# W04 单代理执行计划

用户在父规划批准 W04 后，又明确要求推进到 W04 完成，并确认目录选择器同时支持浏览和输入。任务现为 in_progress；此处原有 planning 门禁已在实施前完成。全程遵守项目 inline/无子代理约束，主会话直接研究、实施和检查，不复用旧 JSONL 代理分派。W01/W02 已归档，正式 W04 不能将 W02 的合成 CLI 探针当生产防护。

## 顺序与交付

1. **契约与规范**：读取 `trellis-before-dev` 与后端目录/DB/HTTP/文件/传输/质量、前端组件/状态/客户端/类型/质量及跨层指南；先固定 W04 的 JSON/错误 fixture、路径/version/identity、配置版本及端点保护矩阵。更新旧 owner spec 中 workspace_id、keep_both、目录上传/偏好状态等与最新父需求冲突的候选文本，不修改已应用迁移。
2. **项目和目录选择**：新增 SQLite 0003 注册根身份与项目 CRUD repository；受鉴权的服务器目录浏览/输入验证；Gin 创建/修改/移除 API、expected_version 冲突、只删元数据。前端项目面板与目录选择器接真实 API，稳定路由和 404 保留。
3. **安全文件内核**：先实现 Linux dirfd/openat2 受限解析与本地可测适配，目标文件身份/强版本、内容分类、根变化拒绝；再开放分页 tree/content/download。测试 symlink、bind mount/特殊文件、同名/重叠根、根移出及外部writer，缺能力 fail closed。
4. **普通文件操作**：固定 operation union、同根安全父句柄 rename no-replace、跨根 copy/move 部分结果、删除预览/token/复验、深度/数量/时间/取消上限。先真实 Go/HTTP/SQLite/目录集成，再接 Explorer 新建、菜单、拖拽、确认和移动端操作。
5. **传输**：配置容量与受版本管理的上传/ZIP状态迁移，私有 staging；块传输/完整 hash/幂等/重启恢复/配置失效与目标确认，桌面目录上传/手机文件上传；文件下载与完成后可取的目录 ZIP。每一项先后端集成与真实解压校验，再接前端进度/取消/部分结果。
6. **外部变化**：扩展已有鉴权边界建立 events WS，按展开目录管理非递归 fsnotify，丢失/overflow/重连触发重列；前端合并事件与请求版本，保持未来编辑缓冲区独立。若文件系统不支持事件，显示轮询兜底状态，不伪称强实时。
7. **全范围复核**：运行 root Go test/race/vet、Linux build/integration、npm lint/typecheck/test/build、共享 fixture 与注册级 HTTP 权限矩阵；启动本地服务做桌面/手机浏览器关键路径截图/交互检查。真实 Debian 仅在既有授权的普通用户隔离临时目录测试，不改系统服务、不对用户真实项目做破坏性试验。按 `trellis-check` 自查，再更新规范/报告；提交与归档走阶段 3.4/finish-work 的用户确认。

## 中间门禁与回滚

- 每一步必须有正向和负向测试，不因 UI 等待后端就注册假成功 API。项目配置移除先于文件副作用则旧请求失败，文件副作用先接受则结果可追溯；上传/ZIP长任务不持项目锁。
- `os.Root` 句柄跟随移出的目录、Landlock 系统白名单、bind mount/魔术文件、硬链接、rename覆盖、外部writer窗口是重点威胁。当前设计若不能通过真实 Linux 负面测试，应停止对应写 API，修改设计并重审，不以路径字符串或事后过滤放行。
- 不以 `rm -rf` 清理非自身路径；测试只在本机/目标机新建私有 fixture 中操作。DB 迁移失败保留旧库并拒绝服务；已经成功的真实文件副作用不可用 Git 撤回，必须记录结果及人工恢复路径。
- W04 验收对应 PRD A01–A08；W03 终端、W05 编辑/草稿、W06 搜索Git、W07 提权和 W08 部署不能作为 W04 的隐式“已通过”。若依赖的用户决策或目标机授权改变，回到 planning 更新 PRD/design 后再实施。

## 2026-09-28 前端纠偏记录

用户指出初版前端技术栈和原型均偏离约束。本轮将 `web/components.json` 设为 shadcn 官方 `base-nova`，通过 CLI 生成/更新 `web/src/components/ui/`，移除直接 `radix-ui` 依赖及调用；业务组件复用生成的 Button、Dialog、Input、Select、DropdownMenu、ContextMenu 等。工作台按原型建立活动栏、多根树、上方文件标签/左右拆分、下方可收起面板、状态栏、手机单视图；使用 React Router、Zustand、react-resizable-panels 和桌面 Monaco。该骨架不扩大 W04 的完成声明：文件内容只读，W05 负责自动保存/草稿；终端面板明确标示尚未接入，W03 负责 xterm/tmux 运行时。`@xterm/xterm` 已锁入依赖但未被渲染或验收，不得据此称终端可用。

已运行前端 lint/typecheck/test/build、Go test/race/vet、Debian 隔离 W04 探针及浏览器桌面/手机检查；最终 W04 验收仍须逐项核对 PRD A01–A08，并保留提交/归档门禁。
