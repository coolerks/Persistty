# 组件与交互

采用函数组件、显式 typed props；组件负责展示/用户意图，hook/api 负责网络和业务协调。基础 UI 必须遵守下述 shadcn/ui 复用约定。Tailwind 使用语义色 token，不散落 light/dark 硬编码颜色。lucide 图标按钮有中文 aria-label，非颜色唯一状态标记。

## 强制约定：先查找并使用 shadcn/ui

当前工程的 `web/components.json` 使用 shadcn 官方 `base-nova` 样式；基础组件由 shadcn CLI 安装到 `web/src/components/ui/`，业务组件从 `@/components/ui/*` 引用。shadcn/ui 是复制到项目的组件源码，不是需要写入 `package.json` 的 `shadcn/ui` 运行时依赖；该样式生成的交互原语为 `@base-ui/react`。不得把直接安装或调用 `radix-ui` 当成已经使用 shadcn/ui，也不得把项目自制的同名按钮/弹窗冒充 CLI 组件。新增组件先核对 `web/components.json`、CLI 生成文件和锁文件，运行 `rg 'radix-ui|@radix-ui' web/src web/package.json web/package-lock.json`；允许依赖树中其他包的传递依赖，但业务代码不得直接引入 Radix。未来若决定切换 shadcn 样式，须单独审查交互 API 与迁移范围，不能混用两套原语。

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

当前 Tabs 是官方 CLI base-nova 生成源码。已锁定的 Base UI 输出 `data-orientation="horizontal|vertical"`，生成样式若使用 `data-horizontal/data-vertical` 必须按实际属性修正选择器，记录差异并验窄屏；不替换交互原语。TabsContent 必须归对应 Tabs root，外部动作按钮是 Trigger 同级，不能嵌入按钮。

固定工作台裁剪区应有就近定位边界：长列表的 sr-only/绝对定位元素可能越出静态祖先并撑大 document，不能仅验 shell.height。实测 document.scrollHeight/clientHeight、scrollWidth/innerWidth，同时证明树/编辑器/历史内部可滚动，不用全站 body overflow:hidden 掩盖。

## 业务交互与验收

U67..U73覆盖下文旧IDE示例：不新增command palette/AI/扩展入口；上下两主区域内部仅左右分组，手机单内容；下方面板X只收起，单会话trash和上方terminal标签X均请求统一终止倒计时，仅controller发起、任一观察端可取消。文件标签X关闭视图保护草稿。移动终端不销毁runtime；边缘拖出展开、按钮/键盘替代入口必备。旧单端确认Dialog不能替代全端倒计时，旧upload Keep Both不是当前用户确认的冲突策略。

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

### 文件树选择与展开定位（2026-09-30）

`Explorer` 的 `selectedFile: OpenFile | null` 表示跨项目根唯一文件选择，目录不输出 `aria-selected` 或选中样式；目录展开仍使用 `aria-expanded`，覆盖 ghost Button 的 `aria-expanded:bg-muted`，避免所有展开目录持续着色。工具栏操作目录另存，点击目录只展开并更新操作目录，点击/定位文件使用父目录作为操作目录。

`ProjectWorkbench` 只在 sidebar `onResize` 检测到 0 → 正尺寸（含初始可见恢复、拖拽、活动栏或状态栏）时传入新 `reveal: { file: OpenFile | null }`；手机从编辑器/终端返回文件视图同样触发。树可见时切换 tab 不改变 reveal、选择或滚动；同一路径再次展开必须传新请求对象。按 focused group/current file 与 folderId 定位，上方为终端时不定位旧文件。

树收到 reveal 后展开目标祖先，沿当前目录分页找到目标子项，待实际节点挂载后 `scrollIntoView({ block: "nearest", inline: "nearest" })`；不存在、分页结束或请求失败即停止，错误保持可重试。不能将路径相同的其他项目根当目标。条目只保留现有 shadcn ContextMenu 右键操作，删除重复的末尾三点 DropdownMenu；删除/重命名确认与共享 action owner 保持原契约。

编辑器/终端原生横向滚动容器统一细轨道：Chromium/WebKit 固定 4px、Firefox 始终 thin，透明轨道。thumb 默认透明，容器 hover 或子项 focus-visible 时使用主题语义色显示；点击遗留 focus 不应持续显示。只切换颜色，不切换厚度/overflow/display，避免 tab 内容上下抖动或滑块 hover 变粗。保留 overflow-x:auto 与焦点/触摸/拖动滚动，不用 overflow:hidden 或全站规则掩盖溢出。

标准 scrollbar-color 规则必须限定在 `@supports not selector(::-webkit-scrollbar)`，WebKit 分支保持 scrollbar-width:auto/scrollbar-color:auto，防止高优先级 hover 标准属性让 Chromium 退回原生轨道。WebKit thumb 用容器的 `--tab-scrollbar-thumb` 变量同步 hover/focus-visible 颜色；浏览器回归同时验默认隐藏、悬停可见、移开再隐藏，三状态与滑块 hover 的轨道高度及 label 坐标均稳定。

回归测试验目录无选中、跨根文件单选、树可见时 tab 切换不 reveal、重复收展/页面恢复/移动端返回、后续分页目标、右键操作完整、终端 ended 关闭零 mutation。浏览器检查展开目录移开鼠标后透明，两类 tab 溢出可滚动且轨道 4px；不能只验初始样式而忽略 hover 后的布局/颜色。合成接口验收不能宣称真实 PTY 或真机通过。
