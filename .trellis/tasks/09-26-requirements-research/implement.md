# Persistty 首版实施计划审查稿

## 阶段与授权

依赖[PRD](prd.md)与[设计](design.md)，状态planning。本文件不是启动指令。用户在最终总结展示后的新消息中明确批准后，才可执行task.py start或实施产品。默认参数、故障取舍及高风险方案同样属于此次审查；实质变动重新审查。

当前任务是调研/规划父交付；下面为可独立验收工作包，不在本轮创建子任务或变更其他任务。以后若拆子任务，每份PRD/implement必须写明直接依赖与输入产物，不能仅靠树位置推断。

## 有序工作包

| ID | 交付与建议所有权 | 直接依赖 | 独立验收 |
| --- | --- | --- | --- |
| W01 | owner spec对齐、配置/认证/SQLite/HTTP/WS骨架；cmd/persistty、internal/config/auth/storage/httpapi、web基础、deploy | 最终新批准 | 保护矩阵、vpn_http显式模式、配置拒绝非法值、登录/撤销；无产品假接口 |
| W02 | 目标Debian生命周期/根安全/提权实验；tests/integration与deploy探针 | W01基础与已批准实验环境 | 独立cgroup/socket、无隐式tmux启动、真实TUI/安全CLI/受限helper可行性报告 |
| W03 | 持久终端、多观察端控制、history/恢复、终止倒计时；internal/terminal、terminal前端 | W01、W02终端实验 | PA01..PA05，真任务/真实WS不以fake代替 |
| W04 | 根注册/切换、内容识别/版本、CRUD/watcher、上传/下载/ZIP；internal/workspace/files/transfer、explorer/workspaces/transfer前端 | W01、W02文件实验 | PA06..PA09基础、PA13..PA14、PA17，原字节hash/部分结果/竞态 |
| W05 | 统一编辑adapter、Monaco/手机基础、autosave/draft/预览/资产与主题；editor/settings/assets | W04文件协议、W01前端/认证 | PA08..PA10、设备边界与资产许可 |
| W06 | 项目搜索预览/替换及只读Git/HEAD色条/引用比较；internal/search/gitview、search/git前端 | W04快照/安全CLI、W05缓冲区接口 | PA11..PA12，CLI对照与预览后变化拒绝 |
| W07 | root-owned单文件helper、授权请求/密码弹窗、安全审计；专用高权限模块和editor操作 | W01认证、W02提权实验、W04版本、W05保存状态 | PA15提权部分，凭据管道/越界/复用/崩溃失败注入 |
| W08 | 多设备E2E、资源/性能、真实部署/升级回滚、文档；tests/e2e/integration、deploy | W03..W07全部 | PA01..PA17矩阵、VPN公网隔离、实际配置/版本与剩余风险清单 |

初始目录约定来自backend/frontend directory-structure.md；尚无源码，不宣称这些目录已存在。每包创建任务时按需加精确文件所有权，worker不得撤回他人改动。共享API/types由协议owner先落定后分派，避免并行修改同一共享文件。

## 实施步骤与关卡

1. 最终新批准后加载owner spec，更新设计列出的规范差异，保留已有安全原则，不机械照抄旧额外attach拒绝/全局偏好/固定allowed_roots/仅手动保存/仅HTTPS规则。
2. 锁定当时维护的Go/前端依赖/Monaco/xterm与实际tmux/systemd/rg/Git版本；记录版本与兼容性。无需为了规划安装依赖。
3. 先认证/根注册/资源ID/错误协议，创建真实最小垂直链路，而非先铺大量空package和假数据界面。
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
- 原子写入与外部writer窗口反例、路径/链接/硬链接/权限、copy/move/delete中途失败，确认后内容变化。
- 外部创建/移动/重命名/删除/修改、watcher溢出与WS重连重新列举；树/打开状态收敛且不清空编辑输入，删除确认后目标替换拒绝旧请求。
- UTF-8/BOM/中文/emoji/混合换行/无末尾LF/非法字节，无后缀/二进制伪装；自动保存时新输入与草稿恢复暂停。
- rg语义/ignore/include边界，Git配置/辅助程序/缺对象/根外gitdir/根和merge提交，预览与实际字节一致。
- 目录同级超过100项/空目录、页面存活续传/Web重启、相同内容跳过、确认竞争、配额、ZIP实际解压/失败提示。
- helper目标/内容/版本绑定、nonce重复/过期/取消、认证失败/日志/代理/argv无凭据，root-owned配置拒绝篡改与符号链接转移。
- VPN入口IPv4/IPv6/路由/公网不可达、Nginx体积/超时/断线、HTTP Cookie限制、Origin/CSRF和SVG隔离。

## 高风险文件与回滚点

deploy/systemd、Nginx/VPN防火墙、helper安装与root-owned策略、认证/会话、DB迁移、终端控制协议、根安全I/O和完整保存提交点是高风险边界。真实文件/任务不能靠Git回滚恢复；先在fixture验证，再获准目标机操作。

Web迁移失败不停止tmux，回退Web/DB兼容版本，不重跑命令。helper回退先停提权接口，普通终端/文件工具继续普通权限。提权与终止响应丢失先查实际状态，不能盲重试；上传/ZIP暂存回收仅删本应用拥有的临时数据。

## task.py start之前

- PRD已收敛、无阻塞用户范围问题，design/implement完整；技术实验是实施关卡，未冒充通过。
- 本任务implement.jsonl/check.jsonl含真实规范/研究/设计条目，旧spec与最新PRD冲突明确按owner规范更新解决。
- 最终总结展示目标、范围内/外、验收、关键决定、风险/参数及产物状态，收到新的明确批准。
- 若改为子任务，先创建独立验收PRD与具体依赖/上下文，再逐个按审查流程启动，不以父任务批准替代变化后的子任务范围审查。

当前没有执行上述产品命令、安装、目标机实验或启动动作。计划通过文档校验不等于产品验收完成。
