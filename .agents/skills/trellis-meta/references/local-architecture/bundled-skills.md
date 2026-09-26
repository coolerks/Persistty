# 内置技能

“内置技能”（bundled skills）是随 Trellis CLI npm 包发布的多文件技能。它们不同于市场技能（用户另行安装到自己的 `.claude/skills/` 或其他平台技能根目录）：`trellis init` 会自动将内置技能写入每个支持平台的技能根目录，`trellis update` 负责同步。它们属于 Trellis 自身，而非第三方内容。

内置技能是 `packages/cli/src/templates/common/bundled-skills/<skill>/` 下的目录，已有自己的 `SKILL.md`（含 YAML frontmatter），以及可选的 `references/`、资源或其他辅助文件。Trellis 原样复制整个目录树到各平台技能根目录，因此引用文件仍可延迟加载，不会被摊平到一个过大的 `SKILL.md` 中。

本页中的 `packages/cli/` 与 `templates/` 路径属于 Trellis CLI 上游源码仓库，用于说明分发实现；用户项目不一定包含这些路径。

## 哪些属于内置技能，以及相邻概念

| 来源路径 | 类型 | 发布方式 |
| --- | --- | --- |
| `templates/common/bundled-skills/<name>/` | 内置技能（多文件） | 整个目录复制到每个平台技能根目录 |
| `templates/common/skills/<name>.md` | 单文件工作流技能 | 包装 frontmatter，写为 `<root>/<name>/SKILL.md` |
| `templates/common/commands/<name>.md` | 斜杠命令或提示 | 写入各平台命令目录（`.claude/commands/trellis/`、`.cursor/commands/trellis-*.md`、`.gemini/commands/trellis/*.toml` 等） |
| `templates/<platform>/skills/` | 平台特定技能 | 仅写入该平台目录（例如 `.codex/skills/`） |
| `.claude/skills/<my-skill>/` 等位置的用户技能 | 市场技能或用户编写的技能 | 完全不由 Trellis 管理 |

Trellis CLI 不会操作自身模板加载器未生成的内容。用户手动放入平台技能根目录的内容不会被触碰。

## 当前内置技能（v0.6.0）

运行时通过列举 `templates/common/bundled-skills/` 下的目录发现技能集合：

| 技能 | 用途 |
| --- | --- |
| `trellis-meta` | 本技能，向用户项目内的 AI 解释本地 Trellis 架构与定制入口。 |
| `trellis-session-insight` | 封装 `trellis mem` CLI，让 AI 了解何时以及如何检索过去的 Claude Code / Codex / Pi Agent 对话日志。 |
| `trellis-spec-bootstrap` | 从真实代码库创建或刷新 `.trellis/spec/` 的平台无关工作流，可选集成 GitNexus / ABCoder。 |
| `trellis-channel` | 能力技能，指导 AI 何时使用 `trellis channel` 进行多代理协作、论坛与线程持久看板，以及分派者等待模式。 |

此列表在运行时发现，因此在 `bundled-skills/` 下添加一个目录即可注册新技能（见下文“添加内置技能”）。

## 内置技能在各平台的落地位置

平台的完整文件集合（命令、工作流技能、代理、钩子、内置技能）只定义一次，由 `packages/cli/src/configurators/<platform>.ts` 中的 `collect<Platform>Templates()` 描述。内置技能的描述包含两个调用：`resolveBundledSkills(ctx)` 读取 `templates/common/bundled-skills/` 下的全部目录，解析占位符，返回扁平的 `{relativePath, content}` 条目列表；`collectSkillTemplates(<skillsRoot>, <workflowSkills>, <bundledSkills>)` 将其合并到平台的 `Map<filePath, content>`，路径为 `<skillsRoot>/<skill>/<relativePath>`。

全部 21 个平台都会接收完整内置技能集合：

