# 工作台 UI 调整验收记录

日期：2026-09-30。当前任务 in_progress；产品实现与自动化验收已接入，真实手机软键盘仍待实机检查。单代理 inline，未提交、未归档，未修改用户现有开发服务或 tmux 会话。

## 实现范围

- 合并项目顶栏，名称/编辑靠左，导航/主题/退出保留。终端仅一个标签操作行，移除原三个常驻行；面板 X 仅收起，标签 X 终止。
- 标签未接管图标、输入确认丢弃触发内容、右键/手机更多、目录 +、自动最小空号与持久改名。历史/返回实时/移动/重试保留，刷新只读列表和状态。
- 上下区域独立冻结目标，全部接管确认后发起一次统一批次；任一成员取消/失权撤销受影响整批。倒计时列名称，逐项展示真实执行结果。
- 后端名称 CAS/镜像对账、批次 POST/GET、WS v3 与缺省 exact v2；共享 fixture。无数据库迁移、无新依赖、无直接 Radix。
- 固定视口和长树裁剪边界；官方 Tabs 属性选择器适配已锁 Base UI。portal 宿主原生 DOM 捕获内部拖拽，避免 React owner 路径绕过当前面板。

## 自动化门禁

| 检查 | 结果 |
| --- | --- |
| gofmt 修改文件 | 无未格式化文件 |
| `go test ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go test -race ./...` | 通过 |
| 前端 lint / typecheck | 通过 |
| 前端 Vitest | 13 文件、51 测试通过 |
| 前端 build | 通过；既有 Monaco 大 chunk 警告保留 |
| Debian + Chromium 强制重启 | 新空实例单独运行，1 条通过 |
| Chromium 常规 + UI 新增 | 全套 12 通过、1 条强制项按计划另跑；追加真实批次后的 UI 专项 5/5 通过，共 14 条不同路径已验证 |
| 本地链接、忽略规则、敏感连接扫描、diff check | 14 个 Markdown 无缺失本地目标；运行资源已忽略；55 个变更文本无匹配私有连接值；diff check 通过 |

Go httptest 默认沙箱禁止绑定 loopback 端口，获准执行后全量通过。Vitest 曾在 build/race 同时运行时触发既有 Explorer 分页测试的时序断言（2 次调用而非 3 次）；未改该业务代码，单独全量重跑 51 项通过，保留为潜在测试时序风险。jsdom 的 Canvas getContext 提示不等于真实画面验收，画面以 Chromium 为准。

## 真实测试与证据

沿用 [Debian/browser harness](../../../tests/integration/debian/browser/README.md)，连接仅从已忽略 `.env` 读取；独立 0700 root、私有 tmux socket、Web/tmux 分离 user unit，15 分钟自动过期。长树 fixture 只排他创建自有 120 个文本文件和 extra 目录。未在用户正式目录运行破坏性测试。

- 强制重启：三个真实计数/HTTP/TUI 负载，原 pane PID/start/cgroup 保持，输入/resize/鼠标恢复、断线零重放、登出不终止。
- 原八条常规路径：桌面/手机输入、控制字节、历史、粘贴、原样链接、三端接管/取消、到期精确终止、按钮/拖拽/收起移动零新增 WS、正常 Web 重启。
- 新五条：观察端字符及合成 paste/IME/beforeinput 触发确认零二进制发送；改名跨端同固定 DOM/零 WS、最小空号复用；下方批次列表与 v2 observer 整批取消，上方继续；双根/长树/Monaco/实时终端和三主题的四尺寸几何；真实 ABC 接管撤销、独立 DE 两项实际 kill/逐项结果及 logout 整批取消。
- 视口：1440×656、1024×540、390×844、844×390。断言 document.scrollHeight=clientHeight=视口高，scrollWidth=innerWidth；桌面长树内部可滚动。模拟手机仍为单视图，快捷栏保留。
- 后端单测补并发八次创建、CAS/幂等、镜像失败修复、ABC 与独立 DE 取消/接管/断连/认证失效、失败零 timer/kill、锁外有界 kill/逐项失败/无重试、严格 HTTP/fixture 与 race。
- 前端 provider 单测补部分接管失败零 batch POST，并如实展示已接管对象、不自动重试。

截图为隔离合成数据：[桌面](research/workbench-desktop.png)、[手机终端](research/workbench-mobile.png)。Playwright 原始 trace/report 位于已忽略 web/test-results，不作为已提交产物。

最终清理返回 units_inactive=true、pane_processes_gone=true、root_removed=true；本轮 tunnel 与 5174 Vite 已停止，用户的 5173/正式 tmux 不在清理范围。旧的两轮到期环境也已同样确认清理。

## 修复与复验

1. Tabs 生成器的方向选择器与已锁原语实际 DOM 不符，手机终端宽度为零；修正选择器并复验，不更换组件系统。
2. observer 的 xterm focus/mouse report 不能当输入意图；保持 disableStdin，捕获用户键盘/paste/IME，onData 只按角色放行，菜单不再误弹接管确认。
3. portal DOM 移到下方面板后，React dragover 不经过该面板，活跃 xterm 上无法 drop；增加仅内部 MIME 的 native capture，真实 dragTo 复验通过，零新连接。
4. 截图测试曾引用未创建的 welcome.txt，改为明确生成的 ui-fixture-000.txt。密集 E2E 登录触发每来源 10 次/分钟保护，新增测试遵守页面真实 60 秒冷却，不改认证限流。
5. 调试轮次超过隔离实例 15 分钟时出现 tmux 503；过期轮次不记通过，精确清理后新实例复验。不得延长或复用失效 fixture 当长期部署。
6. 重跑专项时合法手工重名使全服务名称选择器匹配两项；测试改按稳定 terminal ID 定位，不能为满足测试而禁止用户重名。新增实测沿用现有 terminal POST 200 契约，不改产品响应。

## 未验证项

- 真机 iOS/Android 触屏软键盘、实际输入法组合、visualViewport 与输入位置：仅模拟竖横屏和合成 IME 事件，不声称 AC14 完整实机验收。
- 本轮没有新增所有浏览器引擎、真实触控边缘拖出、超长名称/所有菜单键盘组合的全排列矩阵；桌面 Chromium 的现有折叠/移动和几何断言不代替完整实机矩阵。
- 真实 Debian 已新增 ABC/DE 失权隔离、多成员实际 kill、logout 认证撤销专项；mock race 覆盖更细粒度的取消/到期并发，不能把有限实测称为穷尽所有时序或主机重启恢复。

因此保留 M05 中尚未完全覆盖的检查项与 in_progress，不自动归档。后续用户实机检查与必要的专项测试完成后再作完整验收。
