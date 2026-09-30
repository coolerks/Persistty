# Research: 浏览器传输、编辑草稿与字体图标预览

- Query: 文件与目录上传下载在桌面/移动浏览器的能力边界是什么？续传、草稿恢复、内置字体图标和常见格式预览应如何形成可验收需求？
- Scope: mixed；仓库初始规范与官方 Web 标准、浏览器资料、上游项目源码核对。
- Date: 2026-09-26

## Findings

### 1. 来源与证据等级

本专题依据 `research/requirements-source.md` 的 U04/U05/U07/U08/U09/U10。明确需求包括 Monaco、内置 JetBrains Mono Nerd Font NL 与等宽 fallback、Material Icon Theme 文件名/多后缀映射、深浅主题、文本与图片/SVG 等常见文件预览、二进制正确处理、拖拽文件/目录上传和右键文件/目录 ZIP 下载。

断点续传、IndexedDB 草稿、PDF、十六进制、System 主题、上传冲突三选项和容量默认值属于助手建议或初始规范，不能直接标记为用户确认。移动端支持等级、跨设备草稿同步、浏览器关闭后自动续传、目录一致快照与上传整个目录事务也未获得确认。下述故事与验收是候选，先供澄清，不改变现有 spec。

### 2. 本地文件与相关规范

| 文件 | 内容与引用 |
| --- | --- |
| `README.md:5` | 无产品实现；不能从规范推断已有功能或已安装版本 |
| `.trellis/spec/frontend/index.md:3` | React/Vite/Monaco/xterm 等版本待 foundation 锁定 |
| `.trellis/spec/frontend/editor-terminal-lifecycle.md:4` | Monaco model、保存期间输入与冲突生命周期 |
| `.trellis/spec/frontend/editor-terminal-lifecycle.md:9` | IndexedDB 草稿、恢复选择与存储失败可见 |
| `.trellis/spec/frontend/editor-terminal-lifecycle.md:19` | SVG 源码/隔离预览；PDF/二进制为初始范围 |
| `.trellis/spec/frontend/theme-assets.md:6` | Nerd Fonts 变体锁定、字体加载与重测 |
| `.trellis/spec/frontend/theme-assets.md:7` | 文件名、复合后缀、扩展名映射与可信内置 SVG |
| `.trellis/spec/frontend/theme-assets.md:9` | 第三方资源许可、版本/checksum 与修改记录 |
| `.trellis/spec/backend/transfer-search-git.md:11` | 分块状态持久化、完整校验、原子发布、Web restart |
| `.trellis/spec/backend/transfer-search-git.md:12` | empty-dir manifest、quota/concurrency/TTL |
| `.trellis/spec/backend/transfer-search-git.md:13` | skip/replace/keep_both 与版本冲突 |
| `.trellis/spec/backend/transfer-search-git.md:14` | ZIP、symlink、特殊文件、预览隔离 |
| `.trellis/spec/backend/security-config.md:27` | 编辑/预览/上传尺寸为初始示例值，不是经性能验证的规格 |
| `.trellis/spec/guides/index.md:14` | 本专题自然语言使用简体中文，标识符与路径保留 |

代码模式说明：仓库尚无实现，本表引用的是契约模式而非真实函数。不得声称已验证字体资产、浏览器或上传服务。

### 3. 目录上传与浏览器矩阵

