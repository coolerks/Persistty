# SQLite 元数据

## 1. 范围 / 触发
数据库只保存 Persistty 元数据。tmux、filesystem、git 保持各自权威；不存 Terminal output、文件正文或 dirty draft。初始选择 database/sql + 成熟 SQLite driver；foundation 锁定 driver/version 并记录 CGO、Debian 打包代价，不假定 driver 已存在。

## 2. 签名
repository 以 context 为首参数；SQL 用独立参数。迁移目录约定 `internal/storage/migrations/0001_metadata.sql`，递增不可变。迁移表：`schema_migrations(version INTEGER PRIMARY KEY, checksum TEXT NOT NULL, applied_at TEXT NOT NULL)`。
Terminal schema 必含 id、tmux_session_name（唯一）、display_name、workspace_id、working_directory、created_at、last_accessed_at、tab_order、pinned；名称由后端生成。workspaces、settings、editor_tabs 由对应 task 定义完整 migration，不能先建泛化 JSON 大表替代所有模型。

工作区元数据任务还须定义当前激活编辑器/终端标签、编辑器路径与顺序、Explorer 展开目录、侧栏宽度、终端面板高度、最近工作区和主题的持久化模型。记录 UI 偏好，不保存文件正文或终端输出；不以 active 标签字段推断资源仍存在。具体字段、布局版本、范围校验和并发更新语义须在该任务的迁移及 HTTP 契约中固化，前端恢复规则见 [状态规范](../frontend/state-management.md)。

## 3. 契约
配置连接时启用 foreign_keys；每条连接都生效，不能只在池中随机一条执行 PRAGMA。metadata 服务初期串行写，启用 WAL、明确 busy timeout；读池策略由测试证明。时间保存 UTC RFC3339Nano，boolean 用 CHECK 限制 0/1。数据库及 WAL/SHM 属于同一服务 UID，目录 0700，数据库 0600。
迁移在接收流量前单事务应用 SQL 和版本/checksum，失败回滚并拒绝服务；已应用迁移禁止修改；启动发现数据库版本高于二进制支持时拒绝降级运行。

事务边界覆盖多表不变量；BeginTx -> defer Rollback -> SQL -> Commit。事务内不启动 tmux、执行 git、持有上传流或等待浏览器。SQLite 与 tmux 不能原子提交：create 采用可恢复操作标识与 reconciliation；不得通过“DB 失败就 kill session”误杀已运行任务。

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil { return err }
defer tx.Rollback()
// 使用 tx.ExecContext(ctx, query, arguments...)；此处不运行外部命令。
return tx.Commit()
```

## 4. 验证与错误矩阵
| 条件 | 行为 |
| --- | --- |
| 空库 | 顺序迁移，约束有效 |
| migration checksum 不同/未知高版本 | 拒绝启动 |
| migration SQL 失败 | SQL/版本均回滚 |
| DB busy | 有界等待后 503，不无限重试 |
| tmux 缺失而 DB 有记录 | stale/terminated，不显示 running |
| tmux 查询失败 | unavailable，不标记全部 terminated |

## 5. 优 / 基础 / 错误用例
优：写入 ordering 与 pinned 一次事务，重启可恢复。基础：parameterized query。错误：拼接 display_name 到 SQL，或者将 DB running 字段当进程事实。

## 6. 必需测试
真实临时 SQLite 文件测试空库/重复启动/升级/回滚/checksum/foreign key/唯一约束/事务取消及权限。`:memory:` 多连接不是同一个数据库，不能用其替代文件迁移测试。备份任务验证 SQLite 在线备份或停服务一致备份，不能运行时只复制 .db 忽略 WAL。

## 7. 错误与正确
错误：Commit 失败忽略错误，成功响应。正确：检查 Commit，返回明确失败，再由可恢复协调逻辑核对外部副作用。
