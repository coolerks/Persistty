# SQLite 元数据

## 已批准范围更新（2026-09-27）
旧workspaces/settings/editor_tabs和共享theme/layout不是当前迁移目标。项目定义为project_id/name/version、多folder稳定身份且恰一主folder；项目删除不级联销毁终端或磁盘文件。布局/文件标签/主题归浏览器本地，正文/草稿不进SQLite。W01迁移只建立实际使用的session/项目/folder/终端元数据和迁移checksum，不预建泛化JSON表；后续资源状态只能由实际后端观测，不能以DB running字段宣称进程存活。

## 1. 范围 / 触发
数据库只保存 Persistty 元数据。tmux、filesystem、git 保持各自权威；不存 Terminal output、文件正文或 dirty draft。初始选择 database/sql + 成熟 SQLite driver；foundation 锁定 driver/version 并记录 CGO、Debian 打包代价，不假定 driver 已存在。

## 2. 签名
repository 以 context 为首参数；SQL 用独立参数。迁移目录约定 `internal/storage/migrations/0001_metadata.sql`，递增不可变。迁移表：`schema_migrations(version INTEGER PRIMARY KEY, checksum TEXT NOT NULL, applied_at TEXT NOT NULL)`。
W01 实际 schema 以 [0001](../../../internal/storage/migrations/0001_metadata.sql) 与 [0002](../../../internal/storage/migrations/0002_resource_ids.sql) 为准，保存认证 sessions、projects、folders、terminals；W04 追加 [0003](../../../internal/storage/migrations/0003_folder_roots.sql) 的注册根 device/inode 与 [0004](../../../internal/storage/migrations/0004_transfers.sql) 的 uploads/upload_chunks/archives，见 [W04 契约](workspace-files-contract.md)。资源 TEXT PRIMARY KEY 必须显式 NOT NULL；SQLite 普通表的 TEXT PRIMARY KEY 本身不拒绝 NULL，CHECK 的 NULL 结果也不等于失败。主文件夹使用项目与文件夹的复合外键保证归属，终端关联项目删除时 SET NULL。tmux 身份及后续状态字段由终端任务追加迁移，不预建 workspaces/settings/editor_tabs。

标签、Explorer 展开目录、侧栏宽度、终端面板高度、最近项目和主题归浏览器本地；不保存文件正文或终端输出到 SQLite，不以 active 标签推断资源存在。布局版本、范围校验及恢复规则见 [状态规范](../frontend/state-management.md)。

## 3. 契约
配置连接时启用 foreign_keys；每条连接都生效，不能只在池中随机一条执行 PRAGMA。W01 使用单连接、WAL 和 5 秒 busy timeout；扩大连接池前须验证连接级设置。内部时间使用固定 9 位小数的 UTC 字符串，避免 RFC3339Nano 变长导致 expiry 字典排序错误；API 时间仍为 RFC3339Nano。boolean 用 CHECK 限制 0/1。数据库及 WAL/SHM 属于同一服务 UID，目录 0700，数据库 0600。
迁移在接收流量前单事务应用 SQL 和版本/checksum，失败回滚并拒绝服务；已应用迁移禁止修改；启动发现数据库版本高于二进制支持时拒绝降级运行。
重建有关联的表时，在固定连接且接收流量前临时关闭 foreign_keys，防止 DROP TABLE 触发级联；提交前执行 foreign_key_check。所有成功、失败及取消路径均以独立有界 context 恢复并读取 PRAGMA 确认 foreign_keys=1；恢复失败必须返回错误，拒绝启动。0002 修复 0001 的 NULL ID 漏洞，不修改已应用的 0001 checksum；真实旧库升级保留认证及所有资源元数据，非法旧数据回滚而非自动丢弃。

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
真实临时 SQLite 文件测试空库/重复启动/旧库含 session 与各资源的升级保留/NULL ID 拒绝与升级回滚/checksum/foreign key/唯一约束/事务取消及权限，并检查迁移成功和失败后 FK 均恢复。`:memory:` 多连接不是同一个数据库，不能用其替代文件迁移测试。备份任务验证 SQLite 在线备份或停服务一致备份，不能运行时只复制 .db 忽略 WAL。

## 7. 错误与正确
错误：Commit 失败忽略错误，成功响应。正确：检查 Commit，返回明确失败，再由可恢复协调逻辑核对外部副作用。
