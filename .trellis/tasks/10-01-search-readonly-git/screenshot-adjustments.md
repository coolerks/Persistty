# W05/W06 截图反馈修正（2026-10-01）

用户以五张标注图明确授权修复和调整，沿当前验收任务实施，单代理 inline。

| 项目 | 当前差距、归属与修改边界 |
| --- | --- |
| F01 目录选择溢出 | DirectoryPicker 的 grid/flex 子项受长路径最小内容宽度撑大；约束自身列/输入/路径/列表，不用全站裁剪隐藏问题 |
| F02 只读比较 | GitPanel 弹窗缺全屏及模式选择；复用现有 shadcn Dialog/Tabs/Button，DesktopDiff 公开 options 切换并排/行内，默认并排；大小和模式不重建原编辑 model、不写 Git |
| F03 默认 shell | config.Load 固定 sh；改为未显式配置时检测服务 UID 的系统登录 shell，失败回退 sh；显式配置与开发 --shell 优先，不读取网页参数或仅信任 SHELL 环境，不重启已有 pane |
| F04 无仓库选择器 | GitPanel 无选项仍可打开；加载/空列表时禁用并给出清晰状态，多根发现保持原契约 |
| F05 新仓库报错 | 先用自有 git init 无提交/无 index 的临时仓库复现，修复 snapshot 原因而非掩盖错误；继续拒绝危险元数据/缺对象，保留只读与完整工作树边界 |

UI 查询记录：已核对工程现有组件，并读取官方 [Dialog](https://ui.shadcn.com/docs/components/base/dialog)、[Tabs](https://ui.shadcn.com/docs/components/base/tabs)、[Select](https://ui.shadcn.com/docs/components/base/select)；使用已有组件组合，不添加基础交互原语或依赖。

验证：真实 Git 集成和登录 shell 选择/回退/显式覆盖回归；Playwright 验长路径窄屏、无仓库/无提交状态、模式与全屏切换、关闭后无 pageerror/零写；完整 Go test/vet/race 与前端 lint/typecheck/test/build。远端 Debian、真实手机与正式部署仍按独立验收记录，不因本轮截图修复宣称通过。


## 实施结果

- F01：DirectoryPicker自身minmax列、flex宽度与dvh滚动约束。1440/390超长路径/中文目录名均在popup内，输入/打开/列表/选择/取消按钮边界通过；已目视检查窄屏截图。
- F02：Git比较窗口增加全屏按钮与并排/行内Tabs，默认并排；每次重新打开重置普通窗口/并排。仅updateOptions与automaticLayout改变同一比较实例，关闭沿原清理顺序；未修改原编辑buffer。复用languageForFile语法识别，手机保持只读前后文本。
- F03：未配置shell时实际账户检测，显式YAML/--shell优先。真实本机沙箱外检查选择/bin/zsh；受限沙箱账户目录查询失败则正确回退sh。自有tmux中sh/bash/zsh的pane程序及SHELL通过，隔离HOME/ZDOTDIR。需要重启后端并新建终端才使用新默认，已有pane继续运行。
- F04：发现中/无仓库禁用Select，无本地引用同样禁用，空历史明确显示尚无提交。刷新同时重扫仓库和读取当前状态；修正effect cleanup取消新请求的时序，并断言刷新得到实际200状态响应。
- F05：真实git init无HEAD、无index用例先失败（tool unavailable），修正metadata空目录复制后通过；无提交状态/历史/refs/HEAD与stage比较均准确，源index未创建或字节不变。仓库内容不可表示按repository_unavailable提示，不再混用文件编辑类型错误。用户截图中的所有仓库正文未读取、未修改。

## 实际检查

| 检查 | 结果 |
| --- | --- |
| Go test / vet / race ./... | 最终产品源码通过，包括私有真实tmux shell、Git新仓库/暂存/源只读、既有恶意配置与缺对象 |
| 前端 lint / typecheck / Vitest / build | 最终刷新修正后通过；23文件134项单测，字体/37项许可通过；保留原canvas/chunk告警 |
| Chromium综合回归 | 截图专项2项、W06真实链路2项、原W05 editor-recovery6项，共10项通过 |
| 最终后端＋最终刷新专项 | Chromium4项20.0秒、WebKit4项23.2秒均通过，串行独立输出目录 |
| Linux专属fixture | Git测试Linux/amd64交叉编译通过，非法UTF-8文件名测试只编译未运行；APFS拒绝创建此类文件，不强造“通过” |
| Debian/物理手机 | 本轮未执行，沿统一验收待办；不将桌面WebKit/viewport/本机tmux当实机通过 |

WebKit最初因为子Dialog关闭动画尚未移除时立即查找父“取消”而严格匹配失败；等待子Dialog移除后原场景复跑通过。shell测试最初按sh进程名判断，在macOS看到bash；验证平台实现后仅接受macOS sh的bash别名，其余shell仍严格匹配。未放宽零写、取消、元数据或模型断言，没有原始私有日志/配置输出。

截图保存在已忽略的web/test-results/screenshot-refresh-final与screenshot-refresh-webkit；新fixture仅为自有临时树。所有修正主会话执行，没有commit/push/部署/归档，两个任务仍in_progress。

收尾：54个本地Markdown链接、2个任务JSON、11个Go文件格式及git diff --check通过；本轮两个fixture临时树均已移除，8089/5179测试端口关闭，专属Vite/后端与tmux测试资源已清理。第7次开发日志以--no-commit记录。
