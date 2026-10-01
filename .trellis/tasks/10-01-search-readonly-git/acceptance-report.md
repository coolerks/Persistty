# W05/W06 统一验收进展

日期：2026-10-01。用户明确要求“开始验收W05和W06”；已撤销此前延期安排，两个任务的 `acceptance.state` 为 `in_progress`。主会话单代理 inline，未提交、推送、归档或部署。本文记录实际结果，尚不构成两个工作包最终验收通过。

## 本轮范围和结果

| 检查 | 实际结果 |
| --- | --- |
| Go 全包 test/vet/race | 本轮均通过；有效测试缓存被复用，源码未改变；不等于远端运行 |
| 前端 lint/typecheck | 最终产品和测试修改后通过 |
| Vitest | 23 文件、134 项通过 |
| 生产构建 | 最终修正后通过；字体表、SHA256、等宽字形和37项许可证通过 |
| Chromium 综合回归 | 22项全部通过，1.5分钟；包含既有18项、W05图片/触屏两项及W06两项 |
| Chromium 收尾专项 | 增加未保存旋转及替换默认取消焦点后，四项真实后端专项再次全部通过，21.7秒 |
| WebKit 专项 | 六种图片/主题、触屏旋转、W06搜索替换/Git及取消四项通过；未保存旋转及最终焦点修正后复验四项全部通过，23.9秒 |
| Firefox | 官方Playwright Firefox155已下载，启动时报profile目录不存在；改用独立TMPDIR仍失败。未进入页面，属于环境阻塞，不能记产品通过 |
| agent-browser | 新Vite登录页有内容、控件完整、无Vite覆盖层；截图后关闭浏览器 |
| 私密连接规范本地回归 | 现有18项通过；新增runner连接前拒绝/失败清理/清理失败身份保留3项通过 |
| Linux产物准备 | 当前 runtime-probe 和 files/search/gitview/toolrunner/httpapi 五包测试二进制交叉编译通过，仅是可上传产物 |
| 本轮Debian执行 | 未执行。自动审批在启动SSH/SCP前拒绝向`.env`指定目标发送项目二进制与fixture，等待明确目的地与载荷授权；没有绕过拒绝 |
| 文档/JSON/忽略/diff | 140个本地Markdown链接、4个JSON、25个修改/新增Go文件格式通过，git diff --check通过；测试报告与浏览器下载均在忽略或临时目录，不提交私有连接或真实运行数据 |

前端存在原有jsdom canvas提示，构建存在大chunk提示，Playwright存在NO_COLOR/FORCE_COLOR提示；没有抑制告警或放宽产品门禁。Linux交叉编译、本机缓存测试和WebKit桌面引擎都不等于真实Debian/iPhone验收。

## W05 验收证据

沿用[既有检查](../09-30-workbench-editor-recovery/check-report.md)及[截图反馈专项](../09-30-workbench-editor-recovery/feedback-check-report.md)的真实Debian历史证据，不能声称本轮重跑。最新代码在本机检查：

- E01/E02/E03：BOM/混合换行、自动保存、四组与设备记录、原model/undo、409暂停、真实IndexedDB草稿刷新后零隐式PUT、桌面只读比较后明确保存；存储失败保护最后视图，手机基线变化仅保留/导出。
- E04/E05：既有buffer/draft/view单测及原有安全文件/事件测试通过；本轮没有重新在Debian跑移动/权限/外部进程事件矩阵，历史报告与待办分别保留。
- P01：新增自有3×2 PNG/JPEG/GIF/WebP/AVIF/SVG，按`.txt`命名验证内容识别；HTTP响应原字节hash、尺寸、nosniff/no-store/sandbox和匿名401；Chromium/WebKit实际原生图片解码，在浅色/深色/跟随系统三模式共18次查看。主动SVG和尺寸/容器边界由Go测试覆盖，既有DebianPNG/SVG证据保留。
- L01/L02/L03：布局/恢复、终端位置、同WS/零隐式终端操作、工作台菜单与内部滚动、controller/observer历史边界回归通过。触屏390×844→844×390新增回归，基础textarea/单内容导航/未保存中文emoji/自动落盘保留，document无横向溢出。
- A01：实际自托管字体加载，i/W/Powerline/时钟图标等宽，四种glyph像素图不同于缺字替代；构建校验原字体与许可。未启动真实Oh My Zsh/Powerline shell主题，canvas验证不能代替此项。

