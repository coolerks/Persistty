# D07 历史与 TUI 对照

这是独立实验，不是产品 snapshot/live 实现。依赖锁定官方 `@xterm/headless@5.5.0`（MIT），解析等待 write callback，字节使用 Uint8Array。只操作固定合成历史与 Python 标准 curses 程序，不发送 shell 输入。

真实远端使用私有 .env、安全 transport 和新建 0700 ROOT；预启动私有 tmux，Web 单独用户 unit。实验分别记录初次 attach、capture 后输出超过一屏再 attach、真实 curses 当前画面、两种 resize、detach 后重连和退出。capture 是 grid 导出，并不提供完整 parser/mode 状态或原子 live 截点。

原始固定输出只暂存根 `.cache/`，可提交证据只含 SHA256、字节数、解析断言和脱敏进程身份。headless 证明解析，不证明浏览器 DOM/render、真实浏览器 WS 鉴权、多观察端或生产历史容量。

复现：在本目录 `npm ci`、`npm test`；构建 Linux bridge-probe 后运行 `python3 -B tests/integration/debian/run_remote.py history --binary /tmp/bridge-probe`。运行前须有目标机隔离实验授权；不能因 .env 存在自动执行。记录限时 10 秒/256 KiB/4096 帧；本轮每次 1 秒。正常退出会精确停止自身 units/scope 并删除已知 ROOT；runner 自身 SIGKILL/网络故障仍可能绕过 finally，不保证零残留。
