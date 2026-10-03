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


## Session 11: W06 提交与文件悬浮卡片
<!-- trellis-session: v=2 fp=9ca2d27508dbb84a -->

**Date**: 2026-10-02
**Task**: W06 提交与文件悬浮卡片
**Branch**: `main`

### Summary

完成提交完整消息、作者日期ID、真实文件/增删行数及GitHub链接卡片；历史文件列表和树共用卡片，详情有界缓存与展开复用。

### Main Changes

- 新增详情DTO与私有快照raw/numstat解析，保留重命名原路径及源仓库零写入；仅生成无凭证GitHub公共链接。
- 复用官方shadcn HoverCard，450ms按需加载、8项/2MiB缓存、取消和失败无自动重试；同步PRD与前后端owner。

### Git Commits

(No commits - planning session)

### Testing

- [OK] Go全包test/vet/race通过；前端lint/typecheck/build、最终28文件152项单测通过。
- [OK] Chromium/WebKit各5个不同真实后端用例通过，含完整消息、真实统计、外链本地拦截跳转、缓存复用、合并/分页/无轮询；源HEAD/index/config不变。
- [OK] 21个本地Markdown链接、JSON、忽略规则和diff检查通过；官方本机开发实例重启就绪，8089/5179与专属测试根清理。

### Status

[OK] **Completed**

### Next Steps

- W05/W06仍进行中，原Debian授权、真实手机和设备验收待办保留；不提交、推送或归档。


## Session 12: W07 单文件提权实施与本机验证
<!-- trellis-session: v=2 fp=491ae30c634a7e84 -->

**Date**: 2026-10-02
**Task**: W07 单文件提权实施与本机验证
**Branch**: `main`

### Summary

已完成 W07 源码、API、同缓冲区确认流程与安装审查产物；本机门禁通过。真实 Debian 特权验收待精确目标与独立授权，任务保持 in_progress，W05/W06 统一验收后置。

### Main Changes

- 新增持久一次请求、独立非 root broker/固定 root helper、Linux 元数据阶段提交、默认禁用配置与部署示例；同步 W07 owner 规范、PRD/实施/检查报告。

### Git Commits

(No commits - planning session)

### Testing

- [OK] go test ./...、go vet ./...、go test -race ./...；前端 lint/typecheck/test/build，30文件159单测；最终 Chromium7+WebKit7专项与编辑恢复6项通过。
- [OK] Linux amd64/CGO=0 helper、broker、TTY探针与 files.test 构建通过；记录 SHA256，未执行 Linux/root/远端安装。

### Status

[OK] **Completed**

### Next Steps

- 取得 Debian 地址、现有 UID/GID、隔离文件及安装差异授权，补 sudo/PAM、原生文件/账本/故障、systemd NNP/cgroup、Nginx 零正文落盘；不能提前完成或归档。


## Session 13: 截图修复：标签衔接、终端全屏与登录页
<!-- trellis-session: v=2 fp=c42fe14c99f0f675 -->

**Date**: 2026-10-03
**Task**: 截图修复：标签衔接、终端全屏与登录页
**Branch**: `main`

### Summary

已建立10-03-workbench-screenshot-fixes任务并完成四项实现及本机验收，保持in_progress供复核；未提交或部署。

### Main Changes

- 修复标签原生轨道底缝和文字跳动，删除tracked HEAD，添加终端全屏/拖动吸附及常规比例恢复，统一登录页深浅主题与触屏布局。
- 同步前端owner规范、PRD/design/implement与check-report；保持终端与编辑器实例及既有认证边界。

### Git Commits

(No commits - planning session)

### Testing

- [OK] 前端lint/typecheck/test/build通过；31文件163单测；Go test ./...与go vet ./...通过。
- [OK] Chromium18+WebKit18适用回归，真实后端Git每浏览器1项，共38项最终通过；初轮失败与复验保留报告。
- [OK] 8个本地Markdown链接、JSON、忽略与diff检查通过；5203/5204/8103自有服务和临时Git/SQLite/tmux已清理。

### Status

[OK] **Completed**

### Next Steps

- 用户复核后提交本轮修改；Firefox、Debian/真机软键盘不记通过，原W05/W06/W07/UI任务状态保留。


## Session 14: 截图追加反馈：连续拖动、双向全屏与终端绘制
<!-- trellis-session: v=2 fp=aae047e8ce2985fe -->

**Date**: 2026-10-03
**Task**: 截图追加反馈：连续拖动、双向全屏与终端绘制
**Branch**: `main`

### Summary

取消160px停点，连续拖至38px标签行再吸附；全屏顶部可下拖；修正文字及滚动工具栏衔接；历史已绘制后切换，实时保持尺寸。

### Main Changes

- 更新当前截图修复任务及owner规范，保持单代理、同runtime/model及单WS，无API/依赖/权限变化。

### Git Commits

(No commits - planning session)

### Testing

