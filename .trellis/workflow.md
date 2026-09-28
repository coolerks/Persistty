# 开发工作流

---

## 核心原则

1. **先规划，再编码** — 开始前先明确要做什么
2. **注入规范，不凭记忆** — 通过 hook/skill 加载规范，不靠记忆复述
3. **全部落盘** — 研究、决策和经验写入文件；对话会压缩，文件不会
4. **增量开发** — 每次处理一个任务
5. **沉淀经验** — 每个任务后复盘，将新知识写回规范

---

## Trellis 系统

### 开发者身份

首次使用时初始化身份：

```bash
python3 ./.trellis/scripts/init_developer.py <your-name>
```

创建 `.trellis/.developer`（不纳入 Git）和 `.trellis/workspace/<your-name>/`。

### 规范系统

`.trellis/spec/` 按包和层组织开发规范。

- `.trellis/spec/<package>/<layer>/index.md` — 包含**开发前检查清单**和**质量检查**的入口。详细规范位于索引链接的 `.md` 文件。
- `.trellis/spec/guides/index.md` — 跨包思考指南。

```bash
python3 ./.trellis/scripts/get_context.py --mode packages   # 列出包与层
```

**更新规范的时机**：发现新模式或约定、需要固化缺陷预防规则、作出新的技术决策。

### 任务系统

每个任务在 `.trellis/tasks/{MM-DD-name}/` 下有独立目录，保存 `task.json`、`prd.md`、按需建立的 `design.md`、`implement.md`、`research/`，以及支持子代理的平台使用的上下文清单（`implement.jsonl`、`check.jsonl`）。

```bash
# 任务生命周期
python3 ./.trellis/scripts/task.py create "<title>" [--slug <name>] [--parent <dir>]
python3 ./.trellis/scripts/task.py start <name>          # 设置活动任务（有会话身份时按会话隔离）
python3 ./.trellis/scripts/task.py current --source      # 显示活动任务及来源
python3 ./.trellis/scripts/task.py finish                # 清除活动任务（触发 after_finish hook）
python3 ./.trellis/scripts/task.py archive <name>        # 移入 archive/{year-month}/
python3 ./.trellis/scripts/task.py list [--mine] [--status <s>]
python3 ./.trellis/scripts/task.py list-archive

# 可执行规范上下文（通过 JSONL 注入 implement/check 代理）。
# 支持子代理的平台在 task create 时创建空的 implement.jsonl / check.jsonl。
# AI 在规划阶段填写真实规范与研究条目。清单仍为空时 validate 失败，start 拒绝启动，
# 避免子代理没有规范上下文。确实有意跳过时可显式使用 start --allow-empty-context。
python3 ./.trellis/scripts/task.py add-context <name> <action> <file> <reason>
python3 ./.trellis/scripts/task.py list-context <name> [action]
python3 ./.trellis/scripts/task.py validate <name>

# 任务元数据
python3 ./.trellis/scripts/task.py set-branch <name> <branch>
python3 ./.trellis/scripts/task.py set-base-branch <name> <branch>    # PR 目标分支
python3 ./.trellis/scripts/task.py set-scope <name> <scope>

# 父子层级
python3 ./.trellis/scripts/task.py add-subtask <parent> <child>
python3 ./.trellis/scripts/task.py remove-subtask <parent> <child>

# PR 创建（是否支持以本地 --help 为准）
python3 ./.trellis/scripts/task.py create-pr [name] [--dry-run]
```

> 执行 `python3 ./.trellis/scripts/task.py --help` 查看当前本地可用命令；该列表为权威来源。本仓库脚本不提供 `create-pr` 子命令，上述行为属于模板示例，不能直接执行。

**当前任务机制**：`task.py create` 创建任务目录；存在会话身份时自动设置本会话的活动任务指针，立即触发规划提示。`task.py start` 写入同一指针（重复设置时幂等），将 `task.json.status` 从 `planning` 改为 `in_progress`。状态存放于 `.trellis/.runtime/sessions/`。若 hook 输入、`TRELLIS_CONTEXT_ID` 或平台原生会话环境变量均未提供上下文键，则没有活动任务，`task.py start` 会失败并提示配置会话身份。`task.py finish` 删除当前会话文件，但不改变任务状态。`task.py archive <task>` 写入 `status=completed`，将目录移入 `archive/`，并删除仍指向该任务的运行时会话文件。

### 工作区记录系统

在 `.trellis/workspace/<developer>/` 下记录 AI 会话，便于跨会话追踪。

- `journal-N.md` — 会话日志，**每文件最多 2000 行**；超过后自动创建 `journal-(N+1).md`。
- `index.md` — 个人索引（会话总数、最近活跃时间）。

```bash
python3 ./.trellis/scripts/add_session.py --title "Title" --commit "hash" --summary "Summary"
```

### 上下文脚本

