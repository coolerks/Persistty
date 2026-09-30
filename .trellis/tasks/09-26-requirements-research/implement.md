# Persistty 首版实施计划审查稿

## 阶段与授权

2026-09-30 当前状态：W01/W02/W04 已完成归档；[W03](../09-29-terminal-runtime/check-report.md) T01..T07 与真实 Debian/9 条浏览器验收通过，用户批准提交及归档，不推送。下一工作包为 W05（自动保存/草稿、完整工作台布局和状态恢复），尚未创建或批准新的子任务；首版 W06/W07/W08 也未完成。下方阶段记录保留当时语境，不作为最新待办清单。

2026-09-27最新总结之后收到用户明确“开始实施”。下面的规划门禁已满足，按W01开始分包实施，见[基础子任务](../archive/2026-09/09-27-foundation/implement.md)。后续W02目标机/root操作不包含在本地编码授权内；分包验收不能视为首版全部完成。

已按U67..U73更新三主题、上下两区域仅左右分组/手机单内容、终端展示位置、折叠边缘展开及类型化关闭命令。规划阶段未实施UI；最新总结之后的新批准现已收到。

已按U63..U66更新多文件夹项目、主文件夹、项目维护/移除、路由/书签恢复及多仓库契约。上一版单根方案最终总结不再作为当前实施批准依据。

依赖[PRD](prd.md)与[设计](design.md)，当前in_progress。最终总结之后的新批准门禁已满足，默认参数与故障取舍随批准生效；实质范围变动仍需重新审查，高风险目标机操作另授权。

当前任务管理原始需求与跨包验收，下面为独立工作包。W01 已通过基础门禁并提交 4113358，尚非完整首版；已创建 [W02 隔离实验子任务](../archive/2026-09/09-27-debian-spike/implement.md)，用户批准指定 Debian 的 tmux 安装及自身临时目录/socket/用户单位实验，不包含现有服务、网络或 helper 修改。sudo 非免密，优先普通用户临时解包 tmux。子任务PRD/implement写明直接依赖、边界和输入，不仅靠树位置推断依赖。

W02 第一批及隐私修正已提交 5f392ed；后续 D06 真实 Go/PTY/WS 基础链路经最终二进制独立 Debian 复核通过，详见 [W02 当前总结](../archive/2026-09/09-27-debian-spike/summary.md)。当时尚未提交，W02 尚未整体完成：TUI/history、多端控制及安全 CLI 等门禁保持，不开始大规模正式终端 UI。

D07 后续实测已覆盖一个真实 curses 程序的 resize/重连/退出，并证实直接 capture+attach 有历史遗漏，不能据此定案恢复协议；生产 snapshot/live、浏览器实时链路、完整 TUI 覆盖与多端控制仍待验证。本轮仍未获新提交授权。

随后用户批准 D06/D07，已提交 965b4c6。继续 D08 仅为独立离线单 owner/seq/快照对照；公开 serialize 在 pending 和部分完整状态仍有差异，真实 9 个有限截点也有 1 个不一致，不能批准生产无损恢复。后续新代码单独验收，不与已提交批次混淆。

## 有序工作包

