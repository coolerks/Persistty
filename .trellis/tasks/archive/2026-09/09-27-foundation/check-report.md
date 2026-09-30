# W01 最终审查与自修记录

日期：2026-09-27。审查对象是基础服务与认证工作台，不是完整首版产品。读取 native hook 完整上下文、check.jsonl、PRD/design/implement、前后端具体开发前规范及跨层/复用指南。未分派子代理，未修改规范、部署、README，未提交、归档、安装 root 服务或访问远端，也未占用或停止开发服务器。

## 已修复

- [HTTP 错误分类](../../../../../internal/httpapi/router.go:209)：原来客户端取消进入未知错误分支，返回500并写服务故障日志。现在匹配包装后的 context.Canceled 及已取消 request context，只终止处理，不写故障 JSON/ERROR。新增完整已认证路由和包装错误的[取消测试](../../../../../internal/httpapi/router_test.go:191)。真正未知故障与 DB busy 分类不变。
- [迁移收尾](../../../../../internal/storage/storage.go:166)：原来恢复 foreign_keys 的错误被忽略。现在用独立、有界5秒 context 恢复并读取 PRAGMA 验证为1；任一恢复错误并入迁移错误，Open关闭数据库并拒绝服务，不能带无约束连接接收流量。
- [升级回归](../../../../../internal/storage/storage_test.go:123)：补充真实v1认证会话、密码指纹，断言升级保留登录会话、项目版本/主folder/路径及terminal显示名、项目关联、真实session名称。失败时保持v1版本、旧数据/会话和关联，不残留重建表；成功及失败迁移后均核实外键恢复。原已应用0001未修改。
- [认证异步协调](../../../../../web/src/features/auth/AuthProvider.tsx:5)：旧logout请求在本地过期后可能晚于新login完成，既覆盖React状态，也存在Cookie响应次序风险。过期、卸载或新login先取消旧logout；当前视图的login等待旧logout fetch结算或abort reject之后才发POST，并只允许对应session的有效logout响应清理状态。新增[认证时序测试](../../../../../web/src/features/auth/AuthProvider.test.tsx)：忽略abort的旧logout结算前不能发新login，取消login的迟到响应被丢弃，TTL计时退出不触发资源写入。
- [许可证生成脚本](../../../../../web/scripts/collect-licenses.mjs:4)：URL未显式从node:url导入导致eslint no-undef，已使用Node标准模块导入，无规则豁免。

## 继承修复的核验

上一轮已有0002_resource_ids迁移，真实重建projects/folders/terminals，为资源主键增加NOT NULL，修复SQLite普通TEXT PRIMARY KEY接受NULL并绕过复合FK的问题。已核实NULL身份拒绝、事务回滚、deferred主folder外键、项目删除保留terminal且project_id变null、checksum差异和未知高版本拒绝。新增迁移而非改写已应用0001；metadata清理不触碰真实文件或进程。

根模块go list仅包含cmd/persistty、internal/auth/config/httpapi/storage五个第一方包，web/go.mod是构建边界，没有遍历node_modules内Go样例。前端未引入假PTY或项目写入接口；terminal状态只有unavailable，真实空列表和依赖失败分离。

## 复核覆盖

- 完整Gin注册保护矩阵、独立多设备session、登出/TTL、Cookie flags、Origin/CSRF/XFF、重复和未知JSON字段、正文大小、并发/速率限制、日志秘密扫描及panic脱敏。
- 真实临时SQLite、WAL/SHM权限、token hash、密码变更撤销认证、列表上限、连接重建外键、checksum、高版本、升级失败回滚。
- Go与TS共同fixture、严格unknown解码、非法ID/UTC/主folder/重复ID/safe integer/列表数量；同源fetch、204、401、429、不自动重试POST、路由白名单和项目切换迟到响应。
- 三主题、首帧应用、系统监听清理及独立本地偏好；shadcn官方查询和生成记录见frontend-report，实际业务复用Button/Input/Field/Dialog/Select/Alert/Empty/Skeleton/Badge/Tooltip，没有自造对应基础控件。
- npm精确直接依赖与lock版本/integrity逐项一致；npm ls无缺失包。build读取真实运行依赖LICENSE/NOTICE、生成88项通知并随dist分发，源与dist通知字节相同。shadcn复制组件MIT与react-remove-scroll-bar明确版本补充授权均保留；不把项目MIT覆盖第三方许可，也不声称导入了后续字体/文件图标。

## 最终验证

| 检查 | 结果 |
| --- | --- |
| gofmt（修改的Go源码） | 完成 |
| go test ./... | 通过，五个第一方包 |
| go test -race ./... | 通过，五个第一方包 |
| go vet ./... | 通过 |
| npm --prefix web run lint | 通过 |
| npm --prefix web run typecheck | 通过 |
| npm --prefix web run test | 通过，6个文件、29项 |
| npm --prefix web run build | 通过，包含88项第三方通知 |
| go list ./... | 五个第一方包，无node_modules |
| npm --prefix web ls --depth=0 | 通过 |
| git diff --check | 通过 |

首次完整Go重跑遇到沙箱拒绝访问Go cache，未计通过；随后获准使用工具链/cache执行上述最终test/race/vet，均实际成功。前端初次lint失败已自修后完整重跑。没有跳过或将未运行的环境测试算通过。

## 尚未执行与边界

没有发现当前W01源码范围内尚未修复的阻塞问题。主会话负责最终真实浏览器复验、规范同步和部署文档，证据单独见browser-report；本代理不重复启动服务或将他人反馈当自己的浏览器实测。

AbortController取消客户端等待，不保证服务端没有执行；logout可能已撤销旧session。当前视图取消后等旧fetch结算再登录，防止本视图已知次序覆盖，但不能宣称同浏览器多个标签页的Cookie响应强一致。跨标签认证仍以服务端session及后续401为准，未新增跨标签锁或登录协议。

真实Debian/systemd、Nginx生产分流/CSP、WireGuard公网隔离、Android/iOS真机尚未执行。文件/搜索/Git/传输、真实PTY/TUI持久性、单文件提权helper及完整PA01..PA26属于后续包，不能以W01门禁通过当成这些功能验收通过。

本轮未重新执行Linux交叉编译；backend-report中的早前交叉构建记录不能当成最终修复源码已交叉验证的证据，也不能当成Debian运行验收。

## 主会话追加验证
代理检查结束后，主会话以最终 Go 源码执行 `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build` 成功，产物仅保留在忽略的 .cache；这只是构建验证，不是 Debian 运行验收。最终本地浏览器复验见 [browser-report](browser-report.md)。112 个 Markdown 文件的本地链接（含行号后缀）与 task validate 的 implement 9/check 7 项通过；拟议提交清单见 [commit-plan](commit-plan.md)，未执行 stage/commit/push。
