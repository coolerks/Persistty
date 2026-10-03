# W08 本机实施与检查报告

日期：2026-10-03。用户回复“开始实施”，批准推荐的 main push 自动发布方案；当前主会话单代理完成研究、实现、测试与规范维护。未提交、推送、发布 Release 或安装目标机。

## 交付

- `.github/workflows/release.yml`：main push/手动 main，固定 Action SHA、锁定工具链/npm，工程门禁、两架构包、draft完整上传后Latest发布。required Go skip/失败/空发现拒绝发布，两个本地性能探针单列；真实启动持久性显式开启、包串行，bridgego独立检查。
- `scripts/package-release.sh` / `release-manifest.py`：同源码 bin/web/deploy、实际commit/迁移校验、前后端第三方许可、三资产摘要。Python tar统一成员权限/uid/gid/mtime，避免macOS AppleDouble元数据。
- `deploy/persistty-deploy.py`：标准库独立安装/升级，首次参数/TTY密码，单Release资产ID绑定、HTTPS/私有token、SHA/ELF/清单/安全归档、有界下载/解压、root metadata/锁/不可变版本；保留配置和数据、SQLite一致备份、只操作Web、同版健康幂等/失败重试、兼容代码恢复/不兼容停止，不自动覆盖DB。
- VPN/TLS Nginx、Web systemd、后端完整配置：当前WS/body/API/SPA边界和精确Web可写目录；tmux模板保持独立，W07默认关闭。
- 中文 `deploy/README.md`：完整首次与日常一键操作、私有repo、HTTPS、配置、备份/恢复/迁移/诊断；根README入口；owner部署规范和任务资料。

## 本机验证

| 项目 | 结果及限制 |
| --- | --- |
| 部署隔离回归 | `python3 -W error::ResourceWarning -m unittest discover -s tests/deploy -v`：27项通过，不调用真实systemd/nginx |
| Python/Bash语法 | py_compile与bash -n通过 |
| Actions静态检查 | 固定actionlint v1.7.7通过；外部shellcheck/pyflakes未运行，静态通过不等于远端Actions执行 |
| 前端门禁 | lint、typecheck、31文件163单测、实际build通过；保留既有大chunk及jsdom canvas非失败提示 |
| Go普通检查 | 初次 `go test ./...`（缓存）与vet通过；不以缓存替代后续无缓存实跑 |
| Go无缓存发布模式 | `PERSISTTY_DEV_INTEGRATION=1 go test -p 1 -count=1 -json ./...` 两次全量出现同一既有shell测试清理失败：`TestNewPaneUsesConfiguredShell` 在kill自有tmux后RemoveAll遇`.zsh_history`迟写，目录not empty。其余package与真实启动持久性通过；**全量门禁仍未通过**，不隐藏/删除断言或将该测试列为optional |
| Go vet / 独立bridgego | 单独执行root vet与bridgego test/vet通过；bridgego共29个Go pass事件、无required skip |
| 实际Linux包 | amd64/arm64交叉构建；安装器自身解压/manifest/ELF验证、三个资产SHA验证通过；5条实际迁移，37项前端第三方许可。最终复验通过；包内脚本/README与当前源码逐字节一致，无pyc缓存 |
| 文档/diff | 新增/修改Markdown共54个本地链接与git diff --check通过 |

证据存放于忽略目录 `.cache/release-build.log`、`release-deploy-tests.log`、`release-go-tests*.jsonl`、`release-bridge-tests.jsonl`。部署mock覆盖的是本机隔离行为，不等于系统实机验收。

## 保留的失败与修正

1. macOS原生tar生成`._persistty`，安装器安全解压正确拒绝非约定根成员；修正打包owner，采用标准库tar、规范化权限/时间与gzip头，未放宽下载校验。
2. 首轮Python回归因macOS `/var`→`/private/var`测试路径别名失败，fixture使用真实规范路径；SQLite上下文不关闭连接暴露ResourceWarning，实施显式closing，开启ResourceWarning门禁后通过。
3. sandbox内无缓存Go/bridgego不能绑定本机端口/Unix socket；保留失败后，以已授权隔离测试在允许本机socket的环境重跑，未操作用户实例。
4. required skip审查发现开发启动联调默认关闭；工作流显式 `PERSISTTY_DEV_INTEGRATION=1`，不静默忽略required用例。两项手工性能探针仍是明确optional、未验收性能。
5. 外部环境重跑root Go全量两次均暴露既有Mac shell清理竞态，残留只有本次自有`.zsh_history`，不改范围外终端源码或降低门禁。发布CI在Ubuntu运行，但远端结果尚未知；不能据此宣布通过。按trellis-check“若修复需要触及当前任务范围外的文件，说明情况并停止，不要扩大修改范围”，保留为需另行处理的门禁问题。

## 未执行与边界

- 本地没有Nginx，Docker daemon不可连接，未安装/启动额外系统环境；真实nginx -t、Debian system units、VPN IPv4/IPv6隔离、TLS证书和浏览器通过代理的WS/大body/重启持久性未验收。
- GitHub Actions尚未推送执行，GitHub Release资产API的真实公开/私有下载未执行；资产ID/跳转/摘要链使用隔离fixture，Action SHA从官方repo实际核实。
- 未创建用户、WireGuard/防火墙/证书、sudoers/PAM/W07 helper，也未重启用户tmux或原有开发服务。
- SIGKILL/断电恢复、真实数据库迁移失败恢复、慢网/磁盘满、目标机首次安装/升级需在精确产物审查后授权环境实测；自动备份不代表项目/运行进程已备份。

任务继续 `in_progress`，不将未通过Go全量和未执行远端验收写成完成，不自动归档。部署产物可审查；实际启用前须解决门禁并完成首次目标验收。

## 资源与状态

仅清理本次两个明确自有`persistty-shell-*`临时目录中的`.zsh_history`；其自有tmux已由测试kill-server退出。下载/部署回归TemporaryDirectory自动清理。未删除用户配置、项目、socket、tmux session、开发进程或测试以外的资源。

## 最终本地包

版本 `build-local-test`，当前未提交工作树的本地测试产物，manifest记录基线commit `c5b9a7942590d531b9d85cece2456264346efae7`，不作为已发布commit证明。正式CI从checkout构建并记录实际commit。

| 资产 | SHA-256 |
| --- | --- |
| persistty-linux-amd64.tar.gz | bf5d43dd4082fbaba214bf648a1fda7f0af9c4290a1e9a1183f87b6e60d1adc0 |
| persistty-linux-arm64.tar.gz | e5af896460ca29aca47cb9685211e5ab726602f52a6b16227b03b96ba630c773 |
| persistty-deploy.py | c07b8d436b80948fe016425dc095935585e74dc51359699902a911ff40560da0 |

本地输出在忽略目录 `dist/release/`，未上传。GitHub发布会使用真实唯一build版本，当前生产路径未安装任何产物。