| 平台 | 内置技能根目录 |
| --- | --- |
| Claude Code | `.claude/skills/<skill>/` |
| Cursor | `.cursor/skills/<skill>/` |
| OpenCode | `.opencode/skills/<skill>/` |
| Codex | `.agents/skills/<skill>/` |
| Gemini CLI | `.agents/skills/<skill>/` |
| Pi | `.agents/skills/<skill>/` |
| Kimi | `.agents/skills/<skill>/` |
| Kilo | `.kilocode/skills/<skill>/` |
| Kiro | `.kiro/skills/<skill>/` |
| Antigravity | `.agent/skills/<skill>/` |
| Devin | `.devin/skills/<skill>/` |
| Qoder | `.qoder/skills/<skill>/` |
| Codebuddy | `.codebuddy/skills/<skill>/` |
| Copilot | `.github/skills/<skill>/` |
| Droid | `.factory/skills/<skill>/` |
| Reasonix | `.reasonix/skills/<skill>/` |
| ZCode | `.zcode/skills/<skill>/` |
| Trae | `.trae/skills/<skill>/` |
| OMP | `.omp/skills/<skill>/` |
| Grok | `.grok/skills/<skill>/` |
| Snow | `.snow/skills/<skill>/` |

Codex、Gemini CLI、Pi 和 Kimi 共享 `.agents/skills/` 根目录（上游 Agent Skills 工作区别名）。多个收集器会写入该目录的同一个文件时，它们必须生成字节完全相同的内容。

一份描述，两个消费者：

1. `trellis init` → `configurePlatform(platformId, cwd)` → `writeTemplateMap(cwd, collect<Platform>Templates())`。21 个平台中有 18 个在 `configurators/index.ts` 的注册项直接使用 `fromTemplates(collect<Platform>Templates)`，就是这个组合。Claude Code、Codex 和 ZCode 各自显式定义 `configure`，以处理 `Map<path, content>` 无法表达的工作（可选 `--with-statusline` 参数、刻意保留为空的 `.codex/skills/` 目录、一次性控制台提示）；它们都不会重复定义文件列表。
2. `trellis update` → `collectPlatformTemplates(platformId)`（位于 `configurators/index.ts`）→ 同一份映射，用于检测差异并填充 `.trellis/.template-hashes.json`。

两个消费者读取同一份描述，因此 init 与 update 不会对内置技能生成哪些文件产生分歧。

## 分发接线（代码路径）

将内置技能自动分发到平台技能根目录的机制位于两个文件：

1. `packages/cli/src/templates/common/index.ts`
   - `listDirectories("bundled-skills")` 枚举磁盘上的技能。
   - `listBundledSkillFiles(skillDir)` 递归遍历各技能目录，为每个文件返回 `{relativePath, content}`。
   - `getBundledSkillTemplates()` 返回缓存的 `CommonBundledSkill[]`。

2. `packages/cli/src/configurators/shared.ts`
   - `resolveBundledSkills(ctx)` 将列表摊平为 `ResolvedSkillFile[]`，包含 `<skill>/<relativePath>` 路径与已解析的占位符。
   - `collectSkillTemplates(skillsRoot, workflowSkills, bundledSkills)` 将工作流技能与内置技能文件合并为以 `skillsRoot` 为根的 `Map<filePath, content>`。
   - `writeTemplateMap(cwd, files)` 是把收集到的映射写入磁盘的统一写入器。

每个支持技能的平台都会从自己的 `collect<Platform>Templates()` 调用这两个辅助函数：可直接调用（`claude.ts`、`codex.ts`、`copilot.ts`、`gemini.ts`、`grok.ts`、`kimi.ts`、`kiro.ts`、`omp.ts`、`opencode.ts`、`pi.ts`、`reasonix.ts`、`snow.ts`、`zcode.ts`），或通过 `shared.ts` 中的 `collectBothTemplates(ctx, cmdPath, skillRoot)` 调用。后者为同时具备命令目录与技能根目录的平台（`antigravity.ts`、`codebuddy.ts`、`cursor.ts`、`devin.ts`、`droid.ts`、`kilo.ts`、`qoder.ts`、`trae.ts`）执行相同的两个调用。

## 添加内置技能

结构与分发接线已通用化，因此添加技能只需要文件变更与分发验证。

1. **创建目录树。**

   ```
   packages/cli/src/templates/common/bundled-skills/<my-skill>/
     SKILL.md                     # YAML frontmatter 与正文
     references/                  # 可选
       <topic>.md
     assets/                      # 可选（可按 utf-8 读取的内容）
   ```

2. **编写有效的 `SKILL.md` 文件头。** frontmatter 至少包含：

   ```yaml
   ---
   name: <my-skill>
   description: "AI 应在何时使用此技能。这里填写触发表述。"
   ---
   ```

   各平台自动触发机制匹配的是 `description`，因此它应描述用户意图的触发条件，而不是技能内部实现。

