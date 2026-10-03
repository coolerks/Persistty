# Linux amd64 手工部署包交付

## 范围

2026-10-03 用户取消 Actions 故障跟进，改为本机交叉编译前后端，由用户自行在 Debian 部署。本轮只修改打包脚本、手工文档及任务/规范记录，没有修改产品 Go/前端代码、CI门禁或目标机配置。临时 dev 测试诊断已恢复。

GitHub run `37102646682`（提交 `b007202307a5f4c351f6bd0d312cff6d87cca6da`）的日志有212项测试通过，唯一失败为 `scripts/dev/TestLocalStartupPersistence` 第二次重启。该故障仍未修复，不以本次成功打包或普通 Go 测试替代 CI 验收。原 Ubuntu 容器复现因取消任务及 Docker daemon 不可用而中止，不记通过。

## 交付物

生成目录：`dist/debian-amd64-20261003/`（忽略的本地产物，不提交）。

- `persistty-linux-amd64.tar.gz`：约20MiB，版本 `manual-20261003-b007202`。
- `SHA256SUMS`：仅本轮 amd64 包和保留的独立下载器，不引用 arm64。
- `部署说明.md`：`deploy/MANUAL.md` 的完整副本，包内根 README 同内容。
- 原有 `persistty-deploy.py` 一并保留，手工部署无需运行。

tar SHA256：`a256b259f0df909196570d9844865ddc1987fefa9409c96cd176c651c52248f3`。

包由当前工作树构建，manifest 的 commit 为源码基线 HEAD，上述版本名标记本次手工构建；包含未提交的手工文档与打包变更，不声称该 commit 已包含这些变更。

## 实际验证

| 检查 | 结果 |
| --- | --- |
| `scripts/package-release.sh manual-20261003-b007202 dist/debian-amd64-20261003 amd64` | 成功；前端实际构建，CGO关闭的 Linux amd64 后端交叉编译 |
| `file` 与 ELF 程序头 | x86-64 ELF64，statically linked；无 PT_INTERP/PT_DYNAMIC |
| 实际 tar 安全解压与原部署器 validate_package | 通过，架构/迁移/许可证及必要文件完整 |
| SHA256SUMS | 两项真实文件摘要匹配；不含缺失架构 |
| 前端入口引用、字体、五类 Monaco worker | 通过，指向已打包文件 |
| 后端完整配置、tmux配置、两份unit、Nginx模板 | 与源码模板逐字节一致，实际IP/hash留给目标机填写 |
| 包根README、旁置部署说明 | 与 `deploy/MANUAL.md` 逐字节一致 |
| `go test -count=1 ./...` / `go vet ./...` | 本机通过，短 GOTMPDIR；未开启取消跟进的 dev 启动集成，不记 CI 通过 |
| 前端 npm lint/typecheck/test | 通过；31个测试文件，163项测试 |
| Python部署回归，ResourceWarning作为错误 | 34项通过 |
| Bash语法、无效架构拒绝、diff空白、本地文档链接 | 通过 |

前端构建仍有既有的大 chunk 提示，单测有 jsdom canvas 提示；命令退出均为0。未为无关提示扩展产品修改。

## 部署与未执行项

路径、权限和操作命令见 [手工部署说明](../../../deploy/MANUAL.md)。局域网HTTP、现有手动Nginx；不安装软件，不配置SSL/WireGuard，不保存真实IP。首次安装使用已有非root开发账号，手工更新仅停Web，保留配置、数据及独立tmux。

目标 Debian 的 systemd/Nginx 启动、实际HTTP登录、保存与终端WS由用户按文档验证，本轮未连接或操作目标机。当前W08保持 in_progress，不自动提交、推送、发布或归档。
