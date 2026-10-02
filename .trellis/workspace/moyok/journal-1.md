# 工作日志 — moyok（第 1 部分）

> AI 开发会话日志
> 开始日期：2026-09-26

---



## Session 1: 完成 W02 Debian 隔离实验
<!-- trellis-session: v=2 fp=ea881cc99d663b01 -->

**Date**: 2026-09-28
**Task**: 完成 W02 Debian 隔离实验
**Branch**: `main`

### Summary

按单代理流程完成 D09 Landlock CLI 和 D10 非特权 helper 可行性实验，验证后提交并归档 W02；W04/W07 产品门禁保留。

### Git Commits

| Hash | Message |
|------|---------|
| `d7e55eb` | 约束项目开发为单代理执行 |
| `cb9729c` | 完成 W02 CLI 隔离与提权可行性实验 |

### Status

[OK] **Completed**


## Session 2: 完成 W04 多文件夹项目与文件管理
<!-- trellis-session: v=2 fp=bcb411d46e54ddb8 -->

**Date**: 2026-09-29
**Task**: 完成 W04 多文件夹项目与文件管理
**Branch**: `main`

### Summary

完成多根项目、目录选择、安全文件操作、上传下载与 ZIP、文件事件和 shadcn 工作台；Go/前端/真实 Debian 验收通过，已归档 W04。

### Git Commits

| Hash | Message |
|------|---------|
| `22d4fb8` | feat: 实现 W04 多根项目与安全文件传输 |
| `985c0f2` | feat: 完成 W04 项目工作台与 shadcn 界面 |
| `3b0fa55` | docs: 同步 W04 契约与验收记录 |

### Status

[OK] **Completed**


## Session 3: 完成 W03 持久终端与多端控制
<!-- trellis-session: v=2 fp=7eb9b0a3d01856f8 -->

**Date**: 2026-09-30
**Task**: 完成 W03 持久终端与多端控制
**Branch**: `main`

### Summary

W03 T01..T07 完成：独立 tmux/PTY/WS、稳定 xterm runtime、多端单控制权、全端可取消倒计时、手机快捷键与安全链接。真实 Debian 恢复/产品探针 40 项及完整 9 条浏览器 E2E 通过，Go test/vet/race、前端 lint/typecheck/49 单测/build 通过；连接信息仅来自忽略的 .env，隔离资源精确清理。用户批准三批提交及归档，不推送。W05 编辑与完整布局、W08 正式部署仍未完成。

### Git Commits

| Hash | Message |
|------|---------|
| `0a07afd` | feat: 实现 W03 持久终端后端与 Debian 验收探针 |
| `b48ab4d` | feat: 接入 xterm 工作台与手机终端控制 |
| `6e07bb6` | docs: 固化 W03 终端契约与完整验收记录 |

### Status

[OK] **Completed**


## Session 4: 归档七个已完成任务
<!-- trellis-session: v=2 fp=89bea259b21ca7a3 -->

**Date**: 2026-09-30
**Task**: 归档七个已完成任务
**Branch**: `main`

### Summary

按用户明确要求完成并归档七个任务，修复归档后的文档与 JSONL 引用。保留 W05～W08 未完成及真实手机软键盘等未验收边界。

### Main Changes

- 七个指定任务状态更新为 completed，归档至 .trellis/tasks/archive/2026-09；活动任务归零。
- 补充归档范围、实际代码提交及历史状态说明，修复当前和既有归档链接。

### Git Commits

| Hash | Message |
|------|---------|
| `52e7097728bd1619d9e0a4ac98b753916f299d58` | docs: 修复任务归档引用并记录完成边界 |

### Testing

- [OK] 370 个本地 Markdown 链接、84 个 JSONL 目标、11 个已归档 completed 任务及 0 个活动任务校验通过；git diff --check 通过。
- [OK] 本轮仅修改任务元数据与文档，未重跑 Go/前端或真机产品测试；inline 任务不存在的 JSONL 按 CLI 跳过。

### Status

[OK] **Completed**

### Next Steps

- W05 编辑/自动保存/草稿/完整布局恢复尚未建任务；W06～W08 未完成，手机软键盘等真实设备检查继续保留为后续验收事项。


## Session 5: W06 搜索替换与只读 Git 功能交付，实机验收延期
<!-- trellis-session: v=2 fp=4e4a0ffb03ab9946 -->

**Date**: 2026-10-01
**Task**: W06 搜索替换与只读 Git 功能交付，实机验收延期
**Branch**: `main`

### Summary

单代理 inline 完成 W06 功能及本地开发门禁；任务保持进行中等待用户统一验收，未提交、推送或归档。

### Main Changes

- 安全目录快照/私有 CLI staging、多根搜索与版本保护替换、目标编辑缓冲区保护。
- 多仓库只读 Git、固定 HEAD 历史分页、改名比较、所属仓库 HEAD 行标记和工作台入口。
- 后端/前端 owner、README/AGENTS、任务交付报告与延期验收队列同步。

### Git Commits

(No commits - planning session)

### Testing

- [OK] Go test/vet/race 全包通过，Linux amd64 编译通过；不代表 Debian 运行验收。
- [OK] 前端 lint/typecheck/build 通过，23 文件134单测；10项 Chromium 本地回归通过，收尾另复跑W06两项。
- [OK] 18份 Markdown 的126个本地链接、JSON、Go格式、忽略规则及 diff --check 通过；inline JSONL 校验跳过不计产品验证。

### Status

[OK] **Completed**

### Next Steps

- W05/W06 实机验收等待用户明确安排，后续功能包另建任务；不自动部署或归档。


