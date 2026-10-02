# 技术设计

前半部分保留第一轮历史设计；当前范围和验收按文末第二轮及已同步契约执行。

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


## 第二轮细化（用户 2026-10-02 直接授权实施）

本轮覆盖此前“只调整工作台外观、不改项目页/API”的范围限制；第一轮已由用户提交为 `6881b6c`。按最新五张截图处理：

- 分隔器激活色铺满 4px 间隙，保留 8px 热区；选中编辑器标签贴顶。
- 文件、搜索、Git 活动按钮统一点击收起/再次展开；选中态由实际面板可见性决定。终端入口移至活动栏底部并同步可见性。
- Git 变更/历史标题无常驻背景；比较基线、列表/树形共用已有 shadcn Tabs，改善深浅色对比。
- 项目页采用工作台同一 chrome、8px 微圆角与 1px 框线的紧凑项目条目，保留打开/新标签页/编辑/移除语义。
- 项目设置移至项目名菜单；主题使用图标 Select 下拉入口。
- 顶部居中文件搜索入口，点击或 Ctrl/Cmd+P 打开 Dialog；按名称关键词（不区分大小写、字面子串）搜索多个注册根并打开现有编辑缓冲区。支持中文、键盘上下/Enter、Esc、250ms 防抖、取消及旧结果防回写。
- 新增鉴权 GET `/projects/:id/file-names?project_version=...&query=...`，返回 `{project_version,items:[{folder_id,path}],truncated}`；非空 query 最多 256 字节，无 CR/LF/NUL。最多 100 结果、累计沿用搜索目录项/工具时间/控制文件限额；扫描截断明确展示。
- 名称发现复用安全句柄枚举、私有占位文件树与 rg 忽略规则，普通文件正文不读取，二进制也可按名发现，符号链接/特殊文件不纳入。`.git` 强排除与配置目录排除保留。重叠根相同真实路径去重；响应前复验项目版本，无长期缓存/正文快照。

实施顺序：先修复布局/切换与既有控件；再统一项目页及顶栏；补名称搜索服务/严格 DTO/界面；最后检查实际交互与视觉、前后端门禁、浏览器专项和规范同步。当前主会话单代理完成，不部署，不自动提交/归档，不改变 W05/W06/W07 未完成验收。


第二轮控件查找记录：现有 Select、DropdownMenu、Dialog、Input、Button、Tabs 已足够；对照官方 https://ui.shadcn.com/docs/components/base/select 、https://ui.shadcn.com/docs/components/base/dropdown-menu 、https://ui.shadcn.com/docs/components/base/dialog 。FileQuickOpen 是业务组合，不新增基础控件/依赖；`discover` 抽取现有搜索安全发现，不为名称搜索另设弱安全路径。完整跨层签名见 `.trellis/spec/backend/file-name-search.md`。
