# 下方终端分组合并交付

## 问题与实现

用户部署截图中的四个空下方终端组只有拆分和整体收起入口，无法取消单个分隔。新增每组“合并此终端分组”，复用现有shadcn Button与lucide PanelRightClose图标；已查阅[官方Base UI Button文档](https://ui.shadcn.com/docs/components/base/button)，无新基础组件或依赖。

workspace-view.mergeTerminalGroup将源标签合并到左邻（首组并到右邻），压缩后续索引和lowerActive，保留目的组活动项、空目的组采用源活动项。保留terminalOrder、上方/手机视图与所有终端ID。最后一组不移除，合并按钮消失；X继续收起整个区域。无关闭/接管/创建请求，固定runtime跨宿主保留。状态和生命周期规范已同步。

## 实际验证

- 前端lint/typecheck/test/build通过，32个测试文件、171项测试；新增3项store回归覆盖空组/持久化/再拆分及首中末索引/活动项/顺序/上下与手机隔离。
- 普通私有HTTP源读取生产前端产物，Chromium/WebKit各3项通过：空四组逐个合并/刷新/再拆分/原X整体收起；运行中两条会话合并保留DOM、各一条WS连接，除合法resize外无命令/输入/HTTP mutation；原HTTP UUID初始化/Monaco/重复搜索定位保持。
- 首轮Chromium的空组流程在最后“收起”断言失败：裁剪后的子元素仍保留自身尺寸，Playwright子元素visible不代表外层面板展开。按既有折叠契约改为检验父面板height=0，复验通过；产品代码未为测试调整。失败产物保留在忽略的.cache。
- Go `test -count=1 ./...` / `vet ./...`再次通过，短GOTMPDIR；不启用已取消跟进的dev启动集成，不记原Actions问题解决。
- 实际新包SHA、安全解压/清单、amd64 ELF64静态链接、全部前端文件与浏览器所读产物逐字节一致、原systemd/Nginx/后端/tmux模板、根README及旁置说明检查通过。前端既有canvas/chunk提示保留，命令退出0。

## 包与部署

交付目录 `dist/debian-amd64-terminal-merge-20261003/`（本地忽略产物）；版本 `manual-20261003-terminal-merge`。tar SHA256：`bdafa94a8518ae9af538f1684dd28f5aced2c05a22d1b212ea9246e1ff78e597`。

源码基线HEAD `06355b2`，包含已提交的UUID修复；本轮合并与手工文档变更来自未提交工作树，不声称已包含在该commit。旧包保留，新包前后端和完整配置齐全，使用[部署说明](../../../deploy/MANUAL.md)第7节升级，保留实际配置/数据/独立tmux，浏览器强制刷新。

本轮API/WS为隔离fixture，覆盖生产前端和浏览器连接生命周期，不是Debian真实PTY/Nginx升级验收。目标机本轮未操作，任务保持in_progress，不自动提交、发布或归档。