- [OK] Chromium/WebKit各27项最终通过；旧源码逐帧空白10次，修复后两浏览器均0；前端163单测、lint/typecheck/build与Go test/vet通过。
- [OK] 最新拖动回归真实鼠标按住连续经过140/90/48/38px再拖24px收起；顶部下拖、布局恢复和刷新通过。

### Status

[OK] **Completed**

### Next Steps

- 用户复核并按需提交；物理触控板、手机软键盘、Firefox和Debian/systemd专项仍未执行，不自动部署或归档。


## Session 15: W08 发布包与一键部署本机实施
<!-- trellis-session: v=2 fp=1d80cda7a29c9c32 -->

**Date**: 2026-10-03
**Task**: W08 发布包与一键部署本机实施
**Branch**: `main`

### Summary

main 自动打包/Release 工作流、标准库下载安装升级器、VPN/TLS Nginx/systemd/后端配置及完整中文运维文档完成；27项部署回归、前端门禁、两架构验包、vet/bridgego通过。Go无缓存全量有既有Mac shell清理竞态，真实Actions和Debian验收未执行，任务保持进行中。

### Main Changes

- 新增GitHub发布、amd64/arm64同提交包、SHA/许可/迁移清单
- 部署器保留配置与tmux，SQLite一致备份，兼容代码恢复、不自动回滚数据

### Git Commits

(No commits - planning session)

### Testing

- [OK] 27部署回归；前端163测试及lint/typecheck/build；静态workflow/两架构包验证通过
- [OK] root vet和bridgego29 pass通过；无缓存Go全量shell fixture清理失败已保留

### Status

[OK] **Completed**

### Next Steps

- 审查既有shell fixture清理竞态并重跑Go全量；推送后的Actions与授权Debian正式安装验收


## Session 16: W08 简化为固定局域网 HTTP 部署
<!-- trellis-session: v=2 fp=0477b55e8c65de26 -->

**Date**: 2026-10-03
**Task**: W08 简化为固定局域网 HTTP 部署
**Branch**: `main`

### Summary

按用户最新环境重写为LAN_IP，沿用已有Nginx/Python；无需apt、SSL、WireGuard。

### Main Changes

- 缩短部署文档，现有Nginx一行include，保存实际Nginx/Python路径，新增后端lan_http；以后单命令下载最新Release更新。

### Git Commits

(No commits - planning session)

### Testing

- [OK] 30项部署回归、Go配置专项及普通全量test/vet、前端lint/typecheck/163单测/build、两架构实际打包验包通过；初轮无缓存Mac shell fixture失败保留。

### Status

[OK] **Completed**

### Next Steps

- 尚未提交、推送或执行目标机/GitHub实测；任务继续in_progress。


## Session 17: W08 开源部署地址参数化
<!-- trellis-session: v=2 fp=765ab4fafde4cefa -->

**Date**: 2026-10-03
**Task**: W08 开源部署地址参数化
**Branch**: `main`

### Summary

去除源码、模板、测试、文档与开发记录中的个人实际IP，首次--host指定，后续读取目标机配置一键更新。

### Main Changes

- 模板使用LAN_IP；部署器校验私有IPv4，从同一origin生成Nginx及后端配置；受版本管理记录去敏且保留历史失败结论。

### Git Commits

(No commits - planning session)

### Testing

- [OK] 32项部署回归、Go test/vet、语法/Actions、两架构打包验包及源码/包内个人IP扫描通过；前端未改，本轮build通过。

### Status

[OK] **Completed**

### Next Steps

- 未提交、推送、发布或目标机实测；任务保持in_progress，历史无缓存shell清理竞态未修复。


## Session 18: W08 Actions 首次失败诊断与日志修复
<!-- trellis-session: v=2 fp=8af415b8544c94f7 -->

**Date**: 2026-10-03
**Task**: W08 Actions 首次失败诊断与日志修复
**Branch**: `main`

### Summary

真实run根Go失败但stdout未显示且无artifact，原失败用例无法恢复；修复fail-fast后仍检查JSON与失败日志附件。

### Main Changes

- 拆分CI阶段，保留Go非零退出，打印测试/编译/required skip上下文，failure时上传隐藏目录的指定日志。

### Git Commits

(No commits - planning session)

### Testing

- [OK] 本机无缓存根Go198pass、bridge29pass及vet，Linux测试仅交叉编译通过；34项回归与actionlint通过。

### Status

[OK] **Completed**

### Next Steps

- 提交推送新workflow后读取Ubuntu实际失败详情；未推送、重跑旧run或发布，不声称原Ubuntu根因已修复。


## Session 19: 修复 Actions rg 14 与 Git 2.55 工具兼容失败
<!-- trellis-session: v=2 fp=7156d8f497f00171 -->

**Date**: 2026-10-03
**Task**: 修复 Actions rg 14 与 Git 2.55 工具兼容失败
**Branch**: `main`

### Summary

通过 GitHub 读取 run 37100462607，复现并修复替换能力探测和测试后台维护。隔离 Git 2.55/rg 14 下完整 Go 200 pass、桥接29、race/vet、前端163、部署34及Linux两架构编译通过。原本机环境失败保留；远端须提交推送后验证，未提交发布或安装。

