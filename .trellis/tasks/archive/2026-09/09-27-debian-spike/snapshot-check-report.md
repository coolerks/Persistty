# D08 独立复核报告

## Findings (fixed)

- File: [model.mjs](../../../../../tests/integration/debian/snapshot/model.mjs)。Issue: ring 有界但串行 Promise 待处理队列无容量限制，producer 不等待时可无限保留 payload。Fix: owner/observer 分别限制 64 个待处理请求与 2 MiB payload（含正在应用的请求），超限同步 `queue_full`，不推进 seq、不调用 write；成功/失败均 finally 释放。observer 超大 bytes 在复制前拒绝。回归分别验证两端 event/byte 上限及失败后释放。
- File: [model.mjs](../../../../../tests/integration/debian/snapshot/model.mjs)。Issue: restore 入队闭包保留调用者快照对象，可在执行前被修改而改变序号/尺寸/内容。Fix: 入队前复制字段及冻结字符串值；回归修改原对象后恢复状态仍与连续 owner 一致。已有 bytes/返回 tail 复制行为保留。
- File: [analyze.test.mjs](../../../../../tests/integration/debian/snapshot/analyze.test.mjs)。Issue: CLI 文件权限/非普通文件/符号链接/大小及尺寸边界缺少直接回归。Fix: 增补子进程拒绝测试和尺寸边界；测试资源仅自有临时目录。
- File: [README.md](../../../../../tests/integration/debian/snapshot/README.md)。Fix: 同步 pending 队列限额与 ring 独立边界、拒绝和请求复制契约。未改产品接口或既有 D06/D07 证据。

## Findings (not fixed)

- 公开 serialize API 没有保证 parser pending 保存：七个 UTF-8/CSI/ESC/OSC 中途截点均观测差异；OSC BEL/ST 前的屏幕虽相同，合成 title 事件丢失。完整 scroll-region、charset 场景也不同。属于库能力/生产设计判断，不复制私有状态、不补写或过滤 ANSI 来掩盖。任意截点恢复保持未验收。
- 最终真实记录帧界 5 的零 tail 恢复差异为 `alternate.cells`；序号无缺口仍不等价。不将整体 cells 差异进一步认定为可见文字丢失，也不把其余八个截点推广为通用恢复支持。
- 本地 offline epoch/seq 是模型分配，不是 tmux durable offset；未验证 live observer、浏览器、Web 重启 history owner 或无人观看时的输出持久性。最终生产解析库/schema/服务边界需后续设计，不在本轮决定。
- runner SIGKILL/本机退出/网络故障没有持久资源恢复账本；finally 与成功清理不能保证所有故障零残留。此为既有实验限制，未扩大为生产 runner 或修改公共 transport。

## Verification

- Lint: 根与独立 bridgego `go vet ./...` 通过；修改 Node module 的 `node --check`、`git diff --check` 通过。独立 JS/Python 没有 lint 脚本，不冒称额外 lint 已执行。
- TypeCheck: 本轮为 `.mjs`/Python 实验，无 TypeScript/typecheck 脚本，不适用；未修改产品前端。
- Tests: 根与独立 bridgego 分别 `go test -count=1 ./...`、`go test -race -count=1 ./...` 通过，均实际重跑。bridgego 初次受沙箱 loopback 监听限制，允许本地 PTY/httptest 后通过，没有 skip。
- Tests: snapshot 11 项 Node 测试、history 2 项 Node 测试通过；snapshot/history/bridge/terminal+remote Python 共 37 项通过，无 skip。14 个合成场景：5 个完整基础边界等价，7 个 pending 反例、2 个额外完整状态限制，不将反例算成恢复验收通过。
- Dependencies: `npm ci --ignore-scripts` 显式官方 registry 按 lock 安装成功，再用独立已填充 cache 执行 offline ci 成功；最初用错误 cache/默认 mirror 的 offline 安装失败，不计通过。实际 manifests 为 headless 5.5.0 / serialize 0.13.0 / browser peer 5.5.0、MIT，peer `^5.0.0` 满足。lock SHA512 integrity 由 npm ci 校验，完整 notice 与 peer 包 LICENSE 一致；headless/addon 公共 loadAddon/serialize/restore smoke 通过，不调用私有 parser API或 DOM。
- Context: 当前 implement/check JSONL 均 7/7 validate 通过；文档本地链接、隐私扫描及清理结果如下。

## 最终独立 Debian 证据

自修后新一次成功执行的直接脱敏 runner 输出保存于 [evidence-review.json](../../../../../tests/integration/debian/snapshot/evidence-review.json)，不覆盖实施者 evidence-first/evidence 或 D06/D07。source binary SHA256 `c294742fb8848421b3da04b956199327df981c36ebcdf5b9ee2c2b6319e16501`，没有修改 Go binary。额外一次较早成功复核仅用于检查自修，最终文件对应最后一次执行。

`tui120` 单 WS 连接 120x40，5 帧 2013 bytes，SHA256 `f9b0acc415fdd9fa13c3fed7c8c2cdea9259835f12b2027f6da0e4459591c955`。帧界 0/1/2/4 等价，帧界 5 不等价（`alternate.cells`）；第 1 帧人工 split 1/2/419/838 等价，共 9 个有限截点、8 等价/1 不同。只比较这一连接，不拼接各次 attach，不提交 raw、cells 或 serialized 正文。

六样本 server PID 937314/start_ticks 916362469、pane PID 937317/start_ticks 916362480 及各自 cgroup 不变；心跳 2→50→95→116→137→161 每次增长，stdin 计数始终 0。server/pane 不在 Web cgroup，raw 来自固定合成 workload，不采集业务内容。

最终自身 client/web/tmux 三单位及关联 `tmux-spawn` scope 均 inactive，已知 ROOT 删除；额外只读复核原 server/pane PID+start_ticks 身份均已消失。限定当前 UID、0700、非 symlink 的隔离实验前缀目录未发现未知项，未泛删。两次 reviewer 执行均精确清理；原始本地 history cache 文件数 0，所有本次 exec sessions 已结束。

私有 `.env` 仅由安全 parser 内存读取，不采用 process env；tracked/untracked 可提交文本与 HEAD 实际连接值扫描均 0 命中，仅输出路径/字段/count。`.env` 未跟踪，未输出实际值、argv/stderr/HOME/UID。未使用 sudo、默认 tmux socket或更改系统服务/网络/root helper。W02 保持局部实验状态，未提交、推送、归档。
