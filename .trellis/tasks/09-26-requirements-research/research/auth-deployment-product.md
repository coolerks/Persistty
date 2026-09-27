# 认证、部署与产品边界调研

- 查询：单密码的实际权限、长期连接授权、现有 WireGuard/Nginx 环境、升级恢复与同类方案能解决什么问题。
- 范围：混合；官方资料与本地初始规范。
- 日期：2026-09-26。
- 状态：文档调研；未访问 Debian，未运行产品或部署实验。

## 本地证据

| 文件与定位 | 内容及证据性质 |
| --- | --- |
| `.trellis/spec/backend/security-config.md:4` | 服务 UID 的 shell 权限，allowed_roots 不是终端沙箱；初始契约 |
| `.trellis/spec/backend/security-config.md:36` | 生产 HTTPS 与配置失败拒绝启动；初始契约 |
| `.trellis/spec/backend/security-config.md:37` | Argon2id 参数与有界验证；尚未做目标硬件 benchmark |
| `.trellis/spec/backend/security-config.md:38` | Cookie、DB token hash、TTL、撤销；初始契约 |
| `.trellis/spec/backend/security-config.md:39` | CSRF、Origin、代理信任与存量 WS 撤销；初始契约 |
| `.trellis/spec/backend/database-guidelines.md:16` | DB 与 tmux 不可原子提交，需要协调，不通过误杀任务回滚 |
| `.trellis/spec/backend/database-guidelines.md:40` | 备份需一致性，不能只复制运行中主数据库 |
| `README.md:5` | 没有产品实现，不能声称生命周期或安全已验证 |

## 1. 单密码指的是一个用户身份，不等于一个浏览器会话

U05 要求配置密码、后端强制鉴权。密码应以自适应 hash 保存；[OWASP 密码存储指南](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html) 支持 Argon2id。仓库的 64 MiB/t=3/p=1 比指南最低配置更重，但是否适合闲置电脑需实测，并发 hash 数量必须有上限。不存在“单用户就不需要限流”的推论。

同一人可以有多个认证 session。登录生成新随机凭证可防固定会话攻击，但“新登录撤销所有旧设备”是额外产品决策。当前 security-config 的“旧 secret 失效”未明确仅替换当前 session 还是全局撤销；不能代用户选择。Cookie 标签页共享也意味着同浏览器多个标签不能当成独立账号。依据：[OWASP 会话指南](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)。

候选行为：登录恢复资源；退出登录/TTL 到期只撤销对应控制连接，后台任务继续。是否提供全设备登出、修改密码是否立即撤销全部 session、是否记住登录、TTL/闲置期限，需要澄清。关闭页面是传输离线，不能等同于用户请求终止终端。

## 2. HTTP 鉴权必须覆盖所有资源，WS 授权必须持续有效

