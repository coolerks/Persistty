# W05 截图反馈调整与检查报告

日期：2026-10-01。当前主会话按 inline 单代理执行；本轮代码尚未提交、推送或归档。原 W05 验收保留在 [既有报告](check-report.md)，本文只记录新增截图反馈。

## 调整结果

1. breadcrumb 不再显示常驻“已保存”；未保存文件在标签关闭位置显示灰色 8px 圆点，鼠标悬停或键盘聚焦可关闭，沿用原草稿保护。冲突/失败/暂停仍可见，正常状态保留唯一的无障碍播报。
2. 保存、刷新、下载移到每组标签栏右侧固定区；标签可独立水平滚动，动作复用现有 buffer 与鉴权下载。
3. 语言入口移到底部单一状态栏，显示当前活动文件实际语言，无边框与常驻箭头，不加“自动：”。菜单保留自动重置与全部 91 种模式，手动选择保持既有 Monaco model，分组共享选择，不写文件。
4. 空历史不再隐藏实时终端：无 scrollback 的 LF、零字节、仅空白均保留 live 并提示“暂无历史输出”。提示不拦截首行点击，控制端明确输入后消失；长/短历史维持既有只读 viewport 与底部返回机制，不新增 WS、接管、会话或输入。

根因：tmux 没有 scrollback 时 capture-pane 仍可能返回 LF；零长度 Uint8Array 也为 truthy。旧显示分支把这些响应当有效历史，隐藏 live 后只留下空白。先用两种空历史用例复现失败，再修正 TerminalSession 的响应边界。真实 Debian 发现新增提示遮挡首行点击，已修复 pointer-events 与输入清除，并通过真实键盘输入验证。

## 检查记录

| 检查 | 实际结果 |
| --- | --- |
| `go test ./...`、`go vet ./...`、`go test -race ./...` | 本轮全部通过；隔离 GOCACHE，后端源码未改动 |
| `npm run lint`、`npm run typecheck` | 最终修正后通过 |
| `npm run test -- --maxWorkers=2` | 21 文件、128 项通过；使用锁定脚本，仅限制并发 |
| `npm run build` | 通过；字体 SHA-256/字形、37 项第三方许可通过 |
| 最终 Chromium 专项 | 18 项全通过，1.1 分钟 |
| 真实 Debian 专项 | 2 项通过，24.5 秒：w05-feedback-live 与 w05-live |
| 文档本地链接、忽略、`git diff --check` | 通过；最终报告更新后再复核 |

首次默认并发单测出现未改动组件的超时，未增加超时或删减断言，限制两 worker 后 128 项全通过。浏览器初轮发现标签栏 flex 被重复 flex:none 覆盖、冲突播报重复，均已原地修正；回归使用真实可见元素点击。jsdom canvas 提示与既有构建大 chunk 提示保留，不以 jsdom 证明真实渲染。构建写 public 会触发开发页重载，最终浏览器与构建顺序运行且冻结源码。

可视检查截图：`/private/tmp/persistty-w05-feedback-dirty.png`（Java/灰点/底部入口），`/private/tmp/persistty-w05-feedback-desktop.png`（分组/固定操作/单语言入口）。

本地 Chromium 覆盖 editor-assets 2、editor-recovery 6、terminal-scrolling 6、terminal-geometry 1、terminal-device-attributes 2、workbench-interactions 1。验证灰点/保存消失、底部 SelectValue/单入口、手动语言与原 model、右侧按钮不随标签滚动；覆盖真实 IndexedDB/409/手机草稿保护；空历史三种响应、短历史重复切换、controller/observer、惯性小位移、底部返回、鼠标模式、单 WS/零滚动输入。

## 真实 Debian 与清理

沿用批准的私有 browser harness 和 Linux runtime-probe，不访问截图中用户项目。终端测试仅创建专属项目的新会话，空 history_size=0 时保留 live；通过明确键盘操作输出 120 行，再按浏览器本次实际 HTTP 快照验证三轮上下滚动和返回实时。二进制输入计数取滚动前基线，区分连接初始化自动应答，滚动前后保持不变；本次目标终端只建立一个 WS，pageerror 为空。

真实 w05-live 复验 BOM/混合换行原子落盘、外部冲突/只读 diff/明确保存、三组恢复、PNG/SVG 与匿名拒绝。第一次专属实例达到 RuntimeMaxSec=900 后出现代理断开，按 harness 原规则清理，再用新隔离实例完成最终两项，不改变自动期限。

去敏记录：`/private/tmp/persistty-w05-feedback-debian-start-2.json`、`/private/tmp/persistty-w05-feedback-debian-cleanup-1.json`、`/private/tmp/persistty-w05-feedback-debian-cleanup-2.json`。两个实例清理均确认 units_inactive、pane_processes_gone、root_removed 为 true；专属隧道和 Debian 前端进程已停止。连接凭据没有进入报告/代码/日志输出。

## 复用与规范

复用项目现有 shadcn Button/buttonVariants、Tabs、Select；新增 EditorFileActions/EditorLanguageStatus 为业务薄封装，无依赖或基础 UI 替换。实现前核对 [Select 官方文档](https://ui.shadcn.com/docs/components/base/select)、[Tabs 官方文档](https://ui.shadcn.com/docs/components/base/tabs)、[Button 官方文档](https://ui.shadcn.com/docs/components/base/button)。灰点属于专用标签状态，用语义色与 aria 文本，不另造基础控件。

已同步 frontend/component-guidelines、editor-terminal-lifecycle 与 terminal-runtime-contract；文件保存/认证/PTY 及外部进程契约不变。inline 模式的 JSONL 不存在并由 task.py validate 跳过，skipped 不等于产品验收。

## 验收边界

本轮通过 Chromium 合成滚轮及真实 Debian 后端；真实触控板手感、手机软键盘/输入法与完整图片/平台矩阵仍未验收。W05 保持 in_progress，不以截图批次通过宣称整个任务完成，也不自动部署、提交或归档。
