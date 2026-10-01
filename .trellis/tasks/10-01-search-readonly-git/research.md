# W06 规划研究记录

日期：2026-10-01。主会话直接研究；只读仓库、自有临时 fixture 和本机工具行为探测，没有访问 Debian、用户项目或现有服务。

## 仓库事实与复用位置

- `internal/files/files.go` 的 List/OpenDirectory/ReadContent 与 `secure_linux.go` 的 openat2 实现已拥有路径、根身份、普通文件及跨挂载限制。`internal/files/export.go` 的 CopyVerified 提供有界安全字节复制；复用它生成 snapshot，禁止普通绝对路径 fallback。
- `internal/files/save.go` 的 Save 校验完整 Version（identity/mtime/size/etag），同目录 temp/fsync/rename；`internal/storage/folder_access.go` 的 WithRegisteredFolder 已将发布与项目配置修改串行。替换逐文件复用这些入口，不另造批量无条件覆盖。
- `web/src/features/workspaces/editor-session.ts` 的 EditorScope/FileBuffer 共享真实身份、别名、generation、dirty、saveState、草稿及外部刷新。需要新增针对替换目标的短期保护接口，不能直接调用 scope.protect() 暂停整个项目。
- `DesktopDiff.tsx` 已有独立只读模型及 worker 清理；`DesktopEditor.tsx` 保持 Monaco model/undo。`ProjectWorkbench.tsx` 已有可收展侧栏与手机 files/editor/terminal 单内容导航，但没有 Search/Git；`EntryMenus.tsx` 没有文件夹搜索入口。
- `internal/httpapi/router.go` 所有 protected 路由统一鉴权，写方法检查 Origin/CSRF；`web/src/lib/api/client.ts`/decoder.ts 严格解析协议。新增共享 fixture 延续 tests/contracts。
- 现有 shadcn base-nova 原语包括 Button/Input/Toggle/Select/Tabs/Dialog/ContextMenu/Alert/Empty；搜索/Git 周边复用这些组件，需逐项选择时按官方 registry 添加 Checkbox。没有必要替换布局、Monaco 或原语库。

## CLI 安全方案的证据与限制

[W02 CLI 报告](../archive/2026-09/09-27-debian-spike/cli-report.md)明确 Landlock 探针的系统树许可、特殊文件、已打开 fd 与恶意配置仍有缺口。不能复制它后声称产品只读沙箱已完成。

选择 [process-guidelines](../../spec/backend/process-guidelines.md) 允许的安全输入策略：服务通过原安全根读取，复制受控数据至自身 0700 staging；CLI 不接触真实 workspace，不带用户环境或原始 Git 配置。Git 仅复制经过白名单验证的元数据与当前范围工作文件，移除外部路径入口、禁用辅助程序/隐式更新。安全快照生成和恶意元数据/竞态验证属于实施阻塞关卡，当前没有完成产品级安全适配器。

本机工具实际为 Git 2.54.0（Apple Git-157）、rg 15.2.0；不推断 Debian 版本。合成仓库基础试验：快照仓库 status porcelain v1 -z、HEAD diff 与原 fixture CLI 对照相等，refs 可读，注入的 config include/外部 helper 未执行，临时目录已清理。试验只支持基础可行性，未覆盖 staged/rename/symlink/filters/merge/pack 竞态和产品路径策略。

Git 环境与参数依据 [git 官方文档](https://git-scm.com/docs/git)及[config 官方文档](https://git-scm.com/docs/git-config)：禁用系统/global 配置和 optional locks，禁止 lazy fetch；固定关闭 fsmonitor/hooks/textconv/ext-diff，路径 literal 化。原始 config 只能在 staging 中以不解析 include 的方式读取安全设置，不能将其直接交给工作命令。特殊 repository 格式与外部过滤器不能通过删配置后伪装 CLI 等价。

## 搜索/替换同一引擎

本机合成 stdin 试验确认 rg 15.2.0 `--json --replace` 的 submatch 含 replacement，字节拼接保留 BOM、CRLF/LF 与无末尾 LF；中文/emoji、捕获组、多行替换文本和零宽匹配可由原 offset 精确构造结果。`--passthru` 会增加末尾换行，不能将 stdout 直接当完整替换文件。

[grep-printer 官方 JSON 文档](https://docs.rs/grep-printer/latest/grep_printer/struct.JSON.html)描述 optional replacement；[ripgrep 官方指南](https://github.com/BurntSushi/ripgrep/blob/master/GUIDE.md)与本机 help 的旧描述不能代替实际行为探测。实施时对配置 rg 执行固定合成输入能力探测，锁定支持 JSON replacement 的行为；缺失时明确 tool_unavailable，不静默切换 Go/JS 引擎或升级目标机。是否需要兼容旧 rg 可在开发过程中以同引擎验证解决，任何改变功能语义的替代方案须重新审查。

ignore 规则保留在安全 skeleton 中，由 rg 的实际 ignore engine 筛候选；include 交集不得覆盖强排除或重新纳入 ignore 文件。项目范围外的父级/global ignore/config 不隐式读取。原相对路径由独立映射还原，raw CLI 行文本不作为未验证 UI preview。

## UI 来源与检查安排

核对 [shadcn Dialog](https://ui.shadcn.com/docs/components/base/dialog)、[Tabs](https://ui.shadcn.com/docs/components/base/tabs) 与本地组件目录；正式编辑 UI 时继续核对所需 Input/Toggle/Select/Checkbox 官方文档，保持 base-nova。

本轮仅规划，无 Go/前端产品源码变更；不重跑产品门禁或实机验收，不记为 W06 功能通过。实施中运行必要本地检查，统一实机验收等用户明确安排。
