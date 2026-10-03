# 主题、字体、图标与分发

主题setting为system/light/dark，首次system；effective根据matchMedia计算。偏好在当前浏览器本地持久化，不存SQLite，不覆盖其他设备。首帧在渲染前应用有效主题；system监听并清理media listener，显式light/dark不被系统事件覆盖。
语义 CSS variables 同时覆盖 shadcn/ui、Monaco theme/DiffEditor、xterm.options.theme、context menu/dialog、Explorer/scrollbar；切主题不重建 editor/terminal 或丢 buffer/history。字体/色彩/选区在三模式实测。

内置 JetBrains Mono Nerd Font NL：资产任务锁定 Nerd Fonts release、Mono/NL variant、真实 font-family metadata、格式和 glyph 覆盖。自托管 font-face，等待 document.fonts.load/ready 后 Monaco remeasure/xterm fit；fallback 为 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace。不能只设置 font-family 名字却没分发字体。测试 Oh My Zsh/Powerline/Nerd glyph、中英文、宽度对齐。转换 WOFF2/子集不能意外移除 glyph；未授权不能重新许可。
Material Icon Theme 用锁定上游资源和映射，优先 exact filename -> longest compound suffix -> extension -> language ID -> fallback，folder 独立；处理 Dockerfile/package.json/go.mod/.gitignore/.env/yaml/yml/ts/tsx/d.ts。必要大小写策略及 basename 特例有测试，不仅 extname。icon SVG 作为可信内置资产分发，用户 SVG 安全策略不同。