3. **适当使用占位符。** 内置技能内容会经过 `resolvePlaceholders(file.content, ctx)`。`resolvePlaceholders` 支持的 `{{platform_name}}`、`{{python_cmd}}` 等标记会按平台替换。

4. **无需分发接线。** `listDirectories("bundled-skills")` 自动发现新目录，因此所有平台都会在下次 `trellis init` 或 `trellis update` 时接收它。

5. 发布前**验证分发路径**。历史上，遗漏这些步骤曾导致文档声称技能已内置，但发布的 npm tarball 中缺少文件：

   - 源文件存在于将要打标签的分支上。
   - `pnpm --filter @mindfoldhq/trellis build` 将资源复制到 `dist/templates/common/bundled-skills/<skill>/`。
   - `npm pack --dry-run --json` 包含预期的 `dist/**` 路径。
   - 在全新临时项目中，`trellis init` 写入 `.claude/skills/<skill>/SKILL.md`、`.agents/skills/<skill>/SKILL.md`、`.zcode/skills/<skill>/SKILL.md` 等文件。
   - `.trellis/.template-hashes.json` 列出生成的文件。
   - 在该临时项目中运行 `trellis update --dry-run`，报告 `Already up to date!`（已是最新）。

6. 如果技能在其他项目将升级到的发布版本中新增，应**添加迁移清单项**。没有显式清单项时，文件仍会通过 `trellis update` 标准的“缺失文件”分支落地，但清单可让变更在更新日志中可见。

## 本地覆盖内置技能

没有正式的“项目本地技能”机制（如 `.trellis/skills/`）。内置技能位于平台根目录，覆盖也位于平台根目录。

受支持的模式依赖 `trellis update` 已有的模板哈希差异检测：

1. 直接编辑本地文件，例如 `.claude/skills/trellis-meta/SKILL.md`。
2. 文件哈希会与 `.trellis/.template-hashes.json` 中的记录不同。
3. 下次 `trellis update` 检测到用户修改并保留文件（没有显式 `--force` 时，Trellis 不会覆盖用户修改的文件）。

注意事项：

- 覆盖只适用于所编辑目录对应的平台。例如要同时覆盖 Claude Code 与 Codex 的同一技能，必须同时编辑 `.claude/skills/<name>/` 和 `.agents/skills/<name>/`。
- 之后执行 `trellis update --force` 会覆盖本地编辑。将覆盖纳入版本控制，以便需要时重新应用。
- 同一平台技能根目录中使用不同目录名安装的市场技能（如 `.claude/skills/my-custom-meta/`）不会被 Trellis 修改。目标是添加行为而非修改内置技能时，这通常更合适。
- 团队私有约定属于 `.trellis/spec/` 或独立的市场风格本地技能，不应通过修改 `trellis-meta` 本身保存。见 `customize-local/add-project-local-conventions.md`。

## 从项目中移除内置技能

内置技能没有按项目禁用的参数。可选两种方式：

1. **在各平台技能根目录删除对应目录。** `trellis update` 会发现文件缺失，与 `.template-hashes.json` 比较，并将删除视为其他用户修改；未传 `--force` 时不会静默重建目录。

2. **固定到未发布该技能的 Trellis 版本。** 内置技能集合在构建时确定，因此安装较早的 CLI 版本是永久排除当前版本内置技能的唯一方式。

第三种方式——全局禁用所有内置技能——不受支持。分发无条件执行：`collect<Platform>Templates()` 不接收参数，因此没有参数入口。添加此功能需要修改全部 21 个平台的函数签名，以及 `configurators/index.ts` 中的 `collectPlatformTemplates`。

## 操作规则

- 以 `templates/common/bundled-skills/` 作为内置技能集合的唯一权威来源，不要手动维护逐平台技能列表。
- 不要在内置 `SKILL.md` 中加入平台特定逻辑。平台特定行为应放在 `templates/<platform>/skills/`。
- 不要让内置技能耦合特定 CLI 二进制命令（如 `trellis mem`），却不在技能描述与引用文档中说明依赖；旧版本用户可能没有该命令。
- 不要在内置技能中存储项目私有内容。内置技能公开发布给所有用户；项目规则属于 `.trellis/spec/` 或本地技能。
