# W02 本批提交清单

## 授权与消息
用户在去敏与私有 .env 修正完成后明确要求“帮我提交”。本批仅提交已完成的 W02 底层实验与保密修正，不归档、不 push，不代表完整 W02 或产品验收。

提交消息：`补充 W02 Debian 隔离探针与实测证据`

## 文件清单
- `.trellis/spec/backend/filesystem-guidelines.md`
- `.trellis/spec/backend/index.md`
- `.trellis/spec/backend/logging-guidelines.md`
- `.trellis/spec/backend/quality-guidelines.md`
- `.trellis/spec/backend/terminal-lifecycle.md`
- `.trellis/tasks/09-26-requirements-research/implement.md`
- `.trellis/tasks/09-26-requirements-research/task.json`
- `.env.example`
- `.trellis/spec/backend/remote-validation.md`
- `.trellis/tasks/09-26-requirements-research/research/debian-environment.md`
- `.trellis/tasks/09-27-debian-spike/check-report.md`
- `.trellis/tasks/09-27-debian-spike/check.jsonl`
- `.trellis/tasks/09-27-debian-spike/commit-plan.md`
- `.trellis/tasks/09-27-debian-spike/design.md`
- `.trellis/tasks/09-27-debian-spike/files-report.md`
- `.trellis/tasks/09-27-debian-spike/implement.jsonl`
- `.trellis/tasks/09-27-debian-spike/implement.md`
- `.trellis/tasks/09-27-debian-spike/prd.md`
- `.trellis/tasks/09-27-debian-spike/summary.md`
- `.trellis/tasks/09-27-debian-spike/task.json`
- `.trellis/tasks/09-27-debian-spike/terminal-report.md`
- `tests/integration/debian/files/README.md`
- `tests/integration/debian/files/landlock_linux.go`
- `tests/integration/debian/files/landlock_other.go`
- `tests/integration/debian/files/main.go`
- `tests/integration/debian/files/main_test.go`
- `tests/integration/debian/remote_config.py`
- `tests/integration/debian/run_remote.py`
- `tests/integration/debian/terminal/README.md`
- `tests/integration/debian/terminal/evidence.json`
- `tests/integration/debian/terminal/probe.py`
- `tests/integration/debian/terminal/test_probe.py`
- `tests/integration/debian/terminal/test_remote_config.py`

## 保留的用户修改
- `.gitignore`：用户手动追加的 .env 忽略行，不纳入本批，不撤销。原已提交忽略规则仍覆盖 .env。

私有 .env 本身被忽略、未跟踪，不提交；.env.example 只含虚构值。未发现其他清单外路径。

## 验证
Go test/race/vet 6 个第一方包与前轮真实 Debian 实验通过；本次隐私修正的 Python 18 项本地回归通过，未重新连接远端。可提交文本及 HEAD 实际连接值零命中。具体结果及限制见 check-report.md/summary.md。
