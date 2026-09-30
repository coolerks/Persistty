# 规范设计

## 现状与边界
保留单仓库 backend/frontend/guides 层。目录及代码片段均为用户需求驱动的初始契约，不是已有源码模式。新增 backend 契约文件归属高风险跨层行为，guides 仅做检查入口。

## 文档归属
backend 拥有 API、WS、配置、安全、文件版本、SQLite、tmux 与命令适配器契约；frontend 拥有状态缓存、交互和资源生命周期。通过相对链接共享概念。用户产品需求的完整 Milestone 映射留给 bootstrap 完成后创建的 v0.1 父任务。

## 关键设计约束
- Web 服务只 attach 已在独立生命周期中运行的 tmux server；不能通过放松 Web service KillMode 来声称完成隔离。
- 文件访问采用根目录句柄限定的操作；词法检查和 EvalSymlinks 用于诊断，不作为最终 I/O 安全保证。
- 文件版本包含强 hash；原子 rename 解决半写，不能提供与任意外部 writer 的原子 CAS，必须诚实记录竞态边界并进行回归测试。
- 后端拥有的跨层/基础设施权威契约按 trellis-update-spec 的七段结构书写；前端消费指南引用权威契约，不重复签名和错误矩阵；版本依赖在 foundation/Spike 中锁定和验证。

## 回滚
只修改 project specs 和本 task 文档。保留 framework 文件和用户初始化成果，不回滚其他工作。
