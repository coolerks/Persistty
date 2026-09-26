# 工作代理与代理卡片

> 版本边界：本项目本次检查的 Trellis CLI 为 0.6.17。本文保留模板原有命令示例以便对照，执行前以当前 `trellis channel <command> --help` 为准；包含 `--tag`、`forum list` 或 `thread show` 的旧示例不适用于当前 CLI。当前消息筛选使用 `--kind`，论坛列表为 `trellis channel forum <name>`，主题读取为 `trellis channel thread <name> <key>`。中断事件使用 `interrupt_requested` / `interrupted`；不要把旧文案中的 `interrupt` 当作当前事件名。

当协作代理需要独立执行，并通过频道事件日志汇报时，使用工作代理。工作代理是附着在频道上的已注册子进程（claude 或 codex）；监督进程转发收件箱消息，并将它的输出转换为频道事件。

## 启动

```bash
trellis channel create impl-task --by dispatcher --cwd /path/to/repo
trellis channel spawn impl-task --provider codex --as codex-impl --timeout 30m

echo "Implement the schema for table X per .trellis/.../prd.md" \
  | trellis channel send impl-task --as dispatcher --to codex-impl --stdin

trellis channel wait impl-task --as dispatcher --from codex-impl --kind done --timeout 30m
```

`spawn` 派生一个 `channel __supervisor` 工作代理，发出 `spawned`、流式输出 `progress`，并应以 `done`、`error` 或 `killed` 结束。工作代理在收件箱中保持空闲，直到 `send --to <worker>` 唤醒它（设置 `--inbox-policy broadcastAndExplicit` 时，广播也可唤醒）。

主要 `spawn` 参数：

- `--agent <name>` — 加载 `.trellis/agents/<name>.md`（provider/model/as/系统提示词的默认值）。
- `--provider <claude|codex>` — 覆盖代理卡片；根据适配器注册表验证。
- `--as <name>` — 频道工作代理标识，默认代理名称。
- `--cwd <path>` — 工作代理工作目录（也是 `--file`/`--jsonl` 的路径限制根目录）。
- `--model <id>` — 覆盖模型。
- `--resume <id>` — 恢复已有 claude 会话 / codex 线程。
- `--timeout <duration>` — 在 `30s` / `2m` / `1h` 后自动终止。
- `--warn-before <duration>` — supervisor_warning 的提前量（默认 `5m`；`0ms` 禁用）。
- `--file <path>`（可重复，支持 glob）— 将文件内容注入系统提示词。
- `--jsonl <path>`（可重复）— Trellis jsonl 清单（每行 `{file, reason}`）。
- `--by <agent>` — `spawned` 事件作者（默认 `$TRELLIS_CHANNEL_AS` 或 `main`）。
- `--inbox-policy <explicitOnly|broadcastAndExplicit>` — 默认 `explicitOnly`。
- `--idle-timeout <duration>` — OOM 保护的空闲 TTL（默认 `5m`；`0` 禁用）。
- `--max-live-workers <n>` — 启动时的存活工作代理数量预算（默认 `6`；`0` 禁用）。

成功事件 `spawned` 记录 `pid`、`provider`、`agent`、注入的 `files` 和解析后的 `manifests`，便于后续旁观者审计上下文。

## 代理卡片

`--agent <name>` 解析为 `.trellis/agents/<name>.md`。卡片名称必须匹配 `[A-Za-z0-9._-]+`。默认 Trellis 安装提供两张卡片：

- `.trellis/agents/check.md` — 代码质量审查者。
- `.trellis/agents/implement.md` — 执行实现工作的编码代理。

```yaml
---
name: check
description: 代码质量检查专家。
provider: claude
---
```

Frontmatter 字段填充 `spawn` 的默认值（provider、model、`as`）；Markdown 正文成为工作代理系统提示词中的角色说明。卡片**不会**自动附加任务文件，每次启动都必须显式注入上下文（见下文）。

启动具名代理前，始终检查项目卡片：

```bash
ls .trellis/agents
sed -n '1,100p' .trellis/agents/check.md
```

## 上下文注入

两个参数通过 `context-loader` 组装内容，并将其注入工作代理系统提示词的 `# CONTEXT FILES` 块：