```bash
python3 ./.trellis/scripts/get_context.py                            # 完整会话运行时信息
python3 ./.trellis/scripts/get_context.py --mode packages            # 可用包与规范层
python3 ./.trellis/scripts/get_context.py --mode phase --step <X.Y>  # 单个工作流步骤的详细指引
```

---

<!--
  工作流状态提示契约（修改下方标签块前必读）

  Phase Index 中的 [workflow-state:STATUS] 块是各 AI 平台 UserPromptSubmit hook
  每轮 <workflow-state> 提示的唯一来源。inject-workflow-state.py 和 OpenCode 的
  inject-workflow-state.js 仅读取解析；v0.5.0-rc.0 后脚本不内置备用字典。

  STATUS 字符集：[A-Za-z0-9_-]+。找不到标签时降级为通用“参考 workflow.md 当前步骤”
  提示，刻意保持可见，以便用户发现并修复损坏的 workflow.md。

  不变量（上游 test/regression.test.ts）：每个 [required · once] 步骤都必须在所属
  阶段的状态块中有对应执行要求。状态提示是唯一每轮通道，漏写会导致 AI 静默跳过；
  阶段 1 规划门禁与阶段 3.4 提交遗漏都曾由此发生。

  标签与阶段范围：
    [workflow-state:no_task] → 无活动任务，阶段 1 前
    [workflow-state:task_error] → 活动任务记录不可读，修复后再继续
    [workflow-state:planning] → 阶段 1 全部（status='planning'）
    [workflow-state:planning-inline] → Codex 内联阶段 1
    [workflow-state:in_progress] → 阶段 2 与阶段 3.2–3.4
      （task.py start 至 task.py archive 期间保持 in_progress）
    [workflow-state:in_progress-inline] → Codex 内联阶段 2/3
    [workflow-state:completed] → 当前正常流程不触发：cmd_archive 在同一调用中改状态
      并移动目录，解析器随即失去指针；保留供将来显式 in_progress→completed 转换使用。

  修改清单：
    - 修改状态块后同步核对所属阶段的 [required · once] 步骤。
    - 上游模板维护者通过 trellis update 向下游项目分发块级修改。
    - 上游完整运行时契约：.trellis/spec/cli/backend/workflow-state-contract.md
      （仅是来源说明，本 Persistty 仓库没有该 CLI 包规范）。
-->

## Phase Index

阶段索引（标题为脚本精确定位标记，保留英文）。

```
阶段 1：规划 → 分类、征得建任务同意、编写规划产物
阶段 2：执行 → 任务状态为 in_progress 后才能实现
阶段 3：收尾 → 验证、更新规范、提交、结束会话
```

### 请求分类

- 简单对话或小任务：只询问本轮是否创建 Trellis 任务。用户拒绝时，本会话跳过 Trellis。
- 复杂任务：询问是否允许创建 Trellis 任务并进入规划。用户拒绝时，不进行大范围内联实现；解释情况、明确范围或建议拆小。
- 用户同意创建任务不等于同意开始实现，仍须先完成规划。

### 规划产物

- `prd.md` — 需求、约束和验收标准，不在此放技术设计或执行清单。
- `design.md` — 复杂任务的技术设计：边界、契约、数据流、取舍、兼容性及发布/回滚方式。
- `implement.md` — 复杂任务的执行计划：有序清单、验证命令、审查门禁和回滚点。
- `implement.jsonl` / `check.jsonl` — 子代理上下文使用的规范与研究清单，不代替 `implement.md`。
- 轻量任务可仅有 PRD；复杂任务必须在 `task.py start` 前准备 `prd.md`、`design.md` 和 `implement.md`。

### 父子任务树

一个请求包含多个可独立验收的交付物时，使用父任务。父任务负责原始需求、任务映射、跨子任务验收标准和最终集成审查；除非自身有直接工作，否则通常不作为实现目标。

用子任务管理能独立规划、实现、检查和归档的交付物。父子结构不是依赖系统；一个子任务必须等待另一个时，在子任务的 `prd.md` / `implement.md` 中写明顺序，并保持验收标准可测试。

用 `task.py create "<title>" --slug <name> --parent <parent-dir>` 创建子任务；用 `task.py add-subtask <parent> <child>` 关联已有任务；错误关联用 `task.py remove-subtask <parent> <child>` 解除。

<!-- 每轮状态提示：无活动任务时显示（阶段 1 前） -->

[workflow-state:no_task]
没有活动任务。先对当前请求分类，创建任何 Trellis 任务前先征得用户同意。
简单对话/小任务：只询问本轮是否创建 Trellis 任务。用户拒绝时，本会话跳过 Trellis。
复杂任务：询问是否允许创建 Trellis 任务并进入规划。用户拒绝时，解释情况、明确范围或建议拆小。
[/workflow-state:no_task]

<!-- 每轮状态提示：活动任务记录不可读时显示 -->

