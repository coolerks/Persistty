# W02 当前结果与后续门禁

## 当前交付
在用户授权 Debian 13.4/systemd 257.9 环境完成隔离底层实验、可重复探针及独立检查。sudo 需要密码，因此没有系统安装 tmux；从目标机软件源临时解包 tmux 3.5a-3 与必要 libevent，所有解包文件已清理。未修改现有 Nginx/WireGuard/防火墙/业务服务或安装 helper。

| 项目 | 实际结果 |
| --- | --- |
| D01 环境与资源隔离 | 版本、UID、私有目录/socket/unit/scope 留证，全部实验资源精确清理 |
| D02 底层生命周期 | 模拟 Web stop/restart/SIGKILL、attach 消失、跨 SSH 后原 PID/start time 不变且逐次心跳增长 |
| D03 连接与尺寸 | 空 server 保持，缺 socket -N 不启动；只读 ignore-size 观察端不改变控制端尺寸；产品 history/WS/TUI 未验收 |
| D04 根安全能力 | Linux os.Root 合法访问和越界拒绝、有限目录交换通过；移动句柄跟随身份的限制已确认；Landlock ABI6 仅查询 |
| D05 复核与测试 | 全 Go test/race/vet、Linux 文件3测试、本地/Linux Python3测试、完整终端独立重跑及失败清理通过 |
| D06 真实桥接基础 | Go/PTY/WS 最终二进制独立 Debian 重跑通过，13 次原身份/逐增心跳采样、单次输入无重放、精确资源清理；不是正式多端或浏览器验收 |
| D07 历史/TUI 机制 | 真实 curses 三尺寸/重连/退出与原进程存活通过；raw attach 不含完整历史、capture+attach 遗漏间隙行的反例已实测，不解除生产 snapshot/live 门禁 |

## 证据
- [环境采集](../09-26-requirements-research/research/debian-environment.md)
- [文件结果](files-report.md)
- [终端结果](terminal-report.md)
- [最终复核](check-report.md)
- [Go 桥接实现](go-bridge-report.md)
- [桥接编排与首轮实测](bridge-runner-report.md)
- [桥接独立复核](bridge-check-report.md)
- [最终二进制实测证据](../../../tests/integration/debian/bridge/evidence-review.json)
- [D07 研究](research/history-tui-plan.md)
- [D07 实施](history-report.md)
- [D07 独立复核](history-check-report.md)
- [D07 独立摘要证据](../../../tests/integration/debian/history/evidence-review.json)

## 不能解除的门禁
本次不是完整 W02 验收。W03 正式终端界面仍依赖生产 history snapshot/live、持续负载背压及多观察端控制协议实验；真实 Go/PTY/WS 基础链路与一个受控 curses 程序已通过隔离验证，但不等于产品服务或完整 TUI 覆盖完成。W04 根访问实现可采用已测能力，但 CLI Landlock/安全执行和业务原子保存尚未通过；W07 helper 提权未获安装授权也未验收。不得将底层探针成功写成 PA01..PA26 或完整产品通过。

下一阶段继续 W02，先明确历史快照/live 的单输出 owner 与同步截点，再做受限 CLI；不先铺正式终端 UI。桥接命令捕获虽限制每流内存 1 MiB，磁盘捕获仅有 25 秒超时、无字节配额，不能作为通用生产命令沙箱。

## 版本管理
第一批及隐私修正在用户批准后提交 5f392ed，W01 已提交 4113358，后续用户提交 9bf86d8 保持不动。本轮 D06 未提交，提交需单独批准，不推送或归档。其他活动任务保持原状态，W02 当前保持 in_progress，后续完整实验继续在此任务追踪。

## 连接信息保密修正
最新用户选择以仓库根已忽略的 .env 为唯一来源，不使用同名进程环境变量或默认值。原键及实际值保留，文件格式已兼容，权限从 0644 收紧为 0600；可提交 .env.example 只有虚构值。任务/调研记录、复现命令和 JSON 证据已去敏，证据标注非原始逐字记录。新增受限解析与实验 runner，本地 18 项回归、上下文/语法/JSON/链接检查通过；可提交 UTF-8 文本和 HEAD 的私密值扫描零命中。本轮未远端重跑，不改变前述真实 Linux 实验历史；runner SIGKILL 清理及未知原始输出的限制见最终复核与远端规范。提交授权以后续最新消息为准，私有 .env 永不纳入。

## D06 最终复核
本节是随后 D06 的新结果，不覆盖上节第一批隐私修正的历史。最终 Linux 二进制 SHA256 为 `788be5a918f7593e63d023b77c98a7d9dc88a99ccb51d0d2a00d162cd0bbbe78`，已包含完整依赖及 Go 许可证通知。根模块 test/race/vet 与独立 bridge 模块 12 项测试、race 重复三次、vet 通过；Python bridge 6 项、连接/终端 21 项通过。前端未变化，本轮未重跑前端门禁。

独立远端复核 13 次心跳从 1 增到 59，固定 server/pane 身份，首次明确输入后计数一直为 1；三个精确 unit、派生 scope inactive，私有 ROOT 已删除。早期 allocate 传输失败不能直接断言无目录，前后只读核查当前用户拥有的 0700、非 symlink、指定实验前缀目录均为空，未清理未知资源。runner 被强制结束仍可能绕过 finally，不能承诺任意故障零残留。

## D07 最终复核
最终 Linux binary SHA256 为 `c294742fb8848421b3da04b956199327df981c36ebcdf5b9ee2c2b6319e16501`；新增 record 扩展，不覆盖 D06 旧版本证据。修正 runner 后最终独立重跑 7 个记录逐字节重分块解析状态一致，curses 在 100x30、80x24、120x40、重连及退出后当前画面/实际 pane 模式符合预期。6 次原身份采样心跳 `3,49,94,115,136,160`，stdin 字节计数始终 0，7 attach 已回收；三个 unit、精确关联 scope inactive，ROOT 删除、本地 raw cache 为零。限定当前用户私有 history/bridge 目录只读核查为空，不作全机器零残留承诺。

8 个 true 判定中 3 个是成功观测恢复限制：raw attach 不含早期历史；capture 后追加 attach 仍处于外层 alternate；间隙首行已遗漏。它们不是产品历史恢复通过。xterm headless 只验库解析，浏览器 renderer/实时鉴权链路、serialize/生产 snapshot 协议、长期负载和多端控制未验收。

独立检查修正 Python 优化模式守卫及远端分配前的本地缓存准备顺序，并补 record 超限/取消回归。根 Go test/race/vet、独立 bridge 模块 16 项测试及 race/vet、33 项 Python、2 项 Node tests 通过；没有产品前端变更，不声称前端门禁已重跑。本轮无提交、推送或归档。
