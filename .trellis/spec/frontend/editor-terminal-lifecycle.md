# Monaco、xterm 与草稿生命周期

## 当前批准规则（覆盖旧候选）
默认去抖1秒自动保存且绑定buffer generation/文件版本，Ctrl+S只复用同保存入口；冲突暂停、不自动overwrite。桌面Monaco禁诊断/LSP，手机基础textarea无diff。恢复草稿先服务器复验/diff再明确允许写回，手机基线变化只保留/导出。model/runtime与布局宿主解耦，跨组/移上下不清undo、不dispose原终端连接；controller才发正尺寸，observer只本地fit。文件身份按project/folder/真实资源及view修订隔离。当前文本8MiB/图片16MiB另限像素，PDF/Office/hex均下载不预览。下文旧手动保存/PDF/hex/共享workspace键示例已被覆盖，W05实现前必须按新规则固化真实fixture和测试。

## Monaco
model 以 workspace ID + 正规化相对 path 的 URI 为 key；每文件一个 model，editor view 与 model 生命周期分开。tabs 复用 model 保留 undo/view state；关 tab 或切工作区按 dirty 提示与 draft 策略处理。dispose editor/diff editor、listeners、decorations、models、worker；避免每次 render 重建或给受控 props setValue 清空 undo。
打开 snapshot 得 content/version，buffer 修改只设 dirty。Ctrl/Cmd+S 捕获应用 command，以 base expected_version 调 PUT；成功更新 snapshot，pending 期间继续输入不得被旧保存响应标成 clean（按保存时 buffer generation 比较）。409 保存本地内容并显示 Diff/Reload/Overwrite；diff 使用只读 server model + draft model，不能覆盖旧 buffer。
外部 watcher 变更：clean tab 可安全刷新，dirty tab 只标 external modification。DiffEditor 的 models/disposables 同样释放。find/replace/go to line/语言推断从文件类型 adapter，不让 binary 或超编辑大小进 model。

## IndexedDB 草稿
记录 workspace_id/path/base_version/content/updated_at/schema_version；去抖有界保存 dirty draft，成功 server save 后删除对应版本 draft，不能删掉后续输入。页面 reopen 先读取服务器，再展示 View Diff/Restore Draft/Discard Draft；恢复到 buffer 并保持基于当前 server snapshot 的显式冲突策略，不自动 PUT。配额/权限失败可见，schema 升级不悄悄丢内容；登出不上传草稿。

## 文件预览分类

文件类型以服务端验证后的内容分类与大小为依据，扩展名只用于展示提示，不能使二进制文件进入 Monaco 或文本替换。后端访问与预览隔离复用 [文件安全](../backend/filesystem-guidelines.md) 和 [传输规范](../backend/transfer-search-git.md)，前端不自行放宽策略。

| 类型 | 展示与限制 |
| --- | --- |
| 支持编码的文本/代码 | 在 `files.max_edit_size` 内进入 Monaco，支持语法高亮、查找、替换、跳转行及保存；超限提供明确提示和下载入口，不无界载入 |
| PNG/JPEG/WebP/GIF/AVIF | 在 `files.max_preview_size` 内展示图片；加载失败、解码超限及不支持格式有可见结果 |
| SVG | 可查看源码；安全预览使用隔离策略，禁直接插入页面 HTML、执行脚本或加载外部资源 |
| PDF | 只读隔离预览；不实现编辑，加载失败时可下载 |
| 二进制/未知格式 | 展示大小、类型、修改时间等元数据，按需提供有界十六进制预览和下载；不进入文本编辑或批量替换 |

当前初始上限为编辑 10 MiB、预览 50 MiB，最终使用 [配置规范](../backend/security-config.md) 返回的实际值。不得将超限文件无条件 `fetch` 到浏览器内存再显示“过大”。预览任务须明确内容分类 DTO、受认证流式端点、按需读取和图片解码限制；引用 URL/文件名不作为可信 HTML。切换或关闭预览释放请求、对象 URL 和渲染资源。

验收覆盖伪装为文本扩展名的二进制、格式/大小边界、Unicode 文件名、各图片格式、含脚本/外部资源的 SVG、只读 PDF、超限下载入口、无认证拒绝及切换后资源释放。

## xterm
D07 的独立 @xterm/headless 解析测试只证明固定记录的库解析状态，未安装产品 xterm，也未证明浏览器 renderer/实时 WS 通过。tmux 外层 attach 的 alternate 与 pane TUI 模式不同，历史直接 capture+attach 有遗漏反例；恢复方案遵守 [机制实验边界](../backend/history-validation.md)，不能简单拼接或剥 ANSI。buffer 断言须等待 write callback，并比较 normal/alternate、cell 与 cursor，不以字符串 marker 代替全部状态。

D08 的 [快照截点实验](../backend/snapshot-validation.md) 显示等待 callback 仍不保证 UTF-8/CSI/OSC 完整，serialize 不保存全部状态。客户端不能把运输帧末尾视为安全快照边界；不能用 seq 连续替代两个 buffers、modes 和后续输出的恢复对照。

W03 已通过真实产品验收，当前 owner 为 [前端终端契约](terminal-runtime-contract.md)。runtime 按 terminal ID 保存在工作台 provider，portal 的固定 DOM element 在上下宿主间移动；宿主卸载不 dispose xterm/WS，不自动接管。仅退出整个 scope 才释放视图资源，仍只 detach。addons 按功能引入 fit/web-links，按锁定版本 dispose。ResizeObserver 在可见且尺寸非零时 fit，controller 才发送尺寸，隐藏不发 0x0；server bytes 由有界队列/write callback 排空，保留 split UTF-8。
clear display 只 clear xterm，不执行 shell clear/kill，不清 tmux history。copy/paste 使用浏览器 clipboard 与权限反馈，paste 不伪造控制序列；clickable URL 仅 http/https，禁止 javascript，外链 noopener。URL/标题/输出不是可信 HTML。
scope 退出 dispose xterm/addons/listeners/observer/socket/timers，只 detach；create/terminate 分别来自明确 user command。StrictMode 不重复创建连接或 session。每次 attach 由 tmux 重绘当前画面，普通历史独立整体替换，不拼入 live xterm；旧 D07/D08 负例继续约束后续变更。

## 正反例与必需测试
错误：editor save 成功就 dirty=false，不看期间输入。正确：确认 saved generation 后更新 base，后续输入继续 dirty。
测试 Monaco model reuse/undo/dirty/409/diff dispose/draft 三选择/配额/保存期间输入；xterm mount-cleanup-remount、隐藏尺寸、binary 分块、copy/paste/search/URL/clear/fullscreen、unmount 零 close 请求。真实 browser E2E 验 renderer/font/焦点及恢复，fake terminal 不能替代持久化 acceptance。
