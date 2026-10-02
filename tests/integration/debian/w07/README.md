# W07 Debian 专项（尚未执行）

本目录提供可审查的 TTY 验收客户端；它不安装 helper、不改 sudoers、不自行提升权限。必须先由用户授权精确 Debian 主机、现有账户、安装路径与隔离目标。2026-10-02 本机 Go/TS/浏览器使用普通用户和合成协议，不能替代这里的真实验收。

## 前置与资源范围

按 [安装审查稿](../../../../deploy/elevation/README.md)准备独立 socket/service、root-owned 策略和 nonce 目录。目标为新建的 `/srv/persistty-privileged/example.txt` 等精确测试文件，由管理员设置 root-owned 父目录、0644 可读文本，登记为隔离项目；绝不拿真实系统配置、用户重要文件、正式 tmux/服务测试。记录 UID/GID、mode、`getfacl`、`getfattr` 和启用 SELinux 时的安全标签基线；报告只包含类别/版本/摘要，文件正文和密码不入报告。

编译探针（目标机原生构建或预先交叉编译，输出只在批准的自有上传目录）：

```bash
go build -o /tmp/persistty-w07-probe ./tests/integration/debian/w07/probe
```

运行时参数只传目标元数据；应用密码和系统密码均在本机隐藏 TTY 输入，不通过 SSH argv/env/文件/chat。示例中的 ID/origin 都必须替换成已批准实例：

```bash
/tmp/persistty-w07-probe --origin http://10.66.66.1 --project ISOLATED_ID --folder ISOLATED_FOLDER --project-version 1 --path example.txt --expect applied --ack-write-isolated-target
```

该命令**将向隔离文件追加合成标记**，并验证重复 execute=409、普通复读内容相符。`--expect rejected` 用主动输入的错误系统密码验证零写入；`--expect cancelled` 不询问系统密码、取消 prepared 并验证零写入。任何网络错误只留下 request_id 用于人工查询，不自动重发真实密码/保存；探针不是未确认结果的重试客户端。

## 必须补齐的实际证据

| 维度 | 实际断言 |
| --- | --- |
| sudo/PAM | 无 TTY 的 broker 正确口令成功、错误/空/多轮交互失败；已有 sudo timestamp 不被本命令复用；有效 sudoers 严格无 args/无 SETENV |
| FD / 日志 | stdin 只含认证，FD3 控制、FD4 正文；argv/env/system journal/sudo I/O/Nginx body temp 无合成秘密；禁 core dump |
| 文件安全 | 精确允许列表拒绝其他文件及所有基础设施；symlink/hardlink/FIFO、根/父/leaf替换、同大小同mtime内容变化零覆盖 |
| 元数据 | UID/GID/mode/ACL/user xattr/SELinux 等实际保持；未知安全属性及属性复制失败在 rename 前拒绝；BOM/混合换行/8MiB 边界 |
| 故障与取消 | READY 前终止/断线/过期、配置删除/注销零发布；COMMIT 接受后核对实际状态；rename/目录 fsync/账本失败为 indeterminate，无盲目重放 |
| 一次性 | 并发 execute、重启 Web/broker、账本 nonce 重放、时钟回退拒绝；同目标 helper 锁串行；不同登录不能使用/查询请求 |
| systemd | Web 非 root/NoNewPrivs=1；broker 同开发 UID/NoNewPrivs=0；root helper 短命；stop 只清理自有 broker/helper cgroup，不碰 tmux |
| Nginx | nginx -t；最大转义文本体可用；两类敏感请求零临时正文文件/缓存/日志；只限已批准 VPN HTTP 或 TLS |

执行 Go 的 Linux 文件专项（其中不是所有测试都需要 root，不能把普通用户专项称为特权验证）：

```bash
go test ./internal/files -run Privileged -count=1
go test -race ./internal/elevation ./internal/storage ./internal/httpapi
```

只关闭本次隔离请求、停止明确的新 socket/service、清理本轮精确上传路径和合成目标；保留 nonce 账本以防回滚重放。没有一键扩大 root 访问的安装/清理程序。真实部署/用户手机软键盘以及 W05/W06 统一验收继续各自记录，不合并为通过。
