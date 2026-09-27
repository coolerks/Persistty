# W01 后端实施记录

## 边界与实际交付
只修改`go.mod`、`go.sum`、`cmd/`、`internal/`、`tests/contracts/`及本文。其他代理负责前端、部署、规范和主任务记录；没有撤回他人修改。没有访问远端、安装root服务、操作防火墙、创建tmux/PTY、提交或推送。

已实现普通用户配置读取、TTY密码CLI、Gin认证路由、Argon2id校验、有界登录速率与并发、SQLite认证会话和真实项目/终端元数据读取。未实现的项目写入、文件、终端连接和提权没有注册假成功接口。终端状态仅`unavailable`，不将元数据作为进程存活事实。

## 版本与代价
- 官方Gin发布页核实并锁定`github.com/gin-gonic/gin v1.12.0`：[发布记录](https://github.com/gin-gonic/gin/releases/tag/v1.12.0)。
- 官方包文档核实`modernc.org/sqlite v1.59.0`：[驱动文档](https://pkg.go.dev/modernc.org/sqlite@v1.59.0)。纯Go驱动，无运行时CGO或系统libsqlite依赖；保留上游配套`modernc.org/libc`等间接版本，不能单独升级生成代码依赖。代价为较多生成代码、依赖和构建体积；真实Debian负载与磁盘限制仍未实测。
- `golang.org/x/crypto v0.57.0`、`golang.org/x/term v0.46.0`、`go.yaml.in/yaml/v3 v3.0.4`从官方Go包源核实，实际manifest与sum锁定。最新crypto要求Go1.26，因此将本地1.25.6自动选择的维护工具链显式锁为`toolchain go1.26.8`，模块最低Go1.26.0；[官方维护版本](https://go.dev/dl/?mode=json)包含1.26.8。
- 初次沙箱内下载因DNS限制失败，随后授权下载完成；测试/构建访问Go工具链与缓存也使用授权执行。没有以下载失败代替测试通过。

## 配置与协议
完整字段仅为`server.listen/public_origin/mode/trusted_proxies`、`auth.password_hash/session_ttl`、`storage.path`。未知键、重复键、多YAML文档、非法模式/源/hash/TTL和非私有配置拒绝启动。所有模式后端均明确loopback IP监听；`vpn_http`由同机Nginx绑定VPN地址，`tls`由同机代理终结HTTPS，不从任意代理头推断Cookie安全。只有显式loopback可信代理可用于来源IP；没有宽松CORS。

默认listen为`127.0.0.1:8080`、登录TTL为`168h`。TTL合法范围1分钟至30天。Cookie均HttpOnly、Strict、Path=/、无Domain；TLS为Secure的`__Host-persistty_session`，VPN HTTP/development为`persistty_session`。每次登录CSPRNG32字节独立token，DB仅存SHA256和expiry；CSRF从带固定域前缀的token派生，只返回已认证响应、绑定当前会话。退出仅撤销当前认证；配置密码hash变化在启动事务中撤销所有认证，不触碰终端记录或进程。

Argon2id默认64MiB/t=3/p=1，解析允许64..256MiB/t=3..6/p=1..4，salt16字节/key32字节。新密码CLI最低12字节，最大1024字节；登录接受非空至1024字节，始终限制PHC参数后才hash。CLI只从TTY隐藏输入并确认，不接受argv或管道密码；stdout只输出hash。不能把本地hash速度作为目标Debianbenchmark。

登录来源10次/分钟、全局60次/分钟、Argon同时最多2个；无无界hash队列，来源表有界。会话上限200；项目、每项目folder、终端列表上限200，超过显式413`too_large`，不静默截断。JSON body上限8192字节、深度16、要求有效UTF-8、精确snake_case字段，拒绝重复（包括转义等价）键和额外顶层值。

API只有子设计列出的6个路径、共享envelope与fixture。匿名未知API也401，已认证未知资源404，不返回SPA HTML。所有响应no-store与服务生成request_id。Origin写入严格相等；logout额外CSRF。SQL busy/locked503；未知内部故障500；恢复和日志只记录白名单字段，不记录body、Cookie、password/hash、CSRF、完整URL或panic值。

## 数据与迁移
数据库目录0700、DB/WAL/SHM0600、属于服务用户，拒绝现存不安全权限及symlink。单连接串行、DSN每连接启用foreign_keys/WAL/5秒busy_timeout；没有`:memory:`替代真实文件测试。迁移SQL和版本checksum单事务，checksum变化、高版本、失败均拒绝启动且不部分发布。

projects/folders使用deferred复合FK维持主folder属于本项目、非空关联和正JS安全version；terminal引用project采用`ON DELETE SET NULL`而非级联删除。主题、布局、正文、草稿和终端输出均不入库。会话expiry使用固定9位UTC小数存储，仍是RFC3339合法时间；清理不使用可变小数文本排序或SQLite毫秒舍入，API时间仍RFC3339Nano。

## 验证与启动
- `gofmt -w cmd internal`：完成。
- `go test ./...`：通过，包含CLI/config/PHC/独立会话/TTL/并发限流/真实SQLite迁移与权限/完整注册路由/CSRF/Origin/XFF/日志脱敏/共同fixture。
- `go test -race ./...`：通过。
- `go vet ./...`：通过。
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/persistty-w01-linux ./cmd/persistty`：通过，只是本地交叉编译，不是Debian运行验收。
- `git diff --check`：通过。`go test ./...`额外发现前端已安装依赖中的`flatted/golang`包（无测试）；上述应用包分别显示通过，未修改依赖源码。

主会话可用`go run ./cmd/persistty password`的TTY交互生成临时开发hash，将仅本地临时配置设0600，配置`development`与`http://127.0.0.1:5173`，独立metadata目录0700；再`go run ./cmd/persistty serve --config <临时配置>`，Vite代理8080。没有提供或提交通用开发凭据。长期开发服务、真实浏览器联调由主会话负责；本代理未启动长期服务器。

目标Debian/systemd、真实WireGuard隔离、TUI恢复、多端PTY、helper和真实手机仍由W02及后续任务验证，本记录不宣称完整首版完成。