**文档事实**：File and Directory Entries API 的 `readEntries` 是分批读取；同一个 reader 应顺序读取直到空数组，不应每批重新创建 reader，也不应同时发起读取。Chromium 的常见批次上限为 100。只遍历一次会丢文件。目录 entry 与文件 entry 应分别记录，才能保留空目录。目录选择返回 `FileList`/`webkitRelativePath` 时不能仅从文件推导没有任何文件的目录。[WICG Entries API](https://wicg.github.io/entries-api/#dom-filesystemdirectoryreader-readentries)、[Mozilla 方法说明](https://developer.mozilla.org/en-US/docs/Web/API/FileSystemDirectoryReader/readEntries)

推荐能力链是设计推论：优先使用当前环境可用的目录 handle，回退到 `webkitGetAsEntry`，再提供文件选择或 `webkitdirectory` 的明确降级入口。普通文件上传不应被目录高级 API 缺失阻断。相对路径与 MIME 均是客户端输入，后端仍须验证。不能声称 ZIP 上传等于目录上传，ZIP 导入需要额外解压安全契约。

以下是 2026-09-26 查询上游 BCD 的能力记录，**不是 Persistty 支持版本声明**。移动 OS 的文件提供器、触屏拖放和空目录暴露仍需真机验证；仅检测属性存在不充分。

| 浏览器环境 | Entries 目录 entry | `webkitdirectory` 目录选择 | 外部目录 handle/picker | Monaco 编辑支持 |
| --- | --- | --- | --- | --- |
| 桌面 Chrome/Edge | 有记录 | 有记录 | Chromium 有记录；权限与 HTTPS 限制 | 上游桌面目标，版本待锁定 |
| 桌面 Firefox | Firefox 50 起有记录 | Firefox 50 起有记录 | BCD 标未支持 `showDirectoryPicker`/`getAsFileSystemHandle` | 同上 |
| 桌面 Safari | Safari 11.1 起有记录 | Safari 11.1 起有记录 | BCD 标未支持上述方法 | 同上 |
| Chrome Android | API 有记录，实际目录拖放待验 | 132 起记录完整支持；更早版本有仅属性/崩溃问题 | BCD 132 起有记录 | 官方明确不支持移动浏览器 |
| Firefox Android | 141 起记录 entry API | 142 起完整记录；141 相对路径问题 | BCD 未支持 | 官方明确不支持 |
| iOS Safari | API 记录继承 Safari；实际拖放待验 | 18.4 起完整记录；之前属性可设但无效 | BCD 未支持上述方法 | 官方明确不支持 |

矩阵证据：[Mozilla BCD HTMLInputElement](https://raw.githubusercontent.com/mdn/browser-compat-data/main/api/HTMLInputElement.json)、[DataTransferItem](https://raw.githubusercontent.com/mdn/browser-compat-data/main/api/DataTransferItem.json)、[Window](https://raw.githubusercontent.com/mdn/browser-compat-data/main/api/Window.json)。移动目录选择门槛与桌面不同；第三方 iOS 浏览器不能仅按桌面同品牌能力外推。

**重要范围约束**：Monaco 官方 FAQ 对移动浏览器支持回答为否。页面响应式布局成功不能证明软键盘、IME、光标、滚动、选择、DiffEditor 已获得上游支持。候选选择包括桌面编辑首版、移动只读查看/终端控制，或为移动提供独立编辑降级；选择须等需求澄清，不在本轮替用户决定。[Monaco 官方 FAQ](https://github.com/microsoft/monaco-editor#faq)

### 4. 四种续传不能合并承诺

| 情况 | 可行前提与恢复行为 | 不能推定的能力 |
| --- | --- | --- |
| 网络中断，页面仍存活 | `File` 可读、upload ID 有效，查询后端已收块并补传 | 重发 HTTP 成功不代表文件已发布 |
| Web 服务重启，页面仍存活 | 服务端块数据和状态持久；重新鉴权后查询缺块；complete 幂等与崩溃恢复 | 内存 bitmap 或前端进度不能恢复后端状态 |
| 页面刷新/浏览器关闭再开 | 本地 upload 记录可恢复；通常要求重新选择原文件，或恢复 handle 并复验读取权限 | 已持有的内存 File 不会自动变成永久本地路径授权 |
| 换设备继续 | 新设备必须获得相同源文件，并通过内容身份校验匹配上传任务 | 服务端分块状态不会使另一设备凭空读到原本地文件 |

File/Blob 可序列化，因此也存在将字节复制进 IndexedDB/OPFS 的方案；它消耗本地配额、需复制成本与清理策略，不适合默认承诺保存所有大上传。File System Access handle 可存 IndexedDB，但不等于权限永久保留；`queryPermission`/`requestPermission` 与用户手势仍须处理。Chrome 的持久授权行为受版本、安装状态和用户选择影响。[File API](https://w3c.github.io/FileAPI/#file-section)、[Chrome File System Access](https://developer.chrome.com/docs/capabilities/web-apis/file-system-access)、[Chrome 持久授权说明](https://developer.chrome.com/blog/persistent-permissions-for-the-file-system-access-api)

**设计推论**：重新选文件不能仅检查名称、大小和 mtime 后拼接，三者不是内容身份。必须验证与 upload 已存块/内容指纹一致，最终完整 hash 成功后才能发布；源文件修改或变得不可读应暂停并说明，不能混合旧块和新文件。File API 的 snapshot 不是硬保证：标准对磁盘 snapshot 使用 should，且读取可因状态变化产生 `NotReadableError`。

现有规范使用 index/chunk/hash 状态机；是否采用 tus 需后续设计比较，不能把 tus offset 协议和现有任意 index 上传混称同一种协议。tus 1.0.x 提供 offset 查询、校验和、过期等扩展，说明续传除分块外还需要错误/过期契约。[tus 官方协议](https://tus.io/protocols/resumable-upload)

### 5. 上传冲突、容量和失败表现

**候选契约**：单文件完成才发布，目录按文件显示结果；目录整体成功必须包含所有选中文件和可表示目录。部分成功不能显示全部成功。网络中断应区分待重试、源文件需重选、服务端任务过期、认证需恢复；进度应区分已读/已发送/服务器确认/已校验发布。

skip、replace、keep_both 是初始建议，后续确认可简化首版，但必须避免默默覆盖终端或其他设备写出的文件。replace 在提交时复验版本；keep_both 必须原子争取名称；目录与同名文件冲突、父目录消失、权限改变、目标被替换为 symlink 等需逐项失败可见。批量“应用全部”只适用于本批次。

单文件上限无法限制总磁盘消耗。容量需求应同时覆盖总 staging quota、完成文件空间、重复 chunk、最大活动任务/文件数/目录深度/路径长度、hash worker/上传并发、inode 耗尽、TTL/取消清理、ZIP 压缩 CPU 与断开取消。目标同目录 staging 与完整副本校验可能放大空间占用，不能按单文件字节数预测峰值。现有 20 GiB/8 MiB/32 MiB 只是初始默认形状，尚无 Debian 内存、磁盘、吞吐实测。[本地规范](../../../../../spec/backend/transfer-search-git.md)

资源限制属于设计建议和实验范围，不是本轮已经确定的数值。代理 Nginx 的 body/timeouts、响应 buffering 与后端限制要在部署设计中一起验证。

### 6. ZIP 下载不是项目一致快照

Go `archive/zip` 将逐个文件写入输出，目录以末尾 `/` entry 表示，`Close` 写 central directory。这些机制不创建磁盘快照。**设计推论**：后台 `npm run dev`、AI 或用户继续修改时，ZIP 可组合来自不同时间的文件；单文件版本复验也不能证明整个项目是同一时点。[Go zip.Writer](https://pkg.go.dev/archive/zip#Writer)

候选需求应在普通目录打包与可复现一致快照之间选择。普通打包可要求读取失败/检测变更可见、保留空目录、默认跳过 symlink 并报告、拒绝 device/FIFO、禁止根外路径。更强保证需要文件系统快照或先复制版本集，涉及额外空间、权限和时间，不应暗含在“目录下载为 ZIP”中。

流式响应一旦发出 200 和部分 ZIP，后端后续不能再正常改为 JSON 错误；截断流可能留下损坏文件。可选完整归档先生成再下载、归档作业状态，或带明确结果 manifest 的 best-effort 归档，需按期望规模和 UX 决定。前端不得仅因开始响应/浏览器触发保存就显示“完整下载成功”。直接浏览器下载与应用可追踪任务之间存在观测能力差异，待实验。

### 7. IndexedDB 草稿是本设备恢复能力

草稿绑定 origin 与浏览器 profile，IndexedDB 没有内建跨设备同步。URL 换协议/域名/端口、换 profile/设备后不能假定能找到旧草稿。要跨设备需设计服务端同步与冲突语义；那会把尚未保存到工作区的内容上传服务器，不再只是本地恢复。[Mozilla IndexedDB 概述](https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API/Basic_Terminology)

默认存储 best-effort；配额、清理站点数据、隐私模式会影响保留。`navigator.storage.persist()` 可以请求更强的保留，返回结果必须检查；获准也不阻止用户主动清理，`estimate()` 仅估算。不要以定时写入成功承诺永久保存。[WHATWG Storage Standard](https://storage.spec.whatwg.org/)、[Mozilla quota/eviction](https://developer.mozilla.org/en-US/docs/Web/API/Storage_API/Storage_quotas_and_eviction_criteria)

候选故事：页面误关后，在同浏览器同 origin 看到可恢复草稿，先对照当前服务器版本，再由用户恢复到 editor；不得自动 PUT。存储写入失败要有持续可见状态；浏览器崩溃前未完成事务的最后若干输入不承诺恢复。多标签同一文件草稿需要 tab/revision 仲裁，避免一个标签保存后误删另一个标签的未保存内容。登出后本地保留/清除策略待澄清，单用户不意味着共享浏览器没有数据暴露问题。

### 8. 字体变体、CJK 与加载

`NL` 代表 NoLigatures 变体，`Mono` 代表 Nerd glyph 单格宽度，两者不能互相替代。上游 JetBrainsMono 同时提供 Ligatures/NoLigatures；Nerd Fonts FAQ 建议等宽终端使用 Nerd Font Mono，普通 NF 图标可超过一格宽。**设计建议**：锁定 release 后验证 NoLigatures + Mono 实际文件和内部 family metadata，而不是拼接 CSS 名称；需要粗体/斜体时也核对各 face。[Nerd Fonts JetBrainsMono](https://github.com/ryanoasis/nerd-fonts/tree/master/patched-fonts/JetBrainsMono)、[上游 FAQ](https://github.com/ryanoasis/nerd-fonts/wiki/FAQ-and-Troubleshooting)

字体 fallback 是逐字形匹配，不是整个文件统一换一套字体。不能推断 JetBrainsMono/Nerd 补丁覆盖所有 CJK/emoji；缺字会落到平台字体。generic monospace 不保证所有 PUA Nerd glyph 可用，CSS 对 PUA 有特殊 fallback 规则。中文双格、emoji/变体选择符/组合字符及终端宽度算法需实测；图标存在不等于布局无错位。[CSS Fonts 4 匹配规则](https://www.w3.org/TR/css-fonts-4/#font-style-matching)

`document.fonts.ready` 完成只表示当前加载与布局结束，它本身不因某字体失败而 reject，也不证明每个需求 glyph 均存在。应显式加载目标 face、处理失败，并在字体实际就绪后让 Monaco/xterm 重测；缓存与慢网络、字体失败时 fallback 仍须可用。[CSS Font Loading 3](https://www.w3.org/TR/css-font-loading-3/#font-face-set-ready)

**资源复核事项**：JetBrains 原字体 OFL-1.1；Nerd 补丁 README 列举 CC BY 4.0、Apache 2.0、MIT/OFL 等 icon set，Font Logos 行还标为 `unlicensed`。该标签是上游元数据，不能据此下法律结论或认定实际 release 违法，但也不能只凭原字体 OFL 宣称整个补丁分发已审查。后续导入须核对实际发布包、相关组件许可与归属说明，保留转换/子集记录。[原字体 OFL](https://raw.githubusercontent.com/JetBrains/JetBrainsMono/master/OFL.txt)、[Nerd Fonts 字体 README](https://raw.githubusercontent.com/ryanoasis/nerd-fonts/master/patched-fonts/JetBrainsMono/README.md)

### 9. Material 图标与安全预览

Material Icon Theme 上游映射包含 `fileNames`、`fileExtensions`、patterns、clones；例如 `d.ts` 与 `ts` 是不同映射，Docker Compose 文件名也有专门映射。**设计推论**：按 exact basename、最长复合后缀、扩展名、fallback 解析符合用户目标，但要决定上游 patterns/语言映射/light variants/folder mapping 是否全部纳入；不能只复制图标而丢映射。未知后缀、大小写、隐藏文件与 Unicode 名称需验收。[上游映射源码](https://raw.githubusercontent.com/material-extensions/vscode-material-icon-theme/main/src/core/icons/fileIcons.ts)

查询时上游根 LICENSE 为 MIT，要求分发保留 copyright/license；实际导入 tag/commit、资产子目录和派生资源仍需复核。内置经过审查的 icon SVG 与用户工作区 SVG 属于不同信任级别。[上游 LICENSE](https://raw.githubusercontent.com/material-extensions/vscode-material-icon-theme/main/LICENSE)

SVG 作为 `<img>` 图像按 SVG 2 的 secure animated/static 模式禁脚本与外部引用；作为 inline/直接文档/`object` 的能力不同。**设计建议**：首版用受保护图片入口或源码查看，不用 `innerHTML` 插入用户 SVG；如使用隔离 iframe，`sandbox` 不能独自代替 CSP 外部连接/资源限制，禁脚本/同源/导航等权限且验证跨浏览器。含外部字体/图片的 SVG 可能无法忠实呈现，须把安全限制作为预览行为约束。[SVG 2 模式](https://www.w3.org/TR/SVG/conform.html)、[HTML sandbox](https://html.spec.whatwg.org/multipage/iframe-embed-object.html#attr-iframe-sandbox)、[CSP3](https://www.w3.org/TR/CSP3/)

PDF 属建议范围。浏览器原生 PDF viewer 与 PDF.js 均不能仅凭“只读”认定内容无攻击面。PDF.js 官方 CVE-2024-4367 曾涉及恶意 PDF 在宿主域执行脚本，受影响 <=4.1.392、修复 4.2.67；此历史修复号不是当前版本推荐。若采用 PDF.js，锁定当时维护版本、复核 advisories、限制脚本/外链/worker 与解码资源；不直接导入 demo 的任意 URL 入口。[PDF.js 官方 advisory](https://github.com/mozilla/pdf.js/security/advisories/GHSA-wgrm-67xf-hhpq)

图片压缩文件小并不代表像素解码内存小；预览需求要包括像素尺寸、损坏格式、超限、切换释放 ObjectURL/worker、认证失效等，PDF 还需页数/按需渲染与大文档取消。类型/尺寸预检在后端完成，超限不得先完整 fetch 再告知太大。

### 10. 候选用户故事与失败行为

| 故事 | 正常结果 | 必须可辨认的失败 |
| --- | --- | --- |
| 拖入一个开发目录 | 内容和结构完整列举，逐文件进度，空目录保留能力明确 | 枚举失败、不可读、系统不支持目录、路径无效，不能静默漏项 |
| 上传时网络波动 | 已确认字节不必重传，最终内容校验一致 | 等待重试与任务过期不同；发布前不显示完成 |
| 关闭页面后续传（待确认） | 恢复任务列表并要求重选/授权原文件 | 拒绝授权、源文件变更、找不到文件，不默默从零或混写 |
| 下载项目目录 | 合法 ZIP、Unicode、空目录、范围内文件 | 文件变更/读取失败/流截断可见，不声称一致快照 |
| 误关未保存编辑（待确认） | 同设备草稿恢复前先对照服务器版本 | quota/隐私清理/旧 schema 失败提示，不自动覆盖 |
| 浏览 Oh My Zsh/中英文代码 | 内置 glyph 呈现、列宽与光标对齐 | 字体加载失败可降级；不展示乱码为验收成功 |
| 打开 SVG/图片 | 可读内容并无未授权资源执行/读取 | 损坏/超限/外部资源限制有明确结果 |

### 11. 候选验收场景

1. 桌面各目标浏览器拖入包含超过 100 个同级文件、嵌套目录、多个空目录、零字节文件、中文/空格/emoji 名称的目录；后端枚举总量和逐项 hash 与源一致，错误不中断其他项的结果报告。
2. 移动真机分别用选择器与可用的系统拖放测试目录；无法选择/空目录不可枚举时 UI 正确降级，不凭 API 属性宣称支持。记录 OS、浏览器与文件提供器版本。
3. 在网络断开、Web restart、页面刷新、浏览器重启四种情况各测试大文件；任务查询准确、源文件身份复核、重新授权拒绝、任务过期、hash 不符均不发布坏文件。
4. 上传 complete 前终端改目标文件、两个标签同时 keep_both、目标父目录移走、磁盘/inode 不足与累计 quota；原内容不被静默覆盖，staging 有界且可取消回收。
5. ZIP 包含空目录/Unicode/大文件/symlink/FIFO，同时外部程序修改或删除内容；真实解压核对，观察变更检测局限和部分流失败表现，不把 200 当完成证据。
6. 草稿在同浏览器 reopen、多个标签、保存期间继续输入、服务器外部修改、存储 quota 失败、用户清理、隐私模式结束、域名变化场景下行为明确；另一个设备不会被描述为已自动同步。
7. 字体冷缓存/慢网/404、粗体斜体、Powerline/Nerd glyph、CJK、组合字符、emoji，Monaco/xterm 切深浅主题与 resize 不错位；截图与字体实际元数据留证。
8. 图标测试 `package.json`、`go.mod`、`Dockerfile`、`.gitignore`、`.env`、`a.d.ts`、`a.ts`、yaml/yml、未知后缀、名称大小写与目录，核对锁定上游映射。
9. SVG 含 script/event handler/foreignObject/外部 CSS/image/font/use；监测网络与脚本，不能读取宿主 DOM/Cookie。PDF 如纳入再测脚本/外链/巨页/损坏/worker 取消；超限预览无完整下载。

### 12. 留给需求澄清的问题

- 首版浏览器与设备有哪些？移动端期望终端控制、只读查看还是完整编辑？接受 Monaco 移动降级吗？
- 上传预期最大文件、目录总量和典型网络是多少？断点续传只覆盖断网/Web restart，还是要跨页面/浏览器重启？重选原文件是否可接受？
- 目录上传是否必须保留空目录？系统 API 无法暴露空目录时允许明确降级吗？
- 批量上传冲突如何选择，目录部分成功是否可接受？重复上传是否应保持独立任务？
- ZIP 是便利打包还是备份/可复现快照？打包时项目继续变化，接受报错重试还是完整快照作业？
- 是否需要本地草稿恢复、跨设备同步？登出和切工作区后草稿保留多久？
- “常见格式”除图片/SVG 外是否包括 PDF/音视频/hex？预览失败应仅下载还是提供其它查看方式？
- 是否要求终端所有 Nerd 图标严格单格？内置 CJK 字体还是依赖客户端系统 fallback？

### 13. 待实验而非文档结论

- 目标 Debian 的上传 hash/分块/压缩吞吐、磁盘峰值、inode 和取消回收；Nginx 在慢连接与重启下的行为。
- 目标浏览器真实目录枚举/empty-dir、移动系统提供器与触屏拖放；保存 handle 后关闭浏览器的读取授权。
- Monaco 移动输入/选择与替代体验；桌面 IME、worker/CSP、超大文件边界。
- xterm 的实际 Unicode width 与锁定字体/Oh My Zsh、CJK fallback、主题/缩放/字体失败组合。
- ZIP 响应后错误的用户可观测性；预览脚本/网络隔离与图片/PDF 解码内存。
- 实际 Nerd Fonts release archive 的第三方 LICENSE/NOTICE、glyph coverage 与转换后保留；本轮未下载或安装资产。

## Caveats / Not Found

- 无产品代码、package manifest、浏览器/OS/设备清单或 Debian 资源数据。本轮未执行产品/真机/远端实验，所有性能和最终兼容支持承诺尚未成立。
- BCD 与上游 main/master 内容可变化；上述版本为调研日期读取的上游记录，实施需锁定 tag/commit 并复验目标环境。标准规定与浏览器实际系统文件提供器能力不是同一层证据。
- Nerd Fonts 的 `font-info.md`、目录根 `LICENSE`/`OFL.txt` 入口本轮未获取成功，已改读可获取的 README 和原字体 OFL；不宣称发布包许可清单已完整审查。`unlicensed` 是需追查的上游元数据，不是本轮法律判断。
- 各项容量值、PDF/hex/草稿/续传/移动范围均不得从初始 spec 自动升级为已确认产品需求；本专题只供调研和之后澄清。