[workflow-state:task_error]
无法读取活动任务记录。不要创建或激活另一任务。
检查上述任务目录并修复 task.json；它必须是有效 JSON 对象，且 status 非空。
保留已有字段和产物。无法安全确定正确状态时，重建记录前询问用户。
[/workflow-state:task_error]

### 阶段 1：规划
- 1.0 创建任务 `[required · once]`（必需、一次；先征得建任务同意）
- 1.1 需求探索 `[required · repeatable]`（必需、可重复；形成 `prd.md`，复杂任务另需 `design.md` + `implement.md`）
- 1.2 研究 `[optional · repeatable]`（可选、可重复）
- 1.3 配置上下文 `[required · once]`（必需、一次）— Claude Code、Cursor、OpenCode、Codex、Kiro、Gemini、Qoder、CodeBuddy、Copilot、Droid、Pi、Oh My Pi、ZCode、Snow、Reasonix、Grok、Kimi Code（仅子代理分派平台；内联平台跳过）
- 1.4 激活任务 `[required · once]`（必需、一次；审查通过后运行 `task.py start`，状态变为 in_progress）
- 1.5 完成标准

<!-- 每轮状态提示：阶段 1 全程显示（status='planning'） -->

[workflow-state:planning]
加载 `trellis-brainstorm`，保持规划阶段。
轻量任务可仅有 `prd.md`；复杂任务完成 `prd.md`、`design.md` 和 `implement.md`，在 `task.py start` 前请求审查。
多个交付物时，考虑父任务与可独立验收的子任务；依赖必须写入子任务产物，不能靠树的位置暗示。
子代理模式：开始前填写 `implement.jsonl` 和 `check.jsonl` 的规范/研究清单。
[/workflow-state:planning]

<!-- 每轮状态提示：codex.dispatch_mode=inline 时的阶段 1；主会话直接实现，跳过 JSONL 整理，改由 trellis-before-dev 读取上下文。 -->

[workflow-state:planning-inline]
加载 `trellis-brainstorm`，保持规划阶段。
轻量任务可仅有 `prd.md`；复杂任务完成 `prd.md`、`design.md` 和 `implement.md`，在 `task.py start` 前请求审查。
多个交付物时，考虑父任务与可独立验收的子任务；依赖必须写入子任务产物，不能靠树的位置暗示。
内联模式：跳过 JSONL 清单整理；阶段 2 通过 `trellis-before-dev` 读取产物与规范。
[/workflow-state:planning-inline]

### 阶段 2：执行
- 2.1 实现 `[required · repeatable]`（必需、可重复）
- 2.2 质量检查 `[required · repeatable]`（必需、可重复）
- 2.3 回滚 `[on demand]`（按需）

<!-- 每轮状态提示：in_progress 期间覆盖阶段 2 和阶段 3.2–3.4；状态从 start 到 archive 保持不变，因此正文必须覆盖实现至提交全部必需步骤。 -->

子代理分派协议适用于全部平台和子代理：包括 Codex 原生 `SubagentStart` 注入及子端拉取备用方式、第二类 Gemini/Qoder/Copilot/Reasonix/Trae/Grok/Kimi Code、使用 hook 的 ZCode/Snow 和 `trellis-research`。每个分派提示必须先写 `Active task: <task path from task.py current>`，再写角色指令。Grok Build 使用 `spawn_subagent`，`subagent_type` 为 Trellis 代理名（如 `trellis-implement`）；Kimi Code 分派内置 `coder` / `explore`，附对应 `.kimi-code/skills/trellis-<role>/SKILL.md` 指令。

[workflow-state:in_progress]
工具：`trellis-implement` / `trellis-research` 仅是子代理类型（Task/Agent 工具，不是技能，没有同名 skill）；`trellis-update-spec` 是技能；`trellis-check` 两种形式都有，代码修改后优先使用 Agent 形式验证。
流程：`trellis-implement` → `trellis-check` → `trellis-update-spec` → 提交（阶段 3.4）→ `/trellis:finish-work`。
主会话默认分派 implement/check 子代理。子代理自身豁免：已作为 `trellis-implement` 运行时，不再启动 `trellis-implement` 或 `trellis-check`；已作为 `trellis-check` 运行时，不再启动 `trellis-check` 或 `trellis-implement`。仅主会话负责分派。
分派提示以 `Active task: <task path from task.py current>` 开头。读取顺序：JSONL 条目 → `prd.md` → `design.md`（如有）→ `implement.md`（如有）。
[/workflow-state:in_progress]

<!-- 每轮状态提示：codex.dispatch_mode=inline 时由主会话直接编辑，替代子代理分派模式。 -->

