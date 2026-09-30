# 语言与图标来源核对

日期：2026-09-30。主会话只读研究；未安装依赖、导入产品资产或运行产品验收。

## 仓库证据

- `web/package.json` 固定 `monaco-editor@0.57.0`、`@monaco-editor/react@4.7.0`、npm 11.17.0。
- `web/src/features/workspaces/DesktopEditor.tsx:3` 已导入完整 Monaco；安装包 `esm/vs/editor/editor.main.js` 导入全部基础语言注册与 JSON 等特性。`:33-37` 仅保留少量手写后缀映射，导致其他语言被设为 plaintext。
- 安装包 `esm/vs/languages/definitions/*/register.js` 静态扫描得到 89 个基础语言 ID，另有 JSON 和 plaintext，共 91 个，覆盖用户给出的完整清单。
- 其中六个 FreeMarker 变体、mysql、pgsql、redshift 没有独立 extensions；不能仅靠后缀推断覆盖这些模式。默认 `.ftl/.ftlh/.ftlx` 对应 freemarker2，`.sql` 对应 sql。
- YAML 注册同时包含 `.yaml/.yml`；Python/JavaScript 有 firstLine shebang 元数据；Dockerfile、INI 等有 filenames。
- `Explorer.tsx:130`、`ProjectWorkbench.tsx:150` 使用 lucide 通用图标；目录树根与目录选择器也使用通用 Folder。
- 已有 `web/src/components/ui/select.tsx`，可复用语言模式入口；已查阅 [shadcn Base UI Select 官方文档](https://ui.shadcn.com/docs/components/base/select)，无需自造基础控件。

## 锁定的图标来源

- npm 发布包：[material-icon-theme 5.38.1](https://registry.npmjs.org/material-icon-theme/5.38.1)。下载仅在 `/private/tmp`，实际摘要与 registry integrity 一致。
- 上游仓库提交：`448ab3977ef83b817c2c722ce7cd5034d195b39f`（registry gitHead）。
- 发布包：[material-icon-theme-5.38.1.tgz](https://registry.npmjs.org/material-icon-theme/-/material-icon-theme-5.38.1.tgz)。
- SHA-512 integrity：`sha512-14cFM4NJGdbuo68rIZTq9TSX0f5BtA6VF+eNX3zq23Z5NoEeVtM4Zn4PpZ0ERl++50jCBrywKFWXmOjbxf0xTA==`。
- SHA-256：`d4342dc13a24bd40c4f417337dc19d2d2c42e47f8bb42ca677109bce769f078e`。
- 实际包根 `LICENSE` 为 MIT，copyright 为 `Copyright (c) 2025 Material Extensions`；发布包未发现另外的 LICENSE/NOTICE（`icons/license.svg` 是图标）。实施时保留完整原文，并复核复制资产中有无额外说明。
- 发布 API `generateManifest(config?)` 返回文件名、后缀、语言 ID、目录/根目录开闭、light 与 highContrast 映射及 iconDefinitions；浏览器无需运行上游扩展代码。
- 主会话运行临时发布模块的 `generateManifest()`：默认配置有 2,135 个 fileNames、1,377 个 fileExtensions、200 个 languageIds、1,251 个 iconDefinitions。1,251 个定义的 SVG 均存在于该发布包。
- 默认 activeIconPack 为 angular，folder theme 为 specific；沿用上游默认，不按 Persistty 使用 React 就擅自更改图标包。
- 例子：yaml/yml/yaml.dist/yml.dist → yaml；d.ts → typescript-def；ts → typescript；tsx → react_ts；package.json → nodejs；go.mod → go-mod；Dockerfile、docker-compose.yml/yaml → docker；.gitignore → git。`.env` 通过 `env` 后缀匹配，而非普通 exact filename。
- [锁定提交的 fileIcons](https://github.com/material-extensions/vscode-material-icon-theme/blob/448ab3977ef83b817c2c722ce7cd5034d195b39f/src/core/icons/fileIcons.ts)、[folderIcons](https://github.com/material-extensions/vscode-material-icon-theme/blob/448ab3977ef83b817c2c722ce7cd5034d195b39f/src/core/icons/folderIcons.ts)、[languageIcons](https://github.com/material-extensions/vscode-material-icon-theme/blob/448ab3977ef83b817c2c722ce7cd5034d195b39f/src/core/icons/languageIcons.ts)。patterns 已在上游生成阶段展开，默认 clones 对应 SVG 在包中；无需手工重建规则。

## 限制

这里只证明元数据、发布包资源及许可证存在。真实浏览器着色、worker、浅色对比、离线/冷缓存、键盘交互及所有语言 tokenizer 的加载仍待实施验证。依赖升级后须重新核对元数据路径、ID 集合和生成 API，禁止静默沿用旧快照。
