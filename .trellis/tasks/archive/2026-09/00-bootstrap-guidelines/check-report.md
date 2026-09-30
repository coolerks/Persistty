# Bootstrap 规范审查报告

> 归档状态：2026-09-30 用户明确要求标记完成并归档；本任务已完成收尾。

审查日期：2026-09-26。范围：26 份 `.trellis/spec` 文档，以及本 task 的 PRD、design、implement、研究证据和上下文清单。仅审查文档；没有产品源码或依赖安装。

## 结论

结构、中文、覆盖及机械检查通过。发现的 JSON 示例错误已修复；WebSocket 浏览器认证判断、secret 表述、七段结构范围及外部工具读取边界已由主会话修订并复验。无尚未解决的 bootstrap 文档发现，文档质量门禁通过。本报告不构成产品运行时安全或 acceptance 证明。

## 已修复的发现

- 文件：`.trellis/spec/backend/http-api.md:17`。
  问题：JSON 字符串 `content` 内含实际裸换行，示例不能由标准 JSON decoder 读取。
  修复：审查代理替换为 `hello\n`；5 个 JSON fenced blocks 均经 `json.loads` 验证。
- 文件：`.trellis/spec/frontend/clients.md:7`、`.trellis/spec/backend/websocket-protocol.md:27`。
  问题：原规范要求按 WS 握手 HTTP 401/403 停止重连，但浏览器原生 WebSocket 不向脚本提供该 HTTP 状态，不能直接按此实现。
  修复：主会话补充同源 session probe、已升级连接 1008 停止、未知原因有限重试后手动操作；前后端规则一致，已复验。判断依据为 [WHATWG WebSockets Standard](https://websockets.spec.whatwg.org/) 的握手失败与脚本可观察信息规则。
- 文件：`.trellis/spec/backend/http-api.md:26`（JSON 修复后行号）。
  问题：返回 `csrf_token` 同时声明“任何 secret 不在 JSON”，范围含糊。
  修复：主会话明确密码、password_hash、session secret 禁止；CSRF token 可在已认证响应返回但禁止日志/持久 UI store。与 security-config、logging/state 规则一致，已复验。
- 文件：`.trellis/tasks/00-bootstrap-guidelines/design.md:13`。
  问题：“新契约按七段结构”没有区分权威跨层契约和前端消费指南，结构验收范围不明确。
  修复：主会话明确后端权威基础设施/跨层契约使用七段，前端引用而不重复签名与错误矩阵；7 份权威契约均有完整结构。

### 外部工具的读取边界（主会话已解决并复验）

位置：`.trellis/spec/backend/filesystem-guidelines.md:14`、`.trellis/spec/backend/process-guidelines.md:3`、`.trellis/spec/backend/transfer-search-git.md:17` 和 `:19`。

证据：规范正确指出 `cmd.Dir` 不能防路径竞态，但当前解决表述是“工具遍历结果仍须安全根下复验再读取/修改”。普通 rg 搜索已经读取目标文件并产生 preview；事后复验无法撤销此前的读取。若父目录或 cwd 在检查后被移动/替换，不能仅据后置检查声称工具执行本身满足“最终 I/O 不越界”。这是初始规范的设计缺口，并非已发现的产品代码漏洞。

修复：主会话新增 process-guidelines 的“File API 工具访问边界”，允许经真实 Debian 测试证明的受限只读执行，或 root 句柄控制发现/打开后安全 snapshot 输入；禁止整个 HOME 许可、不安全 fallback，保留 ignore/Git 语义，不满足时不开放 API。filesystem 和 transfer 同步引用；preview 从相同安全 snapshot 重算并绑定 version；新增扫描期间 symlink/父目录交换根外 sentinel 断言，要求内容未被读取。研究记录补充此区别。

审查代理未自行修改原因：涉及外部进程、安全边界和公共实现契约的架构判断，已由主会话处理。复验规范没有把 Landlock 名称或 snapshot 路径当作实测证据，具体设计仍由 owning task 锁定并验证，因此符合 bootstrap 的初始契约边界。

## 尚未修复的发现

无。历史/live 顺序、Debian cgroup、工具隔离、Go 版本/driver、上传配额具体值、资源许可证/glyph 等均已明确归属后续任务并有前置门禁，不视为本次已验证实现。

## 覆盖与一致性

- 后端包/Gin/service/context/error/logging、SQLite 迁移/事务/连接权限、配置与单密码 Argon2id/session/CSRF/Origin/存量 WS 撤销均有明确归属。
- Terminal 长期任务归 tmux，bridge 清理与显式 Close 分离；SQLite 与 tmux reconciliation 不误杀。systemd 独立 unit/cgroup、禁止 Web 自动启动 tmux server、Web restart PID/start-time 断言和主机 reboot 边界明确；Debian 实测仍是后续 Spike 门禁。
- 文件根句柄、symlink/special files、强 hash/版本复验、同目录暂存/权限/rename，以及任意外部 writer 的 CAS 窗口均有规定，不虚称 hash+rename 消除所有竞争。
- 上传 chunks 持久状态、Web restart resume、后端 hash、幂等完成/崩溃协调、quota、empty dirs、keep-both 原子占名；ZIP 不 follow symlink、有界流式读取；Search ignore/binary/.git、preview 与版本冲突；只读 Git 与参数注入测试均覆盖。
- 前端 server truth/UI/draft 分离；Monaco model/undo/diff/保存 generation；xterm bytes/resize/history/StrictMode/只 detach；theme/字体/图标 license 的初始约定与待实测项明确。
- 完整 Explorer/产品 Milestone 的操作清单属于 bootstrap 后的父/子任务，不以本次规范审查替代产品需求和最终 acceptance。

## 验证记录

- 文档机械 lint：PASS。最终 Python 内容检查读取全部 26 specs，63 本地 Markdown 链接存在；5 JSON blocks 可解析；末尾换行及尾随空白无错误。
- 七段权威契约：PASS。database、filesystem、HTTP、security-config、terminal-lifecycle、transfer-search-git、websocket-protocol 各有 1..7 结构、字段、错误/案例/测试断言。
- 模板扫描：PASS。`rg -n 'TODO|TBD|To be filled|Fill in|待补充|待填写|占位模板' .trellis/spec` 仅命中两处“不能建立第二份 TODO”的约束，人工复核为正常正文，非占位。
- 上下文：PASS。`python3 .trellis/scripts/task.py validate 00-bootstrap-guidelines` 返回 implement.jsonl 1 entries、check.jsonl 27 entries，exit 0。
- `git diff --check`：PASS，exit 0。仓库新增初始化目录尚未 tracked，该命令单独不足以验证新文件；上述逐文件检查已覆盖内容。
- 产品 Lint：N/A。没有 Go/frontend 产品源码和 package manifest。
- TypeCheck：N/A。没有 TypeScript 产品代码或 tsconfig；未声称通过。
- Tests：N/A。没有产品 unit/integration/E2E；没有运行或声称 Go、frontend、Playwright、Debian/systemd acceptance 通过。

未 commit、archive 或编辑 Trellis framework。主会话可继续 bootstrap spec update/finish 和任务生命周期；后续产品开发仍须按各 owning task 执行运行时门禁。
