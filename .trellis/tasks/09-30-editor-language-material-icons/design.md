# 集成设计与最终规划

## 目标与边界

采用完整 Monaco registry metadata 修复语言被降级为 plaintext 的问题；采用锁定 Material Icon Theme 默认生成 manifest 与 SVG 实现树/标签一致图标。主会话单代理执行，保持现有 API、安全文本分类、只读编辑、手机基础文本视图与终端生命周期。

## 子任务与共享数据

先[语言设计](../09-30-monaco-language-coverage/design.md)，后[图标设计](../09-30-material-file-icons/design.md)。共享唯一轻量路径语言 adapter，不在 Explorer 引入 Monaco 或 worker。父任务负责集成验收，无独立产品代码，不运行父任务 start。

语言完整清单为 91 个 ID；无后缀语言通过桌面现有内容栏的“语言模式”选择可达。提案保留“自动识别”，手动选择暂存在当前文件视图中，不跨页面刷新永久保存。图标仍按自动文件类型解析，选择 SQL 方言不改变树的路径图标。

图标锁定 material-icon-theme 5.38.1/default manifest，保留 exact filename/最长后缀/语言回退、patterns/clones 展开、目录开闭/root/light。静态资产本地生成和分发，浏览器只加载 lightweight metadata 与必要 img，MIT 与 checksum 清单随包发布。

## 最终评审需要确认的提案

批准本规划将同时批准桌面“语言模式”选择及临时视图记忆策略；这是确保六个 FreeMarker 变体、mysql/pgsql/redshift 等无独立扩展名模式可用的入口。其他用户要求已有明确依据，无另一个阻塞性产品问题。

## 验收与风险

完整 ID 注册还须真实 tokenizer 加载与合法样例 token 验证；别名与示例还须全量资源存在/校验/许可证检查。实施记录前后 bundles、worker、SVG 数量/体积与主题/键盘/布局的浏览器证据。依赖内部路径与上游 API被当前版本锁定，升级须重新核对；页面刷新后手动语言选择恢复自动属于明确提案。

若缺真实隔离浏览器实例，先完成可运行的本地检查与 fixture，再如实列未验收项，不声称元数据研究已证明产品运行通过。回滚无 API/数据库或真实资源销毁，只撤回本次 UI、adapter、生成脚本与依赖配置。
