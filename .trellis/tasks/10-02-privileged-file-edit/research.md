# W07 证据与工程选择（2026-10-02）

## 仓库事实

| 证据 | 结论 |
| --- | --- |
| `internal/files/save.go:54`、`:89`、`:134`、`:146` | 普通保存限制8MiB、完整Version比较、同目录临时文件和原子rename；只保留普通权限，不包含高权限UID/GID/ACL/xattr契约 |
| `internal/files/secure_linux.go` | 注册根dev/inode与openat2 BENEATH/NO_XDEV，修改父目录拒绝所有符号链接；helper应复用该owner，不能EvalSymlinks后绝对路径WriteFile |
| `internal/storage/folder_access.go:42`、`internal/storage/projects_write.go` | 配置变更和文件发布用projectMu裁决；不能把密码输入/认证等待塞进该锁 |
| `internal/auth/service.go:107`、`internal/storage/storage.go:281` | 入口session查验不等于提交时查验，DeleteSession目前不参与发布锁；W07需新增狭窄的会话撤销/发布裁决并验证锁顺序 |
| `internal/httpapi/router.go` error映射、`internal/httpapi/files.go` | EACCES/EPERM/EXDEV/ELOOP都映射forbidden；W07入口必须识别普通保存的真实权限失败，不能所有403都提供提权 |
| `web/src/features/workspaces/editor-session.ts:70` | 一次写入capture epoch/content/generation；失败suspended，成功清本view提交之前草稿，新输入不丢；提权在同一owner扩展 |
| `web/src/features/workspaces/FileEditor.tsx` | 已有失败/比较/导出UI，Dialog默认焦点能力已有，无需独立基础弹窗 |
| `deploy/systemd/persistty.service:17`、`:19` | NoNewPrivileges=true、ProtectSystem=strict；直接从Web exec sudo无法获取root，不能静默移除限制 |
| [D10](../archive/2026-09/09-27-debian-spike/helper-report.md) | 仅非特权模拟；直接truncate、内存nonce不能复用为正式产品 |
| [原始决定](../archive/2026-09/09-26-requirements-research/research/requirements-source.md) U39/U40/U41/U42/U46 | 可读文件权限失败后的单文件保存、root-owned允许范围、网页密码、一次保存、VPN内HTTP已确定 |

## 官方机制核对

- [Debian sudo(8)](https://manpages.debian.org/trixie/sudo/sudo.8.en.html)：-S从stdin读取密码；带command的-k忽略且不更新凭据缓存；-C需要管理员closefrom_override才能保留额外FD；sudo退出1不能单凭退出码区分密码错误与配置失败。参数不得包含目标正文或密码。
- [Debian sudoers(5)](https://manpages.debian.org/trixie/sudo/sudoers.5.en.html)：命令级PASSWD、timestamp_timeout、passwd_tries、env_reset、NOSETENV、I/O日志选项需精确审查；输入日志可能保存管道数据，不能仅验证应用logger。关闭日志仅针对helper命令，不修改全局审计。
- [Linux no_new_privs](https://docs.kernel.org/userspace-api/no_new_privs.html)：标记继承且不可取消，setuid exec不能获得新权限。因此选与Web cgroup独立的非rootbroker；broker可以exec sudo，Web现有NNP保留。
- [shadcn Dialog](https://ui.shadcn.com/docs/components/base/dialog)：授权弹窗复用现有Base UI Dialog组合。项目已有input/field/button/alert，无基础控件缺口。

上述文档核对是机制证据，不是目标机版本、有效sudoers/PAM或安装验收。真实Debian配置必须验证visudo、实际无TTY密码行为、FD保留、失败/取消和安全元数据保持。

## 选择与最小变更

直接Web→sudo会要求放宽Web unit限制；直接PAM需要新增认证原生绑定及独立root服务；复用sudoedit不能绑定本项目nonce/强版本及一次内容发布。选Web→非rootbroker→固定roothelper：增加一个有限职责的进程入口换取Web限制保持，并复用现有系统sudo认证。broker不是AI worker，不承担任何任务分派，仅产品的操作系统权限边界。

授权等待不占projectMu；helper准备好提交后由Web在短发布裁决中确认会话/关联，发送commit并等待有界结果。root防重放账本和Web请求状态各自持久化，正文和密码均不进SQLite/账本。发生不可确定发布结果时查询/复读，不根据网络失败再写。
