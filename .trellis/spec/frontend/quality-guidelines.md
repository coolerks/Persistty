# 前端测试与构建门禁

foundation 固定单一 package manager/lockfile 和 package scripts：lint、typecheck、test、build。以后每 task 用实际 manager 执行四门禁；单测用 Vitest + Testing Library/user-event，端到端用 Playwright。工具/版本由 foundation 验证，不在 bootstrap 安装。lint 不允许 unused/effect dependencies/type bypass，build 校验 Monaco workers/font 与资源路径。

UI 复用门禁：按 [组件规范](component-guidelines.md) 检查项目已有组件与 shadcn/ui 官方查询记录、实际 import 和业务封装。官方已有适用组件或可用组合时必须使用；自定义部分须有查证后的能力缺口及范围说明。未查找即自创或重复实现可用官方组件，先修复再验收；测试仍断言用户行为，不以来源检查代替键盘、焦点和交互测试。

必需 feature tests：Terminal tabs/order/pin/rename/显式 close vs hide、Zustand stores、file tree/context menu/watch updates、file conflict/Dialog/draft、search Unicode 行列、replace preview/选择/冲突、theme、upload 三冲突策略/apply all/resume。测试观察用户行为/请求边界，不镜像 store 实现。

Critical E2E 由 [后端测试](../backend/quality-guidelines.md) 定义运行时断言：Terminal A/B/C 闭页、重连、Web restart、重登录、history/tabs、只 close A；file 外部修改 409；rg ignore + replace conflict；upload/download/空目录/Unicode/大文件/hash/resume/ZIP/symlink；watcher Terminal mkdir/touch/mv/rm；Git CLI 对照。Playwright harness 控制隔离服务而非 mock WS，systemd 必须在 Debian 实测。测试超时/缺依赖/skip 不算通过。

无障碍：键盘 palette/menu/dialog/tabs、焦点返回、resize 最小尺寸、loading/error、dark/light 对比；UI 文案简体中文。auth 安全后端执行，前端测试隐藏按钮不构成安全 review。
W01已有package.json与npm锁文件，lint/typecheck/test/build四门禁必须实际执行。当前只覆盖登录/路由/主题/布局命令模型，真实PTY与完整E2E由后续包交付，不创建空测试满足指标。浏览器viewport模拟不是iOS/Android真机验收；每功能同任务加必要测试，最后full E2E只是整体验收。

局域网HTTP发布必须覆盖非安全上下文：localhost/127.0.0.1通常仍被浏览器视为安全上下文，不能证明私有IP HTTP兼容。`lan-http.spec.ts` 用虚构私有HTTP源，静态产物转取显式本机实例、API/WS用隔离fixture；断言isSecureContext=false且原生randomUUID缺失后检查工作台/文件/搜索定位。执行时记录是否验证生产产物，不能把拦截请求回归记为真实后端、HTTP鉴权或Debian服务通过。

浏览器矩阵与构建串行运行，每轮 `--output` 使用独立的忽略目录；多个Playwright进程共用输出目录会互删trace/network，不能把ENOENT记为产品缺陷或通过。真实后端fixture不能跨并行浏览器共享可写目标。图片区与搜索区共用fixture时明确搜索include范围，保留ignore断言而不放宽结果数量。


E2E对照当前可访问名称与产品状态：图标刷新仍通过aria-label定位，徽标数量不得使折叠按钮定位依赖无数量旧文本。SearchIntent已展开过滤时不重复点击收起；全局终止状态Dialog已恢复时不点击其遮罩下的标签。历史返回live测试先实际产生scrollback，并等待历史xterm解析内容，不能用瞬时加载标题证明已进入历史。语言高亮须等待Monaco公共异步colorize及真实tokenizer注册，保留所有语言token断言，不以固定短延迟判断模块缺失。实际后端/浏览器/构建/race如争用资源应串行复验，保留中间失败；单纯延长总期限不能替代产品断言和HTTP预算。

自动保存前置状态用公开Playwright时钟确定：`page.clock.install()`在导航前，编辑器实际加载后`pauseAt()`，再编辑并关闭或检查灰点；关闭失败后`runFor(1200)`仍须零PUT，手动保存用例恢复时钟且只允许一次PUT。真实自动保存用例保留真实计时，不能统一冻结来隐藏缺陷。真实登录遇429时断言冷却中的禁用状态，等待重新启用后只重试一次并要求200；不禁用限流、不复用未经验证的认证，也不把429当成登录成功。
