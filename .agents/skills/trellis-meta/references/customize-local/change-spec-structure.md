# 修改本地规范结构

用户希望修改 AI 遵循的工程约定、添加规范层或调整 monorepo 包映射时，应编辑 `.trellis/spec/` 和 `.trellis/config.yaml`。

## 先读取这些文件

1. `.trellis/config.yaml`
2. `.trellis/spec/`
3. `.trellis/workflow.md` 的规划产物指引与第 3.3 步
4. 当前任务的 `implement.jsonl` / `check.jsonl`

## 常见需求

| 需求 | 修改位置 |
| --- | --- |
| 添加后端、前端、文档或测试规范层 | `.trellis/spec/<layer>/` 或 `.trellis/spec/<package>/<layer>/` |
| 添加共享思考指南 | `.trellis/spec/guides/` |
| 调整 monorepo 包 | `.trellis/config.yaml` 中的 `packages` |
| 修改默认包 | `.trellis/config.yaml` 中的 `default_package` |
| 控制规范扫描范围 | `.trellis/config.yaml` 中的 `spec_scope` |
| 让任务读取新规范 | 任务的 `implement.jsonl` / `check.jsonl` |

## 添加规范层

单仓库示例：

```text
.trellis/spec/security/
├── index.md
└── auth.md
```

monorepo 示例：

```text
.trellis/spec/webapp/security/
├── index.md
└── auth.md
```

`index.md` 应包含：

- 此层适用哪些代码。
- 开发前检查清单。
- 质量检查。
- 指向具体指南文件的链接。

## 更新上下文

添加规范不意味着每个任务都会自动读取它。当前任务必须在 JSONL 中引用：

```bash
python3 ./.trellis/scripts/task.py add-context <task> implement ".trellis/spec/webapp/security/index.md" "Security conventions"
python3 ./.trellis/scripts/task.py add-context <task> check ".trellis/spec/webapp/security/index.md" "Security review rules"
```

## 修改 monorepo 包

`.trellis/config.yaml` 示例：

```yaml
packages:
  webapp:
    path: apps/web
  api:
    path: apps/api
default_package: webapp
```

编辑后运行：

```bash
python3 ./.trellis/scripts/get_context.py --mode packages
```

通过输出确认 AI 能看到正确的包与规范层。

## 注意事项

- 规范是用户项目约定，可按项目需要修改。
- 不要把临时任务信息放入规范；临时信息应放入任务。
- 不要只把长期约定放在代理或命令中；应保存在规范里。
- 修改规范结构后，检查已有任务 JSONL 是否仍指向存在的文件。
