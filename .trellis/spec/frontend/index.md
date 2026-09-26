# 前端开发规范

当前没有 React/Vite 源码。以下目录/类型/片段为 Persistty 初始约定，不是现有实现。React、TypeScript、Vite、Tailwind、shadcn/ui、lucide-react、Zustand、Monaco、xterm、react-resizable-panels 在 foundation 锁定兼容版本及实际 package manager，不猜测已安装版本。

## 规范索引
| 文档 | 内容 |
| --- | --- |
| [目录结构](directory-structure.md) | feature 组织、依赖方向 |
| [组件](component-guidelines.md) | UI、确认、无障碍 |
| [Hooks](hook-guidelines.md) | effect、取消、stale response |
| [状态](state-management.md) | server truth/UI/draft |
| [类型](type-safety.md) | strict TS、DTO decoding |
| [API/WS](clients.md) | 请求、重连、错误 |
| [编辑器/终端生命周期](editor-terminal-lifecycle.md) | Monaco/xterm/IndexedDB |
| [主题与资源](theme-assets.md) | 统一主题、字体、图标许可 |
| [质量](quality-guidelines.md) | lint/typecheck/test/build/E2E |

## 开发前必读（Pre-Development Checklist）
加载 task PRD/design/implement/JSONL。每个前端任务读目录、组件、状态、类型、质量；网络读 clients 和 backend HTTP/WS；Monaco/xterm 读生命周期，theme/icons 读资源。涉及写入或 close 读 backend 文件与Terminal 契约，不能从 UI 推断服务器安全。

## 质量检查（Quality Check）
按实际 package manager 执行 lint、typecheck、test、build；Playwright 验 AC。检查键盘/焦点、异步资源释放、重连/认证、409、draft、深浅主题。bootstrap 无 manifest 时这些产品命令不适用，只做文档检查，不宣称通过。
