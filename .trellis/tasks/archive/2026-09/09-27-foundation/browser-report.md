# W01 本地浏览器联调

## 环境与边界
2026-09-27主会话运行loopback Go后端8080与Vite5173，模式development，私有开发配置/SQLite位于忽略的.cache/dev。使用TTY密码CLI生成仅本地开发PHC，输入不回显；配置0600、数据库目录0700，测试凭据不是生产密码。使用agent-browser 0.38.1的独立persistty-foundation会话，已关闭此浏览器会话，未接触用户其他标签。

## 实际观察
- 桌面1280x577登录与项目面板渲染，浏览器errors为空，无Vite overlay/空白/横向溢出。
- 填写临时开发密码后真实登录成功，GET projects从实际空SQLite返回items=[]，没有演示项目；刷新仍保持认证。
- /projects/missing-project书签直达显示“项目不存在”，提供项目面板及终端入口；不自动创建资源。
- 手机尺寸390x844显示登录/项目面板，无横向溢出、文字遮挡或桌面多列；这只是viewport模拟，不是软键盘/iOS/Android实机验证。
- 用户菜单切换深色，刷新仍为深色；system下模拟media light/dark实时更新；显式light在系统dark时不被覆盖。
- localStorage键名检查只发现persistty.theme.v1，没有密码/token/CSRF持久化。
- 点击退出后回登录，浏览器同源GET /api/v1/projects实际401 JSON code=unauthenticated，不仅前端隐藏。

## 证据与未执行
截图位于/private/tmp/persistty-login-desktop.png、persistty-projects-desktop.png、persistty-projects-mobile.png、persistty-projects-mobile-dark.png、persistty-login-mobile.png，主会话均检查相关桌面/手机项目截图。截图不是产品正式资产，不写入运行库。

最终复核：数据库实际版本从 1 升到 2；重建后端并重启，已有浏览器 session 仍有效。独立 persistty-foundation-final 会话再次完成 logout 后同源项目 API 返回 401/unauthenticated、重新 login、missing-project 404 页面；390x844 无横向溢出，errors 为空。/private/tmp/persistty-final-mobile.png 已目视检查。该轮使用全新浏览器存储，localStorage 为空；先前主题持久性验证仍如上记录。完成后关闭本任务独立浏览器，保留本地开发服务供用户查看。

未执行Nginx/systemd实机、VPN公网隔离、真实手机软键盘/输入、PTY恢复/多设备控制/倒计时/helper。仅W01行为已观察，不能据此宣布PA01..PA26通过。
