# D06/D07 提交方案（已批准）

## 范围
本轮仅新增真实 Go/PTY/WS 隔离探针、历史/TUI 机制对照、Debian 编排/脱敏证据、回归与任务/规范同步。不改变产品路由、UI、生产配置、系统服务或 root helper。第一批 5f392ed 与用户 9bf86d8 保留。用户最新明确“提交代码，并继续开发”，批准按下列范围提交，不推送；后续新开发另行验证与记录。

建议提交信息：`补充 W02 终端桥接与历史 TUI 隔离实验`。

## 文件清单
- `.trellis/spec/backend/bridge-validation.md`、`history-validation.md`、`index.md`、`quality-guidelines.md`、`terminal-lifecycle.md`、`websocket-protocol.md`、`remote-validation.md`。
- `.trellis/spec/frontend/editor-terminal-lifecycle.md`。
- `.trellis/tasks/09-26-requirements-research/implement.md`。
- `.trellis/tasks/09-27-debian-spike/` 的 `prd.md`、`design.md`、`implement.md`、`implement.jsonl`、`check.jsonl`、`summary.md`、`go-bridge-report.md`、`bridge-runner-report.md`、`bridge-check-report.md`、`history-report.md`、`history-check-report.md`、`research/history-tui-plan.md`、本提交方案。
- `tests/integration/debian/bridgego/` 的 README、Go 源码/测试、独立 go.mod/go.sum、完整 third-party-notices.txt。
- `tests/integration/debian/bridge/` 的 README、probe.py、test_probe.py、首轮 evidence.json 与独立最终 evidence-review.json。
- `tests/integration/debian/history/` 的 README、独立 package.json/package-lock.json、完整通知、固定 workload/probe/解析代码与回归、首轮 evidence.json 和独立 evidence-review.json。
- `tests/integration/debian/run_remote.py` 与 `terminal/test_remote_config.py` 的 bridge 接入/回归。

仅按最终 git diff 路径逐项暂存上述文件，不使用含私有文件的批量添加。.env、二进制、缓存、临时报告与远端运行资源永不纳入。提交前再次检查私密字段、忽略规则与 diff。

## 门禁与保留项
验证及最终 binary hash 见 [桥接独立检查](bridge-check-report.md) 和 [历史独立检查](history-check-report.md)；结果边界见 [总结](summary.md)。一个真实 curses 程序的当前画面已验证，直接历史拼接反例已留证，生产 snapshot/live、浏览器/完整 TUI、多端控制、受限 CLI 和 helper 等仍未验收。W02 仍 in_progress，不归档、不宣布完整首版完成。授权仅覆盖本提交方案已完成批次，不把后续开发自动并入此提交。
