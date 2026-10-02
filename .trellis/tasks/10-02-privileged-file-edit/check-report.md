# W07 实施与质量检查

日期：2026-10-02。用户已批准“开始实施”，单代理 inline；W05/W06 统一实机验收按最新要求延期。本报告区分代码、本机非特权验证和尚未执行的真实系统验收，任务保持 in_progress，不提交或归档。

## 已接入的行为

- 普通保存真实权限失败返回 permission_denied，前端显式申请；默认未配置/非 Linux 不可用，普通文件与终端协议保持。
- 新增 prepare/execute/status/cancel 受保护 API、0005 持久请求 CAS、原会话绑定、限流/配额和重启未知状态；密码/正文不落 SQLite 或 nonce 元数据。
- 独立非 root broker 与固定无参数 root helper，固定 sudo argv、认证/控制/正文分离 FD、私有 socket/peer、在途限制、取消与进程回收；Web 保持 NNP。
- Linux 阶段保存、旧文件/元数据/temp 强版本、同目录原子提交、UID/GID/模式/ACL/xattr/标签保持和 nonce/target锁；未知状态不重放。源码/编译不是真实 root 能力证据。
- 同 FileBuffer 冻结正文/generation，确认弹窗默认取消、可查看快照、清理密码字段；成功只推进捕获 base，新输入/草稿/模型/undo 保留，未知只查询。
- [安装审查稿](../../../deploy/elevation/README.md)、sudoers/策略/socket-service/Nginx 示例，以及 [Debian TTY 探针](../../../tests/integration/debian/w07/README.md)可审查，均未安装/应用/连接目标机。

## 实际检查记录

| 检查 | 本轮结果与范围 |
| --- | --- |
| `go test ./...` | 通过，包含最终容量拒绝/取消和默认禁用测试；不代表 Linux-only 测试已执行 |
| `go vet ./...` | 最终复核通过 |
| `go test -race ./...` | 最终全范围通过，包含会话/发布裁决、Unix/进程与HTTP全部回归 |
| `npm --prefix web run lint` | 最终复核通过 |
| `npm --prefix web run typecheck` | 最终复核通过 |
| `npm --prefix web run test -- --maxWorkers=1` | 30 文件、159 测试通过；与构建/浏览器并行时旧用例超时，串行复核通过，无抑制或产品回退 |
| `npm --prefix web run build` | 通过；字体摘要/资源与37项许可证检查通过；既有 Monaco 大 chunk 警告保留 |
| Chromium | 最终 W07 7项通过；编辑恢复6项另轮通过。真实 model/undo/IDB/输入控件；合成API，不是真实提权或物理手机 |
| WebKit | 最终 W07 7项通过；默认取消/Escape、桌面model/undo、手机textarea、未知只查询、不可用/过期/冲突；截图已检查 |
| Linux `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go vet ./...` | 通过，覆盖 Linux-only helper/账本/文件属性源码 |
| Linux amd64/CGO=0 | 最终 helper、broker、TTY探针及 files 测试二进制交叉编译通过；不运行root/远端 |
| gofmt / Markdown链接 / JSON / diff / 忽略规则 | 31个Go文件、169个本地链接、5个JSON和两份7章owner规范通过；diff与运行数据忽略规则通过 |

最终 W07 浏览器输出在 `/private/tmp/persistty-w07-chromium-complete` 和 `/private/tmp/persistty-w07-webkit-complete`，编辑恢复6项通过输出在 `/private/tmp/persistty-w07-chromium-verified`（该轮旧 textarea 定位导致 W07 六项失败，不能把整轮记通过）。构建产物在 `/private/tmp/persistty-w07-build`。专用 loopback5179 Vite 已停止，不碰用户既有服务或tmux。没有 agent-browser CLI，浏览器验证使用项目锁定的 Playwright 与已有引擎。

## 最终 Linux 构建摘要

`GOOS=linux GOARCH=amd64 CGO_ENABLED=0`；SHA256 仅对应本机未安装产物，后续上传/安装须另行核对。

| 产物 | SHA256 |
| --- | --- |
| `persistty-elevatord` | `fccada0e44eca16cf5bfa02cd56a9d1ce69e83f0605e181f66909f253bf432e5` |
| `persistty-file-helper` | `224f438e5a54c360d0440a8ac87bd808cdea1dcc3b2141020e487fc7169962da` |
| `probe` | `a96f3c5dce7472d5b0138f2e9b767b085070205fc1464bc24573b38ffe83693a` |
| `files.test` | `88a371b0e34d8759e6bebc6b1543b25f0ec681ff57f26e94fa486af95a52fa52` |

## 本轮修复与断言

1. 同请求重复 execute 不覆盖 active attempt；prepared CAS 前注册一次，取消/双击/两槽容量拒绝的时序有测试。认证等待不占 projectMu；注销、项目删除或取消先完成则零发布。
2. pipe writer 必须关闭并 join 后才返回，防止调用者清理 credential/content 与写入并发；短命子进程、超时/提前退出/取消、独立FD、argv/env 无合成秘密均有测试。broker shutdown等待所有handler。
3. root 策略拒绝自修改/认证/动态库/系统管理目录；提交前复查最新策略和可信父链。temp 正文及安全元数据也在提交前复验；privileged hash 有8MiB+1限额与context，不能继承普通20GiB元数据读取限额。
4. xattr 两次读取之间由空变为非空须有长度 guard，失败闭合，不越界panic；未知属性拒绝。Linux-only 执行尚待原生环境。
5. 未知状态查询不能把持久 consumed 请求恢复 prepared/executing；前端查询单次在途；迟到配置响应不得推进base或清草稿。HTTP401/Origin/CSRF/跨session404/重复JSON/日志合成秘密/系统失败不登出有断言。
6. Dialog 统一取消/关闭入口；确认时清字段，期间新输入 paused；用可见UI和实际键盘验证。Monaco 在 Chromium 使用 EditContext、WebKit 使用 textarea，测试通过可访问 textbox 定位，不能依赖单引擎内部 textarea。启动/重载时机与错误缓存路径失败均已修正后串行复验，未绕过产品行为或清undo。

## 未执行，不能写通过

- Linux 原生 `Privileged` 文件专项及真实 root 的 UID/GID/ACL/SELinux、unknown attrs、nonce/时钟/崩溃与原子故障验证。本机 Docker CLI 存在但本地 daemon socket 不存在，Linux 专项尚未执行。
- 目标机真实 sudo/PAM正确/错误口令、timestamp、无TTY、有效sudoers及FD保留，实际root子进程取消。
- 独立 systemd socket/service 的权限、NNP/cgroup/启动停止和精确 ReadWritePaths；Web NNP 保持、tmux零影响。
- Nginx `nginx -t`、最大转义body、敏感正文无临时文件/日志/缓存；具体VPN/TLS边界。

下一步先确认精确 Debian 目标、现有 UID/GID、隔离文件及候选安装差异，再申请目标机操作授权。当前仅授权代码实施，不擅自安装 root helper、改 sudoers/PAM/systemd 或复用正式文件。W05/W06 原待验收项保持 deferred，未补齐项不会被本轮自动化回归替代。