- `--file <path>` — 可重复，支持 glob（`*`、`**`）。读取并拼接每个匹配文件。
- `--jsonl <path>` — 可重复的 Trellis 清单，每行是 `{"file":"<path>","reason":"<why>"}`。reason 保留为各文件内容上方的标题注释。

加载器强制执行的限制：

- 每文件 1 MB 硬上限（超限 → 报错）。
- 每文件超过 200 KB 时向 stderr 发出警告。
- 拼接后的总上下文超过 500 KB 时向 stderr 发出警告。
- 路径穿越限制：所有解析后的路径必须位于 `--cwd` 内。

针对任务目录启动检查代理的示例：

```bash
TASK=.trellis/tasks/05-13-example
trellis channel spawn cr-example --agent check --provider codex --as check-cx \
  --file "$TASK/prd.md" \
  --file "$TASK/design.md" \
  --file "$TASK/implement.md" \
  --jsonl "$TASK/check.jsonl" \
  --cwd "$PWD" --timeout 30m
```

`spawned` 事件同时记录原始 `files` 数组和由 `--jsonl` 展开的 `manifests`，因此审计记录能反映工作代理实际看到了什么。

## 名称与路由

`--as` 有两个含义：

- `send` / `wait` / `interrupt`：发言者身份（产生的事件的作者）。
- `spawn`：其他代理通过 `--to` 寻址的工作代理标识。

同一频道有多个工作代理或提供方参与时，使用显式名称：

```bash
trellis channel spawn cr-feature --agent check --as check-claude
trellis channel spawn cr-feature --agent check --provider codex --as check-cx

trellis channel wait cr-feature --as main \
  --from check-claude,check-cx --kind done --all --timeout 15m
```

`--all` 要求 `--from`，并阻塞直到每个列出的工作代理都产生匹配事件；超时以 **124** 退出，并向 stderr 打印 `timeout: still waiting on ...`。

## 软中断 — `interrupt`

`channel interrupt` 用于协作式改向：追加一个 `interrupt` 事件（reason 为 `"user"`），并在适配器支持时，在提供方层面中断当前轮次，同时传入替代指令。当工作代理应放弃当前轮次、立即处理新输入，并保留会话时使用。

```bash
echo "Stop refactoring the parser — switch to fixing the failing test in src/foo.ts" \
  | trellis channel interrupt impl-task --as dispatcher --to codex-impl --stdin
```

参数：

- `--as <agent>` **（必填）** — 调用者身份。
- `--to <agent>` **（必填）** — 目标工作代理。
- `--scope <project|global>` — 频道范围。
- `--stdin` / `--text-file <path>` / `[text]` — 替代指令正文。

追加事件具有 `kind: "interrupt"`；下游 `wait` / `messages` 筛选器可通过 `--kind interrupt` 订阅以响应改向（例如记录路由改变，或让其他工作代理等待协调者的修正）。

对于应等到工作代理下一轮再处理的低优先级提示，发送带标签的普通消息：

```bash
echo "Check this when you reach the next turn." \
  | trellis channel send impl-task --as dispatcher --to codex-impl \
      --stdin --tag question
```

## 硬中断 — `kill` + `--resume`

工作代理必须**立即**停止时使用 `kill`（例如失控循环、错误指令已经执行，或适配器不响应 `interrupt`）。监督进程按 SIGTERM → 8 秒宽限 → SIGKILL 升级；需要 SIGKILL 时，CLI 写入 `killed` 事件，确保事件日志如实记录。

```bash
trellis channel kill impl-task --as codex-impl
trellis channel spawn impl-task --as codex-impl --provider codex \
  --resume "$(cat ~/.trellis/channels/<bucket>/impl-task/worker.session-id)"

echo "STOP — new instructions: ..." \
  | trellis channel send impl-task --as dispatcher --to codex-impl --stdin
```

`kill` 参数：

- `--as <agent>` **（必填）** — 指定工作代理（位置参数 `<name>` 是频道）。
- `--scope <project|global>`。
- `--force` — 立即 SIGKILL（也会终止内部工作代理 pid）。

副作用：清理 `pid`、`worker-pid`、`config`、`spawnlock` 辅助文件；保留 `log`、`session-id`、`thread-id` 以供取证与恢复。

当 `interrupt` 无法使工作代理收敛时，kill + `--resume` 是确保改向的路径。

