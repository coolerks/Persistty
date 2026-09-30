# Persistty

Persistty 是面向 Debian 开发机的单用户、自托管浏览器终端与轻量文件工作台。最终范围包括持久终端、多文件夹项目、文件管理与自动保存、搜索替换、上传下载和只读 Git。

W01/W02/W04 已完成归档，W03 已通过真实 Debian/systemd 和桌面/手机浏览器验收。当前提供认证、多文件夹项目、文件管理与传输、桌面只读 Monaco/手机基础文本视图，以及真实持久终端、多端单控制权和可取消倒计时终止。自动保存/草稿、搜索/只读 Git、单文件提权和正式部署仍由后续任务交付，不是完整 IDE 已完成。完整规划见[需求](.trellis/tasks/09-26-requirements-research/prd.md)及[实施计划](.trellis/tasks/09-26-requirements-research/implement.md)。Trellis入口为[AGENTS.md](AGENTS.md)和[工作流](.trellis/workflow.md)。

## 本地运行

需要 Go 1.26 工具链和 Node.js 24，依赖版本锁定在 go.mod/go.sum 与 web/package-lock.json。前端使用 npm，不混用其他包管理器。

```bash
go build -o persistty ./cmd/persistty
npm --prefix web ci
./persistty password
./persistty serve --config /绝对路径/config.local.yaml
npm --prefix web run dev -- --port 5173 --strictPort
```

本地配置结构参考[示例](deploy/config.example.yaml)，使用`mode: development`、`listen: 127.0.0.1:8080`、`public_origin: http://127.0.0.1:5173`和绝对私有数据库路径。配置归当前用户，权限0600，数据库父目录0700。密码CLI从TTY隐藏输入，至少12字节；PHC填写auth.password_hash，不把明文放参数或版本库。Vite通过同源`/api`代理后端，打开[工作台](http://127.0.0.1:5173)。通过项目面板选择服务器目录，不手工更改数据库绕过注册身份校验。

真实终端需要先在 Web 生命周期之外启动同 UID 的私有 tmux server，配置中的 `terminal.socket_path` 与 server 一致。参照[独立服务模板](deploy/systemd/persistty-tmux.service)及[部署边界](deploy/README.md)审查实际路径；缺 server 时显示不可用，Web 不隐式启动。Debian 隔离联调步骤见[浏览器验收](tests/integration/debian/browser/README.md)，该临时探针不是长期运行部署。

## 验证

```bash
go test ./...
go test -race ./...
go vet ./...
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run test
npm --prefix web run build
```

共享响应样例位于[基础协议](tests/contracts/foundation.json)和[终端协议](tests/contracts/terminal-runtime.json)，只是契约测试输入，不导入运行中的数据库。W03 真实恢复机制与浏览器验收分别见[恢复实验](tests/integration/debian/recovery/README.md)和[浏览器验收](tests/integration/debian/browser/README.md)。部署示例及尚未完成的正式 Nginx/VPN 安装验证见[deploy](deploy/README.md)。

## 生命周期边界

关闭页面、网络中断、登出以及 Web 服务重启不终止独立 tmux 服务中的任务。多端可查看，仅一个控制端输入；显式终止由控制端发起全端倒计时，截止前任一查看端可取消。面板收起不同于终止，上方终端标签 X 是终止入口。W03 已用计数、HTTP、确定性 TUI 三类负载验证这些路径；界面恢复不重放输入。

普通 tmux 进程无法跨主机重启保存运行中的内存状态，Persistty v0.1 不承诺主机重启后恢复正在执行的进程。程序自行退出、系统终止或管理员停止 tmux 服务也属于真实终止原因。
