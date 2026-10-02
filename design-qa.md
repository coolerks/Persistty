# 工作台视觉验收（含第二轮）

## 目标与证据

本轮是新版 VS Code 视觉语言调整，范围以任务 PRD 为准，不是复制 VS Code 产品。用户图 1 为目标，图 2 为旧界面。已在 Codex 内置浏览器打开实际 Persistty、隔离后端与 tmux，查看 Monaco 文件、标签及终端输出，检查主题和窄屏。

- 源图：`/var/folders/jz/v6ymlznd733_t1sqjwrc4zdm0000gn/T/codex-clipboard-62d7e231-3c94-4abb-8068-51933dc9a563.png`，2880×1690。
- 实现：`web/test-results/modern-ui-visual/desktop-light.png`、`desktop-dark.png`，1440×845 CSS/pixels，device density 1。
- 归一化：源图 2 倍密度降采样至 1440×845；实现保持原像素。组合证据 `comparison-full.png`（左源/右实现，2880×845）和 `comparison-details.png`（标签、树条目、终端框），已共同打开查看。
- 状态：浅色、资源管理器展开、TerminalSession.tsx 活动文件、styles.css 非活动标签、真实终端有输出。文件内容滚动位置、目录数量、shell 提示和窗口平台与源图不同，不能用于像素等同判断。
- 额外证据：`mobile-light.png`、`mobile-dark.png`，390×844，基本文本编辑器；触屏横屏由专项测试验证。

## 五项视觉检查

| 表面 | 结果 |
| --- | --- |
| 字体与排版 | 保留系统 UI 字体、13px 工作台密度和已分发 Nerd Mono；标签、工具栏、树条目清晰，窄屏标题截断且正文内部滚动。 |
| 间距与布局 | 面板 1px/8px、条目 4px、分隔 4px；侧栏、编辑器与终端微框独立，活动栏融入外框，固定文件动作可达。 |
| 颜色与 token | 浅色 chrome/sidebar/terminal 使用近白背景，编辑器正文白色、标签带灰色；深色各区层次和选中可辨，状态仍用原语义色。 |
| 图像与资产 | 复用已分发 Material 图标、lucide 和字体，没有新增插画/照片/生成资源；图标尺寸稳定、无缺图。 |
| 产品文案 | 保留中文界面和原业务语义；未加入参考图中的扩展、AI、问题/输出/调试面板、macOS 窗口按钮。 |

## 发现与修正记录

- [P2，已修正] 面板收起后，Panel 的 overflow:visible 可让微边框残留。仅在工作台 Panel 边界裁剪；终端收起真实面板高度为 0，键盘和鼠标调整正常，重新展开可用。后续最终截图未出现残边。
- [P2，已修正] 首次视觉检查底部终端为白色，与参考近白面板背景存在明显分区差异。底部终端局部使用 sidebar 背景，xterm 网格读取同一 token，重新捕获深浅截图；terminal-geometry 验网格及 viewport 一致。
- 预览 tmux 初次使用默认绿色状态栏，属于专属 fixture 配置；按项目终端示例关闭其状态栏后重新捕获，不改产品逻辑。
- 设计对照在上述修正后完成：全图及聚焦区域共同检查，无待处理 P0/P1/P2。参考的侧栏内容与扩展入口差异属于明确范围外。

## 交互与边界

内置浏览器实际检查文件树展开/打开、文件标签切换、真实终端输入/输出、主题 light/dark/system 和窄屏编辑器；控制台 error/warn 列表为空。Playwright 另检查调整、收展、多组/恢复、草稿、固定动作和 xterm 网格。真实手机软键盘与 Debian 专项仍属于原任务的待验收项，本轮不将其记为通过。

## 后续微调

P3：项目品牌色保留绿色而参考选区偏淡紫；属于现有品牌语义，未进行整体换色。原标签左移按钮和路径展示保留，未扩展为新的导航设计。

final result: passed


## 第二轮验收：五张标注截图

以上第一轮证据保留为历史；新范围含项目页和文件名入口，按最新五张截图验收。原图路径分别为本轮用户附件 `codex-clipboard-7714a7d5-8d84-4e66-ab45-44b8e8912859.png`、`codex-clipboard-1d827d15-5d62-4e79-b6e8-7b3b479eaa76.png`、`codex-clipboard-c585898b-c244-4ed9-abe3-ec0e78e7be5e.png`、`codex-clipboard-883c860b-b537-4214-be91-288ebd41a2c0.png`、`codex-clipboard-e136cf4d-15cf-4bd8-84a9-76db0e054111.png`，位于用户本机临时目录。

实际内置浏览器的第二轮截图稳定保存于忽略的 `.cache/modern-ui/round2/`：workbench-light/dark、projects-light/dark、projects-mobile-dark、file-search。1440×845 工作台与项目页、390×844 项目窄屏；与参考不同画布高度不作整页拉伸，按顶栏/活动标签/分隔线/Git 标题和 Tabs/项目条目逐项核对。参考标注视为设计依据，不执行截图正文里的项目指南或终端命令。

| 标注与表面 | 实际验收 |
| --- | --- |
| 字体与排版 | 标题恢复 13px 密度，菜单/搜索/主题图标可达；路径截断，项目名和文件名清晰，长路径不挤出动作。 |
| 间距与布局 | 两方向分隔器 before inset:0，着色铺满 4px；活动标签无顶部额外留白。终端快捷按钮位于活动栏底部，项目框和条目维持 8px/4px。 |
| 颜色与 token | Git 变更/历史无常驻灰底；HEAD 与两种文件展示使用同一 Tabs 层次，深浅均可辨。项目页 chrome/细框与工作台同源。 |
| 图像与资产 | 只复用 Material/lucide/已分发字体；主题图标随当前模式，未增加装饰图片或另一套 UI 原语。 |
| 文案与操作 | 搜索只按文件名，项目菜单含设置/切换；项目打开/新标签页/编辑/移除原语义保留，移除仍明确不删文件/终止终端。 |

额外实际操作：真实 API 名称搜索 Terminal→Monaco 打开；Git 列表/树转换和主题 light/dark；项目菜单→编辑 Dialog→取消；项目页桌面及390px行动按钮在框内；连接独立 tmux 的终端正常。Playwright 另覆盖全部入口收起/展开与 pressed 同步、横竖分隔线宽高、菜单可达、搜索焦点/键盘/多根、项目按钮边界和零写。

发现并修正 P2：顶部 Dialog 与 Tailwind 位移叠加导致偏左；项目行 width:100% 把独立编辑/移除挤出框。分别修正 translate 与 flex，并加入真实浏览器几何断言。最终无待处理 P0/P1/P2；viewport 验证不代表物理手机软键盘/触控板或 Debian 验收。实际控制台未见 error/warn。