### Main Changes

- 搜索创建前验证 JSON 替换能力，旧 rg 固定 native，增加完整原字节替换回归
- Git fixture 关闭自动维护，原20次失败19次，修复后重复20次通过，保持产品快照保护

### Git Commits

(No commits - planning session)

### Testing

- [OK] 完整根Go200与桥接29、required无skip，search/gitview race和root/bridge vet通过
- [OK] 前端lint/typecheck/163单测/build、部署34回归、Linux amd64/arm64编译与本地链接/diff检查通过

### Status

[OK] **Completed**

### Next Steps

- 用户提交推送main后验证新提交触发的Actions；保持任务in_progress和目标部署未执行边界


## Session 20: 交付 Linux amd64 手工部署包
<!-- trellis-session: v=2 fp=74f078861d9ce95c -->

**Date**: 2026-10-03
**Task**: 交付 Linux amd64 手工部署包
**Branch**: `main`

### Summary

用户停止 CI 跟进，完成 amd64 前后端交叉编译、配置打包与逐文件部署说明；目标机尚未操作。

### Main Changes

- 打包脚本新增可选单架构，默认两架构保持；deploy/MANUAL.md 同步包根 README 和旁置部署说明，沿用现有 Nginx、局域网HTTP，不保存实际IP。
- 原第三次 CI dev 重启失败仍未修复，临时测试诊断已恢复；W08保持进行中，不发布或提交。

### Git Commits

(No commits - planning session)

### Testing

- [OK] 实际 amd64 静态 ELF、SHA、安全解压/清单、前端入口/字体/worker、配置文档逐字节检查通过。
- [OK] Go test/vet、前端 lint/typecheck/163测试/build、Python部署34回归、Bash/文档链接/diff通过；普通Go未开启取消跟进的dev启动集成，不作为CI验收。

### Status

[OK] **Completed**

### Next Steps

- 用户按手工文档在Debian启动systemd与已有Nginx，验证HTTP登录/编辑/终端；目标机未执行项不记通过。


## Session 21: 修复局域网 HTTP UUID 初始化故障并重新交付
<!-- trellis-session: v=2 fp=f3597ddb67190a96 -->

**Date**: 2026-10-03
**Task**: 修复局域网 HTTP UUID 初始化故障并重新交付
**Branch**: `main`

### Summary

三个前端UUID调用统一安全随机回退；旧产物复现相同堆栈，新amd64包通过普通HTTP浏览器回归。

### Main Changes

- 新增共享random-uuid与测试，替换编辑器scope/buffer及搜索定位调用；保持草稿身份、认证、HTTP/Nginx配置不变。
- 交付dist/debian-amd64-http-uuid-20261003，根README和旁置文档说明按第7节保留配置/数据/tmux升级并强制刷新；旧包保留。

### Git Commits

(No commits - planning session)

### Testing

- [OK] 前端lint/typecheck/168测试/build通过，Go test/vet通过；安全解压/manifest、amd64静态ELF、SHA、全部前端产物与文档配置一致性通过。
- [OK] 旧包在虚构私有HTTP浏览器源复现crypto.randomUUID错误；新生产产物Chromium/WebKit各1项工作台/Monaco/重复搜索定位通过。首次harness根目录/Node问题已纠正，未记产品失败。

### Status

[OK] **Completed**

### Next Steps

- 用户在Debian按文档升级并复验；本轮浏览器API/WS为fixture，不宣称真实后端/Nginx或原CI故障已解决。


## Session 22: 补齐下方终端取消分隔入口并交付新包
<!-- trellis-session: v=2 fp=2f8b7df449568122 -->

**Date**: 2026-10-03
**Task**: 补齐下方终端取消分隔入口并交付新包
**Branch**: `main`

### Summary

新增每组合并按钮，空分组可逐个取消，运行终端标签并到邻组且保留原实例连接；交付新版amd64包。

### Main Changes

- workspace-view新增mergeTerminalGroup，压缩组索引并保留活动项/顺序/上方/手机状态；TerminalWorkspace复用shadcn Button和lucide合并图标，X仍整体收起。
- 更新任务/状态与终端规范/部署说明，交付manual-20261003-terminal-merge和SHA，保持HTTP UUID修复、实际配置与独立tmux。

### Git Commits

(No commits - planning session)

### Testing

- [OK] 前端lint/typecheck/171测试/build、Go test/vet、实际SHA/amd64静态ELF/前端全文件/模板文档验包、链接/diff通过。
- [OK] Chromium/WebKit各3项生产前端回归：空四组恢复与刷新/拆分/收起、运行中会话DOM和WS稳定零终止接管输入、HTTP UUID保持。首次收起子元素visible断言不适合裁剪面板，按真实父面板零高度契约修正后复验；未改产品绕过。

### Status

[OK] **Completed**

### Next Steps

- 用户按部署说明第7节升级并强制刷新；本轮API/WS为fixture，不记Debian真实PTY/Nginx验收，不自动提交或归档。