| ID | 交付与建议所有权 | 直接依赖 | 独立验收 |
| --- | --- | --- | --- |
| W01 | owner spec对齐、配置/认证/SQLite/HTTP/WS骨架、项目/folder/terminal协议、布局类型/命令骨架与React Router/Nginx页面分流；cmd/persistty、internal/config/auth/storage/httpapi、web基础、deploy | 最终新批准 | 保护矩阵、vpn_http显式模式、配置拒绝非法值、登录/撤销、直达路由/API404；无产品假接口 |
| W02 | 目标Debian生命周期/根安全/提权实验；tests/integration与deploy探针 | W01基础与已批准实验环境 | 独立cgroup/socket、无隐式tmux启动、真实TUI/安全CLI/受限helper可行性报告 |
| W04 | 项目配置CRUD/主folder/注册身份、项目面板/路由工作台、多根文件树与版本、双根CRUD/watcher、上传/下载/ZIP；internal/workspace/files/transfer、explorer/workspaces/transfer前端 | W01、W02文件实验 | PA06项目/路由部分、PA07..PA09基础、PA13..PA14、PA17、PA19配置/关联裁决，原字节hash/部分结果/竞态 |
| W03 | 持久终端、多观察端控制、history/恢复、终止倒计时、主folder选择、项目解绑/统一入口、布局移动runtime与应用级终止弹窗；internal/terminal、terminal前端 | W01布局/协议骨架、W02终端实验、W04项目注册/配置契约 | PA01..PA05、PA06终端部分、PA18、PA21/PA24/PA26终端命令部分，真实WS/原PID及移动零新建零隐式终止 |
| W05 | 工作台布局/仅左右组/拖放与折叠边缘展开、统一编辑adapter、Monaco/手机基础、autosave/draft/预览/资产与三主题、按项目/视图/设备恢复；layout/editor/settings/assets | W04项目/文件协议、W01布局/认证骨架、W03终端runtime/关闭接口 | PA08..PA10、PA06界面恢复、PA19输入保护、PA20..PA26布局/主题/设备交互，与W03跨层集成 |
| W06 | 多根搜索预览/替换去重及多仓库只读Git/HEAD色条/引用比较；internal/search/gitview、search/git前端 | W04项目版本/快照/安全CLI、W05缓冲区接口 | PA11..PA12、PA19替换范围失效，CLI对照/仓库隔离与预览后变化拒绝 |
| W07 | root-owned单文件helper、授权请求/密码弹窗、安全审计；专用高权限模块和editor操作 | W01认证、W02提权实验、W04版本/配置裁决、W05保存状态 | PA15提权部分、PA19提权关联失效，凭据管道/越界/复用/崩溃失败注入 |
| W08 | 多设备E2E、原型布局/资源/性能、真实部署/升级回滚、文档；tests/e2e/integration、deploy | W03..W07全部 | PA01..PA26矩阵、VPN公网隔离、实际配置/版本与剩余风险清单 |

表中按依赖列出顺序，保留原W编号方便追溯，W04项目契约先于W03。初始目录约定来自backend/frontend directory-structure.md，现有实现以当前源码与子任务验收为准。每包创建任务时按需加精确范围；当前执行策略禁用 subagent，全部由主会话直接完成，不使用原规划的 worker 分派。

## 实施步骤与关卡

1. 最终新批准后加载owner spec，更新设计列出的规范差异，保留已有安全原则，不机械照抄旧额外attach拒绝/全局偏好/固定allowed_roots/仅手动保存/仅HTTPS规则。
2. 锁定当时维护的Go/React Router/react-resizable-panels/前端依赖/Monaco/xterm与实际tmux/systemd/rg/Git版本；记录版本/API与兼容性。无需为了规划安装依赖。
3. 先认证/项目-folder注册/配置version/终端独立ID/错误协议与路由骨架，创建真实最小垂直链路，而非先铺大量空package和假数据界面；项目移除禁止真实资源级联删除。
4. 在隔离、获准环境跑W02；root/helper安装、目标Debian命令与系统服务变更需单独操作授权。能力不足就报告方案边界，不能静默禁用安全校验或扩权。若方案改变MVP行为重新审查。
5. 终端状态/generation/截止序列与文件快照/generation先定契约，再接前端；任务/输入/当前屏幕/元数据/历史分别测。
6. 按工作包补单元/集成/E2E测试，逐包验收；跨层失败注入不能等到最后才做。W07独立安全审查，不以登录成功或VPN可达代替。
7. 真机移动输入、目录枚举/空目录、像素/ZIP/字体与端到端恢复在W08确认；失败明确记录，不把桌面viewport模拟视作真机通过。
8. 检查、构建、启动本地开发服务器提供试用地址；完成最终review与Trellis记录。提交/推送/归档另遵循用户授权，不自行处理现有bootstrap/中文化任务。

## 验证命令与观察结果

文档规划阶段可执行：

```bash
python3 .trellis/scripts/task.py validate .trellis/tasks/09-26-requirements-research
git diff --check
git status --short
```

实现foundation建立脚本后固定以下入口，若未创建则不能声称运行过：

```bash
go test ./...
go test -race ./...
go vet ./...
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run test
npm --prefix web run build
npm --prefix web run test:e2e
```

