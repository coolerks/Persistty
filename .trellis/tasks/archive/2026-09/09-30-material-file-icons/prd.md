# Material 文件与目录图标

> 归档状态：2026-09-30 用户明确要求标记完成并归档；本任务已完成收尾。高亮/图标及相关修复已提交 6cf29aa，终端几何修复已提交 b7ac848；既有验收限制继续保留。

## 目标与背景

用户指定 VS Code Material Icon Theme 的图标与映射，要求 yaml/yml 等别名正确。当前 Explorer 与文件标签均为通用 lucide 图标。研究见[发布包核对](../09-30-editor-language-material-icons/research/upstream-assets.md)，先依赖[语言子任务](../09-30-monaco-language-coverage/prd.md)提供轻量路径推断。

## 需求

- I01：采用锁定 Material Icon Theme 5.38.1 的默认配置及完整生成映射，自托管必要 SVG；默认目录 specific 与 icon pack 保持上游默认。
- I02：文件 exact basename 优先，再最长复合后缀/扩展名，再语言 ID，最后通用 file。小写匹配大小写变体，保留显示原名；隐藏文件/Unicode/未知名可靠回退，不按图标存在文件名猜别名。
- I03：资源管理器文件/目录/项目根与编辑器文件标签使用一致解析，目录开闭状态独立，深浅主题使用上游 light 映射覆盖。目录选择器中的真实目录条目复用该图标 owner；工具栏的功能动作图标继续使用 lucide。
- I04：上游生成阶段展开的 patterns、默认 clones 和文件名映射全部保留；不手写小型白名单，不因资源很多裁掉别名。
- I05：分发保留版本、源提交、下载校验、各资产校验和 MIT LICENSE，构建产物带许可证；开发/生产均不依赖 CDN 或第三方运行时。

## 验收标准

- AC-I01：yaml/yml 及 yaml.dist/yml.dist 相同；d.ts 与 ts 分别使用上游 typescript-def/typescript；tsx 为 react_ts；package.json/go.mod/Dockerfile/.gitignore/.env/docker-compose、大小写、Unicode、未知后缀及有冲突的最长匹配正确。
- AC-I02：完整生成 manifest 的每个关联能解析到内置存在的 SVG，浅色覆盖、默认 clone 与目录展开资源无断链；不以少量示例代替全量资源引用校验。
- AC-I03：同路径在树与标签图标一致；src/.git/node_modules 与未知目录的开闭、项目根和目录选择器按上游映射显示，功能按钮仍有原中文可访问名称。
- AC-I04：深浅/system 切换、窄屏、长名称、左右标签组不溢出；主题切换不重建 Monaco 或终端；图标为装饰性 img，不遮挡键盘/鼠标交互。
- AC-I05：冷缓存加载及构建产物只请求本地资源，图片加载失败有有界 fallback；全部 asset checksum 与许可分发核对通过，前端四门禁及实际浏览器验收通过。

## 范围外

用户自定义 icon packs/颜色/克隆与图标设置页、工作区 SVG 预览、字体、VS Code 扩展 API和所有语言都强制拥有专用图标。上游无映射时按语言或通用图标回退。

## 实施状态

PRD/设计/实施清单已备齐；用户已于 2026-09-30 明确批准开始执行；资源已导入，验证结果见[父任务验收报告](../09-30-editor-language-material-icons/check-report.md)。
