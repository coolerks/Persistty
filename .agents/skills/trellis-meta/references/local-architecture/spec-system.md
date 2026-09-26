# 本地规范系统

`.trellis/spec/` 是用户项目特有的工程规范库。Trellis 的目标不是让 AI 记住约定，而是注入相关规范，或要求 AI 在合适的时间读取它们。

## 目录模型

常见单仓库结构：

```text
.trellis/spec/
├── backend/
│   ├── index.md
│   └── ...
├── frontend/
│   ├── index.md
│   └── ...
└── guides/
    ├── index.md
    └── ...
```

常见 monorepo 结构：

```text
.trellis/spec/
├── cli/
│   ├── backend/
│   │   ├── index.md
│   │   └── ...
│   └── unit-test/
│       ├── index.md
│       └── ...
├── docs-site/
│   └── docs/
│       ├── index.md
│       └── ...
└── guides/
    ├── index.md
    └── ...
```

`index.md` 是每个层的入口，应列出开发前检查清单与质量检查。具体指南放在同目录的其他 Markdown 文件中。

## 包配置

`.trellis/config.yaml` 可声明包：

```yaml
packages:
  cli:
    path: packages/cli
  docs-site:
    path: docs-site
    type: submodule
default_package: cli
```

AI 可运行：

```bash
python3 ./.trellis/scripts/get_context.py --mode packages
```

此命令列出当前项目的包与规范层。配置上下文 JSONL 时，应以此输出为依据。

## 规范如何进入任务

任务进入实施前，如果除了任务产物还需要规范或研究上下文，规划阶段可将相关规范写入 `implement.jsonl` / `check.jsonl`：

```jsonl
{"file": ".trellis/spec/cli/backend/index.md", "reason": "CLI backend conventions"}
{"file": ".trellis/spec/cli/unit-test/conventions.md", "reason": "Test expectations"}
```

子代理或平台前置指令读取这些 JSONL 并加载引用的规范。无子代理支持的平台中，AI 应按工作流直接读取相关规范。

## 规范应包含什么

规范应包含项目可执行的工程约定，而非通用最佳实践：

- 文件应放在哪里。
- 错误处理应如何表达。
- API、钩子与命令的输入输出契约。
- 禁止使用的模式。
- 需要测试的情况。
- 项目特有的陷阱与避免方式。

AI 在实施或调试中学到新规则时，应更新 `.trellis/spec/`，而不是仅在聊天中总结。

## 本地定制位置

| 需求 | 修改位置 |
| --- | --- |
| 添加规范层 | `.trellis/spec/<package>/<layer>/index.md` 与对应指南文件。 |
| 修改 monorepo 规范映射 | `.trellis/config.yaml` 中的 `packages` / `default_package` / `spec_scope`。 |
| 修改 AI 实施前读取的规范 | 任务的 `implement.jsonl`。 |
| 修改 AI 检查时读取的规范 | 任务的 `check.jsonl`。 |
| 修改何时更新规范 | `.trellis/workflow.md` 第 3.3 步与 `trellis-update-spec` 技能。 |

## 边界

`.trellis/spec/` 是用户项目规范，而非 Trellis 内置模板的永久副本。AI 应鼓励用户根据实际项目代码更新规范，不要将 Trellis 默认模板视为不可修改的文档。
