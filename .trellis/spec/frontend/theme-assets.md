# 主题、字体、图标与分发

主题setting为system/light/dark，首次system；effective根据matchMedia计算。偏好在当前浏览器本地持久化，不存SQLite，不覆盖其他设备。首帧在渲染前应用有效主题；system监听并清理media listener，显式light/dark不被系统事件覆盖。
语义 CSS variables 同时覆盖 shadcn/ui、Monaco theme/DiffEditor、xterm.options.theme、context menu/dialog、Explorer/scrollbar；切主题不重建 editor/terminal 或丢 buffer/history。字体/色彩/选区在三模式实测。

内置 JetBrains Mono Nerd Font NL：资产任务锁定 Nerd Fonts release、Mono/NL variant、真实 font-family metadata、格式和 glyph 覆盖。自托管 font-face，等待 document.fonts.load/ready 后 Monaco remeasure/xterm fit；fallback 为 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace。不能只设置 font-family 名字却没分发字体。测试 Oh My Zsh/Powerline/Nerd glyph、中英文、宽度对齐。转换 WOFF2/子集不能意外移除 glyph；未授权不能重新许可。
Material Icon Theme 用锁定上游资源和映射，优先 exact filename -> longest compound suffix -> extension -> fallback，folder 独立；处理 Dockerfile/package.json/go.mod/.gitignore/.env/yaml/yml/ts/tsx/d.ts。必要大小写策略及 basename 特例有测试，不仅 extname。icon SVG 作为可信内置资产分发，用户 SVG 安全策略不同。

导入前逐项检查实际版本 LICENSE/NOTICE：字体原始/补丁/图标许可分别核查；第三方资产清单含 source URL、version/checksum、license、修改/转换记录，分发包含必要 copyright/license/notice。项目 MIT 不能覆盖第三方许可。此任务未导入资产，不声称许可证检查或 glyph 渲染已经通过。来源入口：[Nerd Fonts JetBrainsMono](https://github.com/ryanoasis/nerd-fonts/tree/master/patched-fonts/JetBrainsMono)、[Material Icon Theme](https://github.com/material-extensions/vscode-material-icon-theme)。

测试主题持久/API失败/system event、Monaco/xterm 不丢状态；图标 exact/compound/fallback/目录和 Unicode；生产 build 资产加载路径/许可证随包/字体实际 glyph 截图证据。
