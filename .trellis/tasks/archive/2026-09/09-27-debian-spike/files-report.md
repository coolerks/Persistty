# W02 Debian 文件根实验报告

## 结论与范围

2026-09-27 在已授权的 Debian 环境运行独立 Go 1.26.8 Linux/amd64 探针及测试二进制，退出码均为 0。连接从仓库根已忽略 .env 的 DEBIAN_USER/DEBIAN_IP/DEBIAN_PORT 字段读取，不记录实际值。无需远端安装 Go，无 sudo、无系统配置修改、无业务数据访问。只拥有 `tests/integration/debian/files/**` 与本报告，未修改产品 API、manifest 或其他代理文件。

## 实测证据

| 验证项 | 结果 |
| --- | --- |
| 临时目录权限 | `0700` |
| 根内创建、读取、rename、remove | 通过，内容校验一致 |
| 根内相对 symlink 读取 | 通过 |
| `../`、绝对路径、根外 symlink 读取及写入 | 全部拒绝，实际错误 `path escapes from parent` |
| 根外父目录的新文件创建 | 拒绝，外部未新增文件 |
| rename 的外部源/外部目标 | 分别拒绝，源哨兵保持 |
| 根外父目录下删除 | 拒绝，删除目标保持 |
| 删除根外 symlink 条目 | 成功，仅删除链接，目标内容及条目数不变 |
| 有限父目录交换 | 2,000 次交换，10,000 次文件操作；3,309 次允许、6,691 次拒绝；未读取外部哨兵、外部 3 个条目正文及数量不变 |
| 打开目录移动后的句柄身份 | 句柄继续访问同一被移出原树的目录；这是限制证据，不是当前树成员保证 |
| Landlock 只读 ABI 查询 | ABI 6；没有创建 ruleset 或运行 sandbox CLI |
| 资源清理 | 探针 JSON `cleanup_verified:true`，独立上传目录已精确删除并确认不存在 |

最终 standalone 探针目录 `/tmp/persistty-files-567435381`，首次探针 `/tmp/persistty-files-3111982662`，上传目录 `/tmp/persistty-files-upload.BXt5lk7N`。仅清理这些自身创建的路径，无残留实验服务、socket 或进程。测试用临时目录也由同一 probe 的 cleanup 断言核实。报告不存储业务文件内容。

## 检查与复现

- `gofmt -w tests/integration/debian/files` 已执行。
- 本机 `go test -race ./tests/integration/debian/files`：2 项测试通过（macOS，约 2 秒），Go race 未报告内存数据竞争；这不证明文件系统竞态不存在。
- 本机 `go vet ./tests/integration/debian/files`：通过。默认 Go cache 的 sandbox 权限报错后，改用独立 `/private/tmp/persistty-files-gocache` 成功；未改系统或仓库配置。
- `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build` 和 `go test -c`：通过。
- 真实 Debian `timeout 25s .../persistty-files-test -test.v -test.timeout=20s`：2 项测试通过，RootProbe 约 0.24 秒。
- 独立 JSON 探针有 25 项检查，全部通过；实际参数与复现、失败清理流程见 [探针说明](../../../../../tests/integration/debian/files/README.md)。

全部本地编译/测试会话已结束；远端探针未启动持久服务，SSH 退出后无探针资源。局部检查不能代替主会话整体 Go/frontend 门禁。

## API 依据与限制

实际锁定工具链的 `go doc os.Root` 记录根内解析、并发调用、根移动后绑定身份，以及不限制挂载边界、Linux `/proc` 魔术文件和设备文件等行为。网页的最新版本不用于推断 Go 1.26.8 能力；具体签名以本地工具链与真实编译为准。参考 [Go 官方 os.Root 文档](https://pkg.go.dev/os#Root)。

ABI 查询采用 `landlock_create_ruleset(NULL, 0, LANDLOCK_CREATE_RULESET_VERSION)`，参照 [Linux 官方 Landlock 文档](https://docs.kernel.org/userspace-api/landlock.html)。ABI 可用不证明受限 launcher/exec、授权路径最小化、动态库依赖及 `rg`/Git 行为已验证。

本次没有覆盖目录移出原树后的所有父句柄 TOCTOU、安全命令执行、原子版本保存、特殊文件阻塞、用户真实目录权限或提权 helper。W04 文件 owner 必须明确目录身份与当前路径树的差别；根 API 通过不解除 CLI 与保存验收门禁。
