# 跨层检查清单

- 真实来源是谁：tmux/filesystem/git/SQLite？缓存和 UI 状态能否被误当事实？读 [状态边界](../frontend/state-management.md)。
- 浏览器、WS、Go context 取消分别清理什么？是否误杀 tmux pane？[Terminal 生命周期](../backend/terminal-lifecycle.md) 与 [组件资源](../frontend/editor-terminal-lifecycle.md)。
- 新 API 的 DTO、时间、ID、空值、错误、body limits 是否双方一致并有 fixture？[HTTP](../backend/http-api.md)、[WS](../backend/websocket-protocol.md)。
- 写入的版本来自哪个 snapshot？外部修改、Overwrite 再冲突、上传完成/replace 如何复验？[文件契约](../backend/filesystem-guidelines.md)。
- 认证是否覆盖 preview/download/upgrade/存量连接？Origin/CSRF、代理信任如何验证？[安全](../backend/security-config.md)。
- 文件名、byte offset、UTF-16 column、binary、ignore、symlink 在哪一步转换或过滤？[传输/搜索/Git](../backend/transfer-search-git.md)。
- SQLite 与 tmux/filesystem 的副作用不能原子提交，崩溃后如何 reconciliation，是否误删真实资源？[数据库](../backend/database-guidelines.md)。
- systemd cgroup 的实测证据是否存在？不能以 mock 或 tmux 后台化推断 restart 通过。
- 失败/取消/重连后，fd/goroutine/socket/model/draft 是否释放或正确保留？
- 哪个 integration/E2E 能观察到这个边界错误？新契约在 owner spec 更新，不能只写聊天。
- 终端字符网格由谁拥有？逐段核对输入 owner、每个输出 attach、renderer 与迟加入端的尺寸；只检查 resize ACK 或浏览器容器宽度不能证明 PTY 同步。异常线条/句点先比对真实 PTY bytes，不能用 CSS/正文过滤掩盖错配；契约和实测归 [终端运行时](../backend/terminal-runtime-contract.md)。
- 连续浏览器手势可能锁定原始 target，隐藏 DOM 不代表不再收到事件。验证小数位移、主轴噪声、加载中惯性与完成后惯性，不能仅用单次大滚轮和状态断言推断流畅/不白屏；边界归 [前端终端运行时](../frontend/terminal-runtime-contract.md)。