[workflow-state:in_progress-inline]
流程：`trellis-before-dev` → 编辑 → `trellis-check` → 验证 → `trellis-update-spec` → 提交（阶段 3.4）→ `/trellis:finish-work`。
本项目禁用所有 subagent：研究、实现和检查都由主会话直接完成，不分派 implement/check/research，不用 channel worker 或其他 AI 会话绕过。普通继续/检查请求不解除禁用，约束见 AGENTS.md 与 .trellis/spec/execution-policy.md。
读取顺序：`prd.md` → `design.md`（如有）→ `implement.md`（如有），以及技能加载的相关规范和研究。
[/workflow-state:in_progress-inline]

### 阶段 3：收尾
- 3.2 调试复盘 `[on demand]`（按需）
- 3.3 规范更新 `[required · once]`（必需、一次）
- 3.4 提交修改 `[required · once]`（必需、一次）
- 3.5 收尾提醒

> 说明：步骤 3.1 已合并到 2.2（最后一轮全范围检查）和 3.4（提交前检查）。保留编号以免破坏外部引用。

<!-- 每轮状态提示：completed；正常流程暂不触发，因为 cmd_archive 同时写状态和移目录，解析器失去指针。保留供将来显式状态转换使用，修改方式与其他状态块相同。 -->

[workflow-state:completed]
代码已提交。运行 `/trellis:finish-work`；若仍有未提交修改，先返回阶段 3.4。
[/workflow-state:completed]

### 规则

1. 确定当前阶段，再从该阶段的下一步骤继续
2. 每个阶段按顺序执行，不能跳过 `[required]` 必需步骤
3. 阶段可以回退（如执行发现 PRD 缺陷，则返回规划修复后再执行）
4. `[once]` 步骤已有产物时跳过，不重复运行
5. 根据产物是否存在决定下一步；轻量任务可缺少 `design.md` / `implement.md`，复杂任务缺少则表示规划未完成。

### 活动任务路由

活动任务中用户请求匹配以下意图时，先选择路由，再按需加载阶段详细步骤。

[Claude Code, Cursor, OpenCode, codex-sub-agent, Kiro, Gemini, Qoder, CodeBuddy, Copilot, Droid, Pi, Oh My Pi, ZCode, Snow, Reasonix, Trae, Grok, Kimi Code]

- 规划或需求不明确 → `trellis-brainstorm`。
- `in_progress` 的实现/检查 → 分派 `trellis-implement` / `trellis-check`。
- 反复调试 → `trellis-break-loop`；规范更新 → `trellis-update-spec`。

[/Claude Code, Cursor, OpenCode, codex-sub-agent, Kiro, Gemini, Qoder, CodeBuddy, Copilot, Droid, Pi, Oh My Pi, ZCode, Snow, Reasonix, Trae, Grok, Kimi Code]

[codex-inline, Kilo, Antigravity, Devin, DeepSeek Harness]

- 规划或需求不明确 → `trellis-brainstorm`。
- 编辑前 → `trellis-before-dev`；编辑后 → `trellis-check`。
- 反复调试 → `trellis-break-loop`；规范更新 → `trellis-update-spec`。

[/codex-inline, Kilo, Antigravity, Devin, DeepSeek Harness]

### 流程约束

- 建任务批准不等于实现批准；产物审查后运行 `task.py start`，才能开始实现。
- 轻量任务可仅有 PRD；复杂任务需要 `design.md` + `implement.md`。
- 规划必须保存到任务产物；报告完成前必须运行检查。

### 加载步骤详情

每个步骤运行以下命令获取详细指引：

```bash
python3 ./.trellis/scripts/get_context.py --mode phase --step <step>
# 例如 python3 ./.trellis/scripts/get_context.py --mode phase --step 1.1
```

---

## Phase 1: Plan

阶段 1：规划（标题为脚本精确定位标记，保留英文）。

目标：分类请求，需要任务时征得建任务同意，形成实现前必需的规划产物。

#### 1.0 创建任务 `[required · once]`（必需、一次）

征得建任务同意后才创建任务目录。命令将状态设为 `planning`，写入 `task.json`，创建默认 `prd.md`；有会话身份时自动将新任务设为当前任务：

```bash
python3 ./.trellis/scripts/task.py create "<task title>" --slug <name>
```

`--slug` 只填写便于阅读的名称，**不要**包含 `MM-DD-` 日期前缀；`task.py create` 会自动添加。

任务树先创建父任务，再用 `--parent <parent-dir>` 创建子任务。不能因为有子任务就启动父任务；应启动负责下一可独立验收交付物的子任务。

命令成功后，每轮状态提示自动切换为 `[workflow-state:planning]`，要求 AI 保持规划阶段。

此处只运行 `create`，不要同时运行 `start`。`start` 会将状态改为 `in_progress`，若过早调用，会在产物审查前切换到实现提示；留到步骤 1.4 执行。

若 `python3 ./.trellis/scripts/task.py current --source` 已指向任务，则跳过。

#### 1.1 需求探索 `[required · repeatable]`（必需、可重复）

加载 `trellis-brainstorm`，按技能指引与用户交互探索需求。

