# Persistty

Persistty 是面向 Debian 开发机的单用户、自托管浏览器终端与轻量文件工作台。最终范围包括持久终端、多文件夹项目、文件管理与自动保存、搜索替换、上传下载和只读 Git。

当前正在实现 W01 基础包：Go/Gin 服务、SQLite、Argon2id 密码认证、真实项目/终端元数据读取，以及 React 登录、项目路由和三种主题。项目创建、文件工具、PTY连接和提权尚未交付；空列表来自实际数据库，没有演示项目或伪终端。完整规划见[需求](.trellis/tasks/09-26-requirements-research/prd.md)及[实施计划](.trellis/tasks/09-26-requirements-research/implement.md)。Trellis入口为[AGENTS.md](AGENTS.md)和[工作流](.trellis/workflow.md)。

## 本地运行

需要 Go 1.26 工具链和 Node.js 24，依赖版本锁定在 go.mod/go.sum 与 web/package-lock.json。前端使用 npm，不混用其他包管理器。

```bash
go build -o persistty ./cmd/persistty
npm --prefix web ci
./persistty password
./persistty serve --config /绝对路径/config.local.yaml
npm --prefix web run dev -- --port 5173 --strictPort
```

本地配置结构参考[示例](deploy/config.example.yaml)，使用`mode: development`、`listen: 127.0.0.1:8080`、`public_origin: http://127.0.0.1:5173`和绝对私有数据库路径。配置归当前用户，权限0600，数据库父目录0700。密码CLI从TTY隐藏输入，至少12字节；PHC填写auth.password_hash，不把明文放参数或版本库。Vite通过同源`/api`代理后端，打开[工作台](http://127.0.0.1:5173)。当前项目维护由后续W04交付，不手工更改数据库来绕过注册身份校验。

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

共享响应样例位于[tests/contracts/foundation.json](tests/contracts/foundation.json)，只是契约测试输入，不导入运行中的数据库。部署示例和未验证限制见[deploy](deploy/README.md)；目标Debian/systemd/VPN/提权实验仍需授权及实测。

## 生命周期边界

核心要求：关闭页面、网络中断以及Web服务重启不得终止未来独立tmux服务中的任务。多端可查看，仅一个控制端输入；显式终止由控制端发起全端倒计时，截止前任一查看端可取消。面板收起不同于终止，上方终端标签X是终止入口。该要求须通过W02/W03真实Debian/systemd及TUI测试，不能把W01元数据页面当作持久终端已经实现。

普通 tmux 进程无法跨主机重启保存运行中的内存状态，Persistty v0.1 不承诺主机重启后恢复正在执行的进程。程序自行退出、系统终止或管理员停止 tmux 服务也属于真实终止原因。
