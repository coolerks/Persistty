# W06 工作台搜索与 Git

## 1. 范围与触发

W06前端归 `features/search/SearchPanel`、`features/git/GitPanel`，由 ProjectWorkbench 编排。HTTP/DTO 权威见[后端契约](../backend/search-git-contract.md)。复用现有model、EditorScope、DesktopDiff、API request/error/envelope与shadcn base-nova，不增加Git写操作。

## 2. 签名

`SearchPanel({project,mobile,intent,visible,onOpen})`；`SearchIntent={folderId,path,replace,id}`。`GitPanel({project,mobile,visible})`。`ReadOnlyComparison({original,modified,path,mobile})`：桌面懒加载只读DesktopDiff，手机分区显示原/新文本。`EditorScope.prepareReplacement(files)` 返回 `{protectedIDs,release()}`；FileBuffer.holdReplacement(version) 成功返回一次性释放函数，否则null。API和严格解码分别在 `lib/api/search-git-client.ts`、`search-git-decoder.ts`。

## 3. 状态与行为契约

桌面活动栏和目录右键共享搜索入口；Ctrl/Cmd+Shift+F/H查找/替换，Shift+G打开只读Git。modal打开及输入法合成时不抢快捷键。手机单内容导航新增Search/Git，保留原编辑器/终端组记录。workspace schema v2兼容新增可选 sidebar 与mobileView值，正文/搜索ID/预览ID只存内存。

Search与Git面板隐藏后保留当前会话表单状态；route/配置/认证变更中止请求，清理拥有的搜索/预览。请求AbortController阻止迟到结果；搜索条件修改后旧结果标明需重搜，不允许用新条件与旧结果生成预览。include/exclude用`;`分隔，先选择匹配再预览，可在预览按文件缩小应用范围。逐文件结果与状态查询可见，应用不自动重试。Replace DialogContent通过initialFocus绑定“关闭预览”按钮ref，默认焦点落关闭/取消按钮，不落应用按钮；执行时弹窗内提供“停止剩余替换”，再查询部分结果。

定位先复验服务器版本与本地generation/dirty；失败保留输入并要求重搜。位置请求含project/file/version与一次性ID，DesktopEditor设置selection，手机textarea设置UTF-16范围，不创建新的编辑model或替换正文。

替换保护仅影响目标真实identity及alias。dirty/saving/暂停/冲突/加载/版本不同均跳过；成功持有期间禁自动及显式保存。新输入继续写入草稿；释放后干净buffer刷新，新输入保留且暂停，不让旧autosave覆盖新磁盘。其他文件的autosave继续。

Git面板仓库选择不控制文件的baseline。HEAD baseline按对应文件获取，输入只重算行标记，不因每次按键读Git；保存etag变化、focus/online/15秒轮询刷新，迟到响应按key抛弃。tracked基线与buffer比较，未跟踪/基线不可用显示文字；不把baseline写进buffer。行比较先裁剪共同前后文，LCS≤250000格/20000行，超限以变化区间有界标记；删除锚定现有行。decorations独立collection，theme使用语义色，原model/undo保持。

Git总变更用total_paths，暂存/未暂存用porcelain字段。历史下一页发送首屏head以冻结遍历；刷新才读取新HEAD。比较默认磁盘，可明确选择打开比较时的编辑器快照，界面标明二者。所有比较model关闭后释放；不通过比较窗口保存文件。

## 4. 验证与错误

后端400/409/410/429/503按统一错误文案显示；tool/metadata失败不能显示clean。GET/POST仍服务端鉴权，隐藏按钮不能替代安全边界。结果部分成功、取消或跳过都保留逐文件状态；“已接入”不等于真实设备验收。

## 5. 正常、基础与错误用例

正常：搜索中文/emoji准确选择原model，预览替换显示两边，保存后HEAD修改条保留。基础：无仓库显示Empty，移动比较为只读文本。错误：预览期间新输入保留且暂停，旧响应不能将其标已保存；旧搜索版本不能在新文件定位。

## 6. 必需测试

共享DTO严格解码、非法坐标/状态/新增字段；EditorScope目标保护、脏/暂停/基线不同及处理中新输入；lineChanges插入/修改/删除及超限。Playwright运行本机真实搜索/替换/Git，工作台/草稿/model/undo/终端几何回归。手机viewport仅开发检查，真机键盘/触控板/完整主题矩阵延期。

## 7. 错误与正确

错误：批量暂停整个项目、替换后setValue覆盖buffer、选中Git面板仓库后重算所有文件HEAD。正确：目标buffer独立hold→后端版本复验→释放时刷新/保留新输入；文件所属仓库baseline与面板选择各自管理。
