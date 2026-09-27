# W01 基础服务与认证工作台

## 授权与依赖
父任务为[已批准首版需求](../09-26-requirements-research/prd.md)。用户在最新最终总结后于2026-09-27明确要求“开始实施”，批准范围和设计默认建议。本子任务只是父计划W01的执行分解，不新增产品范围。直接依赖：父规划及来源U01..U73，无其他实现包。

## 变更边界
当前无产品源码，最小差距是没有真实服务、认证、元数据和可访问页面。配置/认证事实归Go服务，项目定义归SQLite，浏览器主题与布局归本地。新增cmd/persistty、internal/config/auth/storage/httpapi、web、deploy、tests/contracts及必要构建文件。每个目录仅创建实际使用代码，不铺空功能包。

本包建立真实认证和持久项目读取端点，项目列表从SQLite读取而不是演示数据；项目创建维护、文件树编辑、真实PTY、Git/搜索/上传/helper由后续包交付，不注册假成功接口，不以空壳工作台宣称全部首版完成。没有用户数据迁移或局部重构。不得操作远端、安装root服务、改变防火墙、提交或推送。

## 验收
- F01 配置拒绝未知/非法字段，显式development/vpn_http/tls模式，不静默降级，普通用户运行。
- F02 Argon2id密码CLI无argv密码；登录多设备独立Cookie、TTL/退出撤销、SQLite仅保存token哈希，Origin/CSRF/并发限流真实执行。
- F03 SQLite文件权限、事务迁移/checksum/高版本拒绝；项目/folder/terminal协议身份隔离且无进程级联删除。
- F04 路由注册级鉴权测试；统一错误、request_id、重复JSON键/未知字段/体积限制、敏感响应no-store，无secret日志。
- F05 React真实登录/退出、项目列表/书签直达/失效项目和终端入口，三种主题独立浏览器保存，桌面/手机适配；未知项目404不伪造资源。
- F06 API fixture在Go与TS验证，前端严格DTO解码，四门禁及Go test/race/vet通过；Nginx页面/API/资源分流，systemd示例不控制终端服务。

## 剩余交付
W02目标Debian实验仍需独立环境与系统操作授权；W04项目配置文件安全、W03持久终端、W05编辑布局、W06搜索Git、W07提权、W08完整E2E按父计划依赖推进。W01不通过不进入依赖包。
