# Debian 文件根隔离探针

独立实验，不导出产品 API。只创建自己的 `os.MkdirTemp` 目录，其下包含安全根、外部哨兵和目录交换攻击者。运行完关闭句柄并删除该目录；JSON 中 `cleanup_verified` 必须为 true。失败不会误记通过，进程退出码非零。

## 本地检查

在仓库根运行，工具链由根 `go.mod` 锁定：

```sh
go test -race ./tests/integration/debian/files
go vet ./tests/integration/debian/files
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/persistty-files-probe ./tests/integration/debian/files
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test -c -o /tmp/persistty-files-test ./tests/integration/debian/files
```

本机非 Linux 的通过不替代 Debian 结果。交叉编译测试二进制不代表运行了 Linux 测试。

## 授权远端运行

先取得目标环境授权，仅使用新建目录。连接参数唯一来源是仓库根已忽略的 `.env`，保留 `DEBIAN_USER`、`DEBIAN_IP`、`DEBIAN_PORT` 原键，不读取同名进程环境变量或默认值。文件必须是当前用户所有的普通非链接文件、权限 0600、最多 4 KiB；安全字面解析支持单/双引号、空白和注释，不执行 `source`、变量插值或命令。非法/重复/缺失字段在连接前失败。

```sh
python3 -B tests/integration/debian/run_remote.py files --probe /tmp/persistty-files-probe --test /tmp/persistty-files-test
```

先按本地检查构建二进制。runner 从 mktemp 获得并校验本次 ROOT，上传并运行探针/测试，finally 精确清理该目录；中间失败也尝试清理，清理失败不能记通过。每次传输最多 60 秒、stdout 内存上限 1 MiB、stderr 丢弃；成功仅输出脱敏 JSON。SSH/SCP 参数由结构化数组构造，IPv6 SCP 使用方括号；关闭 SSH 配置文件避免覆盖目标，只使用默认 key 与已知主机 key，不请求密码。不得归档 `.env`、展开的 argv、原始 stdout/stderr、HOME 或 UID；失败只输出固定类别。

正常探针使用 15 秒 context，竞争阶段最多 5 秒、2,000 次交换和 10,000 次文件操作；普通文件系统调用不受 context 强制打断，外层 timeout 才是进程卡死保护。JSON 为固定数量检查，不输出哨兵正文或业务内容。强制终止可能阻止 defer 清理且没有最终 JSON；因此设置 TMPDIR，将所有子目录限制在同一次 mktemp 返回的已知上传目录中。runner 的 finally 清理仅用于已校验、由本次实验创建的目录；禁止套用到既有目录或批量清除 `persistty-*`。本地 runner 本身被 SIGKILL 或主机退出时 finally 不保证执行，当前没有持久化的清理恢复账本；不宣称此类中断后零残留。

## 解释边界

- `os.Root` 限制其句柄根内路径解析，不替代 API 层对任意 `..` 段、编码、特殊文件的验证。
- 打开的根绑定目录身份。目录移出当前路径树后句柄仍可访问该目录，本探针明确记录这一行为；不能据此宣称目录外部移动后仍有当前树成员保证。
- 竞争实验只覆盖新建目录与外部链接交换，有限通过不是所有调度、挂载、特殊文件或并发保存的形式证明。
- Landlock 仅查询 ABI，不设置规则、不执行受限 `rg` 或 Git；无支持时只报告查询错误，绝不会退化为运行不受限 CLI。
- Git/rg 根隔离、文件版本冲突与原子提交、root helper、真实业务权限均未验收。
