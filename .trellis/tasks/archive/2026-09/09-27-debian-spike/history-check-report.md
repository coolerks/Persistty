# D07 独立检查报告

## Findings (fixed)

- File: `tests/integration/debian/history/probe.py`
- Issue: 优化模式保护写在 assert 中，Python -O/-OO 会同时删除保护、资源身份检查及实验判定。
- Fix: main 入口显式拒绝 sys.flags.optimize；增加两种优化模式的子进程回归，确认在目录访问前失败。

- File: `tests/integration/debian/run_remote.py`
- Issue: history 的本地 cache.mkdir 位于远端 ROOT 分配之后、finally 之前；本地权限错误会绕过已知 ROOT 清理。
- Fix: 缓存目录创建前移至远端分配之前；增加 PermissionError 时零 SSH/SCP 调用的回归。

- File: `tests/integration/debian/bridgego/record_test.go`
- Issue: 记录上限和调用方取消缺少实际 WS 回归，原测试只拒绝非法参数并验证成功记录。
- Fix: 增加 256 KiB 和 4096 帧超限、连接关闭，以及真实 PTY attach 取消后的 Wait/reap 检查。没有改变 record 协议或限额。

## Findings (not fixed)

没有发现需要额外产品决策才能修复的新增缺陷。以下设计限制继续保留，不在本轮扩大实现范围：

- capture + attach 非原子，普通历史被外层 alternate 隐藏，截点间滚动输出发生遗漏；独立实验再次观测到反例，不能作为产品历史恢复方案通过。
- headless 解析及一个受控 curses 程序不证明浏览器 renderer、实时浏览器 WS 鉴权、多观察者/controller、生产 snapshot/live 截点或容量。没有引入 URL token、放松 Origin 或代理绕过认证。
- runner SIGKILL、本机退出或网络失败仍没有持久恢复账本；只验证正常 finally 和已知资源清理。临时命令捕获只有时间界限，不是磁盘 quota。
- D07 同一次远端执行中分次采样，不新增跨 SSH 持续流证据；原 D06 两份 evidence 未覆盖或修改。serialize、Landlock 与 helper 未执行。

## Verification

- Lint: pass。bridgego gofmt 无输出；根模块与独立 bridgego 的 go vet 通过；git diff --check 通过。JS/Python 没有配置额外 lint 脚本。
- TypeCheck: pass（Go 编译门禁）；本轮没有 TypeScript 变更，独立 history 为 JS，无 TypeScript typecheck 脚本。没有冒称产品前端检查已执行。
- Tests: pass。根 Go test/race 使用 -count=1；独立 bridgego test/race 使用 -count=1，16 个 top-level tests、无 skip；Python history 6 + bridge 6 + terminal/remote_config 21，共 33 项；npm ci --ignore-scripts --offline 与 npm test，2 项 Node tests。
- JSONL: implement/check 各 6 项通过；135 个本地 Markdown 链接无缺失（检查时的文档集合）；隐私扫描 301 个可提交文件及 HEAD 零连接值命中；.env 忽略且未跟踪。

## 独立真实 Debian 结果

使用最终 Linux amd64 CGO=0 binary，SHA256 为 `c294742fb8848421b3da04b956199327df981c36ebcdf5b9ee2c2b6319e16501`。修正前后分别独立执行，最终 runner 的摘要保存在 [evidence-review.json](../../../tests/integration/debian/history/evidence-review.json)，保留首次 D07 的 evidence.json。

- 8 项实验断言成立；raw attach 缺少完整历史、capture/attach 滚动间隙丢失、外层 alternate 隐藏已捕获历史是成功观测限制，而不是产品无损恢复验收。
- 7 次实际 WS 记录逐字节重分块后，与原帧解析的 normal/alternate cells、字符宽度、颜色、基本样式和光标一致；使用真实锁定 @xterm/headless 5.5.0、Uint8Array/write callback，MIT 完整通知保留。
- curses 进入、100x30/80x24/120x40 尺寸重绘、同一 pane 重连及退出符合固定判定；同时核对实际程序尺寸、pane alternate_on 与 current/saved capture，不仅依赖 WS resized ack。
- 6 个样本原 server/pane PID、start_ticks、cgroup 不变，heartbeat 为 3→49→94→115→136→160，实际读取 stdin 计数均 0；7 次 attach 均回收。
- 3 个自身 units 与关联 tmux-spawn scope inactive；旧 server/pane 身份不再存活，已知 ROOT 删除，本地 raw cache 数为 0。最终只读核查当前 UID、0700、非符号链接的 history/bridge 临时目录，匹配数 0，没有批量删除未知资源。

所有本次 exec sessions 已结束；没有提交、推送、归档或修改产品/生产配置。本报告不宣称 W02 全部完成。
