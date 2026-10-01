# 本机开发启动契约

## 1. 范围与触发条件

`scripts/dev.sh` 是 Mac/Linux 开发便利入口，不修改产品 API、不安装依赖、不接管现有前后端或正式 systemd 服务。修改此入口时须保持 [W03 生命周期](terminal-runtime-contract.md)。

## 2. 签名

`./scripts/dev.sh start|stop|restart [--config <path>] [--listen 127.0.0.1:<port>] [--port <port>] [--tmux <absolute executable>] [--shell <absolute executable>]`。`stop` 不接受选项。

Bash wrapper 定位仓库根，构建并 exec `scripts/dev` Go 启动器；`stop` 复用已有启动器，不依赖重新构建成功。默认配置 `.cache/dev/config.yaml`；Go 复用 `config.Load/Validate` 与锁定 YAML 库，不用 grep/sed 解析配置。无数据库迁移或新依赖。

## 3. 契约

- 仅 `development`，后端仅明确 127.0.0.1；前端源设为 `http://127.0.0.1:<port>`，端口默认原配置的 public_origin。Vite 代理环境键为 `PERSISTTY_DEV_API_TARGET`，覆盖旧同名环境值，不修改 Cookie/认证规则。
- 明确 tmux 路径优先；只有旧默认 `/usr/bin/tmux` 不存在时从 PATH 查找，不静默替换自定义无效路径。shell 优先级为显式 `--shell` → 显式 `terminal.shell` → config.Load 检测服务 UID 的登录 shell → `/bin/sh`；不能只用继承的 SHELL 环境决定实际程序。已有 pane 不重启。依赖缺失报错，不自行安装。
- 原配置及密码不修改/打印。独立 `.cache/dev/run-*`（0700）存运行配置（0600）和本次后端构建，退出删除此目录；真实数据库、上传目录和 socket 不在清理范围。
- socket 父目录为当前 UID 的私有目录；连接探测带 `-N`。能连接时直接复用；仅 socket 不存在时用固定 `-S/-f start-server` 由 tmux 自行后台化，项目配置禁 exit-empty/exit-unattached，不读取/写入用户 `~/.tmux.conf`。已有不可连接 socket 不删除、不重启 server。
- 前后端有各自自有进程组；退出或任一服务失败只 TERM/有界 KILL 这两个组并 Wait，绝不将 tmux 放入清理组。配置/端口/依赖预检先于创建持久资源。端口占用不 kill 其他进程。
- `start` 创建后台 supervisor，等待真实就绪后返回；重复 start 幂等。`stop` 经私有 Unix socket 请求 supervisor 取消并等待自身前后端清理，不能读取记录 PID、按端口或进程名杀服务。`restart` 先验证配置，再停止和重新启动，默认恢复上次参数，新增选项优先；停止和重启均保留 tmux。
- 控制目录默认 `.cache/dev`，当前 UID、0700；控制 socket/log/参数记录0600，命令用 flock 串行。参数记录仅含 CLI 选项，不复制密码，原子替换并有界解析。无法确认控制 socket 时拒绝重复启动；残留控制 socket 可清理，与禁止删除持久 tmux socket 是两个不同边界。`PERSISTTY_DEV_STATE_DIR` 只用于隔离控制目录（例如测试），不改变产品配置/终端 socket。
- 就绪轮询最长30秒：匿名 `/api/v1/auth/session` 应为401（受保护），前端首页为200；不能把匿名401误判后端坏了。脚本开发入口不承诺开机自动启动。

## 4. 验证与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 原配置权限/格式/模式错误 | 启动失败，无密码输出 |
| 端口占用/缺工具或前端依赖 | 说明处理方式，不启动/杀旧服务 |
| tmux/shell 路径无效 | 可见错误，原配置不改 |
| socket 存在但不可连接、父目录不私有 | 拒绝，不 unlink/kill server |
| 前后端启动失败/超时/退出 | 清理自身前后端；tmux 保留 |
| stop/SIGTERM/再次停止 | 幂等清理自身前后端，同 pane 继续 |
| 重复 start | 返回已运行地址，不启动第二套服务 |
| restart（无新选项） | 使用上次参数重启前后端，pane PID 不变 |
| 控制目录公开/控制路径被普通文件占用 | 拒绝，不覆盖文件或按 PID 杀进程 |

## 5. 正常 / 基础 / 错误用例

正常：Homebrew tmux + 原开发配置，`start --shell /bin/zsh` 后台启动三者，stop 后原 pane 保留，restart 复用同 server/pane PID。基础：已启动私有 server 时只连接，其他端口可用 CLI 选择。错误：把 Persistty YAML 写入 `~/.tmux.conf`；用 `pkill tmux`/全组清理；打印整个配置/hash；自动杀占端口进程。

## 6. 所需测试

`go test ./scripts/dev` 验证运行配置0600、源文件/密码不变、路径/端口拒绝、环境覆盖、匿名401就绪、自有进程组释放、私有参数恢复/覆盖、控制路径拒绝覆盖。`PERSISTTY_DEV_INTEGRATION=1 go test -v ./scripts/dev -count=1` 必须在安装 tmux/npm/Go 的 Mac/Linux 上执行真实后台前后端、认证代理、项目/终端创建及经 Vite 代理的 WS 输入输出，再证明重复 start/stop 幂等、stop 释放端口但 pane PID 存活、restart 恢复参数且 pane PID 不变。默认跳过真实联调时不得声称已验证；隔离测试只清理自己的随机 socket/server/目录。

## 7. 错误与正确示例

错误：`trap 'pkill tmux; pkill node' EXIT`，或者脚本用 regex 从 YAML 读取密码/路径。正确：复用 `config.Load` 后生成私有配置，tmux 独立 daemonize，只保留本次前后端进程组用于退出清理；检查实际就绪，而非固定 sleep。
