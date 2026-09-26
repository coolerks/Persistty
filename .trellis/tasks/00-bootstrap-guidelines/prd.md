# Persistty 项目规范初始化

## 目标
用简体中文替换 from scratch 模板，为 Persistty v0.1 后续任务提供可执行开发契约。当前任务只交付规范，不交付产品代码。

## 背景与证据
仓库只有 README.md（项目名）、LICENSE（MIT）及 Trellis 初始化文件；没有 Go、React、数据库或部署实现。用户在 2026-09-26 明确指定技术栈、功能、安全边界和验收要求。不能伪造源码示例或声称已验证 Terminal 持久化。task.json 初始化时已为 in_progress，本次继续现有任务。

## 范围
- backend：包组织、Gin、服务边界、配置/认证、SQLite/migrations/事务、错误/日志、context/命令执行、文件安全、Terminal 生命周期、测试。
- frontend：React 功能目录、组件、hooks、Zustand/服务器状态、类型/API/WS、Monaco/xterm 资源释放、冲突/草稿、主题/资源许可证、测试。
- guides：简体中文思考清单，链接权威契约，不重复实现规范。
- 跨层实现契约由 backend 归属：HTTP、WS、版本冲突、安全与真实状态来源，frontend 引用。

## 验收标准
- [x] 所有 spec 为简体中文，无空模板和不适用于 Persistty 的 Trellis 框架维护内容。
- [x] 索引包含开发前必读、质量检查及完整有效链接。
- [x] 用户要求的上述范围都有明确归属；高风险契约包含签名、字段、错误矩阵、正反例、断言点。
- [x] 所有未来源码路径/示例明确是初始约定；仓库现状、实现提案、待验证项不混淆。
- [x] 明确 tmux 是会话权威，浏览器/PTY/Go 生命周期不能 kill session，systemd 隔离必须有 Debian 实测。
- [x] 文件 API 防逃逸与保存冲突分别有可执行防护；不能将 EvalSymlinks 或 hash+rename 误称完整并发保证。
- [x] Trellis 上下文校验、占位扫描、相对链接检查、全范围 review 通过并留下报告。

## 不在范围内
产品源码、安装依赖、执行 Terminal Spike、实现 UI、改写 Trellis scripts/skills/hooks。bootstrap 归档后再创建 v0.1 父任务和独立子任务；不建立并行 TODO 系统。
