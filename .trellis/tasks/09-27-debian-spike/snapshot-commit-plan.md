# D08 提交方案（已批准）

## 范围
D06/D07 已在 965b4c6 提交，用户同时要求继续开发；本方案仅列后续 D08 新修改，不更改已提交证据。提交消息：`补充 W02 快照截点与有界序号模型实验`。用户最新明确“提交一下代码吧”，批准本批本地提交，不推送或归档；开发继续保持暂停。

## 文件
- `.trellis/spec/backend/snapshot-validation.md`、`index.md`、`history-validation.md`、`websocket-protocol.md` 及 `.trellis/spec/frontend/editor-terminal-lifecycle.md`。
- `.trellis/tasks/09-26-requirements-research/implement.md`。
- 当前任务 `prd.md`、`design.md`、`implement.md`、`implement.jsonl`、`check.jsonl`、`summary.md`、`research/snapshot-stream-plan.md`、`snapshot-report.md`、`snapshot-check-report.md`、本方案。
- `tests/integration/debian/snapshot/` 的 README、package.json/package-lock.json、完整许可证通知、model/scenarios/analyze、Node/Python 回归及三份独立脱敏摘要证据。
- `tests/integration/debian/run_remote.py` 的最小 snapshot 入口。

提交前按实际文件路径逐项暂存并复核，不加入 .env、node_modules、缓存、raw 字节或二进制。无未识别修改时才提交上述批次，不 amend 或 push。

## 验证与边界
详见 [独立检查](snapshot-check-report.md) 与 [总结](summary.md)。基础完整场景与序号模型有限测试通过，pending/region/charset 和真实 alternate.cells 差异均保留为反例；它们不代表产品无损恢复通过。不确定生产 sidecar/协议，不解锁完整 W02 或浏览器/多端/重启 history 门禁。
