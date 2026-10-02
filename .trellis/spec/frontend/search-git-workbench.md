# W06 工作台搜索与 Git

## 1. 范围与触发

W06前端归 `features/search/SearchPanel`、`features/git/GitPanel`，由 ProjectWorkbench 编排。HTTP/DTO 权威见[后端契约](../backend/search-git-contract.md)。复用现有model、EditorScope、DesktopDiff、API request/error/envelope与shadcn base-nova，不增加Git写操作。

## 2. 签名

`SearchPanel({project,mobile,intent,visible,onOpen})`；`SearchIntent={folderId,path,replace,id}`。`GitPanel({project,mobile,visible})`。`ReadOnlyComparison({original,modified,path,mobile,sideBySide?})`：桌面懒加载只读DesktopDiff，手机分区显示原/新文本。`EditorScope.prepareReplacement(files)` 返回 `{protectedIDs,release()}`；FileBuffer.holdReplacement(version) 成功返回一次性释放函数，否则null。API和严格解码分别在 `lib/api/search-git-client.ts`、`search-git-decoder.ts`。

## 3. 状态与行为契约

桌面活动栏和目录右键共享搜索入口；Ctrl/Cmd+Shift+F/H查找/替换，Shift+G打开只读Git。modal打开及输入法合成时不抢快捷键。手机单内容导航新增Search/Git，保留原编辑器/终端组记录。workspace schema v2兼容新增可选 sidebar 与mobileView值，正文/搜索ID/预览ID只存内存。

Search与Git面板隐藏后保留当前会话表单状态；route/配置/认证变更中止请求，清理拥有的搜索/预览。请求AbortController阻止迟到结果；搜索条件修改后旧结果标明需重搜，不允许用新条件与旧结果生成预览。include/exclude用`;`分隔，先选择匹配再预览，可在预览按文件缩小应用范围。逐文件结果与状态查询可见，应用不自动重试。Replace DialogContent通过initialFocus绑定“关闭预览”按钮ref，默认焦点落关闭/取消按钮，不落应用按钮；执行时弹窗内提供“停止剩余替换”，再查询部分结果。

定位先复验服务器版本与本地generation/dirty；失败保留输入并要求重搜。位置请求含project/file/version与一次性ID，DesktopEditor设置selection，手机textarea设置UTF-16范围，不创建新的编辑model或替换正文。

替换保护仅影响目标真实identity及alias。dirty/saving/暂停/冲突/加载/版本不同均跳过；成功持有期间禁自动及显式保存。新输入继续写入草稿；释放后干净buffer刷新，新输入保留且暂停，不让旧autosave覆盖新磁盘。其他文件的autosave继续。

Git面板仓库选择不控制文件的baseline。HEAD baseline按对应文件获取，输入只重算行标记，不因每次按键读Git；保存etag变化、手动Git刷新、可见后的focus/online刷新，取消固定轮询，迟到响应按key抛弃。tracked基线与buffer比较，未跟踪/基线不可用显示文字；不把baseline写进buffer。行比较先裁剪共同前后文，LCS≤250000格/20000行，超限以变化区间有界标记；删除锚定现有行。decorations独立collection，theme使用语义色，原model/undo保持。

Git总变更用total_paths，暂存/未暂存用porcelain字段。历史下一页发送首屏head以冻结遍历；刷新才读取新HEAD。比较默认磁盘，可明确选择打开比较时的编辑器快照，界面标明二者。所有比较model关闭后释放；不通过比较窗口保存文件。

## 4. 验证与错误

后端400/409/410/429/503按统一错误文案显示；tool/metadata失败不能显示clean。GET/POST仍服务端鉴权，隐藏按钮不能替代安全边界。结果部分成功、取消或跳过都保留逐文件状态；“已接入”不等于真实设备验收。

## 5. 正常、基础与错误用例

正常：搜索中文/emoji准确选择原model，预览替换显示两边，保存后HEAD修改条保留。基础：无仓库显示Empty，移动比较为只读文本。错误：预览期间新输入保留且暂停，旧响应不能将其标已保存；旧搜索版本不能在新文件定位。

## 6. 必需测试

共享DTO严格解码、非法坐标/状态/新增字段；EditorScope目标保护、脏/暂停/基线不同及处理中新输入；lineChanges插入/修改/删除及超限。Playwright运行本机真实搜索/替换/Git，工作台/草稿/model/undo/终端几何回归。手机viewport仅开发检查，真机键盘/触控板/完整主题矩阵延期。

## 7. 错误与正确

错误：批量暂停整个项目、替换后setValue覆盖buffer、选中Git面板仓库后重算所有文件HEAD。正确：目标buffer独立hold→后端版本复验→释放时刷新/保留新输入；文件所属仓库baseline与面板选择各自管理。


## 截图反馈：仓库选择与比较（2026-10-01）

