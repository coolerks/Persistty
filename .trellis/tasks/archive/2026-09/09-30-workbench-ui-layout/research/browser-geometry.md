# 工作台整页溢出只读测量

日期：2026-09-30。使用 CUA 的只读 Playwright DOM evaluate 测量用户已打开的本地项目工作台；未刷新页面、切换标签、输入命令或修改样式。记录仅包含尺寸与公开 CSS 类，不保存项目路径、文件正文、Cookie 或终端输入。

## 结果

| 对象 | 测量值 |
| --- | --- |
| 浏览器视口 | 1440 × 656px，scrollY=0 |
| document | clientHeight=656，scrollHeight=1013，scrollWidth=1440 |
| body / root / app-shell-workspace | 高度均为 656px |
| app-shell-workspace | overflow-y:hidden，bottom=656 |
| workbench | top=48，height=608，bottom=656 |
| workbench-status | height=25，bottom=656 |
| terminal-runtime-host / mount | top≈541.57，height≈89.43，bottom=631 |
| explorer-scroll | top=118，height=513，bottom=631，overflow:auto，position:static |
| 下方 tree-root | top=694，height=319，bottom=1013，position:static |
| 最末越界 sr-only span | top=1012，height=1，bottom=1013，position:absolute，clip-path:inset(50%) |

越界 sr-only 的定位祖先链：span → 文件树 group → tree-root → explorer-scroll → explorer → workbench-sidebar → panel → group → workbench-body → workbench → app-shell-workspace → root → body → html。各祖先 position 都为 static，offsetParent 观测值均为 null；其绝对定位边界未锚定在文件树滚动容器内。

Monaco 内部 lines-content 具有很大的布局高度，但它在引擎自己的裁剪/滚动边界内；不能因为其 bounding box 很大就改写引擎内部样式。

## 推断与验证边界

- 已确认：document 超高，与用户截图现象一致；工作台主框及状态栏本身贴合视口。
- 候选根因：文件树的绝对定位隐藏辅助节点越过静态裁剪祖先，扩展初始包含块的可滚动区域。实施时先为滚动/辅助标签建立正确定位边界，再测 document 的尺寸变化。
- 尚未确认：给定样式修改能完全修复全部场景。规划阶段未做 CSS 注入或 A/B 修改，不能把推断升级为已证实根因或已完成修复。
- 实施必须补：多根目录、长文件树、菜单/tooltip/dialog 打开关闭、编辑器分组、终端上下移动/隐藏、矮窗口、窄屏和手机软键盘下的几何断言。既要 document 不外溢，也要文件树内容仍可内部滚动且键盘可达。
