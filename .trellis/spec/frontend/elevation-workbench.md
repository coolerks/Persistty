# W07 提权确认与编辑缓冲区契约

## 1. 范围与触发条件

2026-10-02 已接入单文件确认流程；真实 sudo/PAM 运行能力仍待独立 Debian 验收。仅 FileBuffer 的普通保存失败 code=`permission_denied` 且文本 dirty/ready、无在途保存时提供“提权保存”。不是所有403都能提权；服务器 prepare 仍检查允许列表、普通可读性和强 Version，默认关闭显示不可用并保留输入。

API/系统安全 owner 见 [后端 W07](../backend/elevation-contract.md)。UI 只处理用户意图，不能缓存系统授权或替代根句柄、nonce、登录/关联发布裁决。

## 2. 签名

`web/src/lib/api/elevation-client.ts` 通过现有同源 request 提供 prepare/execute/status/cancel；严格decoder解码 [共同fixture](../../../tests/contracts/elevation.json)。execute 无自动重试。`FileBuffer` 提供 beginElevation/executeElevation/queryElevation/expireElevation/dismissElevation；`EditorScope.elevation` 为 scope 内唯一临时 Attempt，多个编辑组共用。

`EditorScopeProvider` 只渲染一个 `PrivilegedSaveDialog`。Dialog/Input/Field/Button/Collapsible 全部复用既有 shadcn base-nova 组件，没有新依赖或另造基础弹窗。官方查找记录见 [Dialog](https://ui.shadcn.com/docs/components/base/dialog) 与 [Collapsible](https://ui.shadcn.com/docs/components/base/collapsible)。

## 3. 生命周期与数据契约

开始时暂停当前 buffer 普通自动保存，捕获 content/generation、expected_version（已比较时采用comparison）、file、project version、scope epoch、CSRF。prepare 回复只有当前完整 scope 仍匹配才接入；迟到 prepare 取消并丢弃。确认弹窗显示批准的绝对目标、有效时间、一次范围和可展开的只读冻结正文，新输入不改变本次正文。

初始焦点为取消；只有明确“确认一次保存”执行，Escape/关闭/过期零 execute。单一页脚取消/关闭入口，未提交阶段默认安全焦点。密码用 uncontrolled DOM input ref，不放 React/Zustand/IDB/Attempt；提交立即清空字段，执行阶段卸载输入，effect清理关闭/过期时捕获的DOM值。不承诺浏览器请求body/托管字符串绝对擦除，系统密码与应用密码不同。

执行同步进入 executing，单 buffer 与 scope 防双击；捕获正文附原 CSRF，无授权继承。成功只更新捕获正文与新 Version 的 base，不 setValue/re建model、不清undo；执行中的新输入仍 dirty/paused，需要明确普通保存，不能自动提权。等待draftQueue后按提交 generation 清自己的草稿，再持久新增内容；等待后再次检查 scope，配置变更/关闭不能因迟到响应误清草稿。

执行网络/协议错误为 unknown，保留输入和 request_id，只显示查询；状态请求单次在途，防乱序结果倒退。executing/indeterminate继续unknown；终态更新确认结果，applied才推进base。关闭unknown可保留输入并普通复读比较，不重发密码。冲突清comparison、先读并比较后才能明确普通保存。系统 rejected/authorization_failed是业务结果，不作为应用401登出。

配置/登录变更、重命名/移动、失去有效scope/关闭时取消未接受请求、清密码字段和引用、丢弃迟到结果。Abort不等于服务器回滚，已接受保存需查询/普通复读；失败始终保留当前正文及草稿。暂停状态对后续编辑维持paused，不误显示已保存；原草稿恢复入口文案统一“保存当前输入”。

## 4. 验证与错误矩阵

| 事件 | 可观察行为 |
| --- | --- |
| 通用403/CSRF/越界 | 不提供提权入口 |
| permission_denied | 显式入口；prepare未通过则无密码执行控件 |
| 默认禁用/服务失效 | 可见错误，输入/草稿保留 |
| Cancel/Escape/过期 | 零 execute，prepared可发送DELETE |
| 双击确认 | 一个execute，密码控件立即清空/消失 |
| 保存期间新增输入 | 成功base=授权正文，新内容dirty/paused和草稿保留 |
| 409/rejected conflict | 原内容保留、compare前不能绕版本保存 |
| 网络未知 | 只显式GET查询，绝不自动POST重放 |
| 配置/CSRF/epoch/file失效 | 取消未接受提交；迟到结果不推进base/清草稿 |

## 5. 正常 / 基础 / 错误用例

正常：授权快照“first”，执行中输入“second”，成功base为first而视图/model为second、paused。基础：未配置时对话框显示不可用并保留草稿。错误：授权成功后将当前输入整体标saved，或在重连时重发execute。

## 6. 所需测试与边界

`editor-elevation.test.ts` 验权限入口、关闭/过期、双击、快照合并/草稿、断线查询、配置旧响应及冲突；`elevation-decoder.test.ts` 验共同DTO/字段/状态矛盾。`w07-elevation.spec.ts` 用独立fixture验真实Monaco/undo、手机textarea、焦点/Escape、冻结内容、字段清理、未知查询与不可用/过期/冲突；Chromium/WebKit分别输出，接口为合成协议，不宣称真实特权/物理软键盘已通过。编辑恢复自动化回归不代表重启W05/W06统一实机验收。Monaco 输入控件随引擎切换 EditContext/textarea；桌面键盘测试应定位可访问的 Editor content textbox，不依赖 textarea.inputarea 内部结构。

## 7. 错误与正确示例

错误：`scope.password = input.value; localStorage.setItem("password", ...)`。正确：仅点击确认读取DOM字段，立即清空，把短时值交给唯一execute入口；Attempt只留冻结正文与非秘密状态。

错误：执行结果到达时 `base.content = state.content`。正确：`base.content = attempt.content`，当前正文不同便paused；按attempt.generation清草稿，不删除执行中新generation或其他视图草稿。
