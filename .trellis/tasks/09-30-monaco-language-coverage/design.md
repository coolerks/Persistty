# Monaco 语言设计

## 架构与职责

- 保留 workspaces 当前真实 owner，不提前搬目录。新增轻量 language adapter 与生成 metadata，DesktopEditor 负责引擎及语法加载，ProjectWorkbench 负责视图语言选择。
- 从固定安装包的基础语言注册、JSON 注册及 plaintext 元数据生成统一只读清单（id/aliases/extensions/filenames/firstLine）；采用已有 TypeScript AST API读取字面量，不用 regex 猜 JS 语法或执行第三方任意代码。以 Monaco 主入口实际导入顺序确定重叠扩展的稳定优先级，禁止按文件系统随机顺序覆盖。
- 源 metadata 不匹配已知静态形式或缺语言时生成失败；生成物含 Monaco 版本，自动化核对运行时 getLanguages()，升级不可静默少语言。
- 轻量 metadata 可被图标解析复用，不能因文件树 import 而提前加载 Monaco/editor/worker。引擎继续通过现有 lazy DesktopEditor 加载；语法 loader 沿用 Monaco。

## 推断与选择

顺序：显式 UI 选择 → exact basename → 最长 registered extension → 已注册的首行 shebang → plaintext。扩展匹配包括隐藏文件；首行仅对文件名/后缀未匹配的有界文本执行，正则来自固定 Monaco 定义而非用户输入。相同后缀按上游注册顺序稳定决胜。

自动 filename/extension adapter 也供图标语言回退使用；图标在未读取文件正文时只做路径推断，不为图标额外请求内容。

拟在桌面现有 breadcrumb 操作栏复用 `@/components/ui/select` 显示“语言模式”。选项为自动和全部 ID，常见语言用 aliases 作标签，FreeMarker 变体显示可区分的 Angle/Bracket、Dollar/Bracket 名称；控件和提示用中文。官方 Select 已查阅，复用既有滚动、键盘及焦点支持。宽度有界，保持文件路径可截断。

状态由 workspaces 当前视图 owner 管理，按 projectId/folderId/path 保存临时覆盖；同文件两个分组一致，最后一个视图关闭时清除。跟随既有 relocate/remove 做键迁移或删除，不增加服务器字段、localStorage 版本或独立万能 store。真实文件身份仍由现有文件 API 决定，语言设置不提供访问授权。

DesktopEditor 消费最终 ID，切换时只更新既有 model 的 language；保持 URI、内容、readOnly、worker 与当前 theme。注册 provider 加载不等于诊断开启；复核 JSON/CSS/HTML 现有诊断选项，遵守项目禁诊断契约，不引入 LSP。

## 兼容性、风险与回滚

- 手机不加载 Monaco，不提供本次桌面模式选择；未改变文本/图片/二进制分类、安全上限与后端访问。
- FreeMarker/SQL 方言不能仅靠文件名区分，显式选择保证可达；不以猜测 SQL 内容代替用户选择。
- 生成 metadata 的内部包路径被 0.57.0 锁定；升级时重新研究。支持加载全部语言可能影响构建体积，记录前后主 bundle 与语法 chunks，保持按需加载。
- 回滚 UI/adapter/生成脚本及相应状态键即可；无数据库/API迁移，不触及终端。
