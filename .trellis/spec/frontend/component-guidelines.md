# 组件与交互

采用函数组件、显式 typed props；组件负责展示/用户意图，hook/api 负责网络和业务协调。基础 UI 必须遵守下述 shadcn/ui 复用约定。Tailwind 使用语义色 token，不散落 light/dark 硬编码颜色。lucide 图标按钮有中文 aria-label，非颜色唯一状态标记。

## 强制约定：先查找并使用 shadcn/ui

新增或修改 UI 前，先查找项目已有的 `web/src/components/ui/` 与业务封装，再查询 shadcn/ui 官方组件目录、文档和官方 registry。凡 shadcn/ui 已提供对应组件或可通过其组件组合满足需求的，必须找到并使用，不能一上来手写替代品。此约定用于保持样式、交互和无障碍行为一致，避免重复维护基础控件。

1. 项目已引入对应组件时，直接复用；未引入时，按工程锁定的包管理器、shadcn 配置及官方安装方式添加所需组件至 `web/src/components/ui/`，不复制一份到 feature 中，也不凭记忆重写官方实现。
2. 业务需求通过 props、variants、语义色 token、组合或薄业务封装实现；业务状态和副作用仍归 feature，不塞进基础 UI。外观不同、组件尚未安装或查找失败均不能作为自创理由；查询不可用时说明阻塞并恢复查询，不能宣称官方没有组件。
3. 只有查证官方没有对应组件，且已有组件组合也无法覆盖需求时，才允许编写缺失的专用部分。实施记录须注明查询来源、候选组件、具体缺口及自定义范围；仍复用可用的 shadcn/ui 基础控件。Monaco/xterm 等专用引擎遵守各自规范，其周边按钮、菜单和弹窗同样执行本约定。
4. Review 核对查找记录、实际 import 和自定义范围；有可用官方组件却手写替代实现，或未查找就自创，均视为不通过，先改为复用再验收。不要求为“来自 shadcn/ui”编写镜像测试，测试实际键盘、焦点、取消和业务行为。

示例（下述路径为组件引入后的约定，不表示当前仓库已有源码）：

```tsx
// 正确：复用引入的 shadcn/ui Button，业务动作由调用方传入。
import { Button } from "@/components/ui/button";

type SaveButtonProps = { pending: boolean; onSave(): void };

export function SaveButton({ pending, onSave }: SaveButtonProps) {
  return <Button disabled={pending} onClick={onSave}>保存</Button>;
}

// 错误：未查询或复用 Button，就另造基础按钮的样式与交互。
export function CustomSaveButton({ pending, onSave }: SaveButtonProps) {
  return <button className="rounded bg-blue-600 px-4 py-2 text-white"
    disabled={pending} onClick={onSave}>保存</button>;
}
```

官方组件与安装方式须在实施时核实；不在当前规范阶段安装依赖。组件来源约定不代替产品确认流程、资源生命周期或后端安全契约。

## 业务交互与验收

IDE layout：Activity Bar、Explorer/Search/Git sidebar、editor tabs、Monaco、Terminal tabs；react-resizable-panels 有最小尺寸、键盘可调、持久 layout 版本。错误/loading/empty/unavailable 区分，不能 network failure 显示“没有文件”。command palette Ctrl/Cmd+Shift+P 至少新终端、打开文件/工作区、查找/替换、切换Terminal/sidebar/theme、Open Terminal Here；所有入口复用同一 command dispatcher，不复制副作用。

Terminal “隐藏/分离”只调整 UI。`Close Terminal` 始终 Dialog：`关闭这个终端会终止其中正在运行的程序。`，默认焦点在取消，确认才触发 backend close，pending 禁重复操作；失败保留 tab 和错误，不 optimistic 移除并声称成功。
文件冲突 Dialog 展示 View Diff/Reload/Overwrite；Reload 丢弃 dirty buffer 必须提示；Overwrite 带当前版本再请求，继续冲突保持 Dialog。Replace Preview/Diff/选择不可省略。上传 Skip/Replace/Keep Both 和 Apply to all 只本批生效。

示例 props：`type CloseTerminalDialogProps = { terminalId: string; open: boolean; onCancel(): void; onConfirm(): Promise<void> }`。错误：useEffect unmount 调 close；正确：只有明确 onConfirm 执行业务销毁。
测试用 Testing Library/user-event 从按钮/焦点/键盘断言，不断言内部 JSX 结构；覆盖 Escape/取消零 close 请求、双击只一次、失败仍可重试、409 保护旧 buffer。

## Explorer 操作覆盖

资源管理器必须按文件/目录类型提供对应右键操作，禁用不适用项并说明原因。以下是聊天已明确的 v0.1 需求，后续 Explorer 任务逐项建立验收，不因当前规范阶段没有实现而删减。

| 操作类别 | 必需操作与交互 |
| --- | --- |
| 打开与定位 | 打开、在侧边打开、复制路径、复制相对路径、刷新；复制路径只写剪贴板，不作为服务端访问授权 |
| 新建与重命名 | 新建文件、新建文件夹、重命名；名称校验结果与目标已存在错误可见 |
| 移动与复制 | 移动、复制、剪切、粘贴、创建副本；区分同一工作区的服务器文件操作和本机拖入上传 |
| 删除 | 明确展示将删除的目标并确认；有未保存内容的已打开文件按草稿策略提示，失败不能伪装删除成功 |
| 终端与搜索 | 在此处打开终端、在文件夹中查找、在文件夹中替换；复用终端创建和搜索命令，不通过任意命令接口执行 |
| 传输 | 上传文件、上传文件夹、下载文件、将文件夹下载为 ZIP；支持本机文件/目录拖入，浏览器不支持目录拖入时提供选择器并解释限制 |

菜单、快捷键、命令面板复用同一操作入口。文件操作后按服务端成功结果使列表与相关编辑器快照失效，不能只依赖 watcher；外部变化遵循已有 dirty buffer 保护规则。目标冲突必须显式解决，不能复用上传的“全部应用”来静默覆盖编辑器文件。完整 API 签名和错误映射由对应任务补入 [HTTP 契约](../backend/http-api.md)，操作安全归 [文件契约](../backend/filesystem-guidelines.md)。

测试从用户操作验证菜单类型差异、键盘操作、取消删除零请求、失败保留原列表、重命名后标签定位、复制与上传的来源区分、目录拖入不支持时的替代入口。
