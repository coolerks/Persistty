# D10 单文件提权可行性结论

2026-09-28 只读目标机检查确认 `sudo`、`sudoedit` 二进制与 `/etc/pam.d/sudo` 入口存在；未读取策略内容、未调用 sudo、未提交密码，也未安装 root-owned helper 或改变 sudoers/PAM/systemd。此前 `sudo -n` 需要密码的事实保留，不能推断普通 Web 请求能完成授权。

本地普通用户合成协议五项测试通过：nonce 单次消费且失败不可重放、请求不能转向别的文件、文件身份/内容版本变化拒绝、symlink 替换不写目标、取消与错误口令均拒绝。固定合成口令仅由匿名 pipe 读取，未放 argv/env/URL/日志。该模型不是特权进程，不证明 PAM 交互、root 权限或生产原子保存；直接 truncate 和 Python 内存清理明确不符合 W07 正式要求。

W07 必须单独审批 root-owned helper 安装和精确 sudoers/PAM 策略，并验证：认证与目标文件正文使用分离管道、密码不可进入命令参数/环境/审计正文；一次授权不复用 sudo timestamp（系统 timestamp 与应用 nonce 不同）；随机 nonce 的跨进程/重启持久防重放（本模拟只在进程内记忆）；helper 校验调用者、单文件身份/强版本、父目录与链接、普通文件类型、权限/ACL/SELinux、临时文件与原子提交、失败/崩溃/取消零意外覆盖；systemd 的 `NoNewPrivileges` 与 sudo setuid 路径是否冲突。未获授权前不得将网页弹窗保存称为已可用。

依据：[Debian sudo(8)](https://manpages.debian.org/trixie/sudo/sudo.8.en.html) 明确 `-S` 从 stdin 读密码且策略决定认证；[Debian sudoers(5)](https://manpages.debian.org/trixie/sudo/sudoers.5.en.html) 描述可缓存 timestamp；[Linux no_new_privs](https://docs.kernel.org/userspace-api/no_new_privs.html) 说明 setuid 受限。