需求探索技能要求：
- 每次只问一个问题
- 能研究确定的事实先研究，不问用户
- 优先提供选项，减少开放式问题
- 每次用户回答后立即更新 `prd.md`
- 大范围交付物可独立验收时，拆成父任务与子任务
- `prd.md` 只聚焦需求与验收标准
- 复杂任务开始实现前准备 `design.md` 和 `implement.md`

考虑拆分父子任务时：
- 一个请求包含多个可独立验收的交付物时，使用父任务。
- 父任务负责原始需求、子任务映射、跨子任务验收标准和最终集成审查。
- 子任务负责可独立规划、实现、检查和归档的实际交付物。
- 父子结构不是依赖系统。子任务 B 依赖 A 时，在 B 的 `prd.md` / `implement.md` 写明顺序。
- 启动负责下一交付物的子任务；除非父任务自身有直接实现工作，否则不启动父任务。

需求变化时返回本步骤，修订相关产物。

#### 1.2 研究 `[optional · repeatable]`（可选、可重复）

需求探索中可随时研究，不限于本地代码；可使用现有工具（MCP 服务、技能、网页搜索等）查询第三方库文档、行业实践和 API 参考等外部信息。

[Claude Code, Cursor, OpenCode, codex-sub-agent, Kiro, Gemini, Qoder, CodeBuddy, Copilot, Droid, Pi, Oh My Pi, ZCode, Snow, Reasonix, Trae, Grok, Kimi Code]

启动研究子代理：

- **代理类型**：`trellis-research`
- **任务描述**：研究 <具体问题>
- **关键要求**：研究输出必须保存到 `{TASK_DIR}/research/`

[/Claude Code, Cursor, OpenCode, codex-sub-agent, Kiro, Gemini, Qoder, CodeBuddy, Copilot, Droid, Pi, Oh My Pi, ZCode, Snow, Reasonix, Trae, Grok, Kimi Code]

[codex-inline, Kilo, Antigravity, Devin, DeepSeek Harness]

在主会话直接研究，将结果写入 `{TASK_DIR}/research/`。`codex-inline` 是明确要求工作留在主会话的模式。

[/codex-inline, Kilo, Antigravity, Devin, DeepSeek Harness]

**研究产物约定**：
- 每个研究主题一个文件（如 `research/auth-library-comparison.md`）
- 将第三方库用法示例、API 参考和版本约束写入文件
- 记录发现的相关规范路径，便于后续引用

需求探索与研究可交替进行：暂停讨论研究技术问题，再回到用户讨论。

**关键原则**：研究结果必须落盘，不能只留在聊天中；对话会压缩，文件不会。

#### 1.3 配置上下文 `[required · once]`（必需、一次）

[Claude Code, Cursor, OpenCode, codex-sub-agent, Kiro, Gemini, Qoder, CodeBuddy, Copilot, Droid, Pi, Oh My Pi, ZCode, Snow, Reasonix, Trae, Grok, Kimi Code]

整理 `implement.jsonl` 和 `check.jsonl`，使阶段 2 的子代理得到正确规范/研究上下文。模板可能包含自说明 `_example` 种子行，本地 task create 也可能创建空文件；此处必须填入真实条目。

**位置**：`{TASK_DIR}/implement.jsonl` 和 `{TASK_DIR}/check.jsonl`（已存在）。

**格式**：每行一个 JSON 对象 — `{"file": "<path>", "reason": "<why>"}`，路径相对于仓库根目录。

**应填内容**：
- **规范文件** — `.trellis/spec/<package>/<layer>/index.md` 及任务相关的具体规范（`error-handling.md`、`conventions.md` 等）
- **研究文件** — 子代理需要查阅的 `{TASK_DIR}/research/*.md`

**不应填内容**：
- 代码文件（`src/**`、`packages/**/*.ts` 等）— 子代理在实现时读取，不在此预注册
- 即将修改的文件 — 原因同上

**两个清单的分工**：
- `implement.jsonl` → 实现子代理正确编码所需的规范与研究
- `check.jsonl` → 检查子代理所需的规范（质量指引、检查约定及按需共用的研究）

清单不能代替 `implement.md`。后者是复杂任务供人阅读的执行计划；JSONL 只列出待注入或加载的上下文文件。

**如何发现相关规范**：

```bash
python3 ./.trellis/scripts/get_context.py --mode packages
```

列出所有包、规范层及路径，选择与任务领域相符的条目。

**如何追加条目**：

可直接在编辑器修改 JSONL，或运行：

```bash
python3 ./.trellis/scripts/task.py add-context "$TASK_DIR" implement "<path>" "<reason>"
python3 ./.trellis/scripts/task.py add-context "$TASK_DIR" check "<path>" "<reason>"
```

存在真实条目后可删除 `_example` 种子行（可选，读取端自动跳过）。

