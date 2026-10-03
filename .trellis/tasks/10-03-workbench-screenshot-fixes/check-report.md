# 截图修复检查报告

日期：2026-10-03。首轮四项及追加截图反馈的实现、本机适用验收完成；最新结果见下方追加反馈章节，任务保持 in_progress 供用户复核/提交；未提交、部署或归档，其他 W05/W06/W07/UI 任务状态保留。

## 修改结果

| 需求 | 结果与证据 |
| --- | --- |
| R1 标签底缝 | editor-tab 固定38px，原生轨道继承正文背景；溢出前后文字高度/位置一致，两端滚动与固定动作通过；Chromium/WebKit 深浅截图人工复核 |
| R2 HEAD文字 | breadcrumb 去掉 tracked HEAD 与空状态占位，useGitBaseline/DesktopEditor baseline 保留；真实后端读取/静置无轮询/明确刷新通过 |
| R3 终端全屏 | Button全屏/恢复、分隔器拖入顶部38px吸附；占满右侧区域；单实例/单WS/零输入/零意外写入、多组、收起再开、常规尺寸持久化及刷新通过 |
| R4 登录页 | 统一灰色 chrome、细框线、8px微圆角；桌面双列、触屏横竖屏单列；焦点/Enter、密码清空/提交去重、错误/429冷却与成功返回通过 |

## 实际门禁

| 检查 | 结果 |
| --- | --- |
| web: npm run lint / npm run typecheck / npm test / npm run build | 最终源码四项通过；31文件、163单测；字体/37项第三方许可与生产构建通过 |
| go test ./... | 沙箱外复验全量通过，含自有httptest端口与私有tmux专项 |
| go vet ./... | 通过 |
| Chromium最终回归 | 18项通过，约1.6分钟 |
| WebKit最终回归 | 18项通过，约2.1分钟 |
| 真实本机后端git-requests | Chromium1项、WebKit1项通过；独立8103/5204服务、临时Git/SQLite、自有tmux socket及虚构测试口令 |
| Markdown本地链接 / git diff --check / 忽略规则 | 修改文档的本地链接检查及diff通过；依赖、产物和测试输出继续忽略 |

浏览器最终回归包含 editor-recovery（6）、login-modern-ui（3）、terminal-geometry（1）、workbench-interactions（1）、workbench-modern-ui（7）。每浏览器19项（含真实Git），合计38项；重复复验不计新增用例。

## 首轮问题与复验

- 首轮Chromium7通过/3失败：登录触屏横屏使用双列，是产品差距，媒体条件改为与工作台一致的760px/粗指针组合后通过；其余为测试期待原始429文案和遗漏布局库storage key前缀，按原统一错误契约与锁定库源码修正。
- 第二轮Chromium9通过/1失败：连接数断言取得2。期间并行构建/资产准备且Vite记录样式HMR，不能据此认定全屏动作必然重建连接。改为稳定源码、构建与浏览器分开执行；独立和最终双浏览器单WS/同DOM断言通过，不改终端连接实现。
- 独立终端首次采样发现4px偏移：editors=0时React尚未将separator收为0；测试改为同时等待实际separator=0，保留≤1px的几何标准，复验通过。
- 沙箱初次Vite/Chromium/Go门禁分别遭本机端口、浏览器Mach及tmux权限限制；自动审批允许沙箱外执行后完成。本机shadcn docs CLI因registry DNS失败，官方网页查找成功，复用已有Button/Tabs/Field/Input，未安装依赖。

## 复用与范围审查

trellis-check 与 React best-practices 检查：控件复用、中文aria/title、hook清理、ref临时布局、原始provider/Panel/runtime身份保持。无新增API、数据库迁移、认证/终端协议、依赖或权限提升。常规布局继续由useDefaultLayout/panelStorage拥有，0/100临时全屏不持久化。

同步组件、主题、终端运行时和搜索/Git owner规范；旧规范仅保留HEAD文字的要求已被本轮明确覆盖。未运行Go race：未改Go并发、bridge、watcher或session实现。

## 证据与限制

