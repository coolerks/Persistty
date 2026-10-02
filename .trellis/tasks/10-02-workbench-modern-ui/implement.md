# 执行计划

1. 读取前端规范与参考图，记录视觉方案和复用查询。
2. 保存规划并依据用户「规划，然后执行」授权启动任务。
3. 修改工作台主题 token、面板框、分隔器、标签与树条目，保留状态和生命周期。
4. 补充可观察的浏览器几何/交互验收，复跑 workbench-interactions 与 editor-recovery 专项。
5. 运行 `npm run lint`、`npm run typecheck`、`npm test`、`npm run build`（web/）；`go test ./...`、`go vet ./...`；检查 Markdown 本地链接与 `git diff --check`。
6. 对照源图与渲染截图完成 design-qa.md，记录检查结果并更新主题规范。浏览器 fixture 结果不记为 Debian/真机验收。

## 回滚点
仅产品样式和侧栏语义属性需回滚，测试和报告随任务保留。无需数据库或服务迁移。用户已要求规划后执行，本轮无额外待决产品问题。
