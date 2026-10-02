# 前端开发规范

W01 已有 React/Vite 登录、项目路由、严格 API 解码和三主题；版本及 npm 锁文件见 `web/package.json`/`web/package-lock.json`。W04 已归档，使用 shadcn `base-nova`、Zustand、react-resizable-panels 与按需加载的桌面 Monaco；手机用基础文本视图。2026-09-30 已覆盖完整 91 个 Monaco 模式及 Material 文件/目录图标，真实 Chromium 验收见相应任务报告。W03 已接入真实 xterm/WS，桌面与手机验收通过；W05 已实施共享编辑缓冲区、自动保存、IndexedDB 草稿、四组布局与恢复、预览和字体，并通过自动化与真实 Debian 专项；任务保持进行中，真实手机软键盘验收待完成。下述未交付功能片段仍为契约，不是已验证实现。

## 规范索引
| 文档 | 内容 |
| --- | --- |
| [目录结构](directory-structure.md) | feature 组织、依赖方向 |
| [组件](component-guidelines.md) | shadcn/ui 强制查找与复用、UI、确认、无障碍 |
| [Hooks](hook-guidelines.md) | effect、取消、stale response |
| [状态](state-management.md) | server truth/UI/draft |
| [类型](type-safety.md) | strict TS、DTO decoding |
| [API/WS](clients.md) | 请求、重连、错误 |
| [编辑器/终端生命周期](editor-terminal-lifecycle.md) | Monaco/xterm/IndexedDB |
| [W03 前端终端契约](terminal-runtime-contract.md) | 稳定 runtime、标签动作、观察端输入确认、批次 Dialog 与浏览器验收 |
| [W06 搜索与 Git](search-git-workbench.md) | 面板、替换buffer保护、HEAD标记与只读比较 |
| [主题与资源](theme-assets.md) | 统一主题、字体、图标许可 |
| [质量](quality-guidelines.md) | lint/typecheck/test/build/E2E |
| [W07 提权确认](elevation-workbench.md) | 同buffer冻结快照、密码字段、取消/未知结果与草稿保护 |

## 开发前必读（Pre-Development Checklist）
加载 task PRD/design/implement/JSONL。每个前端任务读目录、组件、状态、类型、质量；网络读 clients 和 backend HTTP/WS；Monaco/xterm 读生命周期，theme/icons 读资源。涉及写入或 close 读 backend 文件与Terminal 契约，不能从 UI 推断服务器安全。

编写 UI 前必须先查找项目已有组件及 shadcn/ui 官方组件；能用官方组件或组合满足的必须复用，具体查找顺序、自定义条件和示例见 [组件规范](component-guidelines.md)。

## 质量检查（Quality Check）
按实际 package manager 执行 lint、typecheck、test、build；Playwright 验 AC。检查键盘/焦点、异步资源释放、重连/认证、409、draft、深浅主题。bootstrap 无 manifest 时这些产品命令不适用，只做文档检查，不宣称通过。

核对 shadcn/ui 查找记录和实际复用情况；未查找就自创、或已有适用官方组件却手写替代品，不通过审查。
