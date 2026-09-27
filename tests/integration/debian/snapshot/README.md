# D08 快照与连续输出截点实验

这是独立离线机制实验，不是产品终端协议或 Node sidecar。真实数据仅来自新一轮授权隔离 history 探针中的 `tui120` 单次 WS 连接；七次 attach 不拼接为连续输出，D06/D07 证据不覆盖。

## 复现

在本目录使用 `npm ci --ignore-scripts`、`npm test`。锁定 `@xterm/headless@5.5.0`、`@xterm/addon-serialize@0.13.0`，并显式安装满足 addon peer 的 `@xterm/xterm@5.5.0`。真实安装 manifest 与 lock integrity 留存；完整 MIT 通知在 [third-party-notices.txt](third-party-notices.txt)。headless/addon 发布包没有 LICENSE，通知取固定源码 tag，并与 browser peer 包 LICENSE 对照。

已授权 Debian 时从仓库根运行：

```sh
python3 -B tests/integration/debian/run_remote.py snapshot --binary /tmp/bridge-probe
```

仅由 [remote_config.py](../remote_config.py) 读取私有 `.env`，不读进程环境替代连接配置。复用 [history 探针](../history/README.md) 的全新 0700 临时目录、私有 socket、精确用户 unit/scope 以及正常 finally 清理。raw 只写入 mkstemp 0600 缓存文件，分析后删除；结果只有版本、字节数、哈希、差异类别和判断。Node 分析有 45 秒限时、2 MiB 文件读取上限，选择记录最多 4096 帧、256 KiB，总计至多九个截点；尺寸限制 200x100，不安装远端 Node。

## 模型契约

一个 owner 的 bytes、resize、snapshot 都进入同一队列。已完成 write callback/resize 后才推进 seq；snapshot 的 `through_seq` 只包含排在它之前且已应用的事件。所有序号仅是本地模型序号，不是 tmux durable offset。每个生命周期不同 epoch。

observer 恢复到快照原尺寸后接 tail，先检查 epoch/连续 seq，再 write/resize；duplicate 不重新应用，gap 停止直到 fresh snapshot，旧 snapshot 不回退游标。ring 同时限制事件数和总 bytes，单事件最多 64 KiB，超出 byte cap 也拒绝；丢失所需 tail 返回 `resync_required`，不跳至 newest。测试覆盖快照排队后 ring 被后续事件填满的确定性场景，fresh resync 只对固定完整基本边界证明一致。

owner/observer 各自的待处理队列另限 64 个请求、2 MiB payload（包含正在应用的请求），超限同步拒绝 `queue_full`，不会分配 seq 或调用 write。请求成功或失败均释放容量，调用者需等待或显式处理拒绝；实验 producer 顺序 await。bytes 与恢复快照在入队时复制，调用者后续修改不影响应用；保留 ring 有界不等于待处理队列有界。

## 断言与限制

正面对照比较两个 buffers 的 cells/宽度/基本属性、wrapped、行数、光标、baseY、active、尺寸及公共 modes；OSC 还比较合成 onTitleChange 事件，不更新 DOM。五个固定完整基础边界（smoke、普通历史/Unicode/SGR/wrapped、alternate 退出+resize、resize、有限 modes）一致。

七个 UTF-8/CSI/ESC/OSC 中途截点都有恢复差异；仅等 write callback 不清除 parser pending。OSC BEL/ST 的屏幕可以相同而 title 事件缺失。两个额外完整序列场景（scroll-region、charset）也不等价。因此序号无缺口、语义序列完整、公共 serialize 可调用都不代表全状态可恢复。没有过滤 ANSI、合并 suffix、复制私有 parser 状态或补写控制序列来消除差异。

真实截点结果允许不一致，`real_single_connection_compared` 只表示有限对照执行完成，不表示所有截点恢复成功。各截点 `equivalent` 和 `differing_fields` 独立留证，成功测试包含成功观测反例。首轮观测第五帧后的无 tail 恢复仍不同，不是 seq gap；随后加差异类别复跑，最终结果见 [evidence.json](evidence.json)。最终 5 帧 2013 bytes：帧界 0/1/2/4 一致，帧界 5 的差异为 `alternate.cells`（包括字符、宽度和基本属性的整体对照，未进一步断定是可见文字丢失）；第 1 帧人工 byte split 1/2/419/838 的最终状态一致。不能把八个成功截点推广为完整库状态支持或任意帧可恢复。

未验证浏览器 renderer、实时多观察者/controller、Web 重启后的 history owner、无页面期间输出、全部终端模式/scroll-region/charset/标题/自定义 OSC、生产解析库与快照 schema。runner 本机退出/SIGKILL/网络失败没有持久恢复账本，不保证任意故障零残留，禁止用通配删除补偿。
