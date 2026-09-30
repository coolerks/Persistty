# Material 图标设计

## 资源生成与分发

- 使用 `material-icon-theme@5.38.1` 精确 devDependency 与 npm 锁文件 integrity。仅构建脚本调用发布 API `generateManifest()`，不将 Node/VS Code 模块或 chroma-js 引入浏览器。
- 上游默认 manifest 保留 fileNames/fileExtensions/languageIds/folder/rootFolder 开闭及 light 映射；patterns/clones 已由上游展开。解析输入均来自锁定生成物，不自行解释上游源码 enum 或复制映射子集。
- 新生成脚本将 manifest 中被引用 SVG 复制至受限的本地目录，包含所有 light variant 和默认 clone；manifest iconPath 重写成由安全 icon ID 对应的本地 URL。先校验源路径处于包 icons 目录、扩展为 SVG，生成时逐个验证存在及摘要。
- 生成物位于明确的 web/public 与 web/src 生成目录，开发/测试/build 之前统一生成，输出通过 .gitignore 排除；保存脚本、package-lock、源版本/提交/integrity记录及上游完整 LICENSE。开发不依赖生成物已存在，干净 npm ci 后能重现；Vite base 下 URL正确。
- 构建附 asset 清单：源 URL、5.38.1、gitHead、发布包 SHA-256/integrity、每个 SVG SHA-256、复制/路径改写记录。SVG 字节不做转换；将完整 MIT LICENSE 加入 collect-licenses 分发清单，devDependency 的资产许可不能被运行依赖扫描遗漏。

## Resolver 与消费者

纯 typed adapter 接收 basename、file/directory/root、expanded、effective theme 与路径推断 languageId。文件按 exact basename → 最长 compound suffix → extension → language → file；对 `.env` 按 env 后缀处理，避免普通 extname 把隐藏文件错当无后缀。小写 lookup，仅由生成表映射 icon ID，用户文件名不能拼成资源路径。light 表按同层覆盖默认表，再执行相同解析优先级。

目录按 rootFolderNames/folderNames 与对应 Expanded 表处理，无映射回退 rootFolder/folder 开闭图标。保留资源管理器 chevron，图标不兼任展开控制。

workspaces 新专用图标组件（不是基础交互控件）使用 `<img alt="" aria-hidden>` 和固定尺寸，禁用图片默认拖拽；失败只回退一次到本地通用图标，避免无限 onError。复用既有主题 owner 或单一监听，不给每个树节点新建 MutationObserver。树、根目录、标签、目录选择器共同使用。

图标根据文件路径与自动语言推断决定，手动选择着色模式不改变树中文件类型图标，避免未打开树节点依赖正文或编辑器视图状态。移动/重命名后按新的 basename 重新解析；不会为图标 fetch 文件正文或访问外网。

## 安全与兼容性

trusted 内置 SVG 与用户工作区 SVG 不混用；用 img 加载，不 innerHTML/object，不修改任何后端用户预览隔离或鉴权。真实 API、文件分类、版本、权限和终端资源生命周期保持现有契约。

采用上游默认 angular icon pack 对其特定文件名沿用官方含义；本轮无图标包切换 UI。主题只支持产品现有 light/dark/system；不新增高对比模式，可保留源清单完整记录。

## 风险与回滚

1,251 个 SVG 会增加分发文件数量，但 img 按可见条目请求、JS 只装轻量映射；记录静态资产总体积，不内联全部 SVG。生成与路径校验必须保证无残留旧版本文件；只清理专属生成目录。

上游 API与包目录由精确版本锁定；升级须复核并重新生成。回滚消费者、生成 script/devDependency/锁文件和 ignore/许可入口，无数据库迁移。遵守已有用户修改保护。