GitPanel发现中或仓库列表为空时禁用Select，显示查找中/未发现Git仓库，空列表不弹空Popup；加载失败显示错误，不伪装空结果。刷新同时重新发现仓库，终端中初始化新仓库后无需切页才能更新；无本地refs禁用引用选择，无历史明确提示尚无提交。

只读比较复用shadcn Dialog/Tabs/Button与lucide全屏图标。每次新比较默认普通窗口、并排，桌面提供并排/行内Tabs和全屏/退出全屏；mobile保持前/后只读文本并可全屏。`ReadOnlyComparison`新增可选sideBySide（默认true），DesktopDiff以updateOptions(renderSideBySide,useInlineViewWhenSpaceIsLimited:false)切换，不重建editor/models/viewModel；使用现有languageForFile语法识别。大小变化由automaticLayout处理，关闭仍按生命周期owner释放。

Fullscreen只改变本弹窗布局到100vw/100dvh，内容区内部滚动，标题/控制/关闭保留；不调用Git写操作、保存/重建编辑buffer或终端。`screenshot-adjustments.spec.ts`真实后端验无仓库、多根、无提交、空引用、全屏与模式切换/默认重置/关闭无pageerror和零写；原editor-recovery继续验草稿/model/undo/恢复。

## Git 加载与背景请求（2026-10-01）

该批旧 Tabs 实现只在打开、切换视图/仓库或明确刷新时读取；当前双区域行为以末节为准。不以固定timer或focus反复全量status。刷新先重新发现仓库，再读取当前视图，发现世代包含project/version/revision，不能先给旧仓库再发一轮请求；切换视图中止旧请求，迟到结果不能更新新视图，失败保留错误且不自动重试。

`baseline-requests.ts` 只共享活跃订阅的相同project/version/folder/path/文件etag键。跨组/StrictMode并发合并，背景基线串行最多一个；保存版本变化读取新基线，focus/online/visibility有30秒冷却（失败同样冷却），隐藏页面不发起请求，没有固定timer。手动Git刷新成功后明确刷新基线，一次刷新revision只通知一次。最后订阅卸载取消、移除排队任务和内存结果；不持久化正文或跨认证复用。Git状态需要用户打开视图或点击刷新更新，不能误宣称实时Git事件订阅。

验证 `use-git-baseline.test.tsx` 的共用、串行、取消/迟到、冷却、失败和保存变化；`git-requests.spec.ts` 用真实隔离后端和浏览器时钟证明两分钟静置没有baseline/status轮询、明确刷新只一轮，并验证慢status时切历史仍可完成。

## Git 双区域与提交图（2026-10-02）

GitPanel 在仓库选择下同时展示上方变更、下方历史。`GitSections` 复用官方 shadcn Collapsible/Resizable，初始 50/50、最小 80px；两区开放时可拖动/键盘调整，收起时保留标题、另一开放区填满剩余空间，两区均可收起。重新展开恢复上次分隔比例。只保存本组件视图状态，不持久化 Git 正文。

`GitFiles` 显示文件名、列表模式的父路径与已知 porcelain 状态、完整路径 title；目录与文件沿用 `FileTypeIcon` 的现有 Material 映射及主题。ToggleGroup 控制列表/文件树，两区独立选择；树只来自当前变更或提交详情，不调用资源管理器逐目录 API。嵌套目录使用 Collapsible，完整路径作为唯一键。

历史 `commitGraph` 依据实际 parents 建立活动 lane，合并分叉/汇合及跨页连续线保留。`GitHistory` 在对应提交下展开文件和 merge 父选择，不把详情放在整个列表末端；分页固定首屏 HEAD。提交文件使用真实详情 stats 的 A/M/D/T/R/C 状态，不从路径虚构状态。图使用主题语义色，展开内容中持续显示经过该行的连接。引用比较入口复用独立 Dialog，保持已打开文件→选定本地引用的能力；只读比较窗口仍保留并排/行内、全屏和打开时编辑快照。

历史不显示“加载更多提交”按钮。在历史自己的 `.git-section-content` 滚动区域距底部不超过 24px 时，发送当前 `next_offset` 和首屏 `head`，追加记录；不监听整个页面、不在 render/effect 初始化时连读所有页。`GitHistory.onMore(): Promise<boolean>` 返回实际请求是否成功；同步 ref 与前台 pending 阻止重复请求，`next_offset < 0` 停止监听。失败/取消不自动重试，保留已有历史与原错误反馈；离开底部再滚回可重试。隐藏区高度为零时不读取，effect 清理滚动监听；加载提示在列表之后，避免向顶部插入提示造成滚动跳动。