就绪门禁：`task.py start` 前，两个清单必须各有至少一个真实 `{"file": "...", "reason": "..."}` 条目；只有 `_example` 种子行不算就绪。

仅在两个文件均已有真实整理条目时跳过。

[/Claude Code, Cursor, OpenCode, codex-sub-agent, Kiro, Gemini, Qoder, CodeBuddy, Copilot, Droid, Pi, Oh My Pi, ZCode, Snow, Reasonix, Trae, Grok, Kimi Code]

[codex-inline, Kilo, Antigravity, Devin, DeepSeek Harness]

跳过本步骤。阶段 2 直接由 `trellis-before-dev` 加载上下文。

[/codex-inline, Kilo, Antigravity, Devin, DeepSeek Harness]

#### 1.4 激活任务 `[required · once]`（必需、一次）

产物审查后，将任务状态改为 `in_progress`：

```bash
python3 ./.trellis/scripts/task.py start <task-dir>
```

轻量任务可仅有 `prd.md`；复杂任务开始前必须有 `prd.md`、`design.md` 和 `implement.md` 且已经审查。子代理分派平台的两个 JSONL 清单也须有真实条目。运行时为兼容可能容忍清单缺失或仅有种子行，但这不表示规划就绪。

命令成功后，提示自动切换为 `[workflow-state:in_progress]`，继续阶段 2/3。

若 `task.py start` 提示会话身份错误（hook 输入、`TRELLIS_CONTEXT_ID` 或平台原生会话环境变量均未提供上下文键），按错误提示配置会话身份后重试。

#### 1.5 完成标准

| 条件 | 是否必需 |
|------|:---:|
| `prd.md` 存在 | ✅ |
| 用户确认任务进入实现 | ✅ |
| 已执行 `task.py start`（status = in_progress）| ✅ |
| `research/` 有研究产物（复杂任务）| 建议 |
| `design.md` 存在（复杂任务）| ✅ |
| `implement.md` 存在（复杂任务）| ✅ |

[Claude Code, Cursor, OpenCode, codex-sub-agent, Kiro, Gemini, Qoder, CodeBuddy, Copilot, Droid, Pi, Oh My Pi, ZCode, Snow, Reasonix, Trae, Grok, Kimi Code]

| 两个 JSONL 清单各有至少一个真实整理条目（种子行不计）| ✅ |

[/Claude Code, Cursor, OpenCode, codex-sub-agent, Kiro, Gemini, Qoder, CodeBuddy, Copilot, Droid, Pi, Oh My Pi, ZCode, Snow, Reasonix, Trae, Grok, Kimi Code]

---

## 阶段 2：执行

目标：将审查后的规划产物实现为通过质量检查的代码。

#### 2.1 实现 `[required · repeatable]`（必需、可重复）

[Claude Code, Cursor, OpenCode, codex-sub-agent, CodeBuddy, Droid, Pi, ZCode, Snow, Oh My Pi]

启动实现子代理：

- **代理类型**：`trellis-implement`
- **任务描述**：按已审查任务产物实现，参考 `{TASK_DIR}/research/`；结束前运行项目 lint 和类型检查
- **分派提示约束**：提示必须以 `Active task: <task path>` 开头，告知代理已是 `trellis-implement` 子代理，应直接实现，不再启动 `trellis-implement` / `trellis-check`。

平台 hook/plugin 自动处理：
- 读取 `implement.jsonl`，将引用的规范/研究注入代理提示
- 注入 `prd.md`，以及存在时的 `design.md` 和 `implement.md`
- Codex 使用 `SubagentStart` 原生注入；代理配置保留子端加载作为备用

[/Claude Code, Cursor, OpenCode, codex-sub-agent, CodeBuddy, Droid, Pi, ZCode, Snow, Oh My Pi]

[Gemini, Qoder, Copilot, Reasonix, Trae, Grok, Kimi Code]

启动实现子代理：

- **代理类型**：`trellis-implement`
- **任务描述**：按已审查任务产物实现，参考 `{TASK_DIR}/research/`；结束前运行项目 lint 和类型检查
- **分派提示约束**：提示必须以 `Active task: <task path>` 开头，明确代理已是 `trellis-implement`，直接实现，不再启动 `trellis-implement` / `trellis-check`。

拉取式子代理定义自动处理上下文加载：
- 用 `task.py current --source` 确定活动任务，读取 `prd.md`，再读取存在时的 `design.md` 和 `implement.md`
- 读取 `implement.jsonl`，要求代理编码前加载每个引用的规范/研究文件

[/Gemini, Qoder, Copilot, Reasonix, Trae, Grok, Kimi Code]

[Kiro]

启动实现子代理：

- **代理类型**：`trellis-implement`
- **任务描述**：按已审查任务产物实现，参考 `{TASK_DIR}/research/`；结束前运行项目 lint 和类型检查
- **分派提示约束**：告知代理已是 `trellis-implement` 子代理，应直接实现，不再启动 `trellis-implement` / `trellis-check`。

