# 远端验证与连接信息保密

## 1. 范围与触发条件
适用于 SSH/SCP 环境采集、部署和 Debian 集成实验。连接身份和地址属于用户私有运维信息，不能因为用户在会话中授权连接，就写入代码、文档、fixture、任务记录或可提交的实验结果。

## 2. 签名
远端连接只读取仓库根目录已被 Git 忽略的 `.env`，字段为 `DEBIAN_USER`、`DEBIAN_IP`、`DEBIAN_PORT`。不读取进程环境中的同名变量，不允许其覆盖文件，也没有默认值或可提交配置 fallback。键名可以入库，实际值不入库。该约定只用于实验/运维入口，不改变产品 YAML 配置契约。
复现入口为 `python3 -B tests/integration/debian/run_remote.py terminal`，或 `files --probe <Linux-probe> --test <Linux-test>`；读取由 [remote_config.py](../../../tests/integration/debian/remote_config.py) 拥有，runner 不输出连接参数。入口执行仍需目标机操作授权，不因有 .env 自动授权连接。

## 3. 契约
文件须为当前用户拥有的普通非符号链接文件、权限恰为 0600，打开使用 O_NOFOLLOW/O_NONBLOCK 并对同一 fd 执行 fstat，读取上限 4 KiB。格式为 UTF-8 的 `KEY=VALUE`，支持空行/注释、等号附近空白和单/双引号包裹的值；只接受三个指定键，不支持 export 或反斜杠转义。不执行 shell、不做变量插值、命令替换或反引号展开，禁止 source/eval。值中美元符号/反引号、重复/未知键、畸形语法、非法 UTF-8/控制字符均拒绝。保留用户原值，格式已兼容时不为格式化重写私有文件。

三个字段均必填。USER 采用 `[A-Za-z_][A-Za-z0-9_-]{0,31}`；IP 为合法 IPv4/IPv6 地址，不接受 `%zone`；PORT 为 1..5 位 ASCII 十进制数字且数值 1..65535。执行入口应在任何 SSH/SCP 前统一检查，错误只显示固定类别或字段名，不显示值、原行或私有路径。SSH 使用 BatchMode、StrictHostKeyChecking=yes、ConnectTimeout；SCP 使用对应的大写 -P，IPv6 目标按工具要求括号化。使用参数列表或正确引用的独立参数，禁止拼接连接值为 shell 代码。

禁止 `env`/`printenv` 全量输出、`set -x`、完整 argv/原始连接错误回显，以及将展开后的命令写入报告。工具错误可包含真实目标，入库前只保留脱敏类别/退出状态。报告只保存环境能力、版本、实验断言与必要的资源关联；HOME、cgroup 中可识别的用户名和 UID 使用明确的脱敏标记。脱敏 JSON 必须注明不是原始逐字证据，不编造或更改实验结果。
已知连接端口字段 port/DEBIAN_PORT/ssh_port/remote_port 的字符串和数字值都需去敏，不全局替换同值 PID/start time/心跳。该结构化去敏不覆盖任意未知原始正文，未知 stdout/stderr 禁止直接归档。传输入口固定错误类别，不打印配置 repr 或失败 argv。

runner 的 finally 只覆盖正常异常；自身 SIGKILL 或本机退出无法保证清理，目前没有持久恢复账本。不得因此宣称零残留；后续远端复现须核实自有资源清理并记录不足，不能用通配清理补偿。

私有 `.env` 不输出全文、不复制入任务记录、不提交；可提交 `.env.example` 只包含格式说明与虚构值，不能充当连接 fallback。禁止把历史会话内容复制为另一份带真实连接信息的文档。

## 4. 验证与错误矩阵
| 条件 | 行为 |
| --- | --- |
| .env 不存在、非 owner/权限宽松/symlink/过大 | 读取或连接前失败，不输出原内容 |
| 重复键、畸形/非法 UTF-8 | 解析失败，不执行其中内容 |
| 任一字段缺失 | 连接前失败，仅报告字段名 |
| 账户/IP/端口非法 | 连接前失败，无远端副作用 |
| 未知或变更 host key | 拒绝连接，不关闭主机校验 |
| SSH/SCP 失败 | 脱敏错误分类，不落盘原始 stderr/argv |
| 证据含连接值或可识别 HOME/cgroup 身份 | 去敏后复核，不纳入提交 |

## 5. 正常、基础、错误用例
正常：解析私有 .env 的三个字段构造连接，报告只说“已授权目标 Debian”。基础：缺少 PORT 时报告其缺失并停止；环境中存在 PORT 也不能补齐。错误：source .env 执行命令替换，或把用户给出的 SSH 命令原样写入 README/fixture。

## 6. 所需测试
本地用虚构 .env 验证注释/引用/缺失/重复/非法/前导选项/控制字符/端口边界、非普通文件和权限，以及命令替换不会执行；用 mock 保证错误输入不会调用 SSH/SCP，并验证环境变量不能补齐/覆盖。私密扫描仅在内存解析 .env，输出路径、字段名、命中数量/结果，禁止输出匹配行。检查脱敏证据有效 JSON、内部关联一致及明确去敏声明。不会为了去敏重新连接目标机或把真实连接信息固化到测试。

## 7. 错误与正确示例
错误：可提交文档存放真实 `user@host`、端口、展开的 HOME 或命令记录。正确：从已忽略的私有 .env 解析并验证字段，用局部值独立传参；格式示例只用虚构值：

```dotenv
DEBIAN_USER=probe_user
DEBIAN_IP=203.0.113.10
DEBIAN_PORT=2222
```

片段不是实际目标配置，也不能自动用于连接。固定远端命令不能包含未经校验的输入，禁止写入展开后的命令或启用执行跟踪。读取入口及复现方式见 [终端探针](../../../tests/integration/debian/terminal/README.md) 和 [文件探针](../../../tests/integration/debian/files/README.md)。
