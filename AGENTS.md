<!-- TRELLIS:START -->
# Trellis 项目指引

本指引适用于在本项目中工作的 AI 助手。

本项目使用 Trellis 管理开发流程。工作所需的知识位于 `.trellis/`：

- `.trellis/workflow.md` — 开发阶段、创建任务的时机和技能路由。
- `.trellis/spec/` — 按包和层组织的开发规范；修改某一层的代码前必须阅读对应规范。
- `.trellis/workspace/` — 各开发者的日志和会话记录。
- `.trellis/tasks/` — 活动及归档任务，包括需求文档（PRD）、研究资料和 JSONL 上下文。

若当前平台提供 Trellis 命令（如 `/trellis:finish-work`、`/trellis:continue`），优先使用命令执行流程。并非所有平台都提供全部命令。

使用 Codex 或其他支持代理的工具时，可在以下位置查找项目级辅助资源：

- `.agents/skills/` — 可复用的 Trellis 技能。
- `.codex/agents/` — 可选的自定义子代理。

此区块由 Trellis 管理。区块外的修改会保留；区块内的修改可能在后续 `trellis update` 时被覆盖。

<!-- TRELLIS:END -->

## 单代理执行约束

- 禁用 subagent。研究、实现、审查、测试和规范维护均由当前主会话直接完成；不得调用 spawn_agent、spawn_subagent、trellis_subagent 或通过 trellis channel/独立 AI worker/其他会话转交工作。
- 不因任务复杂、文件数量、并行效率或“独立检查”自动启用子代理；遇到容量限制时缩小批次、明确进度或向用户说明。
- Trellis 使用 `.trellis/config.yaml` 的 `codex.dispatch_mode: inline`；实现前读取 `trellis-before-dev`，检查使用 `trellis-check` 技能，而非 Agent 类型。原 JSONL 和历史代理报告保留为资料，不代表当前授权。
- 仅用户明确要求解除禁用时才能重新配置。普通“继续”“开始实施”“检查”“调研”不构成解除授权。详细契约见 [.trellis/spec/execution-policy.md](.trellis/spec/execution-policy.md)。

## 项目语言与文档兼容性

项目文档、开发规范和用户界面说明使用简体中文。文件名、引用路径、代码标识符、协议字段、命令和机器解析标记保持原有形式。具体文档维护规则见 [.trellis/spec/guides/index.md](.trellis/spec/guides/index.md)。

## 项目定位与当前状态

Persistty 是面向 Debian 开发机的单用户、自托管 Web IDE，提供持久化终端、文件浏览与编辑、搜索替换、上传下载、只读 Git 查看及后端认证。

W01 工程基础、W02 Debian 实验和 W04 多文件夹项目与文件管理已完成并归档。W03 已接入真实 tmux/PTY/WS 与 xterm，持久性、多端单控制权、倒计时终止、桌面与手机浏览器验收通过；证据见终端运行时规范及对应任务验收报告。桌面 Monaco 和可调整布局已接入，移动端使用基础文本视图；W05 已实施自动保存/草稿、四组布局与本地恢复、有界图片预览及 Nerd Font，并通过自动化和真实 Debian 专项；任务仍进行中，真实手机软键盘验收待完成，正式部署仍属于 W08。不要把依赖已安装、界面已接入和运行时已验收混为一谈，也不创建空包充数。

- 后端：Go、Gin、SQLite（`database/sql`），终端使用 tmux 与 PTY；外部工具包括 `git`、`rg`、`tmux`。
- 前端：React、TypeScript、Vite、React Router、Tailwind CSS、shadcn/ui、lucide-react、Zustand；桌面编辑器使用 Monaco，终端使用 xterm，布局使用 react-resizable-panels。
- 部署：Debian、systemd、Nginx；首版需求规划为 WireGuard VPN 内访问。具体部署与认证契约需在实施前完成审查。
- Go/Node 版本、SQLite driver、前端包管理器及依赖版本由工程基础任务验证和锁定；使用实际 manifest 与锁文件，不猜测版本，不混用包管理器。

2026-10-01 用户已明确恢复 W05/W06 统一验收，此前的验收延期安排已结束。[W06 多根搜索替换与只读 Git](.trellis/tasks/10-01-search-readonly-git/prd.md)已接入功能，本轮本机自动化和 Chromium/WebKit 专项通过，修复触屏横屏模式与替换取消焦点；Debian 产物传输待明确目标授权，真实手机、触控板、shell 主题及 Firefox 环境仍待补齐。两个任务保持进行中，不将未执行项记为通过或完成/归档。结果见[统一验收进展](.trellis/tasks/10-01-search-readonly-git/acceptance-report.md)，实际协议见[W06 契约](.trellis/spec/backend/search-git-contract.md)。

## 需求与规范的使用顺序

