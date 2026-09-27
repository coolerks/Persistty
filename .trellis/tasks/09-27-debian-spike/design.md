# W02 隔离实验设计

## 数据与职责
探针只消费明确实验参数；输出脱敏 JSON/版本/PID/start time/cgroup/判定，不采集 shell 输入、环境凭据或业务数据。临时目录 mktemp 0700，随机私有 socket 与 unit 名称；资源标识在日志中留证。main 负责安装条件及授权、规范；终端探针和文件探针分别拥有独立子目录，不并行修改共享产品文件。
最新连接约定：仅安全解析仓库根已忽略的私有 .env，键为 DEBIAN_USER/DEBIAN_IP/DEBIAN_PORT，不读进程环境覆盖或 fallback，不 source/eval。实际值、展开后的 SSH/SCP 命令与原始错误不落盘；证据关联身份去敏并明确标注，保留原实验判定。遵守 [远端验证规范](../../spec/backend/remote-validation.md)。第一批已提交 5f392ed，当前继续阶段另行验收并请求提交。

## 生命周期
以用户级 systemd-run 分别启动 tmux foreground server 与模拟 Web attach。显式预启动 server，空 server 存活参数按实际版本验证；Web 只通过 -N 连接既有 server。停止 Web 使用正常 control-group 清理，确认 pane 与 server cgroup 均不属于 Web。禁止通过 nohup/setsid 代替 cgroup 证据。跨 SSH 分次采样证明连接生命周期分离；只关闭自身实验单位。

## 文件根
使用已锁定 Go 1.26.8 的 os.Root 在目标 Linux 运行测试二进制，不安装远端 Go。探针安全边界限制其新建临时目录，并单独建立外部 sentinel。读取、创建、rename、remove 与竞争交换按真实 API 检验；外部 sentinel 不得出现在安全读取结果，不得被修改。该结果不替代业务版本/原子保存/CLI Landlock 验收。

## Go/PTY/WS 基础链路继续阶段
在 tests/integration/debian/bridgego 建立独立 Go module，固定既定 PTY/WS 库版本，不将实验路由混入产品 Gin router。服务仅监听 loopback 动态端口；实验 token 为临时文件 0600，不进入 argv/URL/日志/证据。连接检查 token 与固定 loopback Origin，服务只连接已预启动私有 tmux socket，不隐式启动 server，不自动重跑任务。bridge context 取消只回收 attach/PTY/连接；测试必须能观察其释放与原 pane 独立。

runner 负责 tests/integration/debian/bridge 的隔离编排，复用私有 .env 读取与脱敏传输；实际 Go binary 在同一临时目录中的独立用户 unit 启动，tmux 与 pane 不归 Web cgroup。重启前后采样 PID/start time/cgroup 和逐次心跳，检查零任务重建/零自动输入重放。二进制输出/输入与队列、deadline、FD/goroutine 清理分别测试。当前实验最多单活动 attach，不能将其提升为产品多观察端最终架构。

## D07 历史恢复机制与 TUI
D07 沿用 D06 安全 transport 与资源隔离，不覆盖 D06 两份既有证据。仅为固定合成实验输出新增有界记录 client 与新 history 目录；普通用户临时解包依赖，不安装远端 Node/Go。实际 curses 程序与固定阶段文件驱动输出，不向 shell 自动重放命令。Node 解析工具及 xterm 库锁在独立实验 manifest，不能改产品 web 依赖。原始字节仅暂存在已忽略私有文件，提交摘要/hash/断言；原始捕获不是通用用户数据收集入口。

优先验证 raw attach 与 capture+attach 反例；本地 serialize 对照如果进行，只证明确定性固定截点，不声称真实 tmux 原子快照。库解析等待 write callback，分块使用 Uint8Array；不自制 ANSI parser，不正则滤 smcup/rmcup，不利用文本前缀猜测去重。浏览器原生 WS 无法设置当前实验 Authorization header，本轮不改认证/Origin，不使用 URL token 或无鉴权代理；解析通过不记为浏览器渲染/实时端到端通过。研究依据见 [D07 方案](research/history-tui-plan.md)。

## 回滚与限制（所有阶段）
所有探针有超时、就绪轮询、输出界限与 cleanup，只清理新建资源；失败保留必要脱敏证据并报告残留。提权 helper/生产配置不触碰。sudo 密码不可用时只临时解包官方包；缺少 libevent_core 时仅在同一临时目录下载/解包 tmux 必要 Debian 依赖，LD_LIBRARY_PATH 只作用于自身探针，不修改全局环境或提权安装额外依赖。当前仅为 W02 部分实验，完整 W02 门禁仍保留。
