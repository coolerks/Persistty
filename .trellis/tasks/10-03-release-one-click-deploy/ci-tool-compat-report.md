# Actions 工具版本兼容修复

日期：2026-10-03。通过用户明确指定的 GitHub 插件读取 [失败运行](https://github.com/coolerks/Persistty/actions/runs/37100462607)，提交 `fcf530179d6b7c0b0e95e99a2f41f6dde67b1e93`，Ubuntu 24.04、Go 1.26.8、Git 2.55.0、rg 14.1.0。本轮单主会话实施，不自动提交、推送或发布。

## 真实失败与修复

- 搜索多个 Preview 与 rg 对照用例报 `tool unavailable`。rg 14 接受 `--replace --json` 但不输出 replacement；旧空输入探测看不到能力缺口。[官方 rg 15 变更记录](https://github.com/BurntSushi/ripgrep/blob/15.0.0/CHANGELOG.md)确认 JSON 替换从该版本开始支持。现用固定非空捕获/替换样本检测能力，在创建搜索快照前选择并固定 native；保持取消、预算、原字节/坐标、文件版本与 Preview 固定引擎边界。
- Git 历史分页首屏报文件版本冲突。用官方 Git 2.55.0 在本机隔离编译复现：20 次原用例中 19 次失败，包含原版本冲突和对象/元数据消失；Trace2 记录 1050 次自动 maintenance 启动，包含 repack。Git 2.55 默认维护策略改变，[官方记录](https://github.com/git/git/blob/v2.55.0/Documentation/RelNotes/2.55.0.adoc)提供背景。测试 helper 固定 `gc.auto=0,maintenance.auto=false`，只阻止测试自己启动后台写入，不改变产品快照冲突保护。
- 原发布诊断和 required skip 拒绝保持。该 run 已成功保留 `go-test-logs` 附件，后续桥接/前端/打包/publish 因根 Go 失败未执行；不能宣称远端已发布。

## 回归证据

- 官方 rg 14.1.1（隔离目录，不改系统工具）：原对照/批量 Preview 用例重现相同错误；修复后完整 search 包通过。本机 rg 15.2.0 完整 search 包也通过。
- 旧替换 JSON 专项验证具名捕获、美元符号、BOM/混合换行，搜索→选择→预览→Apply，并在探测后移除工具仍保持已固定 native。rg 14/15 都执行真实坐标对照及明确展开期望，未跳过整个对照用例。
- Git 2.55 修复后历史分页及维护隔离各重复 20 次通过，Trace2 无自动 maintenance 启动。维护隔离回归在移除 helper 修复时确定失败（生成 pack），恢复后通过，避免以固定等待隐藏竞态。
- search/gitview `go test -race -count=1` 通过；根与 bridge vet 通过。独立桥接无缓存串行测试 29 个 pass 事件，required 无 skip。
- 使用隔离 Git 2.55.0、rg 14.1.1，`GOTMPDIR=/tmp PERSISTTY_DEV_INTEGRATION=1 go test -p 1 -count=1 -json ./...` 退出0，日志检查器核对200个 pass 事件，required 无 skip。修改规范/任务的9个本地 Markdown 链接、Trellis任务校验和 `git diff --check` 通过。
- 前端 lint/typecheck、31 文件 163 单测、实际 build 通过；原大 chunk 构建提示保留。部署/日志 34 项回归通过。
- Linux amd64/arm64 后端实际交叉编译均成功；**没有执行 Linux 二进制或 Ubuntu 测试**。

## 本机门禁环境说明

首次沙箱运行禁止 Unix/HTTP socket，保持失败证据后按实际监听需要复验。本机完整测试另发现 `.cache/go-tmp` 作为 GOTMPDIR 时，Go testing.TempDir 使控制 socket 地址达到107字节，macOS dial 返回 `invalid argument`，触发启动器停止确认失败；探针已记录真实地址长度和错误，改用 `GOTMPDIR=/tmp` 后控制测试连续10次通过。没有修改启动器产品行为，也不把该环境失败写成 Ubuntu 根因。长目录全量复验还出现本机启动器启动失败，保留原日志；短目录最终全量门禁通过200个 pass 事件，包括本机启动与终端持久性用例。

本轮忽略目录证据：`.cache/ci-tools/rg14-before.log`、`rg14-after.log`、`rg15-after.log`、`git255-before.log`、`git255-after.log`、`git255-regression-before.log`、Trace2 日志，以及 `.cache/release-compat-*`、`release-go-compat-*.jsonl`。

## 远端与部署边界

修复尚在本机工作树，须提交并推送 main 后查看新提交触发的 Actions。重新运行旧 run 使用旧提交，无法验证本轮修复。目标 Nginx/systemd、LAN 浏览器和真实部署下载仍未执行，不升级目标 Git/rg、不重新安装 Nginx/Python、不修改目标账号或 tmux。任务继续 in_progress。
