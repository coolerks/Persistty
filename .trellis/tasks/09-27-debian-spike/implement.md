# W02 执行与验证

1. 主会话已完成只读环境采集；批准来源为用户提供 SSH 与隔离范围之后的明确允许。整理上下文并激活子任务。
2. 终端 implement 负责 tests/integration/debian/terminal/** 和 terminal-report.md；自行核对官方 tmux/systemd 参数，远端只执行批准隔离实验。
3. 文件 implement 负责 tests/integration/debian/files/** 和 files-report.md；交叉构建 Linux 独立探针，在新建目录验证真实 os.Root，不修改产品包。
4. 实施代理不得撤回他人修改；不改 Go manifest/lock 或其他 owner 文件。main 负责文档/规范、整体审查与提交方案。
5. check 代理复核隔离与清理、报告判定、测试可重复性，运行 Go test/vet/race（涉及并发），不得为了复核触碰用户系统服务或扩大授权。
6. main 同步已证实知识及未执行门禁，提供结果/残留清单。提交仍需用户确认，不归档父任务或其他活动任务。

## 当前隐私修正
用户明确要求本批不提交，先剥离真实连接信息。后续复现仅解析仓库根私有 .env 的三个字段；当前改动新增 tests/integration/debian 共享读取/执行入口及本地 mock 回归，不重跑远端。main 维护 .env 权限、虚构 .env.example、规范及文档去敏；不修改用户实际值，不回显全文。检查 tracked/untracked 可提交文本与 HEAD 的敏感匹配只输出路径/count。历史拟议提交清单暂停，不自动扩大或提交。