平台前置指令自动处理上下文加载：
- 读取 `implement.jsonl`，将引用的规范/研究注入代理提示
- 注入 `prd.md`，以及存在时的 `design.md` 和 `implement.md`

[/Kiro]

[codex-inline, Kilo, Antigravity, Devin, DeepSeek Harness]

1. 加载 `trellis-before-dev`，读取项目规范
2. 读取 `{TASK_DIR}/prd.md`，再读取存在时的 `design.md` 和 `implement.md`
3. 查阅 `{TASK_DIR}/research/` 资料
4. 按已审查产物实现代码
5. 运行项目 lint 和类型检查

[/codex-inline, Kilo, Antigravity, Devin, DeepSeek Harness]

#### 2.2 质量检查 `[required · repeatable]`（必需、可重复）

[Claude Code, Cursor, OpenCode, codex-sub-agent, Kiro, Gemini, Qoder, CodeBuddy, Copilot, Droid, Pi, Oh My Pi, ZCode, Snow, Reasonix, Trae, Grok, Kimi Code]

启动检查子代理：

- **代理类型**：`trellis-check`
- **任务描述**：按规范和任务产物审查全部代码修改，直接修复发现的问题，确保 lint 和类型检查通过
- **分派提示约束**：提示必须以 `Active task: <task path>` 开头，告知代理已是 `trellis-check` 子代理，直接审查/修复，不再启动 `trellis-check` / `trellis-implement`。

检查代理职责：
- 按规范审查代码修改
- 按 `prd.md` 及存在时的 `design.md`、`implement.md` 审查修改
- 自动修复发现的问题
- 运行 lint 和类型检查验证

[/Claude Code, Cursor, OpenCode, codex-sub-agent, Kiro, Gemini, Qoder, CodeBuddy, Copilot, Droid, Pi, Oh My Pi, ZCode, Snow, Reasonix, Trae, Grok, Kimi Code]

[codex-inline, Kilo, Antigravity, Devin, DeepSeek Harness]

加载 `trellis-check`，按指引验证代码：
- 规范符合性
- lint / 类型检查 / 测试
- 跨层一致性（修改跨越多个层时）

发现问题 → 修复 → 复验，直到通过。

[/codex-inline, Kilo, Antigravity, Devin, DeepSeek Harness]

**最终检查（阶段 3.4 提交前）**：任务最后一次 2.2 必须覆盖全部范围，不能只检查最新实现片段。用 `python3 ./.trellis/scripts/get_context.py --mode packages` 列出受影响包，读取各包规范索引的质量检查章节，以发现迭代中局部检查遗漏的跨层/跨包问题。

#### 2.3 回滚 `[on demand]`（按需）

- `check` 发现 PRD 缺陷 → 返回阶段 1 修复 `prd.md`，再执行 2.1
- 实现错误 → 回退代码，重新执行 2.1
- 需要更多研究 → 按阶段 1.2 研究，将结果写入 `research/`

---

## 阶段 3：收尾

目标：确保代码质量，沉淀经验，记录工作。

#### 3.2 调试复盘 `[on demand]`（按需）

任务涉及反复调试（同一问题多次修复）时，加载 `trellis-break-loop`：
- 分类根因
- 解释此前修复为何失败
- 提出预防措施

目标是沉淀调试经验，避免同类问题重现。

#### 3.3 规范更新 `[required · once]`（必需、一次）

加载 `trellis-update-spec`，判断本任务是否产生值得记录的新知识：
- 新发现的模式或约定
- 遇到的陷阱
- 新技术决策

据此更新 `.trellis/spec/` 文档；即使结论是“无需更新”，也必须完成判断。

#### 3.4 提交修改 `[required · once]`（必需、一次）

**规范同步前置检查**：拟定提交前，判断本任务是否修复缺陷或发现应写入 `.trellis/spec/` 的隐性知识，以免未来开发者或 AI 重犯。若是，先返回阶段 3.3；规范更新应包含在同一任务提交批次，不能遗忘。

AI 组织本任务修改的批量提交，便于随后执行 `/finish-work`。先提交实际工作，再提交归档与日志等记录，不交错。

**操作步骤**：

1. **检查未提交状态**：
   ```bash
   git status --porcelain
   ```
   记录所有未提交路径；工作树干净时跳到 3.5。

2. **从最近历史了解提交风格**，使拟定消息与之协调：
   ```bash
   git log --oneline -5
   ```
   记录前缀约定（`feat:` / `fix:` / `chore:` / `docs:` 等）、语言（中文/英文）和长度风格。

3. **将未提交文件分为两组**：
   - **本会话由 AI 修改** — 本会话通过 Edit/Write/Bash 工具写入或修改，已知修改内容和原因。
   - **未识别** — 本会话没有修改的未提交文件（可能为用户手动修改、前次会话未完成工作或无关修改），不能擅自纳入提交。

