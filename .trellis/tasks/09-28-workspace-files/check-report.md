# W04 验收检查记录

日期：2026-09-29。状态：实现与质量检查完成，待 Trellis 3.4 提交确认及归档。测试只使用本地/远端私有临时夹具，未对用户真实项目执行删除。

| PRD | 证据 | 结论 |
| --- | --- | --- |
| A01 项目/目录选择 | `internal/storage/projects_write_test.go`、`internal/httpapi/projects_test.go`、`DirectoryPicker.test.tsx`；项目 CRUD、配置版本、主根、书签 404 与仅删除元数据 | 通过 |
| A02 根与关联身份 | `internal/files/files_test.go`、`tests/integration/debian/w04/w04_linux_test.go`；根 device/inode、旧配置版本、逃逸链接与替换后拒绝 | 通过；外部 writer 在最终复验与 rename 之间的窗口仍按 PRD 明示，不宣称绝对 CAS |
| A03 多根树与外部变化 | `internal/files/files_test.go`、`internal/httpapi/events_test.go`、`Explorer.test.tsx`；游标/失效重列、WS rescan 与 polling | 通过；浏览器只读视图不持有 W05 编辑缓冲区 |
| A04 普通文件操作 | `internal/files/files_test.go`、`internal/httpapi/projects_test.go`、`ExplorerActions.test.tsx`；同根/跨根、no-replace、部分失败、删除预览与取消零副作用 | 通过 |
| A05 内容与保存内核 | `internal/files/files_test.go`；BOM、混合换行、二进制、原字节 hash、同 mtime/size 改变、权限位、取消、源叶子身份复验 | 通过；W05 编辑 UI 未纳入 |
| A06 上传 | `internal/transfer/uploads_test.go`、`UploadControls.test.tsx`、Debian W04 测试；错误/重复块、重启续传、相同跳过、首次冲突与确认后再变更、目录项/空目录 | 通过；文件选择器无法枚举的空目录须使用桌面目录拖拽 |
| A07 下载/ZIP | `internal/files/export_test.go`、`internal/transfer/archives_test.go`、`internal/httpapi/transfers_test.go`、Debian W04 测试；原字节、可解压字节/结构、空目录、特殊文件失败、完成后取 ZIP | 通过 |
| A08 质量/跨层 | 共享 `tests/contracts/workspace-files.json` 同时由 Go/TS 测试解码；Go/前端全门禁、真实 Debian、桌面/手机浏览器检查 | 通过；下列细项为实际执行记录 |

## 本轮执行

- `GOCACHE=/tmp/persistty-go-cache go test ./...`：通过（HTTP 测试需要本机回环监听权限；沙箱内首次尝试被拒，授权后通过）。
- `GOCACHE=/tmp/persistty-go-cache go test -race ./...`：通过。
- `GOCACHE=/tmp/persistty-go-cache go vet ./...`：通过。
- `npm run lint`、`npm run typecheck`、`npm run test`（11 文件、43 测试）、`npm run build`：通过。Monaco 独立懒加载 chunk；构建有大 chunk 提示，不是失败。
- Linux amd64 W04 测试交叉编译并通过 `tests/integration/debian/run_remote.py files --probe /tmp/persistty-files-probe --test /tmp/persistty-w04.test` 在已授权 Debian 私有临时目录执行：`tests_passed=true`、`cleanup_verified=true`；连接参数与私有 `.env` 未写入证据。
- `git diff --check`：通过。业务源码和直接依赖扫描未发现 `radix-ui`/`@radix-ui`、调试日志或 lint/type 抑制。

## 浏览器与范围说明

前一轮已在本地授权测试项目中检查桌面 1440x900 的工作台/Monaco 与手机 390x844 的单视图/普通文本视图；本轮复查时测试浏览器会话已过期，页面正确回到登录入口，没有擅自读取或输入真实密码。W04 的浏览器交互还由项目、目录选择、树、上传冲突的 Testing Library 测试覆盖。W03 的真实 xterm/tmux 控制、W05 的编辑/自动保存/草稿、W06 Git/搜索、W07 提权和 W08 正式部署均不在本次通过声明内。
