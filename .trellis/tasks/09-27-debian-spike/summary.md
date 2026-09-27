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

## 证据
- [环境采集](../09-26-requirements-research/research/debian-environment.md)
- [文件结果](files-report.md)
- [终端结果](terminal-report.md)
- [最终复核](check-report.md)

## 不能解除的门禁
本次不是完整 W02 验收。W03 正式终端界面仍依赖真实 Go/PTY/WS bridge、TUI/history/背压及控制协议实验；W04 根访问实现可采用已测能力，但 CLI Landlock/安全执行和业务原子保存尚未通过；W07 helper 提权未获安装授权也未验收。不得将底层探针成功写成 PA01..PA26 或完整产品通过。

下一阶段继续 W02，优先将已测 tmux 隔离接入真实 Go bridge 实验，以及单独验证受限 CLI；不先铺正式终端 UI。命令捕获虽限制每流内存 1 MiB，磁盘捕获仅有 20 秒超时、无字节配额，不能作为通用生产命令沙箱。

## 版本管理
用户先要求暂停提交并剥离敏感信息；修正及复核完成后最新明确要求“帮我提交”，按更新的 commit-plan.md 提交本批，不推送或归档。W01 已提交 4113358；其他活动任务保持原状态。W02 当前保持 in_progress，后续完整实验继续在此任务追踪。

## 连接信息保密修正
最新用户选择以仓库根已忽略的 .env 为唯一来源，不使用同名进程环境变量或默认值。原键及实际值保留，文件格式已兼容，权限从 0644 收紧为 0600；可提交 .env.example 只有虚构值。任务/调研记录、复现命令和 JSON 证据已去敏，证据标注非原始逐字记录。新增受限解析与实验 runner，本地 18 项回归、上下文/语法/JSON/链接检查通过；可提交 UTF-8 文本和 HEAD 的私密值扫描零命中。本轮未远端重跑，不改变前述真实 Linux 实验历史；runner SIGKILL 清理及未知原始输出的限制见最终复核与远端规范。提交授权以后续最新消息为准，私有 .env 永不纳入。
