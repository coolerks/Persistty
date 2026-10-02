# W07 单文件授权安装审查稿

这些产物未应用。目标为 Debian 现有非 root 开发账户；Web 保持 NoNewPrivileges=true，另起非 root broker，sudo 只运行固定无参数 helper。不可通过修改整个 Web unit 来启用 sudo，不影响 tmux unit。真实认证/特权/系统配置验收未完成前，Web `elevation.enabled: false`。

## 精确安装范围

先确认主机、现有账户数值 UID/GID、允许的已有可读文本、对应项目根、Web 配置/数据库/二进制位置及现有 sudo/PAM 策略。示例 `developer`、1000、`/srv/persistty-privileged/example.txt` 都是虚构值，不能直接应用。

| 源产物 | 审核后的安装位置 / 属性 |
| --- | --- |
| `go build ./cmd/persistty-file-helper` | `/usr/local/libexec/persistty-file-helper` root:root 0755 |
| `go build ./cmd/persistty-elevatord` | `/usr/local/libexec/persistty-elevatord` root:root 0755 |
| [policy](policy.example.json) | `/etc/persistty-elevation/policy.json` root:root 0644；目录 root:root 0755 |
| [sudoers](persistty-file-helper.sudoers.example) | `/etc/sudoers.d/persistty-file-helper` root:root 0440；安装前后 `visudo -cf` |
| [socket](../systemd/persistty-elevation.socket) / [service](../systemd/persistty-elevation.service) | `/etc/systemd/system/` root:root 0644；developer 必须替换 |
| nonce 账本目录（新建） | `/var/lib/persistty-elevation` root:开发组 0750；helper 记录 root:开发组 0640 |
| socket（systemd 管理） | `/run/persistty-elevation/broker.sock` 开发账户:组 0600；父目录 root-owned、不准非 root 写 |

所有策略/二进制/目标父目录祖先须 root-owned，不能 group/other writable 或 symlink；例如 `/tmp`、用户 home 和 Debian `/var/run` 链接不能作为策略目标祖先。目标只允许已有 regular nlink=1、无特殊权限位的文件；保留 UID/GID/模式、user xattr、POSIX access ACL、SELinux 标签。未知 security/trusted 属性或复制失败都拒绝，不静默丢失。父目录 default ACL 在 temp 上的结果也复验。根/父/leaf 变更或完整 Version 不匹配均拒绝。

策略 `protected_paths` 必须列出实际 Web 配置、数据库/WAL、可执行文件、前端产物、broker socket 等路径。内建拒绝本功能配置/二进制/账本、sudo/PAM/账户认证、systemd 管理配置和默认 Web 配置/二进制；自定义安装位置不能靠内建默认值自动保护。允许列表只列精确文件，不授予目录通配。

## 应用前后的门禁

1. 构建并保存 SHA256；由管理员安装两个 root-owned 二进制及示例替换后的策略，禁止服务用户覆盖。准备新的账本目录，不复用应用 SQLite 目录。
2. `visudo -cf <候选文件>`，审查 `sudo -l` 的有效规则；确保没有外部规则改变本命令的 PASSWD、closefrom、I/O 记录与认证次数。`-k` 每次忽略已有 timestamp；不缓存密码。
3. `systemd-analyze verify` 两个候选 unit，确认现有账户与 ReadWritePaths 精确目录。只启动新的 socket/service；不要重启 tmux。验证 broker 的 `/proc/<pid>/status` NoNewPrivs=0、Web=1；seccomp 等选项可能隐式强制 NNP，不能仅根据文件写 false 推断已生效。
4. `nginx -t` [Nginx 示例](../nginx/persistty.conf)，仅 W07 两类请求使用 49MiB 有界内存 buffer、关闭 request buffering/正文落文件/缓存。不得开启 debug/request body/Cookie 日志。检查已有 Nginx 前置代理也遵守；普通 HTTP 仅限已批准 VPN。正式整体部署仍属 W08。
5. 保持禁用，使用隔离目标完成 [Debian 专项](../../tests/integration/debian/w07/README.md)，再明确开启并验证普通权限失败入口。真实密码只在本机隐藏 TTY / 网页输入，不写 chat、argv、env 或报告。

helper 首次成功认证后才有权限消费 root nonce；Web 的持久 prepared→executing CAS 已在认证之前消费尝试，错误口令不能通过产品 API 重用。系统同 UID 已可访问 Web 进程与 SQLite，其直接调用 broker 的权限属于开发账户 OS 边界，不能把此功能称为 shell 沙箱。

## 禁用与回退

先将 Web 的 `elevation.enabled` 设 false 并只重启 Web。禁止新请求后核对在途 nonce 实际结果，再停 `persistty-elevation.socket` 和 `persistty-elevation.service`；只撤销本次安装的 sudoers/unit/helper/broker/策略，保留 nonce 账本和已写目标证据以避免回滚重放。不得删除应用 metadata.db 或迁移表以降级，不撤销已发布文件内容，不停止 tmux。不提供一键 root 安装/清理脚本，以便管理员审阅精确差异。
