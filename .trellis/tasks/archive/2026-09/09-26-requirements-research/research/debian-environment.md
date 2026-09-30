# W02 Debian 环境只读采集

日期：2026-09-27。用户在 W01 提交 4113358 后授权通过既有 SSH key 连接指定环境验证。连接信息只从仓库根已忽略 .env 的 DEBIAN_USER/DEBIAN_IP/DEBIAN_PORT 字段读取，不在报告记录实际值。当前仅采集，不将 SSH 连接许可视为 root 安装或现有服务变更许可。

## 实际观察
- BatchMode、StrictHostKeyChecking=yes、ConnectTimeout=10 连接成功，没有输入密码或改变 host key 策略。
- Debian 13.4 (trixie)，内核 6.12.74+deb13+1-amd64，systemd 257.9-1~deb13u1，cgroup v2。
- 当前 UID/GID <remote-uid>，HOME /home/<remote-user>，shell /bin/zsh，LANG zh_CN.UTF-8，字符集 UTF-8。
- 用户级 systemd 状态 running，Linger=yes；/run/user/<remote-uid> 权限 0700 且归 UID/GID <remote-uid>。未开启或修改 linger。
- Git 2.47.3、ripgrep 14.1.1；tmux 不在 PATH，dpkg-query 确认 not-installed，不能运行持久终端实验。
- Nginx 包 1.26.3-3+deb13u2 已安装，但不在当前用户 PATH；不能把 command -v 的缺失推断为包未安装。
- 绝对路径 /usr/sbin/nginx -v 确认运行版本 1.26.3；apt-cache policy tmux 候选 3.5a-3。
- 内存约 15 GiB、采集时 available 约 11 GiB；根文件系统空闲约 304 GiB。HOME 位于 ext 系列文件系统，/tmp 为 tmpfs。这些是当时观测，不是性能验收。

## 操作边界
未运行 sudo、apt install/update，未修改远端文件、用户配置、systemd/Nginx/WireGuard/防火墙，未读取私钥、凭据或业务配置，未查询或销毁已有终端会话。没有 W02 实验通过结论。

后续需要独立临时目录、tmux socket、用户级 transient unit 来验证 Web/SSH 断开与生命周期；只能清理实验自己创建的资源。首先需获准安装缺失的 tmux。Nginx、系统级服务、提权 helper 和公网隔离实验另行限定范围，不随 tmux 安装自动授权。

## 后续授权与执行
用户随后明确允许 tmux 安装及上述隔离实验，见 [W02 子任务](../../09-27-debian-spike/prd.md)。sudo -n apt-get install 未执行安装，因需要密码退出；不索取用户密码，改为尝试临时解包软件源官方 tmux 包并以普通 UID 运行。各实验结果及资源清理在子任务独立记录，前文只读采集阶段不代表其已通过。
