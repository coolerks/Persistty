# W05 技术设计

## 归属与边界

复用 W04 PUT、安全根句柄及 W03 稳定 terminal runtime；编辑/draft/view 位于既有 workspaces feature，API/decoder 统一解码，预览在 files/httpapi owner。无 SQLite 正文/布局写入，无迁移。shadcn 复用 Button/Dialog/Tabs/Select/Alert，不新增交互原语。

## 编辑数据流

项目 scope 持有文件缓冲区，真实 identity 合并重叠根别名；稳定 buffer/model 身份独立于路径。正文保持原始 UTF-8，视图正规化为 LF；最小编辑区间映射回原字节文本，保留 BOM 与未修改区间换行，新换行使用原主要模式。缓冲区保存 base/version/content/generation/save state/project version。

每文件单 inflight PUT，捕获 generation/content/base；成功仅推进对应提交基线，新输入继续 pending。409/权限/消失暂停自动保存。比较读取最新 snapshot，取消不改 base；用户明确保存绑定比较版本，再变仍冲突。clean 外部变化刷新，dirty 保留并提示。

IndexedDB v1 保存 schema/project/folder/path/source root/view instance/generation/base/content/time，单条 8 MiB、全站 64 MiB/100 条。事务复验配额并清理仅成功保存修订，不删除另一视图输入。恢复先服务器复验，desktop diff；恢复到 buffer 暂停 autosave，明确保存才 PUT。手机基线变化只保留/导出。存储失败可见并阻止关闭最后未保护视图；路由退出保留草稿，未鉴权不展示。

文件 rename/move 暂停相关调度、成功迁移路径和草稿并重新读 metadata；delete 停止旧保存且不重建。配置变化先暂停旧关联，利用剩余注册根/相对路径复验真实 identity，覆盖不存在则保留/导出。失效项目仅本地导出。

## API/预览

新增受认证 GET `/projects/:id/folders/:folderId/inspect`：`{kind:text|image|binary,mime,size,width,height,editable,previewable}`。相同 project_version/path 安全边界；stat 后有界读取，不无界 fetch 大文件。GET 同路径 `/preview` 仅提供有界图片，MIME/nosniff/no-store/sandbox CSP。SVG 有界 XML 白名单拒绝脚本/事件/外链/foreignObject/实体/样式引用，用户 SVG 不进入宿主 DOM。图片 16 MiB、16M pixels、单边 8192；浏览器复验解码尺寸。

沿用 PUT content `{project_version,path,expected_version,content}` → `{version}`。JSON body 上限覆盖 UTF-8 内容转义最坏六倍，不提高 8 MiB 正文上限。成功 atomic rename 导致 identity 更新只由响应推进。

## 布局与资产

升级浏览器视图 schema，兼容旧双组；上限四组/每项目 100 文件，min 240px，宽桌面三组。锁定 react-resizable-panels 处理分割/折叠/尺寸，不写通用引擎。窄桌面仅呈现 focused，保留组。desktop/mobile 记录独立；终端位置只保存 ID 与目标组，恢复读真实列表、不创建替代 session。W03 provider 持有 runtime，组移动只改宿主。

Explorer 展开保存 UI 意图，恢复枚举；手机 active/file/terminal 独立。自托管锁定 Nerd Font Mono NL 原字体，记录 source/release/family/SHA256/license，不做子集转换；font ready 后 Monaco remeasure/xterm fit。

## 风险与回滚

未支持 schema 明确回退视图，不清草稿；配额/存储失败不声称恢复保证。回滚 UI 前导出新草稿，既有终端控制/倒计时协议不变。实机/W08 矩阵明确保留；不把模拟手机、fixture HTTP 或本机 tmux 记为 Debian/systemd 实测。
