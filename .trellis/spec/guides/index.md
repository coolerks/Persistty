# Persistty 思考指南

本目录只提供跨层/复用检查清单；具体签名、字段和错误语义由 backend/frontend spec 持有，不在此重复。规范更新通过 trellis-update-spec，任务/PRD 只在 .trellis/tasks；不维护第二份 TODO。Trellis 框架 scripts/skills/codex 不属于 Persistty 产品实现范围。

| 指南 | 何时使用 |
| --- | --- |
| [跨层检查](cross-layer-thinking-guide.md) | 改 API/WS/metadata/文件版本/状态与资源生命周期 |
| [复用检查](code-reuse-thinking-guide.md) | 新 helper/decoder/组件，修改相同概念的多消费者 |

开发前：读取 task 相关索引，再跟链接读 owner spec。review：只采纳有复现/路径/数据流证据的问题，核实信任边界，不能把假设当已证实故障。bootstrap 路径均为约定，后续源码存在后补真实引用。

执行方式遵守 [单代理契约](../execution-policy.md)：禁用 subagent，研究、实现和检查由主会话完成，不以并行或独立评审为由绕过。

## 文档中文化与 Trellis 兼容性

- 自然语言说明、模板提示和用户界面文案使用简体中文；文件名、命令、API/协议字段、技能名称及代码标识符保持原样。
- 翻译标题前检查本地脚本是否精确定位：`workflow.md` 的 `## Phase Index` / `## Phase 1: Plan`、日志的 `## Session N:` / `**Date**:`、`Total Sessions` 和自动维护表头均保留必要机器格式，旁边提供中文解释。
- 保留 `[workflow-state:*]`、平台块、`TRELLIS:START/END`、`@@@auto` 及 JSON/YAML 字段；不为中文化改写运行脚本或模板 hash。
- 文件名不变仍可能使标题锚点失效；翻译后检查本地链接/锚点、代码围栏、阶段提取与任务上下文。工具自动生成的结构可能继续使用英文；本次只翻译文档，不改变生成器。
- 上游 CLI 仓库路径和其他平台示例须注明适用范围；执行以本地 `--help` 与真实文件为准，不能把模板示例当成已存在功能。
