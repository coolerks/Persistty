# D09 Landlock CLI 探针

仅在已授权 Debian 环境与本次 `mktemp` 的 0700 目录运行。连接从仓库根私有 `.env` 读取，不记录连接字段。Linux 二进制在本机交叉构建：

```sh
GOCACHE=/tmp/persistty-go-cache GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/cli-probe ./tests/integration/debian/cli
python3 -B tests/integration/debian/run_remote.py cli --binary /tmp/cli-probe
```

探针只写合成 `workspace` 和 `outside` 哨兵；运行固定 `cat`、`rg`、Git `log`，测试 Unicode/前导短横线文件及阻塞 FIFO 的进程取消。Landlock 无法创建/应用时固定失败，不回退裸执行。`rg` 对故意不可访问的 symlink 报 2，只有内部标记命中、外部标记未出现才算通过。runner 精确清理本次临时根。

实验存在明确限制：动态链接器与工具的系统文件许可不构成任意路径只读沙箱；`/usr` 等系统树在该探针中可读，`/dev/null` 允许写入。有限合成 symlink/目录替换不能证明所有竞态，标准 fd 与已打开 fd 仍需审查。正式 W04 需收紧系统依赖白名单，测试链接指向每个白名单、恶意 Git 配置与外部 helper、特殊文件、持续竞态、命令取消及流量上限；不能直接复制本探针为产品实现。