官方 [OWASP WebSocket 指南](https://cheatsheetseries.owasp.org/cheatsheets/WebSocket_Security_Cheat_Sheet.html) 指出浏览器握手自动携带 Cookie，应验证可信 Origin；已建立连接还需处理认证过期。严格匹配 scheme/host/port，不允许通配或字符串子串匹配。

候选验收：匿名不能打开终端、列表、文件、搜索、Git、上传、下载或预览；撤销后已有连接不能再执行输入/写入；重新登录仍能找回原任务。密码错误与未认证失败不泄露机密，日志不记录 Cookie、键盘输入或终端输出。

上传 complete、批量替换 apply、终端 close 等有副作用的操作都要重新检查有效授权；“创建任务时已登录”不能无限授权后续提交。进行中的写入在撤销前已获准时可能完成，需要任务设计划定提交点，不能承诺撤销能逆转已执行副作用。

## 3. 公网与 WireGuard 是两个部署选择

用户已有公网 IP 与 WireGuard，但没有明确最终网页必须直接对公网开放。[WireGuard](https://www.wireguard.com/) 提供网络隧道；推论：可以先限定经 VPN 访问，减少公开入口，但仍保留用户明确要求的应用鉴权。它不能取代浏览器 HTTPS Origin、Cookie 或终端生命周期管理。

| 选择 | 价值 | 代价与未知 |
| --- | --- | --- |
| WireGuard 内访问 | 复用现有通道，不必开放网页公网入口 | 客户端需 VPN；仍需可信 HTTPS 或明确受限开发模式，内网名称/证书需确定 |
| 公网 HTTPS | 任意符合条件浏览器登录 | 域名/证书/路由/限流/更新/暴露面维护要求更高 |
| 两种入口 | 兼顾方便与备用 | 多 origin/cookie/重定向/证书与会话行为更复杂，不能默认首版支持 |

部署入口也是后续澄清优先项。推荐以单个固定 HTTPS origin 为初始设计方向，具体访问方式由用户决定。生产不应因经 VPN 访问就自动取消 Secure Cookie；依据：[Set-Cookie](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Set-Cookie)。独立域名与子路径部署的静态资产、WS 路由和 Cookie 边界不同，尚未选择。

## 4. 反向代理行为会直接影响“网络稳定”和“大文件可用”

[Nginx WebSocket 文档](https://nginx.org/en/docs/http/websocket.html) 要求正确转发升级头，并说明默认上游无数据 60 秒会断开；心跳/超时应纳入无输出终端测试。这是连接问题，不应造成任务终止。

[Nginx core 文档](https://nginx.org/en/docs/http/ngx_http_core_module.html#client_max_body_size) 中 client_max_body_size 默认 1m，超过返回 413。[proxy 文档](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_request_buffering) 的请求缓存默认开启，可能先收完整请求再送后端。推论：只提高应用的 20 GiB 总文件上限不能证明上传成功；分块请求上限、代理暂存磁盘、应用暂存、超时与取消需要一起核对。没有安装配置或压测，不给出可直接部署的最终参数。

真正 Nginx restart 与 reload 可能影响既有连接不同；验收关注后台进程连续与重连可用，不要求所有网络连接永不关闭。反向代理健康不代表 tmux 健康；状态页面应区分认证失败、服务不可用、终端已退出、终端连接占用。

## 5. 必须复用实际开发用户环境，但权限身份尚待选定

U01 已有 AI 工具、Oh My Zsh 与 Git 环境。运行在另一专用 UID 下可能无法使用原有配置、工具 PATH、凭据或文件权限；运行在原用户下则获得其实际权限。初始规范采用同一非 root UID 管理 Web/tmux，但尚未说明具体 UID。

候选验收：登录后终端中指定 AI 工具、Go、Node/npm、zsh 插件和中文 locale 可用；不通过 Web 修改原用户 shell 配置。环境排查只输出允许的版本与存在性，不能打印全环境或凭据。Terminal shell 的正常开发行为与 File API 根目录保护是不同边界，不能宣传 allowed_roots 限制 shell。

## 6. 升级、备份和故障恢复分别定义

Web 升级只替换/重启 Web；tmux unit 升级或 stop/restart 可能终止任务。不要建立将 Web restart 传播到 tmux 的依赖。官方 [systemd.unit](https://raw.githubusercontent.com/systemd/systemd/main/man/systemd.unit.xml) 解释 PartOf 的停止/重启传播；具体 unit 设计归终端研究。

SQLite 保存标签与偏好，不保存进程内存或文件正文。[SQLite Backup API](https://www.sqlite.org/backup.html) 支持一致在线备份。推论：备份元数据不能恢复活的 AI 进程，也不包含工作区文件；备份范围至少应分别说明配置、DB、工作区文件和草稿。数据库迁移失败时 Web 不服务，原 tmux 任务仍应继续，修复后重新连接。

浏览器缓存旧前端与新 WS 协议不兼容时应明确刷新提示，不进行无限重连或重新创建会话。主机重启、断电、OOM、管理员杀进程、任务主动退出是实际终止原因，不能把“只有显式 Close 才由 Persistty 终止”写成任何情况下都不退出的保证。

## 7. 同类能力比较：用于校准范围，不替换用户选择

| 路线 | 官方可查证能力 | 对本需求的含义（推论） |
| --- | --- | --- |
| ttyd + tmux | [ttyd 官方例子](https://github.com/tsl0922/ttyd/wiki/Example-Usage) 展示 tmux 共享进程连接 | 可参考终端桥接；未证明其具备本产品的工作区、冲突编辑、传输与精确会话管理 |
| code-server | [官方使用指南](https://coder.com/docs/code-server/guide) 介绍密码认证、HTTPS 与开发服务代理 | 提供更完整 IDE，可参考工作流；未核对其服务重启保留原进程的硬性验收，不能据“浏览器 IDE”推定满足 |
| Persistty | 用户指定 Go/React 技术栈与终端优先目标 | 能按核心生命周期定制，需承担协议、安全文件写入与运维验证成本 |

“npm run dev 一直运行”只要求进程存活；“在外部浏览器打开开发页面”另需端口可达/代理/HMR WS。助手提出端口视图不意味着用户已经要求任意端口反向代理；将它留作单独范围问题。

## 用户故事与候选验收

| 场景 | 候选可观察结果 | 阶段与证据 |
| --- | --- | --- |
| 切换 Wi-Fi/手机热点 | 页面重连或提示重新登录，原 A/B/C 仍持续 | 网络 E2E；未执行 |
| 登出/TTL 到期 | HTTP/WS 控制被撤销，原任务心跳不断 | 真实 HTTP/WS；未执行 |
| 密码变更 | 旧密码失效；既有 session 撤销范围按澄清结果 | 身份语义未定 |
| Web 升级/迁移失败 | tmux 不重启，失败可诊断，修复后重 attach | Debian/systemd；未执行 |
| Nginx 大文件/空闲终端 | 分块不过代理上限、暂存有界；空闲后仍可操作 | 真实代理集成；未执行 |
| 备份与恢复 | DB 与文件分别核对；不声称恢复进程内存 | 灾备演练；未执行 |

## 待澄清与待实验

用户决定：访问入口与域名/路径、支持哪些设备、登录保持期限/多设备/密码变更撤销、运行 UID、端口页面预览是否需要、首版备份/升级体验。

后续只读环境采集：Debian release、kernel、systemd/tmux/Nginx、UID/shell、工具版本、可用 RAM/磁盘、浏览器与证书条件。当前未获远端操作任务授权，不主动连接采集。

待实验：目标硬件密码耗时与并发内存、代理心跳与分块、真实鉴权撤销、Web 升级不杀任务、迁移故障恢复、一致备份。官方网页默认值不是用户现有部署实际值。
