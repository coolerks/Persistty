# 技术设计

## 边界与归属
视觉差距归 `web/src/app/styles.css` 的工作台布局和 feature 类，直接修改已有规则，避免叠加另一套覆盖主题。`ProjectWorkbench.tsx` 仅为三个侧栏入口加入 `aria-pressed` 以表达当前视图，已有 Button 继续承担键盘和焦点。

## 视觉结构
- 定义工作台专用 chrome/sidebar/border/tab-strip 语义 token 与统一 8px 面板、4px 条目半径、4px gap。
- 工作台外框用 chrome 背景；侧栏独立框，编辑器与底部终端各组独立框。react-resizable-panels 容器与 ref/ID 保持原位置。
- separator 占据 gap，默认透明，hover/focus/拖动时中央细线着色；不缩小现有拖拽热区。react-resizable-panels 外层默认 overflow:visible，工作台 Panel 局部使用 overflow:clip，避免归零后子框的边框溢出。底部终端局部继承侧栏背景，xterm colors(host) 与 CSS 同源。
- 编辑器白色/深色正文与较深 tab strip 形成层次；选中标签圆角顶边与正文连通，去顶色条和 Trigger 下划线。
- 文件树条目保留缩进与单选语义，条目 wrapper 左右 5px 内收。Git 行用同尺度圆角。
- 移动内容容器单一边框/圆角，编辑器和终端子框避免双重边框；触屏媒体查询沿用既有条件。

## 复用与资产
已检查项目 Button/Tabs/Resizable/Select 与官方文档：
- https://ui.shadcn.com/docs/components/base/button
- https://ui.shadcn.com/docs/components/base/tabs
- https://ui.shadcn.com/docs/components/base/resizable
继续复用已有组件，不新增基础控件。参考图没有需要生成的插画或照片；现有 Material/lucide 图标及系统 UI 字体、Nerd Mono 保留。

## 兼容与回滚
不新增存储/API/依赖，不改变 buffer/model/runtime mount。回滚样式与 aria 属性即可；旧布局尺寸仍兼容。主要风险为边框/gap减少内部像素、圆角裁剪和焦点边界，因此用真实浏览器验证几何、滚动和 Monaco，同时复跑既有恢复/交互专项。
