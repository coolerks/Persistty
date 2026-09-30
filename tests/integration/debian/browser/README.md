# W03 真实浏览器联调

此 harness 运行真实产品 HTTP/WS/tmux 与 xterm，不写用户正式数据库或现有服务。仅从仓库根已忽略 `.env` 读取 SSH 连接，失败只输出 `failed_stage/error_kind`；不输出连接参数。

## 启动与测试

先编译 basename 必须为 `runtime-probe` 的 Linux 探针：

```sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /private/tmp/runtime-probe ./tests/integration/debian/recovery/runtimego
python3 -B tests/integration/debian/run_remote.py browser-start --binary /private/tmp/runtime-probe
```

返回随机私有 root 与 loopback port。使用返回 port 建本机 18080 tunnel；下面的 `PORT` 和 `ROOT` 必须替换成返回值，不是固定部署路径：

```sh
python3 -B tests/integration/debian/browser/tunnel.py PORT
PERSISTTY_DEV_API_TARGET=http://127.0.0.1:18080 npm --prefix web run dev -- --port 5174 --strictPort
```

打开 `http://127.0.0.1:5174`，隔离登录密码为 `isolated-test-password`，项目为 `W03 隔离终端`；这些是合成 fixture，不可用在真实部署。安装 Playwright Chromium 后，在 `web/` 执行：

```sh
PERSISTTY_E2E_BASE_URL=http://127.0.0.1:5174 PERSISTTY_E2E_PASSWORD=isolated-test-password PERSISTTY_E2E_PROJECT='W03 隔离终端' PERSISTTY_E2E_REMOTE_ROOT=ROOT PERSISTTY_E2E_FORCE=1 npm run test:e2e
```

完整九条验收必须使用新建的空实例：首条建立恰好三个计数/HTTP/TUI 会话，随后跨 SIGKILL/正常 Web 重启检验 pane 身份与负载。其他测试覆盖输入一次性、三端倒计时、新端加入、精确终止、宿主移动零新 WS、手机按键/链接/主题和非空截图。截图和 Playwright 输出在已忽略 `web/test-results/`；不提交含私有信息的报告。最新 2026-09-30 完整九条全部通过。

## 精确清理与限制

```sh
python3 -B tests/integration/debian/run_remote.py browser-cleanup --root ROOT
```

清理检查 own units inactive、own pane processes gone、root removed，然后停止本轮 tunnel/Vite。远端 root 0700、配置0600；临时解包固定 tmux 3.5a 包及 libevent，不进行系统安装。tmux 与 Web 是不同 user unit，`RuntimeMaxSec=900`，**15 分钟后自动结束测试资源，禁止当作长期开发机部署或运行真实工作负载**。正式部署属于 W08。

故障经验：SIGKILL 后用 `systemctl restart` 同 transient Web unit，不先 `reset-failed` 导致定义被卸载；cleanup 才 reset。确定性 curses fixture 按 tmux terminfo 使用鼠标模式，不手动强开 SGR mouse；bracketed paste 单独测试。Chromium loopback offline 不一定断开 WS，离线测试同时派发真实页面 offline 事件，并验证输入立即失效，不靠后台仍在收帧判断网络可写。
