# 工作台视觉验收

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