- 合成接口/真实xterm专项输出：`web/test-results/screenshot-fixes/chromium-final/`、`webkit-final/`。
- 真实后端专项输出：`web/test-results/screenshot-fixes/git-chromium/`、`git-webkit/`。
- 首轮与中间失败trace保留在同根的chromium、chromium-recheck、chromium-terminal目录，未用最终通过覆盖失败证据。
- 便捷预览：[登录页](../../../web/test-results/screenshot-fixes/login-light.png)、[终端全屏](../../../web/test-results/screenshot-fixes/terminal-fullscreen.png)。
- 本轮私有服务/fixture清理证据保留 `.cache/screenshot-fixes/cleanup.json`；用户已有开发服务未操作。
- 真实Debian/systemd、Firefox、物理手机软键盘及触控板未执行；本轮浏览器模拟不作为这些环境的通过证据，也不改变原任务未完成项。
- 构建保留已有大chunk提示，Vitest保留jsdom canvas环境提示；均未降低门禁或添加依赖掩盖。


## 追加截图反馈与最终复验

本节覆盖随后两张用户截图，已获直接执行授权。用户明确覆盖旧160px最小高度：分隔器应连续拖到顶部编辑标签区域，再继续拖才收起编辑器。

| 反馈 | 最终行为与验证 |
| --- | --- |
| 160px处提前停住 | editor minSize=38px、collapsedThreshold=1px；真实鼠标按住后连续经过140/90/48/38px，均匹配目标≤1px，再拖到24px才吸附。1px容差防止浮点尺寸在38px边缘提前收起 |
| 全屏顶部不能下拖 | 水平separator最大化仅0px布局，保留顶部8px热区；真实鼠标下拖220px后继续调整。按钮恢复、多组、收起再开、持久化和刷新通过；同Monaco/live DOM、单WS、零输入/写入 |
| 标签/路径/底部文字未居中与工具栏缺口 | 标签行与固定动作38px，滚动容器42px/负4px下边距；轨道覆盖路径顶部，固定工具栏齐平；标签20px、路径18px、状态16px明确行高，Range文字中心误差≤2px，深浅两端截图复核 |
| 滚动时偶发内容消失 | 确认历史字节到达即隐藏live、异步write/rAF尚未绘制的空白窗口。历史改为独立覆盖层，绘制及viewport同步后显示；live仅visibility隐藏且尺寸保留。首次读取、刷新、三视口、返回实时连续逐rAF采样：旧实现10个空白采样帧；最终Chromium/WebKit均0，同时单连接、零shell输入、固定DOM |

最终浏览器：`chromium-verified` 27项通过（1.6分钟）、`webkit-verified` 27项通过（2.3分钟），共54次浏览器执行、27个不同用例。包含editor-recovery 6、login 3、device-attributes 2、geometry 1、scrolling 7、interactions 1、modern-ui 7。新的终端专项使用真实xterm/decoder及隔离合成HTTP/WS；此轮未重新启动真实后端Git服务，上一轮证据保留。

中间结果保留：首次Chromium13通过/1失败（动态阈值切换导致同一次向下拖动停在160px，移除动态阈值后通过）；旧源码对照首先发现实时宿主display:none无几何，再用完整逐帧对照捕获10次空白；`chromium-continuous`在38px边缘提前收起，改1px容差后逐点通过。一次修复命令使用了错误工作目录而未修改文件，检测后中止测试并正确应用。`chromium-final`是用户新反馈到达后主动中断的旧阈值批次（15项通过、1项中断、11项未运行），不记为最终通过；最终证据在`*-verified`，不覆盖中间trace。

本轮Go test/vet全量命令通过（缓存命中）；前端最终npm lint/typecheck/test/build四门禁通过：31个单测文件、163项通过；字体和37项许可及生产构建通过。保留已有jsdom canvas和大chunk提示，未抑制或降低门禁。scope、WS effect依赖和后端协议未改，新增绘制就绪状态绑定具体historyBytes并沿现有active/rAF清理；独立历史覆盖层位于手机快捷行之外。React及trellis-check审查通过，无新增依赖、API、迁移、权限或日志正文。

预览：[标签与工具栏衔接](../../../web/test-results/screenshot-followup/tabs-light.png)、[全屏](../../../web/test-results/screenshot-followup/terminal-fullscreen.png)、[返回实时画面](../../../web/test-results/screenshot-followup/history-live.png)。完整输出在`web/test-results/screenshot-followup/`，服务5203已停止；没有修改用户其他服务。真实Debian/systemd、Firefox、物理手机软键盘和触控板仍未执行；用户设备上的全部偶发滚动现象不能从本机合成测试推定已完全消除。
