# 前端测试与构建门禁

foundation 固定单一 package manager/lockfile 和 package scripts：lint、typecheck、test、build。以后每 task 用实际 manager 执行四门禁；单测用 Vitest + Testing Library/user-event，端到端用 Playwright。工具/版本由 foundation 验证，不在 bootstrap 安装。lint 不允许 unused/effect dependencies/type bypass，build 校验 Monaco workers/font 与资源路径。

UI 复用门禁：按 [组件规范](component-guidelines.md) 检查项目已有组件与 shadcn/ui 官方查询记录、实际 import 和业务封装。官方已有适用组件或可用组合时必须使用；自定义部分须有查证后的能力缺口及范围说明。未查找即自创或重复实现可用官方组件，先修复再验收；测试仍断言用户行为，不以来源检查代替键盘、焦点和交互测试。

必需 feature tests：Terminal tabs/order/pin/rename/显式 close vs hide、Zustand stores、file tree/context menu/watch updates、file conflict/Dialog/draft、search Unicode 行列、replace preview/选择/冲突、theme、upload 三冲突策略/apply all/resume。测试观察用户行为/请求边界，不镜像 store 实现。

Critical E2E 由 [后端测试](../backend/quality-guidelines.md) 定义运行时断言：Terminal A/B/C 闭页、重连、Web restart、重登录、history/tabs、只 close A；file 外部修改 409；rg ignore + replace conflict；upload/download/空目录/Unicode/大文件/hash/resume/ZIP/symlink；watcher Terminal mkdir/touch/mv/rm；Git CLI 对照。Playwright harness 控制隔离服务而非 mock WS，systemd 必须在 Debian 实测。测试超时/缺依赖/skip 不算通过。

无障碍：键盘 palette/menu/dialog/tabs、焦点返回、resize 最小尺寸、loading/error、dark/light 对比；UI 文案简体中文。auth 安全后端执行，前端测试隐藏按钮不构成安全 review。
当前文档阶段无 package.json，四门禁和 Playwright 均不适用；不创建空测试满足指标。每功能实现同任务加必要测试，最后 full E2E 只是整体验收。
