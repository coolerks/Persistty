# 技术设计

## 边界
- styles.css 与 ProjectWorkbench 的标签布局承担 R1；标签背景延伸到原生滚动轨道底部，保持完整高度，复用 Tabs 原语与原生滚动，不新增基础控件。
- FileEditor 仅删除 tracked HEAD 展示，useGitBaseline 和 DesktopEditor baseline 保留。
- ProjectWorkbench 统一管理下方所有终端组的最大化；editors Panel 可折叠到0，Group/Panel/provider/宿主身份不变。进入前记录常规布局，最大化布局不交给 useDefaultLayout 持久化。利用现有分隔器和 collapsedThreshold 在顶部标签区域吸附，按钮提供键盘替代入口。退出恢复进入前尺寸；页面刷新恢复常规布局。
- TerminalWorkspace 只接收最大化状态和操作回调，复用 Button 与 lucide 图标。移动端仍采用现有单内容终端视图。
- LoginPage 用已有 Field/Input/Button 的业务组合；styles.css 提供工作台语义背景、细边框和微圆角、适量品牌说明；原 submit/auth 状态逻辑不动。

## 兼容与验证
无数据库/API/schema 迁移。风险是原生滚动条在各浏览器的布局差异、折叠最小尺寸与持久化回调顺序、xterm ResizeObserver 和焦点。真实浏览器测几何、滚动、大小恢复、缓冲区及请求边界；登录测错误/限流与成功跳转。回滚本任务前端文件和规范增量即可。

## 复用查找
已有 Button、Tabs、Field、Input 与 react-resizable-panels 满足需求。shadcn docs CLI 因 npm registry DNS 不可达失败，改用官方 https://ui.shadcn.com/docs/components/base/button 、field、input、tabs 文档核对。不安装或重生成基础组件。旧 UI 任务第四轮仅保留 HEAD 的要求由本次 R2 覆盖。

## 追加修复边界
全屏的分隔器禁用导致不能下拖，改为0px布局保留顶部8px热区；编辑器最小高度为38px，collapsedThreshold=1（1px容差避免浮点尺寸在38px边界提前收起）；向上连续拖到标签行后继续拖才折叠，向下重新展开后继续调整。动态修改阈值会在同一次手势中重置约束，已移除；旧160px最小值导致提前卡住，由最新用户截图需求覆盖。标签滚动条延伸到路径栏顶部4px，内容固定38px，路径栏和固定动作保持各自行高；正文背景提供轨道底色。状态文字移除全高盒子/align-content，用明确行高自然居中。TerminalSession 的历史字节到达即隐藏 live，而 write/rAF尚未完成存在空白窗口；历史改为实时上方独立覆盖层，解析/viewport同步后显示，实时保持尺寸，仅切换visibility；HTTP/WS协议及后端不改。涉及样式、ProjectWorkbench、TerminalSession及对应E2E/owner规范。
