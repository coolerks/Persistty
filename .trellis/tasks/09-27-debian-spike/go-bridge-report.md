# W02 D06 Go / PTY / WS 实施记录

## 修改边界
只新增 `tests/integration/debian/bridgego/` 独立实验 module、源码、测试、README、通知及本记录。根 Go module、Gin 产品路由、数据库、前端和生产部署未修改。本 owner 未执行 SSH、未读取私有 `.env` 或真实 token；远端执行、脱敏证据及精确清理由 runner owner 负责。

## 协议与清理
CLI 与固定字段见 [实验 README](../../../tests/integration/debian/bridgego/README.md)。服务使用 loopback 动态端口，token 文件同 fd 检查当前 owner/0600/普通文件，拒绝符号链接；64hex token 仅作 header。Origin 固定比较，单 active attach 409 是本实验限制。Go PTY 与 WS 双向 binary，不按 Unicode 转字符串；严格 resize 字段及上限，控制/输入超限关闭；队列和写 deadline 有界。

Web 或 WS 取消仅关闭 fd、终止精确 attach child、Wait 并等待 pump 退出，无创建/kill tmux session/server 路径。`-N` 禁止无 server 时隐式启动。stats 分别记录启动与 Wait 计数、当前 FD/goroutine；exercise 七次实际 attach 结束后 active false，started==reaped。FD/goroutine 相对基线预算分别 +2/+4，此有限检查不等于任意负载无泄漏证明。

exercise 固定输入一次，UTF-8 字符在两次 WS binary 内切断；Python pane 计数与 ACK 由 runner 提供。observe/hold 不发送输入，因此恢复本身不会重跑输入。完整 TUI/history/snapshot、产品多观察端/controller/倒计时、认证撤销和反向代理未验收。

## 依赖与通知
Go 1.26.8、PTY v1.1.24、WS v1.8.15 精确锁定在本 module，`go mod tidy` 保存真实下载校验。采用[PTY 官方包资料](https://pkg.go.dev/github.com/creack/pty@v1.1.24)、[coder 官方包资料](https://pkg.go.dev/github.com/coder/websocket@v1.8.15)，不宣称 latest。模块 MIT/ISC 与 Go 工具链 BSD 许可证全文从实际下载文件读取，见 [嵌入通知](../../../tests/integration/debian/bridgego/third-party-notices.txt)，二进制 `licenses` 可查看。

## 本地验证
在独立 module 下执行 `go test ./...`、`go test -race ./...`、`go vet ./...` 均通过，无 skip；共有 9 个顶级测试（含表驱动子用例）。真实 httptest 网络与 PTY 使用已明确申请的本地沙箱升级，不把环境限制视为通过。覆盖 token/配置、Origin/认证/占用、非法 UTF-8 与 NUL 字节透明性、严格控制解析/尺寸/输入与控制限额、15 次连接循环的 child Wait 与服务取消。PTY raw-mode 用明确输出就绪，不靠固定 sleep。

`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build` 成功，忽略目录 `.cache/debian-bridge-probe` 的最终 SHA256 为 `5a0559258ec0582d6446385be12c60be4412fb05a2c31ea5c871f7580f0d9513`。根 `go test ./...` 不会覆盖该 nested module，因此不能代替以上检查。

真实远端初轮由 runner 报告成功，最终此哈希版本的独立复核与清理仍以主会话/check 的报告为准；本记录不自行宣布完整 W02 或终端产品验收通过。
