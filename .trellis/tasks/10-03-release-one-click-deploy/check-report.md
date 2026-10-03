# W08 开源部署参数化复验

最新实际失败与业务兼容修复见 [工具兼容检查报告](ci-tool-compat-report.md)：已读取第二次 Actions 的真实错误，修复 rg 14 替换探测和 Git 2.55 测试后台维护；保留旧失败证据与远端未验证边界。

最新 Actions 首次失败跟进见 [CI检查报告](ci-failure-report.md)：已修复隐藏日志与附件保留；本机同参数Go/桥接及34项回归通过，原Ubuntu失败用例仍待新workflow日志确认。

日期：2026-10-03。用户明确要求开源内容不包含个人实际IP；本轮已从源码、模板、测试、文档、任务和开发日志移除。首次 `install --host LAN_IPV4` 校验并保存地址，Nginx与后端从同一值生成；`update` 读取目标机保存的地址，仍是一条命令。部署模板的 `LAN_IP` 必须由安装器替换，不能直接启用。

## 最新本机结果

- 32项部署回归通过：RFC1918三网段、拒绝公网/loopback/配置注入、部署设置序列化读取、后端与Nginx同源生成，以及原下载/备份/生命周期回归。
- `go test ./...`、`go vet ./...`通过；config包本轮实际执行，部分未改包使用缓存。初轮无缓存macOS shell fixture清理竞态未修复，原失败证据仍保留。
- Python/Bash语法与actionlint通过。前端未修改；本轮打包实际build通过，lint/typecheck/163单测沿用上一轮证据。
- `build-generic-local-test`两架构实际包均通过SHA/manifest/ELF、安全解压、包内全部文件个人IP扫描与实际模板渲染。包内文档/部署器/模板与源码逐字节一致。
- Git受版本管理及未忽略待提交文件扫描无个人IP；49个本地Markdown链接与diff检查通过。

证据：忽略目录 `.cache/release-deploy-tests-generic.log`、`.cache/release-go-generic.log`、`.cache/release-build-generic.log`、`dist/release/SHA256SUMS`。本地产物来自未提交工作树，不作为已发布commit证明。

| 最新资产 | SHA-256 |
| --- | --- |
| persistty-linux-amd64.tar.gz | b174f38ad5caef25f2bceda291ff986d8c79ad36b629fd158a2a90c33a5fc130 |
| persistty-linux-arm64.tar.gz | 39e99c81bdddadd72efbdffcd2614c23f3bf774f727519bbeb8463d4716ee34e |
| persistty-deploy.py | 82aa2457702e1be1b8f38655c0f7ca632ca3d0839fea1bd76ed2a24818ff63b1 |

目标Nginx/systemd/LAN浏览器与GitHub发布未执行；未提交、推送、安装或归档。任务保持in_progress。

---

## 上轮记录（真实地址已脱敏，以下包摘要为旧产物）

# W08 局域网部署重写检查

日期：2026-10-03。按用户最新要求，访问采用局域网 HTTP（实际地址已脱敏为 `LAN_IP`），沿用已有 Nginx/Python，不安装系统软件，不配置 SSL 或 WireGuard。原方案与检查历史见 [初轮报告](check-report-initial.md)。当前主会话单代理完成本轮修改，未提交、推送、发布或安装目标机。

## 本轮交付

- [部署文档](../../../deploy/README.md)精简主流程：main 自动打包、现有 Nginx 添加一行 include、首次安装、以后 `sudo /usr/local/sbin/persistty-deploy update`。
- 固定 LAN Nginx 模板与新增后端 `lan_http` 模式：私有 IP HTTP Origin、loopback 后端、现有鉴权和 CSRF 边界；移除部署包的 TLS 模板。
- 部署器记录实际 Nginx/Python 路径，支持已有 Nginx 主配置路径；通过原程序执行 `-t` 和 `-s reload`，不操作 `nginx.service`。默认服务账号为 sudo 原账号。
- 保留 GitHub 两架构打包、下载校验、数据库备份、配置与数据保留、失败恢复及独立 tmux 生命周期。

## 本机验证

| 检查 | 实际结果 |
| --- | --- |
| 部署回归 | `python3 -W error::ResourceWarning -m unittest discover -s tests/deploy -v`：30 项通过；覆盖实际程序路径、原 Nginx 主配置、Python shebang、LAN 模板及更新安全边界，系统操作使用隔离替身 |
| 语法与 Actions | Python 编译、Bash 语法、actionlint v1.7.7 通过；外部 shellcheck/pyflakes 未执行 |
| Go | 本轮修改的 `go test ./internal/config` 通过；`go test ./...` 与 `go vet ./...` 通过，部分未改包使用缓存 |
| 前端 | lint、typecheck、31 文件 163 单测及实际 build 通过；既有大 chunk 与 jsdom canvas 提示保留 |
| 实际包 | `scripts/package-release.sh build-lan-local-test` 构建 Linux amd64/arm64；两包均通过部署器自身安全解压、manifest、ELF 和摘要校验；包内脚本、配置和 README 与源码一致，无 TLS 模板 |
| 文档 | 46 个本地 Markdown 链接及 `git diff --check` 通过 |

本轮日志位于忽略目录 `.cache/release-deploy-tests-lan.log`、`.cache/release-go-lan.log`、`.cache/release-build-lan.log`。部署隔离测试不代表真实系统安装验收。

初轮无缓存 Go 发布模式出现的既有 macOS shell fixture 清理竞态没有在本轮修复；本轮普通全量缓存测试通过不能替代该失败证据。CI 保留 required Go 门禁，实际 GitHub 执行结果尚未知。

## 本地包

版本 `build-lan-local-test`，输出在忽略目录 `dist/release/`，未上传。manifest 基线 commit 为 `c5b9a7942590d531b9d85cece2456264346efae7`；这是未提交工作树的本地验证产物，不作为已发布 commit 证明。

| 资产 | SHA-256 |
| --- | --- |
| persistty-linux-amd64.tar.gz | 8592bb5a262930395342b1bd3a616e18968981396839870c3f3a8c6f8fb15cb8 |
| persistty-linux-arm64.tar.gz | bbb1d06bea0b99ca3e65142c46df139019454c3bb03fac061156afdfaed0c48c |
| persistty-deploy.py | 627c1fe8c15e5e4c6e34e2cb022334048bab6f3df9b2b0b00419f4e9971abc1c |

## 未执行

目标电脑已有 Nginx 的真实语法检查、reload、systemd 首次安装/升级、LAN 浏览器 WebSocket 与终端重启持久性尚未执行。GitHub Actions 和公开/私有 Release 的真实下载也未执行。不安装 Nginx/Python、不修改目标主配置、不重启用户 tmux，不把替身测试记为实机通过。任务保持 `in_progress`，不自动归档。
