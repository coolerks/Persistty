# W02 隔离实验设计

## 数据与职责
探针只消费明确实验参数；输出脱敏 JSON/版本/PID/start time/cgroup/判定，不采集 shell 输入、环境凭据或业务数据。临时目录 mktemp 0700，随机私有 socket 与 unit 名称；资源标识在日志中留证。main 负责安装条件及授权、规范；终端探针和文件探针分别拥有独立子目录，不并行修改共享产品文件。
最新连接约定：仅安全解析仓库根已忽略的私有 .env，键为 DEBIAN_USER/DEBIAN_IP/DEBIAN_PORT，不读进程环境覆盖或 fallback，不 source/eval。实际值、展开后的 SSH/SCP 命令与原始错误不落盘；证据关联身份去敏并明确标注，保留原实验判定。遵守 [远端验证规范](../../spec/backend/remote-validation.md)。本批不提交。

## 生命周期
以用户级 systemd-run 分别启动 tmux foreground server 与模拟 Web attach。显式预启动 server，空 server 存活参数按实际版本验证；Web 只通过 -N 连接既有 server。停止 Web 使用正常 control-group 清理，确认 pane 与 server cgroup 均不属于 Web。禁止通过 nohup/setsid 代替 cgroup 证据。跨 SSH 分次采样证明连接生命周期分离；只关闭自身实验单位。

## 文件根
使用已锁定 Go 1.26.8 的 os.Root 在目标 Linux 运行测试二进制，不安装远端 Go。探针安全边界限制其新建临时目录，并单独建立外部 sentinel。读取、创建、rename、remove 与竞争交换按真实 API 检验；外部 sentinel 不得出现在安全读取结果，不得被修改。该结果不替代业务版本/原子保存/CLI Landlock 验收。

## 回滚与限制
所有探针有超时、就绪轮询、输出界限与 cleanup，只清理新建资源；失败保留必要脱敏证据并报告残留。提权 helper/生产配置不触碰。sudo 密码不可用时只临时解包官方包；缺少 libevent_core 时仅在同一临时目录下载/解包 tmux 必要 Debian 依赖，LD_LIBRARY_PATH 只作用于自身探针，不修改全局环境或提权安装额外依赖。当前仅为 W02 部分实验，完整 W02 门禁仍保留。
