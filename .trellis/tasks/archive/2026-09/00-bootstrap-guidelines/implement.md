# 执行与验收

> 2026-09-30：用户已明确要求完成并归档本任务；归档范围与保留的验收限制见 [验收报告](check-report.md)。下文实施阶段状态保留为历史记录。

1. 阅读 AGENTS.md、workflow、全部现有 spec/task 及 skills；确认无产品源码。
2. 补齐本任务需求/设计/执行文档和研究证据，绑定现有 in_progress task。
3. 单一负责人按 trellis-spec-bootstrap 更新所有 project specs；不编辑产品代码或框架。
4. 用 trellis-check 全范围审阅：覆盖、链接、占位、语言、七段契约、跨层一致性。
5. 完成 trellis-update-spec 判断，将高风险研究结果落入归属 spec。
6. Phase 3.4 展示提交分组和未识别 dirty files，按 workflow 请求一次确认。
7. 确认后提交本任务文件，再按 trellis-finish-work 归档、记录 journal。
8. bootstrap 完成后，按已授权范围创建 Persistty v0.1 父/子任务并进入规划；复杂开发任务仍须最终计划审批。

## 验证
- python3 .trellis/scripts/task.py validate 00-bootstrap-guidelines
- 扫描 .trellis/spec 中模板占位、空章节及陈旧英文框架指南。
- 检查所有本地 Markdown 链接和索引文件集合。
- git diff --check（新文件另检查末尾换行/尾随空白）。
- 无 go.mod/package.json，不运行或宣称产品测试已通过；文档质量与产品验收分开。

## 回滚点
规范完成、检查完成、用户确认提交三个节点分别留 task 记录；未批准时保持 in_progress，不提前归档或开始产品实现。

## 当前结果
2026-09-26：26 份规范与全范围 trellis-check 已通过；check-report.md 留有覆盖与复验记录。产品测试不适用，Spike 未运行。PRD 文档验收项均已完成。当前停留 Phase 3.4，等待一次性提交方案确认；未 commit、未 archive，之后才进入 trellis-finish-work 和产品任务规划。