导入前逐项检查实际版本 LICENSE/NOTICE：字体原始/补丁/图标许可分别核查；第三方资产清单含 source URL、version/checksum、license、修改/转换记录，分发包含必要 copyright/license/notice。项目 MIT 不能覆盖第三方许可。Material 图标已按下文契约分发；W05 已分发 Nerd Font，验证范围见下文；真实手机和 shell 主题实际输出矩阵仍需专项验收。来源入口：[Nerd Fonts JetBrainsMono](https://github.com/ryanoasis/nerd-fonts/tree/master/patched-fonts/JetBrainsMono)、[Material Icon Theme](https://github.com/material-extensions/vscode-material-icon-theme)。

测试主题持久/API失败/system event、Monaco/xterm 不丢状态；图标 exact/compound/fallback/目录和 Unicode；生产 build 资产加载路径/许可证随包/字体实际 glyph 截图证据。


## 内置 Material 图标生成契约（2026-09-30）

### 1. 范围与触发条件
更新图标版本、映射、资源路径或构建入口时遵守本契约。owner 为 `web/scripts/generate-material-icons.mjs`、`web/src/features/workspaces/file-icons.ts` 和 `FileTypeIcon.tsx`；文件树、根目录、文件标签及目录选择器复用，功能按钮保留 lucide。

### 2. 签名
在 web/ 执行 `npm run prepare:assets`，顺序生成语言和图标资源；predev/prelint/pretypecheck/pretest/pretest:e2e/prebuild 自动调用。`fileIconFor({ path, kind?: "file" | "directory" | "root", expanded?: boolean, theme?: "light" | "dark" }): string` 返回受控 ID；`fileIconURL(id: string): string` 返回 Vite BASE_URL 下的本地 URL。无新增 HTTP/WS/DB 契约。

### 3. 契约
精确 devDependency `material-icon-theme@5.38.1`，源提交/integrity/发布包 SHA-256 记录在 `web/licenses/material-icon-theme-source.json`，实际依赖来源以 npm lock resolved 为准。仅构建脚本调用上游默认 `generateManifest()`，保留 patterns/clones 展开、languageIds、目录/root 开闭和 light 映射，不手写子集或自行选择框架图标包。

生成 `web/src/generated/material-icons.json` 和 `web/public/material-icons/`，二者忽略，干净依赖安装后可重建。1251 个定义的 SVG 仅在字节变化时原子替换，完整发布后才清理过期 SVG；重复生成不清空目录、不触发无变化 HMR。assets.json 含每项 SHA-256，MIT 原文随图标和第三方许可清单分发；不能只扫描运行依赖而漏掉构建依赖的分发资产。

basename/关联键小写匹配，显示原名保留；`.env` 的 env 后缀参与匹配；exact basename 优先，再最长后缀、语言 ID、通用 file。light 表按层覆盖默认表，目录/root 独立。未知类型经 plaintext 语言映射使用上游 document，无语言关联才通用 file；手动语言选择不改变路径图标。不为图标 fetch 正文，不根据用户文件名拼资源路径。

组件为固定尺寸装饰 img，空 alt、aria-hidden、禁图片拖拽；失败仅回退一次到对应通用图标。`use-effective-theme.ts` 用单一 MutationObserver 服务所有消费者，不为每个树节点创建 observer。用户 SVG 预览隔离规则保持独立。

### 4. 验证与错误矩阵
| 条件 | 行为 |
| --- | --- |
| 版本/integrity 或 LICENSE 不符 | 停止生成，重新审查 |
| 非受控 ID、源路径越界、SVG 缺失或关联断链 | 停止生成，不发布缺图构建 |
| constructor/__proto__ 等文件名 | 仅查 own entries，不命中对象原型 |
| 图片加载失败 | 单次本地通用图标回退，不无限重试 |
| 文件/后缀/语言都无映射 | 通用 file |

### 5. 正常 / 基础 / 错误用例
正常：yaml/yml/yaml.dist/yml.dist 同 yaml，d.ts 为 typescript-def、ts 为 typescript，package.json 优先 nodejs。基础：src 开闭分别 folder-src/folder-src-open，浅色 toml 为 toml_light。错误：只复制 SVG 丢别名、内联用户 SVG、因为生成物已存在而跳过干净构建验证。

### 6. 所需测试
`file-icons.test.ts`、`FileTypeIcon.test.tsx` 验最长后缀、隐藏/大小写/Unicode/原型名、语言回退、目录/主题、失败回退；生成脚本全量验证引用，构建核对分发 checksum/许可。`web/tests/e2e/editor-assets.spec.ts` 验本地图片请求；`editor-workbench.spec.ts` 在专属 fixture 上验真实树/标签和布局，未提供 fixture 而跳过不能算通过。

### 7. 错误与正确示例
错误：`return '/icons/' + userFilename + '.svg'`。正确：`fileIconURL(fileIconFor({ path, theme }))`，仅映射到受控资源；prepare:assets 先验证完整资源，再构建。


## W05 字体分发契约（2026-10-01）

1. 范围：字体升级、替换、加载或 editor/terminal 测量变化时适用。
2. 签名：在 web/ 运行 `node scripts/verify-font.mjs`，build 先校验字体，再 collect-licenses、类型检查与 Vite 构建。CSS `@font-face` 名为 `Persistty Nerd Mono`，本地资源为 `fonts/JetBrainsMonoNLNerdFontMono-Regular.ttf`。
3. 契约：锁定 Nerd Fonts v3.5.1 Mono/NL Regular 原始 TTF，未转换/子集。真实 family 为 JetBrainsMonoNL NFM，SHA-256 `d9a80146bbf2ff23b187316a3e82ee2a2ceab8220267fe09ac150ca46ab2c3bd`；source/names/hash 记录在 `web/licenses/jetbrains-mono-nerd-font.json`，原始 SIL OFL 1.1 与汇总许可随构建分发。Monaco fonts.load 后 remeasureFonts/layout；xterm 先 fallback，fonts.load 后更新 options.fontFamily 触发字符服务复测，再 fit，observer 不发送 controller resize。
4. 矩阵：hash/name/glyph/等宽/许可不符停止构建；字体网络加载失败保留 fallback，不伪报 glyph 通过；异步完成后宿主已卸载不重测已释放 runtime。
5. 用例：正常本地 TTF 加载后 Nerd glyph 与 ASCII 等宽；基础离线资源可访问；错误只写 family 名未分发，或加载字体后只 fit 却缓存旧字符尺寸。
6. 测试：verify-font 验 name/cmap/hmtx 中 U+0041、U+E0B0、U+F017、U+F120；生产构建带字体与原许可；editor-recovery 验 fonts.check，terminal-geometry 验字体加载后的 controller/observer 字符几何和连接复用。真实 Oh My Zsh/Powerline/手机软键盘矩阵不从字体表推定通过。
7. 错误与正确：错误 `terminal.options.fontFamily = "JetBrains Mono Nerd Font"` 但无资源；正确先自托管已校验资源、fonts.load，更新正确别名并触发测量，保留 fallback 与生命周期门禁。


## 工作台微圆角与细框线（2026-10-02）

工作台外观归 `web/src/app/styles.css`；`--workbench-chrome/sidebar/border/tabs` 在浅色与深色定义，面板半径 `--workbench-panel-radius: 8px`、条目半径 `--workbench-item-radius: 4px`、分隔间隔 `--workbench-gap: 4px`。侧栏、各编辑器组和各终端组独立 1px 边框，禁止用阴影堆叠或整页卡片替代工作台结构。文件树 wrapper 左右 5px 内收；Git 行与终端标签保持微圆角。

编辑器选中标签使用正文背景及顶部 6px 圆角，不恢复主题色顶线或 TabsTrigger 下划线。未选中保留 hover，focus 继续使用既有 shadcn 原语。右侧文件动作与可滚动 TabsList 仍同级，不把按钮放进滚动标签容器。侧栏按钮用 `aria-pressed` 表达实际展开且属于当前 sidebar；收起时全部侧栏按钮取消选中，onResize 读取真实面板尺寸同步状态，包含拖动、底部按钮、关闭及恢复。三个入口同一 toggleSidebar，点击当前打开入口收起，重点击展开。终端按钮位于活动栏底部并按实际尺寸同步选中。

底部 `.workbench .terminal-pane` 局部令 `--background: var(--workbench-sidebar)`，让 xterm 的 `colors(host)` 与 viewport 同时继承底部面板背景；上方终端跟随编辑器背景。不能只改父面板 background 而让 xterm 网格留下不同底色，也不能改全局 `--background` 影响登录、项目和 Monaco。

react-resizable-panels 4.14.1 的 Panel 外层有内联 `overflow: visible`，子框边框在尺寸归零时仍可能产生最小盒子。仅在 `.workbench-body [data-panel]` 使用 `overflow: clip !important` 约束局部框边界；不靠 body 裁剪隐藏 document 溢出。默认透明 separator 占 4px，hover/focus/拖动时整个 4px 间隙显示主题色，现有 8px 热区不缩小。收起验收读取真实 `[data-panel]#terminal` 的 0px 高度，而非用仍有最小 border-box 的子 section 的 `toBeVisible()` 推断面板状态。

移动单内容区统一一层 1px/8px 外框、4px 外边距，内层 editor-group/terminal-pane 取消双框。保留 `(max-width: 760px), (hover: none) and (pointer: coarse)`，触屏横屏不能切为桌面 Monaco。

验证见 `web/tests/e2e/workbench-modern-ui.spec.ts`：键盘与鼠标调整、收展、固定动作与滚动标签、深浅主题、触屏 390×844/844×390、document 几何及零额外 mutation；继续复跑 editor-recovery、workbench-interactions 和 terminal-geometry。浏览器模拟不代替 Debian/物理手机验收。


第二轮选中标签去掉 tab-row 顶部留白，使其与框顶衔接。主题 SelectTrigger 使用所选主题的 Monitor/Sun/Moon 图标，保留 aria-label/title、键盘/option 与 system 监听，复用原下拉组件；仅局部隐藏 chevron。项目名 DropdownMenu 负责项目设置/切换项目，顶栏不重复放属性入口；居中搜索调用既有 Dialog/Input/Button，移动端缩短提示并隐藏快捷键标记。

项目页在原功能与确认 Dialog 外使用 workbench chrome/sidebar/border、8px 主框/4px 条目、紧凑标题/文件夹图标与路径，保留打开/新标签页/编辑/配置移除边界。项目数量不改变命令/后台资源的语义。

Dialog 原语的 Tailwind translate 与 transform 是独立属性；顶部搜索弹窗用 `translate: -50% 0; transform: none` 设置水平居中、top:12%，不能再叠加 translateX。真实浏览器检查 dialog 中心与 viewport 中心一致，且窄屏宽度/上下边界容纳。


第三轮截图细化：分隔器 `::before` 保持完整 4px 间隙，半径 2px 形成圆头；8px 热区不改变。Git 局部 TabsList 为 28px、3px 内边距/6px 圆角，Trigger 为 22px/4px 圆角，选中用正文底色，无描边与阴影；focus ring 保留。不修改基础 Tabs 或其他功能标签。总仓库刷新使用已有 RefreshCw/Button icon-sm，aria/title“刷新仓库”。

FileQuickOpen 的 Input 与 DialogClose 组合在 `.file-quick-input-row` 中，关闭按钮 right:4px/top:50%/translateY(-50%)，关闭/输入的中心一致；禁用 DialogContent 默认关闭按钮，避免两个关闭入口叠加。

项目选择页用欢迎页结构：品牌标题、开始使用/已有项目两栏，760px 以下单栏；项目名称、路径、打开位置 Dialog、编辑与移除配置确认保持原语义。没有独立终端导航或 runtime；旧 `/terminals` 和 `/terminals/:id` 在鉴权后 Navigate 到 `/projects`，不发送终端创建/attach/终止请求，项目工作台终端不受影响。


第四轮终端折叠时，ProjectWorkbench 的水平 Separator 保留原位置/ref/Panel布局身份，依据实际 onResize 可见性 disabled、height:0、关闭伪元素热区；再次 expand 后恢复4px与拖动/键盘。不能卸载 terminal runtime 或终止进程来消除间隙。状态栏固定25px、16px文字行高，直属文本块使用16px行高自然高度，由footer的align-items:center居中，不使用满高align-content盒子，语言入口/按钮同栏居中。浏览器断言侧栏与editor底边误差≤1px、文字Range中心误差≤2px，并等待 Separator 的实际展开属性后再验伪元素宽高。

## 登录页与标签衔接（2026-10-03）

LoginPage 的 login-main 使用 workbench-chrome，login-frame 使用正文背景、1px workbench-border 和8px面板半径，桌面说明/登录表单双列，宽度≤760px或粗指针无悬停时单列；触屏横屏沿用单列并允许页面纵向滚动。Field/Input/Button 继续复用，密码清空、错误、429冷却、提交去重及安全返回路径逻辑不变。login-modern-ui 浏览器专项验证深浅主题、390×844/844×390、焦点/Enter、错误/冷却和成功返回项目页；真实软键盘不从模拟推定。

编辑器原生滚动轨道使用 --background，终端仍透明；选中背景与正文衔接，标签行、固定动作与标签内容固定38px，editor-tabs为42px并margin-bottom:-4px，原生4px轨道覆盖路径栏顶部，避免挤压内容或在固定工具栏下方留下缺口。标签明确20px行高、路径30px高度/18px行高；溢出前后文字中心不变。Firefox 的 scrollbar-color 同时提供轨道背景；WebKit 用 ::-webkit-scrollbar-track。不得仅改灰缝而隐藏滚动入口。workbench-modern-ui 与 workbench-interactions 覆盖溢出前后坐标、滚动到两端、hover/focus稳定、轨道与正文同色及固定动作。
