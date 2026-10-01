# Monaco、xterm 与草稿生命周期

## 1. 范围与触发条件

W05 编辑/自动保存/草稿/预览与既有 Monaco、xterm 的生命周期 owner。修改 model、buffer、异步请求或宿主移动时适用。桌面 Monaco 可编辑且诊断/LSP 关闭；手机 textarea 基础编辑，不提供 diff。保存与预览协议归 [后端 owner](../backend/editor-preview-contract.md)，本地记录归 [状态规范](state-management.md)。

## 2. 签名

`FileBuffer.change(text, changes?)` 接受归一化 LF 视图与 Monaco rangeOffset/rangeLength/text 数组；`save(explicit=false)`、refresh/compare/restore/discardDraft/reloadCompared/persist/suspend/resume 管理保存状态。`DesktopEditor` 接收 modelURI/onChange/isModelOpen；DesktopDiff 接收原始 original/modified 和 language，只读。后端使用 W04 强版本 PUT 与 W05 inspect/preview；无 LSP 或语言服务网络 API。

## 3. 契约

### 文本与保存

首次读取原始 UTF-8 snapshot，不删除 BOM、不统一落盘换行。视图去前导 BOM 并归一化 LF；编辑按 Monaco 同时变更 offset 映射回原始正文，保留未修改段的 CRLF/LF/CR 与末尾换行，插入部分采用原始主换行。手机使用最小公共前后缀更新；不能用一次全量 LF setValue 代替原始正文。

默认去抖 1 秒自动保存，按钮与 Ctrl/Cmd+S 复用同一入口。请求捕获 content/generation/epoch、完整 expected_version 和 CSRF；单 buffer 单写入在途。成功仅将提交正文变为 base，新输入继续 pending；失败暂停且保留输入，不隐式重试。409 必须先重新读取并比较，再明确保存本地内容，后端仍复验，手机冲突只保留/导出或加载服务器。

watcher/15 秒轮询/focus/online 触发复验。clean 更新；dirty 暂停并显示冲突。在途保存时跳过读取，读取返回时再次核对发起时 base/epoch，不能让旧读响应倒退刚保存的强版本。配置更新/关闭/退出暂停请求与计时器；取消 HTTP 不代表已发送的服务器写入回滚，后续重新读取实际文件。

### 草稿与 model

恢复必须先读服务器，再展示候选草稿与桌面只读 diff；恢复到 buffer 后 paused，明确保存才 PUT。手机基线变化不提供恢复按钮，只能导出；未变化可明确恢复。IDB 失败可见且保留内存，成功保存不清理其他视图草稿。配额与 schema 见状态 owner。

buffer.uri 在重命名/移动/重绑后保持稳定，重叠根按服务端 identity 共享调度。同文件多组复用 model/undo；DesktopEditor keepCurrentModel，最后桌面或手机别名视图关闭后延迟到 React 脱离再释放。最后干净标签关闭移除 buffer 缓存，重开重新读取并正常保存；脏输入保留 paused，明确保存才恢复。scope 退出释放 model 视图，StrictMode 即时重挂不误释放。禁止通过受控 setValue 重建 URI/清空编辑 undo。

DiffEditor 使用公开 createDiffEditor/createModel/createViewModel；清理顺序是 setModel(null)、viewModel.dispose() 取消 diff worker、editor.dispose()、两份 model.dispose()。仅销毁 editor 再立即销毁 models 会产生异步 “no diff result available”，必须验关闭 diff 后的 pageerror。

### 语言

`web/src/features/workspaces/file-language.ts` 提供 `languageForFile(path: string, content?: string): string` 及完整清单。`web/scripts/generate-language-metadata.mjs` 从锁定 Monaco 0.57.0 的真实注册 AST 提取 91 个 ID（89 基础语言、JSON、plaintext），沿主入口注册顺序，不维护少量后缀白名单。非静态元数据、版本或集合变化须停止生成并复核。图标可复用轻量 metadata，但文件树不能因此提前加载引擎/worker。

推断为 exact basename → 最长 registered extension → 有界首行上游 shebang → plaintext，名称小写匹配。桌面 breadcrumb 用既有 shadcn Select 提供自动识别及全部模式；FreeMarker 六变体、mysql/pgsql/redshift 无独立后缀，保留手动入口。选择只改变 model language，不写文件/启用诊断/LSP；JSON/CSS/HTML/TS/JS 的诊断关闭，语法/worker 本地动态加载。

