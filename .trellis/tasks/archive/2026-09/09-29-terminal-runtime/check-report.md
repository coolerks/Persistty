# W03 验收记录

日期：2026-09-30。当前主会话单代理完成研究、实现、审查及测试，无 subagent。用户批准三批提交、归档和日志，不推送。

## 范围与实现

完成 [PRD](prd.md) T01..T07：独立 tmux 真实终端、项目主/指定文件夹创建、真实状态与元数据恢复、普通历史/活动画面分离、多端单控制权、服务器统一可取消倒计时、真实 xterm 工作台及手机控件。复用既有数据库 schema，无新增迁移；正式 Nginx/VPN 安装、单文件提权和 W05 编辑/完整布局仍未交付。

## 验收矩阵

| 项 | 实际证据与断言 | 结果 |
| --- | --- | --- |
| T01 生命周期 | 真实独立 user unit；计数/HTTP/确定性 curses TUI 三 pane 跨正常与 SIGKILL Web 重启保持 PID/start_ticks/cgroup，计数继续、HTTP 可达；关页/离线/登出后重新登录可输入，缺 server 不启动 | 通过 |
| T02 恢复/历史 | tmux 3.5a 普通 attach 恢复拆分 UTF-8/OSC、alternate 和 resize；120 行有界普通历史与实时编号连续且不拼接；实际浏览器 TUI 键盘/方向/鼠标/resize、粘贴及真实画面非空 | 通过 |
| T03 多端 | 双端接管/旧端撤销、旧 generation 输入拒绝、observer 不 resize、有限重连零输入重放；慢观察端不停止 pane；race 测试 | 通过 |
| T04 终止 | 三端/期间新连接相同 request/deadline；观察端取消、重复请求拒绝、接管/失效/断开/Web 重启取消；实际到期仅目标结束，其余保持 running | 通过 |
| T05 项目 | 主/所选 folder 与改主后新终端裁决；旧 cwd 不改；旧配置拒绝；真实产品探针移除项目保留 session | 通过 |
| T06 UI | 按钮/拖拽上下移动、刷新/面板收起展开均同 ID/同 runtime/零新 WS；上方 X 走倒计时；手机所有真实快捷字节、浅/深主题窄屏无横向溢出；桌面 Ctrl 链接及手机确认/取消原样新标签 | 通过 |
| T07 质量/安全 | Go 全包 test/vet/race；前端 lint/typecheck/49 单测/build；9 条真实 Chromium E2E；Cookie/Origin/CSRF/注销矩阵；隔离资源精确清理 | 通过 |

## 可复现检查

```sh
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run test
npm --prefix web run build
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /private/tmp/runtime-probe ./tests/integration/debian/recovery/runtimego
python3 -B tests/integration/debian/run_remote.py recovery --binary /private/tmp/runtime-probe
```

真实恢复/产品探针 40 项全部为 true，units/scopes/root 清理为 true。浏览器完整 9 条在新隔离实例顺序运行，43.9 秒全部通过；步骤见 [浏览器 README](../../../../../tests/integration/debian/browser/README.md)。浏览器 cleanup 再验证 units inactive、pane processes gone、root removed；本轮 SSH tunnel 与 Vite 联调进程也已停止。连接参数仅来自已忽略 `.env`，记录无 SSH 用户/IP/端口、密码、Cookie/token、输入或真实文件正文。

## 修复与复用检查

- 稳定 provider/portal runtime 保留宿主移动时的 xterm/WS，删除自动重新接管补偿；StrictMode、移动、隐藏与 scope 退出有回归测试。
- 离线立即撤销 ready，忽略 dispose 后晚到帧；有界输出/慢端隔离；owner 错误路径加锁；原 pane 不随 attach 或项目清理销毁。
- 使用既有 shadcn Button/Dialog/Select/Alert/Empty/Badge，并通过官方 CLI 添加 Toggle，Ctrl/Alt 使用 pressed 状态；查找先于编写，CLI `docs button dialog select empty badge`、`docs toggle`、`docs alert`，核对 [Base Toggle](https://ui.shadcn.com/docs/components/base/toggle) 与 [Base Alert](https://ui.shadcn.com/docs/components/base/alert)。沿用 `base-nova`/`@base-ui/react`，无直接 Radix 依赖/导入。
- 协议 fixture 与双端 decoder、HTTP/WS handler/service/repository 数据流已复核；终端日志不记录输入/输出，失败探针只记录阶段和错误类别。

## 限制与非阻断提示

原始 [2026-09-29 机制证据](../../../../../tests/integration/debian/recovery/evidence.json) 保留当时 `integrated_web_t02_accepted:false`，不改历史；最新产品通过记录见本报告。控制模式 snapshot/raw 拼接的 UTF-8 反例仍成立，不采用该模型。

Go 沙箱内首次 test 因 loopback bind 被拒，已在允许监听的执行环境用 `-count=1` 重跑完整 test/race；未跳过测试。49 条前端单测通过但 jsdom 提示 canvas getContext 未实现，真实渲染由 Chromium 验证；构建成功但仍有 Monaco/主包超过 500 kB 警告，未隐藏提示或改阈值。移动测试为 Chromium 窄屏，不冒充真手机键盘/设备全矩阵。普通 tmux 不保证主机重启恢复，管理员停 tmux 会终止任务。隔离探针有 900 秒 TTL，只用于验收，不作为长期正常使用部署。
