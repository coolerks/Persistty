# 综合实施与验收

1. 用户在后续消息明确批准最新最终规划；批准前所有任务保持 planning，不运行 task.py start 或编辑产品代码。
2. 读取 trellis-before-dev；激活语言子任务（inline 允许空 JSONL，遵循实际 CLI选项），按[语言清单](../09-30-monaco-language-coverage/implement.md)实现并检查。
3. 顺序激活图标子任务，按[图标清单](../09-30-material-file-icons/implement.md)实现，复用同一轻量路径 adapter；不得分派代理。
4. 主会话用 trellis-check 进行两个子任务及父任务综合自审；对最终代码执行前端四门禁，按 AGENTS.md 运行 Go 回归门禁。专项浏览器用例使用现有 web/tests/e2e 和显式隔离实例，记录实际执行命令与结果。
5. 联合验收：全部 91 个语言可达及 token 加载、无后缀模式选择、yaml/yml一致、d.ts最长后缀、树/标签一致、完整 SVG引用/许可、深浅主题/worker本地加载、模型/终端稳定及窄屏无溢出。
6. 更新 frontend editor-terminal-lifecycle/theme-assets/state owner中相关契约；检查 Markdown 本地链接、规范一致性、忽略规则和 git diff --check。只维护与本功能相关契约，不搬改旧规范中的无关历史。
7. 形成验收记录，明确未运行/失败/环境缺项；用户未要求提交或归档，保留任务与可审阅 diff。无产品修改时规划阶段不运行产品测试或宣称通过。