集成计划须提供独立命令/fixture，覆盖：
- AI/后端/npm三原PID/start time在关页/断网/登出/Web/Nginx重启的连续性；整机重启不自动执行。
- 两电脑两手机/多tab，lease/旧输入/尺寸/控制转移、取消截止竞态、Web重启取消pending。
- 桌面上方/下方至少三列并排、无内部上下drop目标，组内/跨组拖动和文件下拖拒绝；会话移上/移回保留terminal_id/控制generation/原PID及有界输出，不重复创建/重放。
- 下方面板X收起零终止请求、上方terminal tab X/单会话trash仅controller发起统一倒计时；面板收起或会话移动时仍有应用级弹窗，observer接管必须显式，取消/失败标签保留。
- 左栏/下方面板缩小折叠、边缘外拖展开、键盘/按钮替代入口、布局持久/窄窗口约束；隐藏不发0x0，observer不发PTY resize；文件model/undo/草稿和终端历史不因宿主移动丢失。
- system/light/dark切换及系统实时变化、显式主题不被覆盖、首帧/监听清理、Monaco/DiffEditor/xterm/菜单一致；手机仅一个内容、软键盘与快捷栏、设备布局记录分隔且不覆盖桌面组。
- 多文件夹同名/重叠根、跨根copy/move、搜索去重、多个/嵌套仓库HEAD隔离；项目改名/主folder调整保留书签，配置竞争返回冲突，旧保存/替换/上传不能穿透失效关联。
- 项目移除不删文件/不杀原PID，统一终端入口在重启/重登录后可访问；原终端cd到项目外不改变文件API范围，原pending倒计时继续按终端规则裁决。
- 书签直达/刷新/登录返回、浏览器后退/前进、当前/新tab、已移除项目/失效目录、同浏览器同项目多tab独立缓冲区、草稿存储失败不丢输入；Nginx不能把API/WS/缺失静态资源回退为HTML。
- 重叠根移除后同文件复验改绑仍可编辑、旧请求拒绝且model/输入/草稿不丢；失效项目页草稿导出不重建项目或旧访问权限。
- 原子写入与外部writer窗口反例、路径/链接/硬链接/权限、copy/move/delete中途失败，确认后内容变化。
- 外部创建/移动/重命名/删除/修改、watcher溢出与WS重连重新列举；树/打开状态收敛且不清空编辑输入，删除确认后目标替换拒绝旧请求。
- UTF-8/BOM/中文/emoji/混合换行/无末尾LF/非法字节，无后缀/二进制伪装；自动保存时新输入与草稿恢复暂停。
- rg语义/ignore/include边界，Git配置/辅助程序/缺对象/根外gitdir/根和merge提交，预览与实际字节一致。
- 目录同级超过100项/空目录、页面存活续传/Web重启、相同内容跳过、确认竞争、配额、ZIP实际解压/失败提示。
- helper目标/内容/版本及project/folder/config绑定、关联移除与接受执行的两种裁决顺序、nonce重复/过期/取消、认证失败/日志/代理/argv无凭据，root-owned配置拒绝篡改与符号链接转移。
- VPN入口IPv4/IPv6/路由/公网不可达、Nginx体积/超时/断线、HTTP Cookie限制、Origin/CSRF和SVG隔离。

## 高风险文件与回滚点

deploy/systemd、Nginx/VPN防火墙及SPA回退、helper安装与root-owned策略、认证/会话、项目/folder/终端DB关系、配置变更与文件发布裁决、终端控制协议、根安全I/O和完整保存提交点是高风险边界。真实文件/任务不能靠Git回滚恢复；先在fixture验证，再获准目标机操作。

Web迁移失败不停止tmux，回退Web/DB兼容版本，不重跑命令。helper回退先停提权接口，普通终端/文件工具继续普通权限。提权与终止响应丢失先查实际状态，不能盲重试；上传/ZIP暂存回收仅删本应用拥有的临时数据。

## task.py start之前

- PRD已收敛、无阻塞用户范围问题，design/implement完整；技术实验是实施关卡，未冒充通过。
- 本任务implement.jsonl/check.jsonl含真实规范/研究/设计条目，旧spec与最新PRD冲突明确按owner规范更新解决。
- 最终总结展示目标、范围内/外、验收、关键决定、风险/参数及产物状态，收到新的明确批准。
- 若改为子任务，先创建独立验收PRD与具体依赖/上下文，再逐个按审查流程启动，不以父任务批准替代变化后的子任务范围审查。

规划时上述命令与实验未执行；当前W01已有实际本地安装、产品测试和开发服务，记录归子任务，仍未执行目标机系统实验。分包通过不等于首版全部验收完成。
