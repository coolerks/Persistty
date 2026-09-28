# W01 设计

继承[父设计](../09-26-requirements-research/design.md)。当前实现仅基础服务及认证，真实SQLite项目读接口供路由接入，后续项目写入归W04。使用Go/Gin、Argon2id、database/sql SQLite，前端React/Vite/React Router与shadcn官方基础控件。实施代理核实官方维护版本并锁定依赖、记录选择，不随意跟旧示例复制API。

## 共享契约
- `POST /api/v1/auth/login`请求`{"password":"..."}`，响应`data={authenticated:true,expires_at:<UTC>,csrf_token:<string>}`。
- `GET /api/v1/auth/session`已登录同登录响应，匿名401。`POST /api/v1/auth/logout`带CSRF，成功204。
- `GET /api/v1/projects`响应`data={items:Project[]}`，最大200，超限显式错误不静默截断。`GET /api/v1/projects/:id`真实存在返回Project，否则404。
- `Project={id,name,version,main_folder_id,folders:Folder[]}`；`Folder={id,path}`，项目version正安全整数；folders非空、主folder在列表中。后续W04负责注册普通目录身份、配置修改与失效裁决；W01不接受客户端写入目录。
- `GET /api/v1/terminals`读取真实已有元数据，响应`data={items:Terminal[]}`；当前无创建入口，不自动启动tmux。`Terminal={id,display_name,project_id:string|null,working_directory,state:"unavailable"}`，W03才能增加已实验状态与attach协议。
- 成功envelope为`{data,request_id}`；错误`{error:{code,message},request_id}`。CSRF仅内存，Cookie不可读，不持久化任何凭据。受保护资源无session先401，未知API也不回HTML。
- 布局协议只定义水平editor/terminal组和四种类型化命令，不实现假终端关闭或连接。主题本浏览器system/light/dark，未知本地快照忽略而不写后端。

具体配置字段、DB迁移及真实API fixture由后端owner定义并落规范，前端按上述契约解码，不各自扩展协议。前端可用Vite同源代理至loopback后端进行真实测试。
