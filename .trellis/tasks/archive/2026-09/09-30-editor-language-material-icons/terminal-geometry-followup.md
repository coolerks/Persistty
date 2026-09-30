# 终端黑色底边与最后一行裁剪补充

## 用户反馈与证据

用户在字符网格同步修正后继续反馈“还是有黑线”，随后明确当前黑线消失但最后一行遮挡，并提供提示符/光标下半部分被状态栏裁掉的截图。本轮直接读取当前 Chrome 页面的 DOM/计算样式，没有输入命令或调整用户布局。

当前页宿主高 240.875px，padding 为 5px 8px；xterm 根内容高 230.875px，但 renderer 网格高 240px、底部 635.125px，超过宿主底部 631px。最后一行实际被 overflow:clip 裁掉。此前 tmux 录制证明的是输出中的边界字符/句点，不能据此断言所有黑线都来自 tmux。

## 根因与修正

1. FitAddon 0.11.0 读取父元素 computed height/width，只扣 `.xterm` 自身 padding，不扣父元素 padding。此前内边距放在宿主，controller/历史 fit 多算可用尺寸。将相同内边距移至 `.xterm`，保持正尺寸、角色权限与独立历史规则。
2. 移动内边距后原页面直接露出完整黑边：`.xterm-viewport` 默认背景为 rgb(0,0,0)，而 `.xterm-scrollable-element` 的主题背景为 rgb(255,255,255)。xterm 6 的底层 viewport 覆盖整个根元素，网格整数舍入后的余量/内边距会露出黑色。为实时/历史 viewport 显式设置 `var(--background)`；不修改合法下划线或终端正文。

修正后的原页网格高 225px，最后一行底部 620.125px，宿主底部 631px；右侧也完整容纳。viewport 与 scrollable 背景均为 rgb(255,255,255)，实际截图中提示符/光标完整、无黑边。无需刷新，原开发页面通过 CSS 热更新生效；没有重启用户服务、终止进程或新建用户终端连接。

## 回归与质量检查

- 新增 `terminal-geometry.spec.ts`：真实 xterm/FitAddon 加隔离 transport，旧代码在几何断言失败（越界 7.84375px）；修正后验证三个桌面视口、浅/深主题、实时末行/光标、历史底部行、padding 边界、底层背景一致、ANSI 下划线保留、单连接/零输入。
- 前端 lint、typecheck、test、build 实际通过，20 个文件/110 条单测；保留既有 jsdom canvas 和大 chunk 提示。随后补充几何断言和截图，再执行 lint/typecheck 与浏览器验收。
- 隔离本机 Vite 5175 上几何、滚动两角色、设备应答两角色、工作台交互共 6 个浏览器用例实际通过。HTTP/WS 合成验证不代替 Debian/systemd 持久性验收；本轮仅改 CSS/回归/文档，没有后端代码变动。
- owner 规范已固化 FitAddon 内边距和 xterm viewport 背景契约。Markdown 5 个文档/19 个本地链接与 git diff --check 实际通过；不提交或归档。

## 防止重复修补

根因类别为第三方 API 隐含假设与几何测试缺口。此前文字存在/行数正确不能证明光标整行未被裁剪；只验证 tmux 输出也不能覆盖 CSS 黑色底层。以后先区分输出字符、renderer 网格、可用 DOM 边界和底层背景，分别验证；不能用裁剪或正文过滤代替修复。

## 收尾

最终严格边界的 6 个浏览器用例通过；测试文件最后修改后 lint/typecheck 再次通过。独立 Vite 5175 已停止（session 93520，退出 130），仅保留忽略目录中的合成截图/测试证据。用户原服务与页面保持运行；本轮修改未提交或归档。