合并提交不在详情中常驻展示“比较父提交”下拉框；提交区域复用已有 shadcn ContextMenu/RadioGroup，在右键菜单选择编号和短 ID，勾选实际当前父（首次默认第一父）。`onParent(commit, parent)` 显式传所属提交，即使该条收起或当前详情属于其他提交，也直接展开并只读取所选父，不能先读默认父再读所选父。RadioItem 用 `closeOnClick`，选择后关闭菜单；pending 时禁用选项。作者/时间保留。详情中的文件列表和树共用 `.git-commit-files`：向内 3px、1px 主题 border 左边框贯穿当前提交文件范围，不影响上方当前变更区。

打开/仓库或项目版本变更/明确刷新时先读取历史，再串行读取状态，预留另一个工具槽给编辑基线。状态有独立 AbortController/loading/error，不禁用提交展开或仓库切换；历史/详情/比较操作取消尚未完成的状态读取，取消后用户可明确刷新变更，不自动重试。初始化 continuation 同时检查 effect 生命周期与其实际初始 signal，不能让旧请求在新仓库启动状态读取。隐藏/卸载清理两类请求；无固定 polling/focus 状态请求。手动整体刷新保持重新发现及一次基线失效通知。


验证：`commit-graph.test.ts` 验实际合并分叉/汇合及追加分页的 lane 连续，文件树保留同名完整路径与改名元信息；`GitPanel.test.tsx` 验项目版本变化后旧历史不能继续读取旧状态、慢状态可取消而不回填或重试。`git-layout.spec.ts` 在真实隔离后端验 50/50、拖动/收展恢复、Material 文件/目录图标、真实 merge 两条 parents 及父选择→对应文件树→只读比较，并对源 HEAD/index/config hash。`git-requests.spec.ts` 验两分钟无轮询及慢状态期间展开历史；比较、无仓库、无 HEAD 和 W05 原 model/草稿回归保留。

`GitHistory.test.tsx` 验仅历史滚到底触发、未完成请求去重、失败不连重试/重新滚入重试、pending/隐藏/末页不读及卸载释放；`git-pagination.spec.ts` 用专属 105 提交真实仓库验证 50→100→105、慢第二页期间零重复、冻结 HEAD、追加无丢失/重复及末页停止。

## 提交与历史文件悬浮卡片（2026-10-02）

复用官方 shadcn HoverCard（Base UI PreviewCard），Trigger render 组合现有 CollapsibleTrigger/Button，鼠标停留 450ms 打开、移出 200ms 关闭，移入卡片可操作链接。卡片有界宽高、内部滚动、长消息 pre-wrap/break-anywhere，React 文本渲染；提交显示作者、日期、完整 message/ID、实际文件数及文本增删行数、二进制数和实际父基线。数据加载时明确 loading，失败局部显示统一错误；缺统计不显示零。已展开提交直接使用当前详情，不另发请求。

`useCommitDetails(project, repo, visible, revision, busy)` 以 project ID/version、repo、刷新世代、commit、明确 parent 为键，最多 8 份且 JSON UTF-16 估算总计最多 2MiB 的 LRU 内存缓存。悬停/重悬停/展开复用 read；默认父必须规范化为第一父（根为空），不能因空参数和第一父实际等价而重读。悬停主动取消慢 status，前台展开/比较/引用操作优先并取消预览；快速移出/换提交、隐藏、卸载和 scope 变化中止预览，迟到响应不得回填或写缓存。失败不自动重试，离开后重新悬停或明确展开才重试；无 timer polling、本地持久化、跨认证缓存或后台预加载所有提交。

历史文件列表及树共用 FileHoverCard，直接使用详情 stats，不发逐文件请求；显示完整路径、状态、old_path 和增删行数，两个 null 表示二进制而非零。API decoder 严格验证 fields、状态、安全整数/null 成对、stats/files 顺序及路径唯一、message 长度；github_url 必须空或精确 github.com 公共提交 URL 且 ID 与 commit.id 一致。GitHub 按钮只在合法 URL 非空时出现，真实 anchor 使用 target=_blank、rel=noopener noreferrer；不透传远端凭证、不预访问远端。

正常：悬停长标题查看多行正文→链接打开→展开复用同一响应→文件显示真实 +2/-1；基础：根提交空树、无 origin 无链接、二进制只给明确提示；错误：503 只在卡片显示失败且不循环重试，新项目不接受旧响应。反例：每次 mousemove 发详情、把 null 转 0、把 origin 原串作 href；正例：官方延迟卡片→有界 scope cache→严格 canonical URL→用户点击 anchor。

`useCommitDetails.test.tsx` 验缓存复用/逐出、scope/隐藏清理、迟到响应、busy 优先及失败重试边界；共享详情 fixture 和 decoder 验完整消息、统计及非法 URL/半 null。`git-hover.spec.ts` 在真实隔离后端验截断标题与完整卡片、移入可点击链接、一次详情供悬停和展开、列表/树的真实文本/二进制/改名信息，以及源 HEAD/index/config 不变。外链浏览器测试通过本地拦截响应验证跳转，不记为访问真实 GitHub 的证据。
