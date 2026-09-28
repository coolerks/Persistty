# W02 D09/D10 单代理最终检查

2026-09-28，按项目 inline/无子代理约束由主会话直接实施与复核。只新增隔离探针、受控远端入口和任务/owner 文档；未更改产品服务、系统服务、sudoers/PAM、Nginx/WireGuard 或 root helper。此前 D01–D08 的负面证据保持不变。

## 运行结果

- 真实 Debian D09：九项合成判定全部通过，`rg`/Git 存在，临时 ROOT 精确删除；阻塞 `cat` 由本次进程取消回收。只记录布尔/退出码，原始 stdout/stderr 未写入报告。
- 真实 Debian D10：只读确认 sudo、sudoedit 与 PAM sudo 服务入口存在；无提权或策略读取。
- 本地：根 `go test ./...`、`go test -race ./...`、`go vet ./...` 通过；Linux/amd64 CLI 交叉构建与 `go vet` 通过；Python helper 5 项及既有远端配置/终端 21 项通过。
- `task.py validate` 通过，修改的 Markdown 本地链接无缺失，`git diff --check` 通过。私有 `.env` 不跟踪；扫描 330 个可提交路径，真实连接字段匹配路径数为 0，仅输出计数。

## 人工风险复核

- D09 不是完整 root-only 沙箱：动态系统目录仍可读，外链指向白名单路径未验证；Landlock 与词法路径限制分别承担不同防线，不能把直接越界拒绝记作 Landlock 证据。
- D10 不是真实 PAM/特权保存：模拟只在本进程记录已用 nonce，直接 truncate 非原子，Python 内存清零不可证明；W07 需另行授权与实现。
- 远端 runner 成功路径只清理其自身随机 ROOT；若宿主强制终止 runner，`finally` 不保证执行，仍需按已知 ROOT 人工核查，不通配清理未知目录。
- 本轮未改 web 前端，未运行前端检查；D01–D08 的历史证据和 W03/W04/W07 产品验收仍按各报告保留。

本任务的隔离可行性范围可判为完成；正式产品门禁未解除。提交/归档须按用户本轮批准和 Trellis 收尾步骤单独执行。
