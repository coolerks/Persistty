# 2026-09-30 工作台交互补充

用户在高亮与图标验收后直接要求修复四处交互，本轮在已批准任务中继续实施，不另建任务、不提交或归档。

## 需求与边界

- 已结束会话可用 X、右键关闭当前/其他/全部关闭浏览器标签。关闭状态按终端 ID 保存在本浏览器，刷新不会重新显示；仅明确 terminated 可关闭视图，running 仍使用既有接管确认/批次倒计时，unavailable 不当作已结束。保留服务端元数据与历史。
- 文件树移除条目末尾三点，全部既有操作继续由现有 shadcn ContextMenu 提供。
- 文件树只高亮文件，目录展开与工具栏操作目录保持独立。树可见时切换文件标签不改变树的高亮/展开/滚动；从隐藏恢复（按钮、状态栏、拖拽或初始恢复）才按当前编辑器文件展开祖先、选中并滚动到可见位置。多项目根与分组按 folderId 区分。移动端返回文件视图同样定位。
- 编辑器与终端 tab 的横向滚动条使用一致的细轨道与主题色，保留触摸板、触摸、键盘及拖动滚动。

## 归属与实施

TerminalRuntime 是关闭意图 owner；TerminalTab 解除已结束按钮禁用，TerminalWorkspace/EditorGroup 根据临时视图状态过滤。文件树选择与 reveal 请求属于 Explorer，ProjectWorkbench 只在资源管理器从收起变为展开时生成请求，不订阅 tab 变化来自动 reveal。EntryMenus 移除 DropdownMenu 重复入口；styles.css 局部修改 tab 轨道，不更换 Tabs/Monaco/xterm 引擎。

读取 frontend 目录/组件/状态/类型/质量/生命周期/终端协议与 backend 终端契约。已查找项目组件及 [shadcn ContextMenu](https://ui.shadcn.com/docs/components/base/context-menu)、[ScrollArea](https://ui.shadcn.com/docs/components/base/scroll-area)；右键复用已安装 ContextMenu，滚动条仅为既有原生滚动容器的 CSS 调整。

## 验证

补充回归单测：已结束标签关闭零终止请求、混合批次 running 确认、状态未知保护、目录不选中/可见时 tab 切换不移动/多根 reveal/分页定位/菜单保留。运行前端 lint/typecheck/test/build；隔离浏览器验右键菜单、树展开定位和两类标签横向滚动/浅深主题；Go test/vet 按仓库门禁检查。结果写入本文。

## 实际验收结果

- 前端 lint、typecheck、test、build 全部通过；最终为 19 个测试文件、107 条测试。保留原 Explorer 5 条安全操作/分页/拖放用例，其中两条旧三点菜单点击改为右键；新增 3 条树选择/定位/菜单回归、3 条 runtime 关闭回归与 1 条持久视图偏好用例。既有 jsdom canvas 提示与大块构建提示保留。
- `workbench-interactions.spec.ts` 的完整 Chromium 交互用例通过（最终 9.6 秒）：实际 ProjectWorkbench/Explorer/TerminalTab/Monaco/shadcn 组件；合成接口与 events WS 隔离，不接触用户服务。展开目录离开 hover 后透明；切换 tab 不移动树；重新展开自动找到分页中的 target.ts；右键原有操作全部可达；两类 tab 原生滚动宽度溢出、可改变 scrollLeft，轨道 4px；已结束上下标签 X、关闭其他/全部、刷新/重载后过滤，unavailable 保留；初始侧栏收起布局恢复及手机返回树定位通过；零 mutation 请求、零 pageerror。
- 主题动画结束后检查选中文件的 dark 背景/文字语义色，已人工查看浅深截图；截图位于忽略的 web/test-results。页面使用合成文件，无真实正文。该验收不表示真实 tmux/终止倒计时执行、Debian 持久性或真机验收。
- Go `go test ./...`、`go vet ./...` 通过，部分结果缓存；未改 Go 并发/bridge/watcher，无新增 race 要求。文档链接、围栏和 `git diff --check` 已检查。
- 使用 trellis-check 主会话审查，按 trellis-update-spec 更新组件交互和前端终端 owner 契约；已有 running 确认/批次与 runtime 宿主保留测试继续通过，不改后端协议、数据库或终端进程。关闭仅保存本浏览器 UI 偏好。
- 本轮专属 Vite 5175 验收进程已停止；未操作用户 5173/8080 服务、文件或终端。保留全部未提交修改及任务，不自动提交/归档。
