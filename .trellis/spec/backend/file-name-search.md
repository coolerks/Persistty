# 文件名快速搜索与打开（2026-10-02）

## 1. 范围与触发条件

顶部文件名搜索归 `features/workspaces/FileQuickOpen`，服务归 `internal/search.Service.Names`，路由复用已有鉴权组。普通文件只枚举名称，不生成正文快照；这与内容搜索/替换是不同请求。内容搜索、Git、文件保存和终端生命周期沿用原 owner，未新增数据库或配置。

## 2. 签名

`GET /api/v1/projects/:id/file-names?project_version=<正整数>&query=<关键词>`。统一成功 envelope 的 data 为 `{project_version,items:[{folder_id,path}],truncated}`。`Service.Names(ctx, project, version, query) (FileNames,error)`；TS 客户端 `searchGitAPI.fileNames(project, version, query, signal)` 与 `decodeFileNames`，共享 fixture 键 `file_names`。

UI `FileQuickOpen({project,onOpen})`，onOpen 传现有 `{folderId,path}`；桌面仍经 WorkspaceView/EditorScope 打开，移动端切到既有文本编辑视图。不会创建替代 model 或自动保存正文。

## 3. 请求、响应与资源契约

- query 原始 UTF-8 最多 256 字节、无 NUL/CR/LF，去首尾空白后非空。字面文件 basename 子串匹配，不区分大小写；没有 regex/glob/目录/绝对路径参数，也不按内容匹配。客户端单输入最多 256 字符并补 UTF-8 字节检查。
- 覆盖当前项目所有注册根，按配置根顺序及各根相对路径排序；重叠根同一真实路径去重，保留第一个根身份。结果最多 100 项，只有实际超限或扫描截断才标 truncated；不是完整项目原子快照。
- `discover` 是与内容搜索共享的安全发现 owner：句柄枚举/复验，私有 0700 staging 中建立空占位文件及安全复制的 ignore 控制文件；固定 rg `--no-config --files --hidden --null` 只访问 staging，不交付源 cwd，不使用 shell。普通文件正文不复制，二进制/大文件也可按名称发现。
- `.git` 强排除；范围内 `.gitignore,.ignore,.rgignore,.git/info/exclude` 与搜索配置的默认排除目录继续生效，不读取父级/全局规则。符号链接与特殊文件不纳入结果。只安全读取有界 ignore 控制文件，失败不静默忽略。
- 累计目录项/深度/单目录、ignore 控制字节、工具输出、并发槽与 timeout 沿用搜索/files/toolrunner 配额。context 取消释放槽与 staging。没有跨请求名称/正文缓存或搜索 snapshot ID。响应前复验项目版本。
- 顶栏点击或 Ctrl/Cmd+P 打开 shadcn Dialog；输入法合成、已有 Dialog 或多余修饰键时不抢快捷键。250ms 防抖，改词/关闭/卸载/项目配置变化 abort；迟到响应必须检查 signal，按项目 ID/version/关键词 key 展示，并拒绝版本或 folder_id 不属于当前项目的结果。
- Input 自动获焦，上下键选择、Enter 打开、Esc 关闭，鼠标或 Tab 可访问每个 Button。空词、加载、无结果、过长、截断和 API 失败可见；不自动重试。标题入口不伪造 combobox/listbox 原语；组合已有 Dialog/Input/Button 与文件图标。

## 4. 验证与错误矩阵

| 条件 | 结果 |
| --- | --- |
| 未认证 | 401，沿用统一认证 client/session 处理 |
| 缺失/非法 project_version | 428 version_required |
| 空/非法/超长关键词或项目标识 | 400 invalid_request |
| 项目缺失 / 项目配置变化 | 404 / 409 conflict，不在旧版本打开文件 |
| 符号链接/特殊文件 | 不返回，不读取目标正文 |
| 目录项/结果上限 | 200 truncated=true，提示缩小关键词 |
| ignore 控制文件或工具输出过限 | 413，明确失败 |
| 并发满 / 工具缺失 / 超时 | 429 / 503 tool_unavailable / 503 timeout，沿用统一映射 |
| 取消/迟到结果 | 中止请求/无 UI 回写，无后台重试 |

GET 沿用后端鉴权，不新增写权限；打开结果仍由既有文件接口重新鉴权与校验。query、路径、控制正文及普通正文不得进入日志。

## 5. 正常、基础与错误用例

正常：输入 `资料` 找到隐藏目录的 `资料😀.BIN`，选择第二注册根文件，在原编辑器打开；基础：没有匹配显示空状态，空词不请求；错误：改词后旧请求迟到不能显示旧文件，版本变化要求刷新，不用旧结果绕过 409。

## 6. 所需测试

- `file_names_test.go`：中文/大小写、四类 ignore、默认排除目录、隐藏文件、二进制及 80MiB 文件只按名称发现、根外目录/叶链接不返回、重叠根去重、100 结果与扫描截断、非法 UTF-8/控制字符/版本/取消，以及 staging 清理。
- `search_git_test.go`：GET 未认证/缺版本/配置冲突/非法 query、正常路径身份/无正文、日志不含名字/query/正文/源路径，共享 Go DTO roundtrip。
- TS 严格 fixture 解码、根外/.git/新增字段/超限拒绝；`FileQuickOpen.test.tsx` 验防抖、abort、迟到回写、第二根身份、快捷键、过长/版本错误、Esc 后不请求。
- `workbench-modern-ui.spec.ts` 验真实浏览器焦点/位置/键盘选择/关闭与多根编辑身份；实际本机预览另验证真实服务名称发现→Monaco 打开。继续跑内容搜索/替换的全量单测，防止共享 discovery 改变 ignore/include 行为。

## 7. 错误与正确示例

错误：`rg --files` 直接以用户注册根为 cwd，或调用内容搜索后拿匹配正文代替名称；每次输入立即发送且接受旧响应。

正确：安全枚举→私有占位树/有界 ignore→固定 rg→有界名称 DTO→防抖/abort/key/版本校验→现有编辑器按 `{folderId,path}` 打开。