## W06 验收证据

后端安全/rg/Git临时真实集成与HTTP会话隔离通过，细项见[功能报告](implementation-report.md)及[后端owner](../../spec/backend/search-git-contract.md)。本轮本机真实后端浏览器覆盖：

- S01～S04：指定搜索文件与忽略候选仍被排除，emoji UTF-16位置打开原编辑器；Chromium用真实复制核对选中`hit`，WebKit验证原Monaco选区。其它模式/范围/去重/截断/特殊文件由Go集成覆盖，未将真实DebianGit/rg行为算作通过。
- R01～R03：同源预览/只读diff，外部修改后冲突或前端版本保护跳过且零覆盖；重新预览明确应用、状态查询，取消等待中的传输零写入；前端目标buffer保护/处理中新增输入由单测覆盖；默认关闭焦点有实际浏览器断言。
- G01～G04：当前磁盘只读比较、明确选择编辑器快照、根提交历史、手机前后只读文本、HEAD修改标记；对临时源HEAD/index前后hash验证一致。多根/嵌套/rename/merge/缺对象/危险配置、固定HEAD分页由真实本机Git集成覆盖。真实DebianCLI对照与大仓库性能仍待目标授权。
- I01：桌面入口、手机单内容、CSS裁剪、原工作台/草稿/终端几何与设备应答回归通过；真实触控板/手机输入法不能用鼠标wheel或hasTouch代替。

本地fixture没有真实tmux，终端专项使用已有专用browser fixtures；本轮没有重新证明systemd持久性。W03已有真实验收证据仍有效，W08正式部署范围不变。

## 发现并修复

1. W05横屏判断仅看760px宽度，触屏旋转到844px会切换桌面恢复记录及Monaco，手机导航消失。失败用例先复现，`useMobile`与相关CSS采用`(max-width: 760px), (hover: none) and (pointer: coarse)`；宽鼠标桌面仍为多组，粗指针无悬停触屏保留单内容。新增未保存输入旋转及实际落盘断言，更新生命周期owner。
2. W06只描述默认取消焦点，未显式绑定，实际浏览器断言失败。`DialogContent.initialFocus`指向“关闭预览”按钮ref，保持现有shadcn/Base UI原语；浏览器复测焦点与替换全链路，更新owner。
3. 本轮初次同时运行两组Playwright并共用输出目录，产生trace/network ENOENT；这是主会话测试编排错误。已串行、独立`--output`复跑并通过。新增图片fixture使默认搜索跳过数增加；W06测试明确include自己的sample和ignored文件，保持零跳过与ignore断言，没有删减安全断言。

## 待完成与下一步

1. Debian：用户确认使用私有`.env`目标并传输本项目Linux产物/合成fixture后，运行[五包runner](../../../tests/integration/debian/w06/README.md)及现有隔离browser harness。核对实际Git/rg能力、安全目录读取、UI集成、真实字体/终端输出和精确unit/目录清理，不修改正式服务。
2. 真实手机：待用户提供设备/系统/浏览器，并协调其可访问的隔离地址。按[手机操作清单](../09-30-workbench-editor-recovery/phone-acceptance.md)执行；不把hasTouch、WebKit桌面或viewport计作软键盘验收。
3. 真实触控板惯性与Firefox启动环境：分别记录物理操作和可用浏览器条件，再恢复受影响项。现有Chromium/WebKit结果不能替代Firefox未执行项。

两个任务保持进行中。待授权/设备条件补齐后继续本报告，不归档或宣布完整验收。

收尾已停止本轮专属Vite与本地fixture进程，确认5179/8089端口关闭、`/private/tmp/persistty-w06-browser-169302694`已删除。Linux测试产物保留在临时目录等待授权，不是远端残留；本轮未连接Debian。开发日志记录为第6次会话，没有自动提交。


## 后续截图反馈修正

用户随后以五张截图授权修正目录对话框、比较全屏/行内、默认shell、无仓库选择及无提交仓库。最新结果见[截图修正记录](screenshot-adjustments.md)；此前的本机报告是当时结果，不能替代本批回归。实机验收条件仍待补齐。

## Git 延迟与重复请求反馈（2026-10-01）

本机性能修复与安全/浏览器回归见 [performance-adjustments.md](performance-adjustments.md)。本次修复不改变上文真实Debian与实机验收的待办状态。