### 预览

内容分类与大小由服务端确定，路径后缀不能让图片进入文本。文本最多 8 MiB，图片最多 16 MiB、8192 单边、16,000,000 像素。支持 PNG/JPEG/GIF/WebP/AVIF 与严格白名单 SVG；SVG 用受认证 img URL，不内联用户 HTML，可切换源码编辑。脚本/外链 SVG 仅源码/下载。PDF/Office/未知二进制及超限只下载。图片加载失败可见，不冒充已通过格式验收；完整策略与 DTO 见后端 owner。

## 4. 验证与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 保存期间输入 | 新 generation 保留 pending，继续下一次正常保存 |
| 409 / 403 / 文件消失 / 离线 | 暂停，保留正文与草稿，明确处理 |
| 恢复草稿 | 不 PUT，先 paused 再明确保存 |
| stale read/compare/config 响应 | epoch/base/config revision 不符便丢弃 |
| 不安全 SVG / 不支持或超限 | 可见错误及原字节下载，不进图片或文本错误路径 |
| 移动 model/runtime 宿主 | 保留身份和连接；不 create/terminate/input |

## 5. 正常 / 基础 / 错误用例

正常：带 BOM/混合换行文本改一行，真实原子写入保留其他行字节。基础：未知 UTF-8 文件 plaintext 可编辑。错误：恢复草稿立即覆盖磁盘，或切语言/移动组导致 model 重建。

## 6. 所需测试

editor-session 单测覆盖同时输入/多光标/强版本/迟到读取/409/IDB 失败/多视图隔离/身份与关联；editor-recovery 验真实 model/undo、IDB、关闭 diff 无错误及恢复零隐式写入；editor-assets 验所有语言真实着色；真实 Debian w05-live 验原子保存、外部冲突、PNG/SVG、匿名拒绝。图片格式全浏览器矩阵和真实手机软键盘尚未完成，不能从头部识别单测推定。

## 7. 错误与正确示例

错误：`base = response; dirty = false`。正确：base 只对应捕获的提交正文，当前正文不同便仍 pending；先排空草稿队列，再按提交 generation 清理。错误：只 dispose diff editor；正确：先分离并取消 viewModel，然后释放 editor 与 models。

## xterm
D07 的独立 @xterm/headless 解析测试只证明固定记录的库解析状态，未安装产品 xterm，也未证明浏览器 renderer/实时 WS 通过。tmux 外层 attach 的 alternate 与 pane TUI 模式不同，历史直接 capture+attach 有遗漏反例；恢复方案遵守 [机制实验边界](../backend/history-validation.md)，不能简单拼接或剥 ANSI。buffer 断言须等待 write callback，并比较 normal/alternate、cell 与 cursor，不以字符串 marker 代替全部状态。

D08 的 [快照截点实验](../backend/snapshot-validation.md) 显示等待 callback 仍不保证 UTF-8/CSI/OSC 完整，serialize 不保存全部状态。客户端不能把运输帧末尾视为安全快照边界；不能用 seq 连续替代两个 buffers、modes 和后续输出的恢复对照。

W03 已通过真实产品验收，当前 owner 为 [前端终端契约](terminal-runtime-contract.md)。runtime 按 terminal ID 保存在工作台 provider，portal 的固定 DOM element 在上下宿主间移动；宿主卸载不 dispose xterm/WS，不自动接管。仅退出整个 scope 才释放视图资源，仍只 detach。addons 按功能引入 fit/web-links，按锁定版本 dispose。ResizeObserver 在可见且尺寸非零时 fit，controller 才发送尺寸，隐藏不发 0x0；server bytes 由有界队列/write callback 排空，保留 split UTF-8。
clear display 只 clear xterm，不执行 shell clear/kill，不清 tmux history。copy/paste 使用浏览器 clipboard 与权限反馈，paste 不伪造控制序列；clickable URL 仅 http/https，禁止 javascript，外链 noopener。URL/标题/输出不是可信 HTML。
scope 退出 dispose xterm/addons/listeners/observer/socket/timers，只 detach；create/terminate 分别来自明确 user command。StrictMode 不重复创建连接或 session。每次 attach 由 tmux 重绘当前画面，普通历史独立整体替换，不拼入 live xterm；旧 D07/D08 负例继续约束后续变更。
