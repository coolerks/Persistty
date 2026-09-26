# 修改本地技能、命令、提示与工作流

用户希望修改 AI 入口、自动触发规则或显式命令行为时，应编辑本地平台目录中的技能、命令、提示或工作流。

编辑前，先分类要修改的技能：

- **上游内置技能**：`trellis-meta`、`trellis-spec-bootstrap`、`trellis-session-insight`、`trellis-channel`。权威来源位于 Trellis CLI 仓库的 `packages/cli/src/templates/common/bundled-skills/<name>/`；`trellis init` / `trellis update` 时由 `getBundledSkillTemplates()` 自动分发到每个平台的技能根目录。本地修改由 `.trellis/.template-hashes.json` 跟踪，下次更新时会标记。
- **项目本地技能**：`.{platform}/skills/` 下的其他技能。由用户拥有，`trellis update` 不会刷新它们。

本文件其余部分使用“技能”指本地文件；这两种情况的覆盖与冲突规则不同。

## 先读取这些文件

1. `.trellis/workflow.md`
2. 目标平台的技能、命令、提示或工作流目录
3. 相关代理或钩子文件
4. 检查 `.trellis/spec/` 中是否已有项目规则
5. `.trellis/.template-hashes.json`：确认将要编辑的技能属于上游管理（有记录）还是项目本地（无记录）

## 选择哪种入口

| 目标 | 建议 |
| --- | --- |
| AI 应自动了解某项能力 | 添加或修改技能。 |
| 用户希望手动通过命令触发 | 添加或修改命令、提示或工作流。 |
| 团队项目约定 | 优先使用 `.trellis/spec/` 或项目本地技能，不要写入内置技能目录。 |
| 为用户自己的项目调整内置技能（如 `trellis-meta`） | 创建不同名称的项目本地同级技能来覆盖意图，或编辑 `.trellis/spec/`。内置技能目录的修改需要在每次 `trellis update` 时选择保留（keep）才能继续维护。 |
| 将变更贡献回上游 | 编辑 Trellis CLI 仓库中的 `packages/cli/src/templates/common/bundled-skills/<name>/`，而不是部署后的副本。 |
| 修改 Trellis 流程语义 | 同步 `.trellis/workflow.md`。 |

## 修改技能

技能通常具有以下结构：

```text
<skill-name>/
├── SKILL.md
└── references/
```

`SKILL.md` 应简短，负责触发与路由。较长内容放到 `references/`，让 AI 按需读取。

frontmatter 的 description 应说明何时使用技能。例如：

```yaml
description: "定制本项目的部署工作流与发布检查清单时使用。"
```

不要使用“有用的项目技能”这类含糊描述，它们可能导致错误触发。

### 内置技能与项目本地技能

相同的目录结构对应两种非常不同的所有权模型：

| 方面 | 内置（`trellis-meta`、`trellis-spec-bootstrap`、`trellis-session-insight`、`trellis-channel`） | 项目本地 |
| --- | --- | --- |
| 权威来源 | Trellis CLI 仓库中的 `packages/cli/src/templates/common/bundled-skills/<name>/` | 用户项目内部 |
| 分发 | `trellis init` / `trellis update` 时，由 `getBundledSkillTemplates()`（`packages/cli/src/templates/common/index.ts`）自动分发到每个平台技能根目录 | 用户或其他技能创建，不会被移动 |
| 哈希跟踪 | 每个文件记录在 `.trellis/.template-hashes.json` 中；更新时提示冲突 | 不跟踪 |
| 本地编辑 | 允许，但下次更新时会标记为“用户已修改” | 可直接编辑 |
| 正确的定制方式 | 添加一个名称不同的项目本地新技能，补充或替代内置技能 | 直接编辑文件 |

如果目标是“让项目 AI 在讨论发布说明时采用不同做法”，通常应创建项目本地技能，而不是修改 `trellis-meta/`。

## 修改命令、提示或工作流

显式入口应说明：

- 用户如何触发。
- 需要读取哪些 `.trellis/` 文件。
- 需要运行哪些脚本。
- 完成后如何报告。

如果命令只是重复工作流规则，应让它引用或读取 `.trellis/workflow.md`，而不是再维护一份流程副本。

## 常见路径

| 平台 | 入口目录 |
| --- | --- |
| Claude Code | `.claude/skills/`、`.claude/commands/` |
| Cursor | `.cursor/skills/`、`.cursor/commands/` |
| OpenCode | `.opencode/skills/`、`.opencode/commands/` |
| Codex | `.agents/skills/`、`.codex/skills/` |
| Gemini CLI | `.agents/skills/`、`.gemini/commands/` |
| Kiro | `.kiro/skills/` |
| Qoder | `.qoder/skills/`、`.qoder/commands/` |
| CodeBuddy | `.codebuddy/skills/`、`.codebuddy/commands/` |
| GitHub Copilot | `.github/skills/`、`.github/prompts/` |
| Factory Droid | `.factory/skills/`、`.factory/commands/` |
| Pi Agent | `.agents/skills/` |
| Reasonix | `.reasonix/skills/`（无独立命令目录；斜杠命令由平台内置） |
| ZCode | `.zcode/skills/`、`.zcode/commands/` |
| Kilo / Antigravity / Devin | 工作流与技能 |

上述目录都是四个内置技能的部署目标。每个平台在 `trellis init` 时接收完整副本，在 `trellis update` 时刷新；无需手动接线。

## 添加项目本地技能

用户希望记录团队私有定制时，应创建项目本地技能。不要把项目私有内容写进内置技能目录，因为 `trellis update` 会覆盖它。

```text
.claude/skills/project-trellis-local/
└── SKILL.md
```

多平台项目应在各平台技能目录中添加等价版本，或在支持共享层的平台（Codex、Gemini CLI）上使用 `.agents/skills/`。

名称不能与内置集合冲突：

- `trellis-meta`
- `trellis-spec-bootstrap`
- `trellis-session-insight`
- `trellis-channel`

重复使用名称会导致下次更新时 `getBundledSkillTemplates()` 覆盖项目本地副本。常见做法是添加项目名称前缀，如 `acme-trellis-deploy`、`acme-trellis-onboarding`。

## 注意事项

- 不要将所有平台的语法混到一个文件中。
- 不要只修改一个平台入口却声称支持所有平台。
- 不要将长期工程约定藏在命令里；应写入 `.trellis/spec/`。
- 不要手动编辑任何 `.{platform}/skills/` 目录下的 `trellis-meta/`、`trellis-spec-bootstrap/`、`trellis-session-insight/` 或 `trellis-channel/`，并期待变更自动持久保留；它们是内置技能，会由 `trellis update` 刷新。应贡献上游，或添加项目本地技能补充它们。
- `trellis update` 对内置技能文件报告“由你修改”的冲突后，只有接受手动维护差异时才选择 **keep**；否则接受覆盖，并将意图重新实现为项目本地技能。
