# 组件与交互

采用函数组件、显式 typed props；组件负责展示/用户意图，hook/api 负责网络和业务协调。可访问性复用 shadcn/ui Dialog/Menu/Tabs；不自己拼无焦点管理的弹层。Tailwind 使用语义色 token，不散落 light/dark 硬编码颜色。lucide 图标按钮有中文 aria-label，非颜色唯一状态标记。

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
