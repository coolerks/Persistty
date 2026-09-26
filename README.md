# Persistty

Persistty 是面向 Debian 开发机的单用户、自托管 Web IDE，计划提供持久化终端、文件浏览与编辑、查找替换、上传下载、只读 Git 查看和后端认证。

目前项目处于开发规范准备阶段，尚无可运行的产品实现。开发规范与任务由 Trellis 管理，入口为 [AGENTS.md](AGENTS.md) 和 [开发工作流](.trellis/workflow.md)。

核心要求：浏览器仅负责查看和控制终端，关闭页面、网络中断以及 Web 服务重启不得终止 tmux 中的任务。重新登录后应恢复终端列表并允许重新连接；只有明确确认“关闭终端”才由 Persistty 终止对应会话。该要求仍须通过后续 Terminal Lifecycle Spike 和真实 Debian/systemd 测试验证。

普通 tmux 进程无法跨主机重启保存运行中的内存状态，Persistty v0.1 不承诺主机重启后恢复正在执行的进程。程序自行退出、系统终止或管理员停止 tmux 服务也属于真实终止原因。
