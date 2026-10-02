# W07 实施清单

状态：已按用户“开始实施”完成 M01～M07 的源码与审查产物；M08 本机门禁与规范同步完成，M09 真实特权验收待精确目标授权。task.py start 已运行，任务保持 in_progress。主会话 inline，无子代理。已勾选项只表示实施完成，不替代 Linux/root/远端专项证据。

## 顺序与门禁

- [x] P01 建立任务、核对首版授权与D10、研究源码owner和官方sudo/NNP机制。
- [x] P02 完成PRD/design/implement；将W05/W06统一验收延后且保留证据。
- [x] P03 展示最终总结并取得后续明确批准；重读trellis-before-dev、PRD/design/implement和对应owner规范，运行task.py start。
- [x] M01 锁定共享fixture与领域types/errors；config严格校验、默认disabled、Unix传输协议和root策略解析。先实现越界/重复字段/限额/调用UID拒绝测试。
- [x] M02 实现Linux短命helper、安全元数据保持与同目录原子提交、持久nonce/target锁、READY/COMMIT/APPLIED及indeterminate。拒绝普通用户调用生产root入口；非Linux不可用；Linux失败注入/特权专项独立标注。
- [x] M03 实现非rootbroker、SO_PEERCRED、固定sudo参数、密码/控制/正文分离FD、限流/deadline/取消与结果查询；自身非root不能假定kill root有效，helper有独立deadline/断连退出。
- [x] M04 新增实际请求元数据迁移/repository CAS；认证撤销与项目发布裁决；prepare/execute/cancel/status服务。验证等待认证不占项目锁、旧关联/旧session不发布、重启不重放。
- [x] M05 接HTTP完整保护矩阵、permission_denied准确映射、执行专用budget；共享fixture供Go/TS decoder，禁止密码和sudo stderr透传。
- [x] M06 前端同FileBuffer一次写入/冻结generation/结果合并与草稿保护；shadcn授权弹窗、默认取消焦点、凭据清理、失效/未知结果可恢复；桌面/手机共用。
- [x] M07 产出root-owned策略/sudoers/broker socket-service/Nginx示例、独立Debian探针及精确安装/回滚清单；本阶段只写可审查产物，不执行root安装。
- [x] M08 trellis-check主会话完整验证并同步backend elevation-contract及frontend owner、索引、README；更新实施/检查报告，区分已实现与真实特权验收。
- [ ] M09 对具体Debian目的地、安装/策略/服务差异和合成目标另请授权，获准后真实专项。缺条件仍可交付代码/构建/本机报告，但不能把W07记为已验收或归档。

## 预计文件与修改理由

| 范围 | 修改理由 |
| --- | --- |
| cmd/persistty-elevatord、cmd/persistty-file-helper、internal/elevation | 独立非root执行器与短命root边界、固定协议/策略/nonce |
| internal/files/save.go、Linux安全写入及新privileged实现/测试 | 复用安全路径/Version/提交内核，补owner/ACL/xattr保持及故障语义；原Save由回归证明不变 |
| internal/storage/migrations/0005_elevation_requests.sql、elevation_requests.go、folder_access.go、storage.go | 持久请求CAS和session/关联发布裁决；不长期占DB事务 |
| internal/config与internal/httpapi/elevation.go、files.go、router.go | 默认禁用配置、注册保护/预算、准确权限错误；普通API不绕过安全 |
| web/src/lib/api/elevation-client.ts、elevation-decoder.ts、client.ts | 共享DTO严格解码和无自动execute重试 |
| web/src/features/workspaces/editor-session.ts、FileEditor.tsx、PrivilegedSaveDialog.tsx及测试 | 同缓冲区接提权快照和草稿/model保护；复用现有基础UI |
| tests/contracts/elevation.json、tests/integration/debian/w07、web/tests/e2e/w07-*.spec.ts、本机fixture | 实际协议、安全边界与浏览器行为证据 |
| deploy/elevation、deploy/systemd、deploy/README.md及Nginx示例 | root策略/允许命令/独立unit/内存正文缓冲/安装回滚可审查 |
| .trellis/spec/backend、frontend及任务报告 | 实施后同步实际owner契约；候选接口不冒充现有API |

最终新增文件名可按实作归属局部调整；任何增加提权读取、目录授权或改变Web安全设置均需重新审查。不改Trellis框架/代理配置，不做W05/W06剩余实机验收，不提交/推送/归档。

## 关键验证

1. 真实普通文件：BOM/混合换行、同大小同mtime变化、根/父/leaf替换、symlink/hardlink/FIFO、metadata变化、原子提交故障、落盘Version；敏感哨兵不能被读取/覆盖。
2. SQLite/CAS：nonce单次消费、并发执行一次、其他session404、过期/取消、重启/错误结果不恢复prepared、配置移除与注销/到期、rename后结果故障indeterminate。
3. 进程：三个独立FD、argv/env无密码正文、短/超长/错误认证输入、EOF/超时、FD关闭、无无界队列、sudo stderr脱敏、roothelper独立超时；合成口令不会证明PAM通过。
4. HTTP：所有新路由401/Origin/CSRF、重复/未知key、UTF-8/版本/体积、限流、状态查询、错误不使系统密码失败登出应用。
5. 前端：默认取消焦点、关闭零execute、双击一次、过期/409/配置切换、保存中新输入/草稿隔离/多组model、未知结果只查询、密码不进入持久状态；Chromium/WebKit分别运行独立输出且串行。
6. Debian：具体授权后visudo有效策略、无TTY正确/错误密码、timestamp不复用、FD3/4、真实UID/GID/ACL/xattr/SELinux、安全链接/替换/崩溃/root取消、Web NNP保留及broker独立、Nginx body不落盘。仅清理本轮精确路径/unit/PID，不动正式服务或tmux。

## 完整代码门禁（实施后实际运行）

```bash
go test ./...
go vet ./...
go test -race ./...
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run test
npm --prefix web run build
go build -o /private/tmp/persistty-w07-build/persistty-elevatord ./cmd/persistty-elevatord
go build -o /private/tmp/persistty-w07-build/persistty-file-helper ./cmd/persistty-file-helper
```

Linux构建使用实际GOOS=linux/GOARCH=amd64/CGO_ENABLED=0并隔离输出；不在计划阶段运行不存在的包。E2E使用现有锁定Playwright与隔离fixture，无真实sudo时明确模拟边界。更新后gofmt、Markdown链接/任务JSON/忽略规则/git diff --check；未运行/跳过不记通过。

## 高风险与回滚点

helper策略、保存元数据和发布锁为高风险，不以UI通过替代安全检查。Web继续非root/NNP，授权broker的安装/命令策略单独审查。元数据或凭据管道契约不满足时停止开放提权能力，不回退到rootWeb/shell/无版本写入。禁用elevation后普通保存/终端保持，已落盘文件不由代码回滚撤销。迁移不可修改checksum或删除已应用表来兼容旧版本。
