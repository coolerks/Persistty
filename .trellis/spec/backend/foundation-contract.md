# W01 基础协议

## 1. 范围与触发
用于首个可运行认证服务与浏览器路由。原bootstrap示例与2026-09-27批准的多项目/VPN HTTP/独立多设备登录冲突时按本协议及对应owner更新执行。不提前开放未实现文件/终端写接口。

## 2. 签名
API前缀`/api/v1`：POST auth/login、GET auth/session、POST auth/logout、GET projects、GET projects/:id、GET terminals，注册归[router](../../../internal/httpapi/router.go)。CLI为persistty serve --config与persistty password，归[main](../../../cmd/persistty/main.go)；密码CLI仅TTY输入，不接受密码参数。[配置](../../../internal/config/config.go)及[迁移](../../../internal/storage/migrations/0001_metadata.sql)为实际字段与schema来源。

## 3. 契约
成功`{data,request_id}`，失败`{error:{code,message},request_id}`，字段snake_case。login请求只有password；session响应authenticated=true/expires_at/csrf_token，匿名401；logout携带X-CSRF-Token，成功204。项目列表data.items最大200，超限明确错误。Project为id/name/version/main_folder_id/folders，Folder为id/path，version正JS安全整数，非空folders与唯一主folder；项目详情不存在404。Terminal为id/display_name/project_id（可null）/working_directory/state，目前只有unavailable，未做真实观测不得标running。列表读真实SQLite，空为items=[]。不开放写项目/创建PTY/通用shell端点。

共享[fixture](../../../tests/contracts/foundation.json)由Go序列化测试与[TS decoder](../../../web/src/lib/api/decoder.ts)消费。session secret只在HttpOnly Cookie，CSRF由domain-separated SHA256(secret)派生，DB仅token hash/expiry，不持久化CSRF明文；客户端CSRF仅内存。模式明确development/vpn_http/tls，Go仅loopback；所有写入严格Origin，认证写入另CSRF。数据库布局与theme不进DB。

资源界限：JSON 8KiB、password至多1024字节（CLI新设至少12）、Argon2并发2、来源10次/分钟与全局60次/分钟、最多200有效sessions及200列表条目。限流map有界；超限显式错误，不增加无界worker。配置TTL1分钟到30天、默认168h。Argon2 PHC参数有上下界；password_hash变化在启动事务内撤销全部认证sessions，不操作终端。SQLite driver modernc.org/sqlite纯Go无需CGO，时间存储固定9位UTC避免expiry字典排序错误，响应仍RFC3339Nano。

## 4. 验证与错误矩阵
匿名资源401；Origin/CSRF错误403；非法/重复JSON键/未知字段400；超body413；限流429；不存在404；SQLite不可用503；内部故障脱敏500。unknown API不回HTML。密码和token不进request_id/日志/JSON错误。配置错误在listen前失败，Cookie模式显式不可降级。

## 5. 正常、基础、错误用例
正常：两个设备分别登录，登出其中一端不撤销另一端；基础：空库登录后读取空项目列表。错误：为了展示页面写入演示项目、缺资源自动创建、从缓存推断terminal running。

## 6. 所需测试
真实临时SQLite迁移/权限/checksum/rollback、高版本拒绝；完整Gin注册保护矩阵、TTL/注销、Cookie/Origin/CSRF/XFF、重复键、未知字段、并发限流和日志秘密检查；Go与TS fixture合法/非法decoder；真实本地服务登录/退出/直达项目404/SPA与API分流。
迁移修复必须通过新版本 SQL，不重写旧 checksum；旧库升级保留 session 和所有资源，NULL ID 数据拒绝且回滚。请求 context 取消属于正常取消，不记录为服务器 500/ERROR。根 Go 模块测试通过 web/go.mod 隔离 npm 中的外部 Go 源码，不把 node_modules 当作产品包测试。

## 7. 正反例
错误：`response.json() as Project`后直接渲染；正确：统一client将unknown交由decoder检查一次，再传给组件。错误：读取DB失败返回空项目列表；正确：503和错误状态，空列表只代表读取成功且真实无条目。