## Session 6: W05/W06 恢复统一验收与两项修正
<!-- trellis-session: v=2 fp=ba041155e3df27b8 -->

**Date**: 2026-10-01
**Task**: W05/W06 恢复统一验收与两项修正
**Branch**: `main`

### Summary

本机门禁及 Chromium/WebKit 回归通过；Debian 传输待明确目标与载荷授权，真实手机仍待设备条件。

### Main Changes

- 修复触屏横屏切换桌面模式与替换预览默认取消焦点，并加入实际回归、六种合成图片和隔离 Debian 包 runner。

### Git Commits

(No commits - planning session)

### Testing

- [OK] Go test/vet/race、前端 lint/typecheck/134 单测/build 通过；Chromium 综合22项及收尾4项、WebKit收尾4项通过；Python安全回归21项通过。

### Status

[OK] **Completed**

### Next Steps

- 获得 .env 指定 Debian 目标及测试产物传输授权后继续真实专项；协调手机、触控板及 shell 主题操作，排除 Firefox 启动环境阻塞。


## Session 7: W05/W06 五张截图反馈修正
<!-- trellis-session: v=2 fp=bef5a72ab50531fd -->

**Date**: 2026-10-01
**Task**: W05/W06 五张截图反馈修正
**Branch**: `main`

### Summary

完成目录溢出、比较全屏/行内、默认账户 shell、无仓库禁用及无提交 Git 快照；本机回归通过。

### Main Changes

- 复用 shadcn 和同一 Monaco 实例；Git 空 metadata 目录保留，shell 按服务 UID 检测并仅影响新 pane。

### Git Commits

(No commits - planning session)

### Testing

- [OK] Go test/vet/race、前端134项/lint/typecheck/build、Chromium10项及最终Chromium/WebKit各4项通过；Linux专属测试仅编译。

### Status

[OK] **Completed**

### Next Steps

- 重启后端后新建终端应用默认 shell；Debian目标授权、真实手机与触控板验收待补齐，任务不归档。


## Session 8: W06 Git 延迟与轮询修复
<!-- trellis-session: v=2 fp=08bcbdf1daf38d1c -->

**Date**: 2026-10-01
**Task**: W06 Git 延迟与轮询修复
**Branch**: `main`

### Summary

按语义收窄安全快照，优化Darwin批量句柄，移除Git定时轮询与共享串行基线。Halo本机只读复测与Chromium/WebKit回归通过。

### Main Changes

- Git基线与历史/详情/比较不再复制无关工作树；已发现仓库位置仍每次校验身份与版本
- 基线请求共享/串行、焦点冷却、失败不自动轮询；刷新发现世代去重

### Git Commits

(No commits - planning session)

### Testing

- [OK] go test/vet 全量通过；全量race与最终files/gitview race通过
- [OK] 前端 lint/typecheck、137单测、build；Chromium11与WebKit5通过

### Status

[OK] **Completed**

### Next Steps

- W05/W06真实Debian与手机/触控板等验收仍待完成，详见任务验收报告


## Session 9: Git 超时预算与双区域提交图调整
<!-- trellis-session: v=2 fp=c65a8b89dbcae28c -->

**Date**: 2026-10-02
**Task**: Git 超时预算与双区域提交图调整
**Branch**: `main`

### Summary

修正 HTTP/工具预算和忽略树快照，接入可拖动可折叠的变更/历史面板及 Material 文件树；重启已管理的 5173 本机实例使修复生效。

### Main Changes

- Git 原生 ignore 剪枝保留 tracked 文件，所有操作使用同一预算 context；历史按拓扑顺序及真实父关系绘图。
- 官方 Collapsible/Resizable/ToggleGroup 组合默认平分双区域，提交详情内联展开，保留父选择/本地引用/只读比较。

### Git Commits

(No commits - planning session)

### Testing

- [OK] Go 全包 test/vet/race 通过；最终受影响三包再验 test/race 与完整 vet。前端 lint/typecheck/build，26 文件 142 单测通过。
- [OK] Chromium 13 个不同用例、WebKit 7 项通过；Halo 只读 status 约 2.43 秒，源 HEAD/index/config 哈希不变。21 本地链接/任务 JSON/diff 检查通过；Linux Git 测试二进制编译通过。

### Status

[OK] **Completed**

### Next Steps

- W05/W06 保持进行中；真实 Debian 目标授权、真实手机输入法、触控板、shell 主题和 Firefox 仍按原验收队列记录。


## Session 10: Git 历史触底加载与右键父提交选择
<!-- trellis-session: v=2 fp=129dfd243767233e -->

**Date**: 2026-10-02
**Task**: Git 历史触底加载与右键父提交选择
**Branch**: `main`

### Summary

移除手动分页按钮并实现触底加载；父提交选择移到右键菜单，文件范围增加3px缩进和左边框。

### Main Changes

- 复用已有 ContextMenu/RadioGroup，选父关闭菜单且直接展开指定提交；保留默认第一父、只读比较与冻结 HEAD。

### Git Commits

(No commits - planning session)

### Testing

- [OK] 前端 lint/typecheck/test/build 通过，27文件146项单测；Chromium/WebKit各3项真实后端回归通过，105提交三页、慢请求去重、末页停止和真实merge父切换。
- [OK] git diff --check、20个本地Markdown链接、task JSON通过；隔离8089/5179实例及自身测试根已清理。

### Status

[OK] **Completed**

### Next Steps

- 继续W05/W06统一验收原待办，真实手机/触控板/Firefox与Debian目标授权仍待补齐；不提交、推送或归档。
