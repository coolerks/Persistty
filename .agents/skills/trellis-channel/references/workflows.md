# 工作流

> 版本边界：本项目本次检查的 Trellis CLI 为 0.6.17。本文保留模板原有命令示例以便对照，执行前以当前 `trellis channel <command> --help` 为准；包含 `--tag`、`forum list` 或 `thread show` 的旧示例不适用于当前 CLI。当前消息筛选使用 `--kind`，论坛列表为 `trellis channel forum <name>`，主题读取为 `trellis channel thread <name> <key>`。中断事件使用 `interrupt_requested` / `interrupted`；不要把旧文案中的 `interrupt` 当作当前事件名。

根据意图选择以下模式。多轮工作优先使用持久频道，单次问题优先使用 `channel run`。

## 模式 A：多轮头脑风暴

当用户说“和 codex/claude 讨论一下”、“brainstorm”或“拉一个 agent 进来一起看”时使用。

```bash
trellis channel create brainstorm-storage-layer --by main \
  --task .trellis/tasks/05-XX-storage-adapter

trellis channel spawn brainstorm-storage-layer \
  --agent architect --provider codex \
  --file .trellis/tasks/05-XX-storage-adapter/prd.md \
  --file .trellis/tasks/05-XX-storage-adapter/design.md \
  --as cx-arch --timeout 30m

trellis channel send brainstorm-storage-layer \
  --as main --to cx-arch --text-file /tmp/brainstorm-r1.md

trellis channel wait brainstorm-storage-layer \
  --as main --kind done --from cx-arch --timeout 10m
```

不要在一次回答后停止。阅读回答，找出模糊之处，发送新的追问，重复直到结果可以执行。

至少包含以下轮次：

1. 方向分岔：应放入现有机制还是新建机制？
2. MVP 边界：v1、v2，以及哪些情况会迫使 v2 的内容回到 v1。
3. 数据契约：事件、schema、元数据、状态权威来源、兼容性。
4. CLI / UX 契约：命令名、参数、错误、默认值、歧义。
5. 跨层风险与测试：共享辅助函数、容易偏离之处、阻塞发布的测试。

可选轮次：

- 运维：日志、调试、工作代理卡住、终止/重启、恢复。
- 迁移/发布：破坏性变更状态、清单、变更日志、文档站点。
- 反方审查：让协作代理提出反对当前方案的论据。

每次追问都应要求具体文件路径、命令、schema、被否决的替代方案及阻塞发布的问题。需要作出决定时，不接受含糊回避。

## 模式 B：实现 / 检查代理

当用户要求分派实现或审查工作时使用。

```bash
TASK=.trellis/tasks/05-12-foo
trellis channel create cr-foo --task "$TASK" --by main

trellis channel spawn cr-foo \
  --agent check \
  --jsonl "$TASK/check.jsonl" \
  --file "$TASK/prd.md" \
  --file "$TASK/design.md" \
  --file "$TASK/implement.md" \
  --cwd "$PWD" --timeout 15m

trellis channel send cr-foo --as main --to check --text-file /tmp/cr-brief.md
trellis channel wait cr-foo --as main --kind done --from check --timeout 15m
trellis channel messages cr-foo --kind message --from check --tag final_answer
```

实现工作使用 `--agent implement` 并发送实现简报。检查工作包含准确的差异范围、相关规范及已运行的验证。

## 模式 C：并行审查者

使用同一个频道，并为工作代理设置不同名称。

```bash
trellis channel create cr-feature --by main --ephemeral

trellis channel spawn cr-feature --agent check \
  --jsonl "$TASK/check.jsonl" --file "$TASK/prd.md" --file "$TASK/design.md" \
  --timeout 15m

trellis channel spawn cr-feature --agent check --provider codex --as check-cx \
  --jsonl "$TASK/check.jsonl" --file "$TASK/prd.md" --file "$TASK/design.md" \
  --timeout 15m

trellis channel send cr-feature --as main --to check --text-file /tmp/cr-brief.md
trellis channel send cr-feature --as main --to check-cx --text-file /tmp/cr-brief.md
trellis channel wait cr-feature --as main --kind done --from check,check-cx --all --timeout 15m
```

`--all` 表示列出的每个工作代理都必须发出匹配事件。

## 模式 D：单次工作代理

```bash
trellis channel run --provider codex --message "say hi in 3 words" --timeout 1m
trellis channel run --agent plan --message-file /tmp/plan-question.md --timeout 10m
```

成功时，`run` 会移除临时频道。发生错误、超时或被终止时，保留频道并打印路径以便检查。

## 模式 E：论坛频道

适用于问题论坛、主题式反馈、发布待办、代理发现和内部变更日志。完整模型见 `forum.md`。

## 模式 F：接手已有主题

如果用户提供论坛/主题名称，自行恢复上下文：

```bash
trellis channel forum <board> --scope global
trellis channel thread <board> <thread> --scope global --raw
trellis channel context list <board> --scope global --thread <thread>
trellis channel messages <board> --scope global --raw --thread <thread>
```

输出约束摘要，不要倾倒对话记录：

- 用户层面的问题
- 影响本仓库的上下文文件
- 当前版本与未来版本的需求
- 当前代码/设计是否满足需求
- 下一步行动或要追加的评论
