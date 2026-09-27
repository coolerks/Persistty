# 状态归属

## 已批准范围更新（2026-09-27）
下文bootstrap的SQLite共享布局/theme/workspaces规则已被U63..U73替代：SQLite只持有项目/folder版本及终端共享元数据，浏览器本地按project_id/schema/view_instance/设备模式保存标签、布局、展开项与主题，草稿在IndexedDB。手机单内容记录不能覆盖桌面多组布局，恢复不自动写文件/夺控制/创建终端。旧workspace_id+path键改为project/folder/真实资源身份并区分view修订；密码和CSRF不得持久化。下文旧示例不能作为未覆盖这些新规则的实施依据。

| 来源 | 权威事实 | 前端处理 |
| --- | --- | --- |
| tmux | Terminal 是否 alive | backend observation，重连重查 |
| filesystem | 文件正文/版本 | 服务端快照缓存 |
| git | branch/status/diff/log | 服务端快照缓存 |
| SQLite | metadata/settings/workspaces/order/tabs/layout/theme | API 读取/明确 mutation |
| Zustand | 当前浏览器 UI | 活跃 panel/tab、展开节点、连接状态、选择 |
| IndexedDB | 本浏览器未保存 draft | 显式恢复/丢弃，绝不自动写服务器 |

server state 不放 Zustand 当真相；foundation 按实际需要采用 TanStack Query 等成熟 cache 或功能内有取消/失效的 request hook，选型落 task/spec 后再实现。Zustand 只保存跨组件 UI 状态，小 Dialog/filter 用 local state，derived state 用 selector 不复制。按 feature 切片/selector 订阅，不建一个全应用万能 store。

Terminal IDs 来自 API；tab ordering/pinned/name 通过 API 持久化 SQLite，前端可展示 pending 状态，失败回滚。tab UI 移除不能删除 server session。首次加载/重登录/restart 根据 server metadata 和 tmux observation 恢复，不用 localStorage running=true。
settings/layout/theme 同样 API 持久化，localStorage 可用于首帧主题 hint，但 API 成功值为权威；hint 错误最终同步。不得把 secret/服务凭证放 store persist。

## 工作区界面恢复

SQLite 保存可恢复的工作区界面偏好：编辑器标签路径与顺序、当前激活标签、终端标签顺序/固定/名称、Explorer 展开目录、侧栏宽度、终端面板高度、最近工作区及主题。Zustand 保存这些偏好的当前浏览器投影和待提交状态，不能只做本地持久化而丢失 Web 服务重启后的恢复。每项偏好有版本与合法范围；具体字段、更新冲突策略和迁移由工作区任务固化于 [数据库规范](../backend/database-guidelines.md)。

恢复顺序：认证 → 读取工作区和持久偏好 → 查询真实文件与终端状态 → 恢复界面 → 对可编辑文件提示本浏览器草稿。不存在或权限已变化的文件显示不可用，不自动重建；不存在的终端按服务端观测显示已终止，不创建同 ID 替代进程。展开目录重查列表，不能将上次缓存当当前目录事实。布局按当前屏幕限制尺寸，激活项不存在时采用有效标签作为回退。未保存正文只属于 IndexedDB，不写进 SQLite 编辑器标签元数据。

验收关闭浏览器、重新部署前端、重启 Web/Nginx 后重新登录，名称、顺序、固定、激活项、展开目录和合理布局可恢复；真实终端任务存活须另按持久化 E2E 验证，不能以界面恢复替代。
editor dirty 是 buffer 与 base snapshot 的差异。workspace_id+path 为 draft key，draft 包含 base version 和时间，server snapshot 与 draft 分离；切 workspace 不丢 dirty 状态。IndexedDB 写失败显示“草稿未保存在本地”，不伪造恢复保证。

错误：`store.set({running:true})` 作为 reconnect 依据。正确：本地 connection=disconnected，重新 GET observation。测试 selector/actions/恢复次序/服务失败 rollback/draft 命名空间，断言不会触发隐式 file save/terminal close。
