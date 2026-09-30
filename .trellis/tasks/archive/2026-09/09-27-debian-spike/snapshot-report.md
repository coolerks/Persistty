# D08 实施报告

## 修改与依赖

新增独立 [snapshot 实验](../../../../../tests/integration/debian/snapshot/README.md)：manifest/lock、完整 MIT 通知、串行 owner/observer 模型、Node 比较/测试、Python runner 回归、两份脱敏摘要。`run_remote.py` 新增 snapshot 入口，以默认不变的 history runner 取得新一次固定记录，使用独立 analyzer，严格只选 `tui120` 一次连接。产品包、Go bridge/record 协议与 D06/D07 既有证据没有修改。没有提交、推送、归档。

实际安装锁定 headless 5.5.0 / serialize 0.13.0 / peer xterm 5.5.0，npm 官方 registry、lock integrity 和实际包 manifest 对照；没有 legacy-peer-deps 或关闭校验。公共 `loadAddon`、`serialize`、同尺寸新实例 `write` callback 的 smoke 通过，未调用 DOM/HTML API或复制私有 parser 状态。完整许可证通知在实验目录保留；headless/addon 发布包省略 LICENSE，使用固定 5.5.0 源码 tag 通知并对照 peer 包 LICENSE。

## 模型与固定场景

- bytes/resize/snapshot 串行排队，应用完成才发布 seq；epoch 防不同 owner 混流，duplicate 零 write/resize、gap 停止、stale snapshot 不回退。
- ring event/byte 双上限与单事件拒绝、序号范围/epoch 校验、tail 被裁剪时返回 resync；确定性快照排队后填满 ring，与完整边界 fresh snapshot + tail 对照。
- 五个完整基础场景等价：smoke、普通 history/Unicode/SGR/wrapped、alternate 退出+resize、resize 完成后、有限公共 modes。
- 七个 pending 截点均不等价：UTF-8 前 1/2 字节，CSI 参数、ESC，OSC BEL 前、ST 前与 ST 中 ESC 后。OSC BEL/ST 前截点屏幕相同，但合成 title 事件丢失，不能只比较画面。
- 完整 scroll-region、charset 场景仍不等价，序列完整也不是全状态保存证明。比较两 buffers 的行/光标/baseY/cells/宽度/基本样式/wrapped、active、尺寸、公共 modes与 title事件；未过滤转义、合并 suffix 或补写序列掩盖差异。

## 真实 Debian 结果

运行两次新的隔离 history workload，经既有 Go/PTY/WS record 取得固定记录。最终 binary SHA256 `c294742fb8848421b3da04b956199327df981c36ebcdf5b9ee2c2b6319e16501`；没有重建或修改 Go。

最终直接保存 runner 的脱敏 JSON [evidence.json](../../../../../tests/integration/debian/snapshot/evidence.json)。首轮为注明整理方式的 [evidence-first.json](../../../../../tests/integration/debian/snapshot/evidence-first.json)，最终增加差异类别后复跑。两次同一合成连接记录的 hash/大小相同，但各自是独立执行，不拼接证据或 raw 帧。

单连接 `tui120` 为 120x40，5 帧 2013 bytes，SHA256 `f9b0acc415fdd9fa13c3fed7c8c2cdea9259835f12b2027f6da0e4459591c955`。五个帧界 0/1/2/4/5 和第 1 帧的人工 split 1/2/419/838 共九个有限截点中，八个等价；帧界 5 后 snapshot、零 tail 不等价，差异类别 `alternate.cells`。这不是 seq gap，说明库序列化状态对当前对照仍不足；未进一步将 cells 整体差异认定为可见文字丢失。`real_single_connection_compared` 仅代表执行对照，不代表恢复验收通过。

最终六样本 server/pane PID/start_ticks/cgroup 不变，心跳 2→50→95→116→137→161，读取 stdin 计数均 0。自身三个 units 与关联 tmux-spawn scope inactive，已知 ROOT 删除；本地 raw 0600 cache 删除。未碰默认 socket、系统服务、WireGuard/Nginx、root helper；没有读取 process env 代替私有 `.env`。

## 检查与保留门禁

- npm install / `npm ci --ignore-scripts --offline`（使用已填充隔离 npm cache）、`npm test`，8 项 Node 测试通过，无 skip。
- Python snapshot 4 + history 6 + bridge 6 + terminal/remote 21，共 37 项通过。
- 根与独立 bridgego `go test ./...`（基线缓存结果）、`go vet ./...` 通过；两模块 `go test -race -count=1 ./...` 实际重跑通过。
- `git diff --check` 通过；所有本次 exec sessions 已退出。未配置 JS/Python额外 lint 或 TypeScript typecheck，不冒称这些门禁已运行。

本轮仍是 W02 局部机制实验。公开 serialize 的 parser pending、scroll region、charset及真实 alternate cells 差异阻止任意截点恢复定案。没有选择生产 Node sidecar/schema、浏览器渲染、live 多观察者/controller、Web 重启 history owner、无人查看期间输出与完整容量策略；Landlock/helper 也未验收。runner SIGKILL/本机退出/网络错误没有持久账本，不保证所有故障零残留。本轮只核实成功执行的已知资源清理，未知 ROOT 不通配删除。main 还需独立检查后更新规范及提交方案。
