# 仓库分析

目标是在编写规则前发现项目的真实架构。不要从通用规范模板开始填空。先阅读代码，再让规范结构随之形成。

## 分析顺序

1. 读取现有 `.trellis/spec/` 目录树，记录哪些文件是模板、已过期或已针对项目编写。
2. 查看包清单、构建脚本、工作区配置与顶层文档，确定包与运行层。
3. 使用 GitNexus 了解执行流程、模块集群、依赖枢纽与对影响敏感的区域。
4. 使用 ABCoder 或语言原生工具获取精确签名、类型、类边界与实施示例。
5. 将任何发现转化为规范规则前，直接读取有代表性的源码与测试文件。

## 需要记录的内容

| 领域 | 问题 |
|------|-----------|
| 包边界 | 每个包负责什么？哪些导入跨越边界？ |
| 运行层 | 哪些代码属于 CLI、后端、前端、worker、共享库、测试专用或工具？ |
| 核心抽象 | 哪些类型、服务、store、命令、路由或适配器定义系统结构？ |
| 数据流 | 用户输入从何处进入、如何验证，状态在哪里持久化？ |
| 错误处理 | 如何表示、记录、展示与测试失败？ |
| 配置 | 默认值、环境配置、生成文件与模板位于何处？ |
| 测试 | 哪些测试风格可作为新工作的可靠示例？ |

## GitNexus 使用

先广泛查看，再查看具体符号：

```text
gitnexus_query({query: "CLI command execution flow"})
gitnexus_query({query: "template generation and migration"})
gitnexus_context({name: "SymbolName"})
gitnexus_cypher({query: "MATCH (n)-[r]->(m) RETURN n.name, type(r), m.name LIMIT 30"})
```

使用 GitNexus 结果寻找重要文件与流程。核对相关源码文件前，不要将图输出作为最终权威依据引用。

## ABCoder 使用

规范需要精确代码形态时，使用 ABCoder：

```text
list_repos()
get_repo_structure({repo_name: "package-name"})
get_file_structure({repo_name: "package-name", file_path: "src/example.ts"})
get_ast_node({repo_name: "package-name", node_ids: [{mod_path: "...", pkg_path: "...", name: "SymbolName"}]})
```

ABCoder 尤其适合记录构造函数模式、函数签名、类型契约与引用链。

## 分析笔记

分析时保留简短笔记，包含：

- 包或层名称。
- 定义本地模式的文件。
- 规范应传达的规则。
- 在旧代码、注释、测试或迁移路径中发现的反模式。
- 应创建、删除、重命名或合并的规范文件。
