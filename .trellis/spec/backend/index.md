# 后端开发规范

## 适用状态
Persistty是Debian单用户浏览器终端与轻量文件工作台。W01 已有配置、认证、SQLite 和元数据读接口，见[基础协议](foundation-contract.md)；W04 项目与文件接口见[W04 契约](workspace-files-contract.md)。W03 真实终端已通过 Debian/systemd 与浏览器验收，见[运行时契约](terminal-runtime-contract.md)；W05 自动保存/草稿仍未交付。规范用简体中文，标识符/协议字段保持英文。

## 规范索引
| 文档 | 归属 |
| --- | --- |
| [目录与边界](directory-structure.md) | Go/Gin/service/domain |
| [配置与认证](security-config.md) | Argon2id、session、CSRF、默认值 |
| [SQLite](database-guidelines.md) | schema、migrations、事务 |
| [错误](error-handling.md) | 领域错误与 HTTP 映射 |
| [日志](logging-guidelines.md) | slog、脱敏、审计 |
| [远端验证](remote-validation.md) | 私有 .env 连接、SSH/SCP 与证据去敏 |
| [命令与取消](process-guidelines.md) | context、git/rg/tmux 适配器 |
| [本机开发启动](local-development.md) | Mac/Linux 一键启动、自有进程清理与 tmux 保留 |
| [Terminal 生命周期](terminal-lifecycle.md) | tmux、PTY、systemd Spike |
| [W03 终端运行时契约](terminal-runtime-contract.md) | 终端 HTTP/WS、历史/控制/终止与验收门禁 |
| [Go 桥接实验](bridge-validation.md) | 独立模块、真实 WS/PTY、D06 门禁 |
| [历史/TUI 实验](history-validation.md) | capture/attach 反例、库解析与 D07 门禁 |
| [快照截点实验](snapshot-validation.md) | 序号/ring 模型、解析状态限制与 D08 门禁 |
| [文件安全与版本](filesystem-guidelines.md) | 根句柄、冲突、原子保存 |
| [传输与搜索](transfer-search-git.md) | 上传/ZIP/rg/替换/只读 Git |
| [HTTP 与共享概念](http-api.md) | 跨层权威契约 |
| [W01 基础协议](foundation-contract.md) | 已批准基础API与共享fixture；覆盖旧bootstrap字段 |
| [W04 项目与文件 API 契约](workspace-files-contract.md) | 已实现的多根项目、文件、传输、事件与测试矩阵 |
| [WebSocket](websocket-protocol.md) | 帧、重连、背压、watcher |
| [质量与测试](quality-guidelines.md) | 后端门禁、真实集成与安全检查 |

## 开发前必读（Pre-Development Checklist）
先加载当前 task 的 prd/design/implement 和 JSONL。所有后端任务读目录、错误、质量；涉及外部输入再读配置/认证、HTTP；文件/命令/Terminal/DB 按归属读完整文件，不能只读索引。跨层任务同时读前端索引和 guides。

## 质量检查（Quality Check）
核对相关 task AC 与规范；检查 context、权限、参数、事务、错误映射、资源释放和日志脱敏。有源码后运行 go test ./...、go vet ./...；风险模块按质量规范运行 race/真实集成。文档阶段检查链接/占位/一致性；无源码时产品测试记为不适用，不能写通过。
