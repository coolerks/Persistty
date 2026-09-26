# 中文化与规范补充审查报告

审查日期：2026-09-26。范围为 AGENTS.md、README.md、`.trellis/**/*.md` 和 `.agents/skills/**/*.md`。依据本任务 PRD、design、implement、check.jsonl、研究上下文及 `/private/tmp/persistty-chinese-original/` 原文快照审查；没有实施产品功能、提交或归档任务。

## 已修复的发现

- 文件：`.trellis/spec/frontend/state-management.md`。
  问题：状态归属表的文件系统与 Git 两行仍使用自然语言 `server snapshot cache`。
  修复：改为“服务端快照缓存”；状态权威来源和缓存行为不变。
- 文件：`.agents/skills/trellis-session-insight/SKILL.md`。
  问题：触发模式引用仍称示例包含英语与中文，但本任务已将全部表达译成中文。
  修复：改为“用户表达示例（已统一译为中文）”；主会话从审查暂存目录安装，技能名称和触发语义不变。
- 文件：`.agents/skills/trellis-channel/SKILL.md`。
  问题：引用页已有 CLI 0.6.17 兼容性说明，入口仍以现在时称 CLI 帮助提供 `--tag`，容易把旧模板规则当作当前参数。
  修复：在入口补同一版本边界，说明旧模板来源、当前 `--kind` 及中断事件；保留原命令例子，不修改 CLI。

## 未修复的发现

无本次范围内尚未解决的文档问题。未发现需要主会话作产品或架构决定的新增冲突。

旧模板命令及上游源码路径保留用于对照，并已有适用边界说明；这不构成对应命令在本地全部可运行的保证。自动日志生成器仍可能产生英文机器结构，本次按要求保留生成器及定位格式。

## 语义审查

- 原有任务创建、规划审查、代理分派与递归豁免、机械修复边界、提交确认、禁止自动 push 等约束保留；技能 name、frontmatter 键及平台/状态/自动区块标记保持兼容。
- Shell 可执行行与原快照一致；其他带语言的代码围栏仅发现两处 YAML description 的自然语言翻译，未改变代码、字段或参数。翻译解释性 Markdown 模板和注释符合设计边界。
- 工作流保留脚本精确依赖的 `Phase Index` 与 `Phase 1: Plan` 标题，旁边提供中文解释；工作区保留 `Total Sessions`、`Session N`、`Date` 与自动维护表头。
- Explorer 操作补充引用既有安全根、版本冲突和草稿规则，源/目标独立校验、默认零覆盖、移动失败保留源、删除确认与部分失败均有验收归属。
- 工作区恢复继续以 SQLite 元数据、真实文件/tmux 观测和 IndexedDB 草稿分别为权威，不把缓存当资源存活事实，不保存正文或终端输出。
- 图片/SVG/PDF/二进制预览复用既有认证、隔离、大小和资源释放约束；编辑 10 MiB、预览 50 MiB 与配置初始上限一致，运行值仍由配置提供。
- README 明确当前没有产品实现、终端生命周期仍待 Spike 及真实 Debian/systemd 验证，并说明主机重启不能恢复进程内存。

## 验证

- 文档 Lint：通过。原文集合保留、末尾换行、尾随空白、围栏闭合、本地文件链接/锚点、机器标签与 Shell 示例对照均通过；`git diff --check` 通过。新增未跟踪文档也由逐文件检查覆盖。
- 文档结构检查：通过。包含本报告及主会话提交方案共 91 份 Markdown、80 个本地链接、6 个 JSON 围栏、13 个工作流步骤和 8 个状态块；检查脚本退出码为 0。
- 上下文验证：通过。`python3 .trellis/scripts/task.py validate 09-26-chinese-docs` 检查 implement/check 各 1 个真实条目，退出码为 0。
- 主会话补验：实际 `.codex/hooks/inject-workflow-state.py` 使用当前会话身份返回中文 in_progress 正文；12 个技能 frontmatter 键集合保持不变，均通过。
- TypeCheck：不适用。没有 TypeScript 产品代码或 tsconfig，本次未改程序。
- 产品测试：不适用。没有 Go/React 产品源码或依赖清单，未运行 unit/integration/Playwright/systemd 验收。文档解析检查通过不等于产品运行或安全验收通过。

文档质量门禁通过；后续功能实现仍须在所属任务中锁定完整 HTTP/迁移契约，并执行规范要求的运行测试。
