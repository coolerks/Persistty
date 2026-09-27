# 快照截点与序号模型实验

## 1. 范围与触发条件
W02 D08 是独立离线机制实验：单串行 owner 为字节/resize 赋本地序号，对照公开 headless/serialize 快照与 tail。它不是产品协议、Node sidecar 或 tmux durable offset；序号无缺口不代表终端解析状态完整恢复。产品 snapshot/live 门禁仍保持。

## 2. 签名
复现入口：`python3 -B tests/integration/debian/run_remote.py snapshot --binary /tmp/bridge-probe`。连接仅由 [私密配置 owner](remote-validation.md) 读取根 .env，复用授权的私有 history 资源隔离；解析模块见 [snapshot README](../../../tests/integration/debian/snapshot/README.md)。独立 npm 锁 headless 5.5.0、serialize 0.13.0 与其 xterm 5.5.0 peer，不改产品依赖。

模型快照 `{epoch, through_seq, cols, rows, serialized}`；tail 返回 `ok`、`epoch_mismatch`、`invalid_cursor` 或 `resync_required`，有效尾部携带同 epoch 连续事件。具体签名属于实验模块，不注册 HTTP/WS 产品端点。

## 3. 契约
bytes、resize、snapshot 同队列；已完成 write callback/resize 后才发布 seq，snapshot 只覆盖它之前已应用的事件。observer 写入前检查 epoch/seq；duplicate 零 write/resize、gap 停止应用直到新有效快照，旧快照不能回退同 epoch 游标。ring 同时限制事件数和字节数，单事件超限拒绝；已裁剪的 tail 返回 resync，不能跳过缺口。ring 有界与待处理队列有界是不同要求，需分别测试。

实验 owner/observer 待处理队列限 64 请求、2 MiB（包含正在执行者），超限同步返回 queue_full，不分配 seq、不触发 write/resize；成功或失败都在 finally 释放容量。入队 payload 必须复制，尤其 restore 快照不能由调用者在等待期间修改；bytes 单事件大小先检查再复制，避免未验证的大分配。

快照恢复到相同尺寸，await write callback 后再应用 tail/resize。仅使用公共 addon API，不复制私有 parser 状态、不剥 ANSI、不合并 prefix/suffix 掩盖差异。callback 不承诺 UTF-8/CSI/OSC 已完成；serialize 未覆盖全部终端状态。固定完整序列边界也不能自动推出 scroll-region/charset 等状态已保留。

真实输入只选择新一次单 WS 固定合成记录，不拼多个 attach 为连续流。raw 私有暂存后删除，证据仅 hash、计数、差异字段和脱敏资源身份。有限截点的 `equivalent` 与“已执行比较”分别记录，负面对照成立不称恢复成功。

## 4. 验证与错误矩阵
| 场景 | 判定 |
| --- | --- |
| 完整基础快照 + 连续 tail | 比较两个 buffers/cells/width/style/wrapped/cursor/baseY/modes，不只 marker |
| UTF-8/CSI/OSC pending 后快照 | 记录差异反例；OSC 还比较固定合成 title 事件，不更新 DOM |
| 完整 scroll-region/charset | 实际对照，不预设完整序列即可恢复 |
| duplicate/旧 epoch/gap | 副作用前忽略或拒绝，gap 后不自动续写 |
| ring 裁剪/过期 snapshot | 明确 resync，不越过缺口、不无限缓存 |
| 输入文件/帧/事件/尺寸超限 | 副作用前拒绝，不采集用户正文或泄露私密配置 |
| 真实记录某截点不同 | 保存字段类别；cells 整体不同不直接断定可见文字丢失 |

## 5. 正常、基础、错误用例
正常：固定完整基本边界恢复后，再按连续 seq 应用 tail，与连续 owner 状态相同。基础：公共 loadAddon/serialize/restore smoke。错误：帧已被 write callback 处理就认为 parser pending 清空，或序号正确就把快照恢复标为无损。

## 6. 所需测试
独立 npm ci/test、Python 安全入口/限额/证据回归；根 Go 与 bridgego 基线分别 test/race/vet，不把模块排除误报为已覆盖。Node 必须覆盖完整正面、pending 负面、mode/region/charset 限制、序号与 ring/队列容量边界。独立 Debian 重跑保留单独证据，精确清理 unit/scope/原资源身份/ROOT/raw cache，不覆盖 D06/D07。

有限库状态比较、离线 observer、单记录赋序号不解除浏览器 renderer/实时多端、Web 重启或无网页期间 output owner/history、生产容量/背压、最终解析引擎与 snapshot schema 门禁。

## 7. 错误与正确示例
错误：`await terminal.write(chunk) -> serialize -> live suffix` 被认为任意截点都安全。正确：串行序号只保证事件顺序；另验证解析状态保存能力，保留未完成序列及完整状态差异，生产恢复设计不得直接采用本轮未通过的公共 serialize 全状态假设。
