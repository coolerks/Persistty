# 多文件夹项目与路由补充调研

日期：2026-09-27。依据用户U63..U66；仅文档研究，无产品实现、依赖安装或浏览器/目标机实验。本文件补充初次单根调研，最新需求以[PRD](../prd.md)为准。

## 官方能力证据

- React Router的声明式安装文档支持在Vite React应用中使用BrowserRouter；无需为了项目路由引入SSR或新的Node生产服务。[安装文档](https://reactrouter.com/start/declarative/installation)
- SPA部署必须把客户端页面路由回退到入口HTML；这不意味着把API/WS或缺失静态资源也回退成HTML。[SPA部署文档](https://reactrouter.com/how-to/spa)
- VS Code多根文档展示多根文件树、按根筛选搜索和多仓库概览，可作交互参考；Persistty不因此引入其扩展、工作区文件或Git写功能。[多根工作区](https://code.visualstudio.com/docs/editing/workspaces/multi-root-workspaces)

## 仓库证据与冲突

- 现有frontend/state-management.md是初始共享偏好约定；用户U37已确认各浏览器独立界面，新项目模型按项目分命名空间，不能恢复成多设备互相覆盖布局。
- 现有backend/database-guidelines.md尚未定义多根项目/孤立终端数据模型，无产品代码或数据库可迁移。实施前由owner更新规范，不把候选字段宣称为现有API。
- 旧U53/U56的单根状态栏已由U63替代，U55的目录权限边界及文件API安全原则保留。U65要求移除项目后原终端仍有入口，不可用外键级联删除终端或调用tmux kill。

## 设计推论与待实施验证

项目使用不可复用稳定ID，名称/主文件夹改变不使书签失效。关联文件夹使用独立ID与配置版本；所有旧保存/替换/上传/CLI调用须查当前关联及真实目录身份，不能把浏览器旧缓存当有效权限范围。

文件身份去重与路径显示分开：两个根下的src/index.ts不是同一文件；重叠根下同一目标不能执行两遍替换，两个仓库的HEAD/缓存也不能混合。根外Git元数据沿用既有明确注册验证原则，不能因多仓库识别默许根外内容读取。

建议按项目恢复本浏览器最后有效视图，同一项目多tab初始化后独立编辑，保存恢复快照有顺序版本；不实时同步dirty buffer。项目/目录失效只标不可用并保护草稿，不自动重建。

实验验收：Nginx直达/刷新/登录返回路由、原生新tab/opener隔离、项目移除原PID连续、同名/重叠根、配置变更与提交竞态、草稿存储失败/其他端调整、多个仓库/嵌套仓库/范围外gitdir。尚未执行，路由版本与扫描资源限额实施时锁定。