## 工作代理 OOM 保护

OOM 保护防止孤儿/空闲工作代理累积并耗尽主机资源。每次 `spawn` 时运行，在每个项目分区实施两项策略：

- **空闲 TTL** — 清理最后活动时间早于配置阈值的工作代理（默认 `5m`；`0` 禁用）。
- **存活工作代理预算** — 如果同一项目分区已存活的工作代理多于 N 个，拒绝新启动（默认 `6`；`0` 禁用）。

优先级（从高到低）：

1. CLI 参数：`spawn` 的 `--idle-timeout`、`--max-live-workers`。
2. 环境变量：`TRELLIS_CHANNEL_WORKER_IDLE_TIMEOUT`、`TRELLIS_CHANNEL_MAX_LIVE_WORKERS`。
3. `.trellis/config.yaml` 中的 `channel.worker_guard`。
4. 内置默认值（`5m`、`6`）。

启动时向 stderr 写入清理通知，便于操作人员查看清理了哪些空闲工作代理，以及为何拒绝新启动。保护机制对临时 / `channel run` 工作代理一视同仁，适用相同的空闲 TTL 和预算。

审计当前状态时，通过 `channel list` 的 `WORKERS` 列查看工作代理，并检查 `~/.trellis/channels/<bucket>/<channel>/` 下各频道的 `pid` / `worker-pid` 辅助文件。

## 工作代理收件箱 API

收件箱是唤醒工作代理的频道接口。路由由两个设置控制：

- **收件箱策略**（`spawn --inbox-policy`）：
  - `explicitOnly`（默认）— 仅由 `send --to <worker>` 或 `interrupt --to <worker>` 唤醒。
  - `broadcastAndExplicit` — 也由广播唤醒（不带 `--to` 的 `send`）。
- **投递模式**（`send --delivery-mode`）：
  - `appendOnly` — 无论工作代理状态如何，都追加事件。
  - `requireKnownWorker` — 如果 `--to` 指定的工作代理从未被启动，则失败。
  - `requireRunningWorker` — 如果具名工作代理当前不存活，则失败。

当调用者期待协作代理正在运行时，更严格的投递模式可防止消息静默丢失。

收件箱相关子命令：

- `send <channel> [text]` — 追加 `message` 事件。
  - `--as <agent>` **（必填）** — 作者。
  - `--to <agents>` — CSV；一个 → 字符串，多个 → 数组；省略时广播。
  - `--stdin` / `--text-file <path>` / `[text]` — 正文来源。
  - `--delivery-mode <appendOnly|requireKnownWorker|requireRunningWorker>`。
- `interrupt <channel> [text]` — 软中断改向（见上文）。
- `wait <channel>` — 阻塞直到匹配事件到达。
  - `--as <agent>` **（必填）** — 筛选上下文中的 `self`。
  - `--from <agents>` — CSV 作者列表。
  - `--kind <kind[,kind...]>` — CSV（OR 语义）；支持 `interrupt`、`done`、`progress` 等。
  - `--to <target>` — 默认自身代理（广播 + 明确发给自己）。
  - `--include-progress` — 也在进度事件到达时唤醒。
  - `--all` — 要求每个 `--from` 代理都匹配（超时 → 退出 **124**）。
  - `--timeout <duration>` — `30s` / `2m` / `1h` / `1000ms`。
- `messages <channel>` — 查看 / 筛选 / 跟踪事件流。
  - `--follow` 跟踪尾部，`--kind` / `--from` / `--to` 筛选，`--raw` 每行输出一个 JSON，`--no-progress` 隐藏进度噪声。

典型分派者循环：

```bash
# 1. 唤醒工作代理。
echo "Run the failing test and report." \
  | trellis channel send impl-task --as dispatcher --to codex-impl --stdin \
      --delivery-mode requireRunningWorker

# 2. 阻塞直到完成。
trellis channel wait impl-task --as dispatcher \
  --from codex-impl --kind done,error --timeout 30m

# 3. 阅读最终回答。
trellis channel messages impl-task --from codex-impl --last 1 --raw
```

所有发出事件的子命令（`send`、`interrupt`、`post`、`context add` / `delete`、`title set` / `clear`、`thread rename`）都在 stdout 上以单行 JSON 打印追加事件，便于通过脚本使用收件箱层。
