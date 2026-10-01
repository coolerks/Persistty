# W04 项目与文件 API 契约

## 1. 范围与触发条件

W04 已实现多文件夹项目、目录选择、文件树/内容、文件操作、上传、下载、ZIP 与文件事件。修改 `internal/storage`、`internal/files`、`internal/transfer`、`internal/httpapi` 或对应 `web/src/features/workspaces` 时适用。终端运行时属 W03，自动保存与草稿属 W05；文件界面在 W04 只读，W05 已接入既有强版本 PUT；保存与预览补充见 [W05 契约](editor-preview-contract.md)。完整字段形状以代码和 [共享 fixture](../../../tests/contracts/workspace-files.json) 同步校验，不把此文当成可跳过测试的替代品。

## 2. 签名（API / DB）

所有路径以 `/api/v1` 为前缀。项目：`GET/POST /projects`、`GET/PATCH/DELETE /projects/:id`、`GET /directories?path=<absolute>`。文件：`GET /projects/:id/folders/:folderId/{entries,content,metadata,download}?project_version=<int>&path=<relative>`；`PUT .../content`；W05 新增同路径 GET `inspect/preview`（见补充契约）；`POST /projects/:id/{file-operations,delete-preview}`。传输：`POST /uploads`、`GET/DELETE /uploads/:id`、`PUT /uploads/:id/chunks/:index`、`POST /uploads/:id/complete`、`POST /archives`、`GET/DELETE /archives/:id`、`GET /archives/:id/download`。`GET /events` 为经认证、Origin 校验的 WebSocket。

`0003_folder_roots.sql` 保存 `folder_id,device,inode`；`0004_transfers.sql` 保存 `uploads`、`upload_chunks`、`archives`。上传暂存和 ZIP 文件在私有磁盘目录，不将正文放入 SQLite。迁移校验和不可通过改写已应用 SQL 来修正。

## 3. 契约（请求 / 响应 / 配置）

- 项目由稳定 `id`、`name`、正整数 `version`、`main_folder_id`、非空 `folders[{id,path}]` 组成。创建使用 `folder_paths[] + main_index`；更新使用 `expected_version`、`name`、`add_paths[]`、`remove_folder_ids[]`，主文件夹由 `main_folder_id` 或 `main_added_index` 指定；删除仅清元数据，并要求 `expected_version`。目录浏览只接受服务器绝对路径，返回当前路径、上级路径和目录项；输入与选择使用同一后端校验。
- 普通文件请求必须同时绑定项目 ID、文件夹 ID、相对路径和项目配置版本。`entries` 每页默认 100、最大 200，使用 `next_cursor` 继续；游标失效返回冲突并从首项重列。`content` 仅返回有效 UTF-8 文本和强 `version`；二进制走原字节下载。`version` 包含 `identity,mtime,size,etag`，hash 基于原字节。`PUT content` 必须带 `expected_version`。
- `file-operations` 的 `kind` 仅为 `create_file/create_directory/rename/copy/move/delete`。非删除的已有来源带 `source_folder_id,source_path,expected_identity`，普通文件还必须带 `expected_version`；目录操作使用身份及目录成员复验。目标带 `target_folder_id,target_path`；删除必须先取得短时、绑定会话/身份/版本的 `delete-preview.token`。部分成功返回 207 和 `state=partial,target_created,source_removed,failure_code`，前端不能把它当完整移动成功。
- 上传创建带 `project_id,folder_id,project_version,path,batch_id,size,sha256`，确认替换还带确认时捕获的 `expected_version`。块 PUT 带 `X-Chunk-SHA256`，`GET /uploads/:id` 返回已收到的块索引用于同一浏览器仍持源文件时的续传。相同内容跳过；不同内容默认 409；仅提供 skip/replace，不提供 keep_both。目录上传由已验证的逐项建目录与文件上传组成，拖拽能保留空目录；手机只展示文件上传。ZIP 在后台完成后才开放下载，状态为 pending/ready/failed/cancelled。
- JSON 正常为 `{data,request_id}`，错误为 `{error:{code,message},request_id}`；下载为原字节/ZIP，不套 JSON。写入使用会话 Cookie、同源 Origin 和 `X-CSRF-Token`。传输大小、分块、暂存/批次配额与 TTL 使用 `transfer` 配置及 `config.example.yaml`，不得在前端绕过服务端上限。

## 4. 验证与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 未认证 / Origin 或 CSRF 不匹配 | 401 / 403；下载、WS 与写入同样受保护 |
| 缺项目/文件版本 | 428 `version_required`；不执行磁盘副作用 |
| 项目配置已变、根 device/inode 变化、源/目标身份或强版本已变 | 409 `conflict`；客户端重新读取，不自动重试写入 |
| 相对路径逃逸、符号链接越根、内部跨挂载、特殊文件 | 拒绝；Linux `openat2` 无法提供约束时 fail closed |
| 目标已有不同内容、上传确认后又变、删除预览后变 | 409；不能隐式覆盖或重用旧确认 |
| 块/全文件 hash 不符、配额/TTL 超限 | 明确失败；只回收本应用暂存，不报告完成 |
| 跨根移动复制成功但来源删除失败 | 207 `partial`；保留目标和来源的真实结果 |

## 5. 正常 / 基础 / 错误用例

- 正常：创建两个根的项目，选择主根，浏览、移动文件、上传 ZIP 后解压核对，改项目名后书签 ID 不变。
- 基础：空目录可列出，目录树按游标完整遍历；移除项目只删除 SQLite 配置，不删除真实文件或终端。
- 错误：旧项目版本访问新关联、同 mtime/size 内容被外部改写、确认替换后目标再次改变、目录移出注册根、ZIP 读中变化都不得作为成功写入或完成下载。

## 6. 所需测试（断言点）

`go test ./...`、`go test -race ./...`、`go vet ./...`；文件单测覆盖原字节 hash、BOM/换行、身份复验、no-replace、删除预览、部分移动；传输单测覆盖重复/错误块、重启续传、同内容跳过、替换二次冲突、ZIP 特殊文件。HTTP 测试覆盖鉴权/CSRF/Origin、下载、WS、共享 fixture。`tests/integration/debian/w04` 必须在真实 Linux 私有临时树跑，并核对 ZIP 条目与根替换。前端按锁文件执行 lint/typecheck/test/build；浏览器检查桌面项目/目录选择/树与手机单视图，不把 W03/W05 占位误报为可用。

## 7. 错误与正确示例

错误：仅用 `path` 找文件，或在用户点“替换”时重新读取目标版本并立即覆盖；这会混淆两个根，也会吞掉确认之后的新修改。

正确：请求保持 `project_id + folder_id + project_version + relative_path`；弹窗保存首次冲突时的 `expected_version`，完成上传时服务器复验，遇 409 提示文件再次变化并要求重新确认。
