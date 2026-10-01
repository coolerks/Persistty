# 状态归属

## 1. 范围与触发条件

W05 工作台、编辑缓冲区、本地视图和 IndexedDB 草稿的 owner 契约。修改 workspace-view、editor-session、editor-drafts 或恢复入口时适用；覆盖初始化阶段的 SQLite 共享布局候选。

| 来源 | 权威事实 | 前端处理 |
| --- | --- | --- |
| tmux | 进程存活与当前画面 | API/WS 观测，不以本地 running 推断 |
| 文件系统 | 原始正文、身份、强版本 | 受认证 API 快照，写入复验 |
| Git | 分支/status/diff/log | 服务端只读快照 |
| SQLite | 项目/文件夹版本、终端共享元数据 | 明确 API mutation；不存正文或浏览器布局 |
| Zustand/localStorage | 本浏览器视图、主题 | 标签/组/展开/尺寸；设备记录独立 |
| EditorScope | 当前文件缓冲区与保存调度 | 按真实身份合并别名，generation 区分输入 |
| IndexedDB | 本浏览器未保存草稿 | 明确恢复/丢弃/导出，恢复不自动写回 |

## 2. 签名

`useWorkspaceView` 提供 open/close/move/split/unsplit、openMobile/closeMobile、placeTerminal/selectTerminal/orderTerminal、expand/relocate/remove。`orderTerminal(projectId, id, before, visibleOrder?)` 补全首次恢复时服务端标签顺序，再插入指定位置。

localStorage `persistty.workspace-view.v1` 使用 Zustand persist `version: 2`；只 partialize `projects`。`EditorScope(project, storage?, client?)` 提供 open/protect/refresh/configure/relocate/dispose。`IndexedDraftStorage` 数据库为 `persistty.editor-drafts`，版本 1，`drafts` 对象仓库以 `id` 为 key；提供 list(projectId)、put(draft)、remove(id, maximumGeneration)。无新增 SQL、环境变量或认证协议。

## 3. 契约

每项目有四组 `groups/active`，focused、split，独立 mobileFiles/mobileActive/mobileView/mobileTerminal，terminals 的 top/bottom 与 group 0..3，terminalOrder、upperActive/lowerActive、lowerCount 和 expanded。面板尺寸用 react-resizable-panels 的 useDefaultLayout，经 panelStorage 校验独立存储；语言覆盖不持久化。桌面/手机视图不互相覆盖，同浏览器内同文件仍共用 buffer。

读取时验证 schema、ID、相对路径、字段范围与数量，旧 version 0/1 的双组迁移到四组。最多 100 个项目、每项目桌面/手机各 100 个标签、500 个终端 ID、1000 个展开目录；localStorage 单记录最多 2 MiB 字符。尺寸必须为 1..4 个 0..100 数值且总和为 100；无效记录用默认视图并提示。恢复认证后重新读取真实资源，不创建文件、终端或发送输入；不存在资源显示失败/已终止。

草稿 `schema:1` 保存 id/projectId/file{folderId,path}/sourceRoot/viewId/generation/base{kind,content,version}/content/updatedAt。base.version 采用完整 identity/mtime/size/etag；每 scope 生成独立 viewId，同文件不同视图不会互删。草稿单正文最多 8 MiB，总 base+content 最多 64 MiB、100 条；不自动淘汰。put 串行且 IDB 事务核对配额；保存成功只移除本视图提交 generation 及之前的记录，后续输入重新保护。

认证先于服务器读取，服务器读取先于草稿比较。源路径和身份均保留，配置移除需对剩余覆盖根复验完整版本才能重绑；失败保留输入供导出。密码、Cookie、CSRF、token 不进入持久 store 或日志。最后标签关闭、显式离开链接、登出与文件操作前 protect；失败保留页面/输入。beforeunload 提醒脏输入；强制关闭浏览器无法保证未落盘的输入恢复。

## 4. 验证与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 视图 schema/尺寸/路径无效 | 默认视图并可见提示；草稿独立保留 |
| localStorage 不可写 | 内存 fallback 并提示下次可能不能恢复 |
| IDB 配额/权限/schema 失败 | 显示草稿失败，保留内存，不伪造保护成功 |
| 文件基线变化 | dirty 暂停并保留双方；clean 安全刷新 |
| 切设备/移动标签 | 保留记录、buffer/model 和终端 runtime |
| 关联或项目消失 | 受认证草稿导出；不复活文件或进程 |

## 5. 正常 / 基础 / 错误用例

正常：桌面三组刷新恢复，手机独立打开另一个文件，回桌面仍三组。基础：从旧双组记录迁移，无终端时显示空区域。错误：localStorage 有终端 ID 就自动创建会话，或恢复草稿立即 PUT。

## 6. 所需测试

workspace-view 单测验四组、数量/路径/schema、移动/合并/设备隔离；editor-session 验并发输入、草稿失败、身份合并、关联迁移及 stale response。editor-recovery 浏览器验真实 IndexedDB、409/只读 diff/明确保存、刷新零隐式 PUT、终端位置恢复零 mutation。终端进程持久性另见 [W03 owner](terminal-runtime-contract.md)；视口模拟不代替真机软键盘验收。

## 7. 错误与正确示例

错误：保存响应返回便将 dirty=false，删除所有同 path 草稿。正确：更新本次提交的 base，保留新的 generation，只删除本 viewId 提交及之前的草稿。错误：`running=true` 本地恢复作为 tmux 事实；正确：保留标签 ID，重新查询 API 状态。

## 临时文件语言覆盖（2026-09-30）

`web/src/features/workspaces/workspace-view.ts` 新增 `languageModes: Record<projectId, Record<fileKey, languageId>>` 与 `setLanguage(projectId, file, mode: string | undefined)`。fileKey 为 folderId + NUL + path；只接受完整已注册 ID，undefined 恢复自动，未知值拒绝。手动模式跨同文件多组共享、不同项目/文件隔离，不进入 persist.partialize；刷新页面恢复自动，未新增 API/SQLite/localStorage 字段。

关闭一个分组仍有同文件视图时保留；最后视图关闭、remove 后清理；relocate（含父目录与跨文件夹）迁移键；move/unsplit 保留。覆盖仅为 UI 语言提示，不代表文件真实类型、资源授权或正文状态。正确：读服务器快照后传最终 language 给既有 model；错误：选择语言写回文件，或将全部 store 状态持久化使临时覆盖跨刷新残留。workspace-view 单测验项目隔离、分组、关闭/删除/重命名与不持久化，真实工作台验同文件分组同步。
