# 主题、字体、图标与分发

主题setting为system/light/dark，首次system；effective根据matchMedia计算。偏好在当前浏览器本地持久化，不存SQLite，不覆盖其他设备。首帧在渲染前应用有效主题；system监听并清理media listener，显式light/dark不被系统事件覆盖。
语义 CSS variables 同时覆盖 shadcn/ui、Monaco theme/DiffEditor、xterm.options.theme、context menu/dialog、Explorer/scrollbar；切主题不重建 editor/terminal 或丢 buffer/history。字体/色彩/选区在三模式实测。

内置 JetBrains Mono Nerd Font NL：资产任务锁定 Nerd Fonts release、Mono/NL variant、真实 font-family metadata、格式和 glyph 覆盖。自托管 font-face，等待 document.fonts.load/ready 后 Monaco remeasure/xterm fit；fallback 为 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace。不能只设置 font-family 名字却没分发字体。测试 Oh My Zsh/Powerline/Nerd glyph、中英文、宽度对齐。转换 WOFF2/子集不能意外移除 glyph；未授权不能重新许可。
Material Icon Theme 用锁定上游资源和映射，优先 exact filename -> longest compound suffix -> extension -> language ID -> fallback，folder 独立；处理 Dockerfile/package.json/go.mod/.gitignore/.env/yaml/yml/ts/tsx/d.ts。必要大小写策略及 basename 特例有测试，不仅 extname。icon SVG 作为可信内置资产分发，用户 SVG 安全策略不同。

导入前逐项检查实际版本 LICENSE/NOTICE：字体原始/补丁/图标许可分别核查；第三方资产清单含 source URL、version/checksum、license、修改/转换记录，分发包含必要 copyright/license/notice。项目 MIT 不能覆盖第三方许可。Material 图标已按下文契约分发；Nerd Font 尚未导入，不声称字体 glyph 渲染已通过。来源入口：[Nerd Fonts JetBrainsMono](https://github.com/ryanoasis/nerd-fonts/tree/master/patched-fonts/JetBrainsMono)、[Material Icon Theme](https://github.com/material-extensions/vscode-material-icon-theme)。

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
