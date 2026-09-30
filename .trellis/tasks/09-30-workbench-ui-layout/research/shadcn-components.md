# shadcn/ui 组件查找记录

日期：2026-09-30。规划阶段仅查询，没有安装、覆盖或修改组件、依赖及锁文件。

## 项目证据

- `web/components.json` 与官方 CLI `info --json` 一致：Vite、TypeScript、Tailwind v4、`base-nova`、Base UI、lucide，npm 锁文件。
- 已有 `context-menu`、`dropdown-menu`、`dialog`、`tooltip`、`button`、`field`、`input`、`toggle`、`alert` 等组件。缺少 `tabs`。
- 最初沙箱网络解析失败；改用已缓存的官方 CLI、获准只读联网后 `info --json` 和 `docs context-menu dropdown-menu tabs dialog tooltip` 成功。未运行 `add`。

## 官方来源与使用结论

| 来源 | 本任务使用 |
| --- | --- |
| [Context Menu](https://ui.shadcn.com/docs/components/base/context-menu) | 标签右键；复用已有 Group/Item/Separator 与 destructive variant |
| [Dropdown Menu](https://ui.shadcn.com/docs/components/base/dropdown-menu) | 多目录新建与手机“更多”；复用已有组件，不新增手写浮层 |
| [Tabs](https://ui.shadcn.com/docs/components/base/tabs) | 官方提供 Tabs/List/Trigger/Content；实施时按 base-nova CLI 添加缺失组件 |
| [Dialog](https://ui.shadcn.com/docs/components/base/dialog) | 名称确认、重命名与统一倒计时；保留 Title/Description 和可访问焦点 |
| [Tooltip](https://ui.shadcn.com/docs/components/base/tooltip) | 标签目录和状态、图标命令说明；触屏仍有可点击替代入口 |
| [Base UI Tabs](https://base-ui.com/react/components/tabs) | 核对受控选择、键盘操作和稳定宿主组合，不使用 Tab 切换卸载真实运行时 |

上述网页已读取。CLI 同时返回对应 Base UI API 地址；`.md` Tabs API 抓取失败后已读取官方 HTML 页面，不据失败自行重写原语。

标签的接管、更多、关闭按钮与 TabsTrigger 组合时不得形成 button 内嵌 button；使用外层业务布局与同级官方按钮，TabsTrigger 始终归 TabsList。若某种原型排布受官方组合限制，先调整业务容器，不改写基础组件替代其键盘/焦点行为。

实现前重新加载 `shadcn` skill 并核对文档；不引入直接 Radix，不切换组件样式，不批量重装已有组件。

## 实施记录

实施阶段已重新加载 skill，在 `web/` 运行官方 `npx shadcn@latest add @shadcn/tabs --yes`，仅新增 `src/components/ui/tabs.tsx`，manifest/锁文件无变化。其余组件均复用项目已有源码，无直接 Radix import。

生成样式采用 `data-horizontal/data-vertical`，但锁定 Base UI 的实际 DOM 为 `data-orientation="horizontal|vertical"`，已仅修正对应 Tailwind 属性选择器；交互原语仍为官方 Tabs。窄屏零宽终端因此修复，真实 Chromium 三主题/竖横屏验证。TabsContent 归各自 root，按钮保持同级，不以内嵌按钮构造标签。