4. **拟定提交方案**。将 AI 修改文件按逻辑组织（每个完整变更单元一个提交，不是每文件一个），每项包含 `<提交消息>` 与文件列表；底部单列未识别文件。

5. **一次展示方案，一次确认**。格式：
   ```
   拟议提交（按顺序）：
     1. <message>
        - <file>
        - <file>
     2. <message>
        - <file>

   未识别的未提交文件（未纳入任何提交，请确认是否包含）：
     - <file>
     - <file>

   回复“ok”/“行”执行；可回复修改意见，或“我自己来”/“manual”取消。
   ```

6. **确认后**：按顺序为每批运行 `git add <files>` 和 `git commit -m "<msg>"`；不 amend，不 push。

7. **拒绝时**（用户回复“不行”/“我自己来”/“manual”或反对方案）：停止，不拟第二套方案；用户手动提交，待其确认后跳到 3.5。

**规则**：
- 任何位置都不使用 `git commit --amend`；按工作提交 → 归档提交 → 日志提交三个阶段处理。
- 此步骤绝不向远端 push。
- 用户认可分组但希望改消息时，修改消息后再确认一次；若反对分组，则转为手动提交。
- 一次展示整个批量方案，不按每个提交分别询问。

#### 3.5 收尾提醒

完成上述流程后，提醒用户可运行 `/finish-work` 收尾（归档任务、记录会话）。

---

## 定制 Trellis（供分支维护者参考）

本节面向修改 Trellis 工作流本身的开发者。通过本文件定制工作流文案；脚本只负责解析。

### 修改步骤含义

修改上方阶段 1/2/3 的相应步骤正文，保持以下关键不变量：
- 无活动任务时先分类，创建 Trellis 任务前征得用户同意。
- 规划须区分仅需 PRD 的轻量任务，与开始前必须有 `prd.md`、`design.md`、`implement.md` 的复杂任务。
- 每条必需执行路径都必须在 `/trellis:finish-work` 前保留可到达的阶段 3.4 提交提醒。

所有标签块位于上方 `## Phase Index`，紧接各阶段概要：

| 范围 | 对应标签 |
|---|---|
| 无活动任务（阶段 1 前）| `[workflow-state:no_task]`（阶段索引示意图后）|
| 活动任务记录不可读 | `[workflow-state:task_error]`（先修复已有任务再继续）|
| 阶段 1 全部（已建任务 → 准备实现）| `[workflow-state:planning]`（阶段 1 概要后）|
| Codex 内联阶段 1 | `[workflow-state:planning-inline]` |
| 阶段 2 + 阶段 3.2–3.4（实现、检查、收尾）| `[workflow-state:in_progress]`（阶段 2 概要后）|
| Codex 内联阶段 2 + 阶段 3.2–3.4 | `[workflow-state:in_progress-inline]` |
| 阶段 3.5 后（已归档）| `[workflow-state:completed]`（阶段 3 概要后，**目前正常流程不触发**）|

### 修改每轮提示文案

直接修改相应 `[workflow-state:STATUS]` 块正文。模板维护者修改后通过 `trellis update` 分发；定制本项目时重新开始 AI 会话即可，无需改脚本。

### 添加自定义状态

新增标签块：

```
[workflow-state:my-status]
每轮提示文案
[/workflow-state:my-status]
```

约束：
- STATUS 字符集：`[A-Za-z0-9_-]+`（允许下划线和连字符，如 `in-review`、`blocked-by-team`）
- 生命周期 hook 必须将 `task.json.status` 写为自定义值，否则该标签不会被读取
- 生命周期 hook 在 `task.json.hooks.after_*` 中，绑定 `after_create / after_start / after_finish / after_archive` 之一

### 添加生命周期 hook

在 `task.json` 添加 `hooks` 字段：

```json
{
  "hooks": {
    "after_finish": [
      "your-script-or-command-here"
    ]
  }
}
```

支持事件：`after_create / after_start / after_finish / after_archive`。`after_finish` 不等于状态变化，它仅清除活动任务指针；“任务完成”通知使用 `after_archive`。

### 完整契约

工作流状态机的运行时契约、所有状态写入点、伪状态（`no_task` / `stale_<source_type>`）、hook 可达性矩阵等详细内容，可参考以下模板上游入口。本项目没有 CLI 包规范；本地 hook 实际位于 `.codex/hooks/inject-workflow-state.py`，不能将上游路径当作已存在文件：

- `.trellis/spec/cli/backend/workflow-state-contract.md` — 上游运行时契约、写入点表和测试不变量（本项目未生成）
- `.trellis/scripts/inject-workflow-state.py` — 模板示意解析器路径；本地对应 `.codex/hooks/inject-workflow-state.py`（仅读取 workflow.md，无内置状态正文）
