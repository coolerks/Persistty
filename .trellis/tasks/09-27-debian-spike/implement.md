# W02 执行与验证

1. 主会话已完成只读环境采集；批准来源为用户提供 SSH 与隔离范围之后的明确允许。整理上下文并激活子任务。
2. 终端 implement 负责 tests/integration/debian/terminal/** 和 terminal-report.md；自行核对官方 tmux/systemd 参数，远端只执行批准隔离实验。
3. 文件 implement 负责 tests/integration/debian/files/** 和 files-report.md；交叉构建 Linux 独立探针，在新建目录验证真实 os.Root，不修改产品包。
4. 实施代理不得撤回他人修改；不改 Go manifest/lock 或其他 owner 文件。main 负责文档/规范、整体审查与提交方案。
5. check 代理复核隔离与清理、报告判定、测试可重复性，运行 Go test/vet/race（涉及并发），不得为了复核触碰用户系统服务或扩大授权。
6. main 同步已证实知识及未执行门禁，提供结果/残留清单。提交仍需用户确认，不归档父任务或其他活动任务。

## 已提交隐私修正
用户先暂停提交并要求剥离真实连接信息，修正后批准提交，第一批及隐私修正已在 5f392ed。后续复现仅解析仓库根私有 .env 的三个字段；main 维护规范及文档去敏，不修改用户实际值、不回显全文。检查 tracked/untracked 可提交文本与 HEAD 的敏感匹配只输出路径/count。

## 当前 D06 继续阶段
1. Go implement 只负责 tests/integration/debian/bridgego/** 与 go-bridge-report.md：隔离模块依赖/真实 Go-PTY-WS 服务与测试、Linux binary；先向 runner owner 提供精确 CLI/就绪协议。不改根 module 或其他 owner 文件。
2. runner implement 负责 tests/integration/debian/bridge/**、run_remote.py 必要分支及关联 Python 回归、bridge-runner-report.md：私密连接入口、隔离用户单位、start/check/cleanup、真实远端执行；不修改 bridgego 代码或产品。共享协议先协商再接入。
3. check 在两 owner 定稿后做整体 review/自修，覆盖认证/Origin/二进制限额/取消/原进程存活/清理与去敏，并独立重跑。主会话维护规范/任务总结/提交方案；本轮不改生产配置、不安装 root helper。

## D07 恢复机制继续阶段
1. 单一 history implement owner 负责 tests/integration/debian/history/**、bridgego 有界记录 client 的最小扩展及 run_remote.py 必要 history 入口、history-report.md。保留 D06 既有行为与证据，不改产品或主会话规范；先确认锁定 xterm 库的实际官方包/API/许可证，再做解析。取证只涉及固定合成数据及 curses 程序，不记录密码、业务文本或连接字段。
2. 最小门禁覆盖普通历史与 raw attach 的区别、capture+attach 的外层缓冲区/持续输出窗口限制、真实 curses 进入/退出/resize/reconnect 和原 pane 身份、分块解析等价性及精确清理。未执行/能力不足必须明确报告，负面实验可证明方案限制而非自动修补为产品方案。
3. 实施定稿后 history check 独立复核并重跑必要真实实验/根与隔离模块门禁，主会话同步知识。所有资源有界且仅自身清理；本轮仍不提交、推送、归档。

## D08 同步截点继续阶段
D06/D07 已提交 965b4c6，用户要求继续开发。snapshot implement 单独负责 tests/integration/debian/snapshot/**、run_remote.py 最小安全 snapshot 入口及必要回归、snapshot-report.md；不改已有证据、产品依赖或 bridge 协议。先核实公共 API/实际包兼容，然后做固定完整边界正面对照及 pending 截点负面回归、epoch/seq/ring 边界、新一次真实固定 WS 记录的离线恢复对照。实施完成后独立 snapshot check，自修仅本实验范围；main 更新规范及后续提交方案，不自动将新改动并入 965b4c6。

## D09/D10 单代理收尾阶段
用户最新要求推进到 W02 完成；项目已禁用子代理，后续研究、实现、检查均由主会话直接完成。先实施 D09 隔离 Landlock CLI 探针及真实 Debian 复核，再做 D10 不提权 helper 可行性报告/非特权协议测试。两阶段均不触碰生产服务或安装 root helper；发现需要新权限时停止对应操作并明确请求授权。全部门禁、去敏、精确清理与规范同步后，只在 D01..D10 任务验收真实完成时归档 W02；生产 W03/W04/W07 的待验收内容保留在后续任务，不将实验成功冒充产品完成。未经新批准不提交或推送。