先读取当前任务的 `prd.md`，再读取存在的 `design.md`、`implement.md` 及相关规范。后端入口为 [.trellis/spec/backend/index.md](.trellis/spec/backend/index.md)，前端入口为 [.trellis/spec/frontend/index.md](.trellis/spec/frontend/index.md)，跨层修改同时读取 [思考指南](.trellis/spec/guides/index.md) 及其对应文档。

当前需求研究见 [已归档需求研究](.trellis/tasks/archive/2026-09/09-26-requirements-research/prd.md)，设计审查稿见 [设计审查稿](.trellis/tasks/archive/2026-09/09-26-requirements-research/design.md)。其中的多文件夹项目、VPN HTTP 部署、终端控制与终止流程、自动保存等内容与初始规范存在差异，设计稿列出了需要更新的 owner 契约。研究稿不代表实现已获批准；遇到冲突先确认最新已批准需求，再同步对应规范，不能自行选取旧默认值或将候选协议视为现有 API。任务归档后通过 `.trellis/tasks/` 查找迁移后的记录，不依赖原路径一直存在。

## 代码组织与职责

目录在对应功能实施时按需建立，详细约定见 [后端目录规范](.trellis/spec/backend/directory-structure.md) 和 [前端目录规范](.trellis/spec/frontend/directory-structure.md)。

- `cmd/persistty/`：CLI 入口；`internal/`：配置、认证、HTTP、终端、文件、搜索、传输、只读 Git 与存储等功能包。
- `internal/storage/migrations/`：受版本管理的 SQLite SQL 迁移；数据库文件与上传暂存属于运行数据。
- `web/src/app/`：应用入口与布局；`web/src/features/`：按功能组织；`web/src/components/ui/`：基础组件；`web/src/lib/`：共享 API 与 WebSocket 客户端。
- `tests/integration/`、`tests/e2e/`：真实集成和端到端测试；`deploy/`：Debian/systemd/Nginx 部署示例。

Gin handler 负责输入校验与响应映射，业务服务不依赖 `*gin.Context`，repository 不依赖 HTTP。外部进程使用固定参数列表和 `context`，不把用户输入拼接为 shell 命令。前端由 app 编排 features，基础 UI 不依赖业务；协议字段和错误语义以对应后端契约为准。

## 必须保持的产品边界

- 终端进程与网页、Web 服务生命周期分离。闭页、隐藏面板、断网、登出或 Web 服务重启不得终止 tmux 任务；界面恢复不得自动重跑命令。应用发起终止必须经过明确操作及已批准的确认流程。普通 tmux 不保证主机重启后恢复运行中的进程。
- 项目或文件夹配置移除不得删除真实文件或终止已有终端。真实文件/进程、持久元数据和浏览器视图分别管理，不能通过清理元数据推断应销毁真实资源。
- HTTP、WebSocket、下载和预览均由后端鉴权，写操作遵守 Origin/CSRF 与文件身份、版本检查。终端 shell 遵守服务用户权限，Web IDE 不构成 shell 沙箱；Web 服务以非 root 用户运行。
- 文件写入必须遵守安全目录访问、冲突检测与原子保存契约；保留未保存输入和草稿，不以自动重试绕过冲突。Git 面板只读，不引入隐式写入或联网操作。
- 密码、Cookie、token、终端输入和文件正文不得进入日志。单文件提权按专门审查后的执行边界实施，不能提升整个 Web 服务权限。

## 验证与版本管理

现有产品工程须按以下门禁实际运行，另检查 Markdown 本地链接、规范一致性、忽略规则和 `git diff --check`。未运行或跳过的检查不能声称通过。

产品工程建立后，按实际目录与工具运行以下检查：

- Go：格式化修改的源码，运行 `go test ./...`、`go vet ./...`；并发、终端 bridge、watcher 或 session 变更补充 `go test -race ./...`。
- 前端：在 `web/` 使用锁定的包管理器执行实际 `package.json` 中的 `lint`、`typecheck`、`test`、`build` 脚本；单测方向为 Vitest/Testing Library，端到端为 Playwright。
- 终端持久性和部署行为须在真实 Debian/systemd 环境验证。测试资源隔离并仅清理自己的进程、socket 和 tmux 会话；缺依赖或跳过不能记为已验收。

保留 `go.mod`、`go.sum`、前端锁文件、SQL 迁移、部署示例、测试 fixture，以及 `.trellis/`、`.agents/`、`.codex/` 中受版本管理的项目资源。依赖、构建结果、报告、本地 `.env`/根目录配置及 `/data/`、`/tmp/`、`/logs/` 等运行目录按 `.gitignore` 排除；这些运行目录是本地约定，部署路径仍由后续配置确定。可提交的环境模板使用 `.env.example` 或 `.env.*.example`，配置模板使用 `config.example.yaml` 等明确示例名称，并只包含虚构值。
