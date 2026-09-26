# 复用检查清单

先用 `rg` 搜索目标概念及消费者，确认是否已经有 owner。

- 路径验证/安全文件操作应复用 workspace/file owner，不能上传/ZIP/search 各自拼一套较弱逻辑。
- JSON/WS/rg/git payload 由一个 decoder/parser 归属；组件不重复 `as` 取字段。阅读 [类型](../frontend/type-safety.md)、[命令](../backend/process-guidelines.md)。
- theme 颜色与 mode 单一来源；palette/context menu/快捷键调用同一 command。
- 同值不一定同概念；仅在需要同步变化且职责相同时抽取，不按重复次数机械建万能 helper。
- 短期 attach 清理与长期 tmux close 不能抽成一个“通用销毁”函数；[生命周期](../backend/terminal-lifecycle.md)。
- Search preview 与 apply 共用替换计算，但仍分别验证文件 version；[搜索契约](../backend/transfer-search-git.md)。
- 修改共享字段时检查 Go DTO/TS decoder/fixture/API doc/error/UI/test 全链；抽象后仍测试行为。

发现新惯例更新 owner spec/index；不改写 Trellis template sync 或 framework 代码来实现产品复用。
