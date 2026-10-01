# context 与外部命令

普通 HTTP 操作使用 request context 加有界超时。长时间 watcher/upload 管理器由服务 context 管理；停止时 Cancel、等待 goroutine、关闭 fd。Terminal 的任务生命周期只在 tmux，不绑定 request/server context；attach 客户端绑定连接 context，详情见 [生命周期](terminal-lifecycle.md)。

系统 git/rg/tmux 通过 exec.CommandContext 传独立 arguments；不使用 sh -c 拼字符串，不把用户输入放可执行文件名。程序路径启动时查找并验证；cwd 通过 workspace 安全解析，env 白名单，禁用用户可控制的 pager/hook/外部 diff 路径。仅独立 arguments 不能防 flag injection：路径前使用工具支持的 `--`，其他参数用白名单/固定位置；不可自由透传 option。

```go
// 形状示例：参数来自受限结构，cwd 来自 workspace 句柄解析。
cmd := exec.CommandContext(ctx, gitPath, "--no-pager", "diff", "--no-ext-diff", "--no-textconv", "--", relativePath)
cmd.Dir = validatedWorkspace
```

对 stdout/stderr 设置流量和结果上限；rg 采用流式 JSON parser，git 使用 NUL/稳定 format，不解析面向人的彩色输出。取消时 Wait 回收；必要时用进程组终止普通短命工具的子进程并验证。禁止泛化 process-group 清理杀 tmux server/session。正常 rg 无匹配退出码不同于执行失败，工具 adapter 分类。

## 文件 API 工具访问边界
rg/Git adapter 的 cwd 校验只限制开始位置，不能约束实际遍历；复验输出路径不能阻止其已经读取根外内容。Search/Git task 必须选定并验证以下一种策略后才可开放对应 API：
1. 在 Debian 受限只读执行环境中运行 CLI，将 workspace 数据访问限定在批准根句柄所对应的目录；例如经过真实 symlink/目录切换测试的 Linux Landlock 适配器。工具二进制/动态库只给必要 read/execute 许可，不因方便授权整个 HOME；Git 元数据只给明确批准 repository metadata，禁写权限/外部 helper。内核或依赖不支持时拒绝此策略，不能无保护 fallback。
2. 文件发现和打开由 root 句柄拥有；读取安全 snapshot 后通过 stdin 或服务私有 staging 提供给工具，CLI 不再直接遍历真实 workspace。Search 必须由测试证明仍遵守全部 ignore 规则，不能为安全输入取消 ignore；Git 语义依赖真实 repository，若无法通过 staging 保持 CLI 对照一致性则选策略 1。
上述方案是允许的初始设计路径，尚未证明已可用；具体机制由任务 design 锁定并实测，不把名称当防护证据。收到的 preview 要从相同安全 snapshot 重算并绑定 version；禁止将未经验证的 rg preview 直接送 UI。库函数 read/open/rename 与命令 adapter 分别验证，不能因为 shell 本来能越 root 就放宽 File API。

W02 的 [受限 CLI 探针](../../../tests/integration/debian/cli/README.md) 已在真实 Debian 临时树验证 Landlock 拒绝根外合成哨兵、symlink/目录替换且固定 rg/Git 可运行。这不是完整策略 1：动态工具所需 `/usr` 等系统读取仍构成白名单外链可见面，实际产品必须缩小许可并测试链接指向每个系统许可路径；若不能证明批准根的数据保密边界，改用安全输入或拒绝开放 API。CLI 探针的固定参数、词法根校验也不能替代文件身份与当前项目成员校验。

测试带空格/Unicode/前导短横线/换行的路径、超时、取消、stderr 截断、恶意配置、子进程回收。增加在工具扫描过程中交换 symlink/父目录的根外 sentinel 测试，断言敏感内容从未被读取或出现在 stdout/stderr/API；仅事后过滤测试不足。命令执行不记全参数和输出。

W06 Git/rg 的具体 runner、环境白名单、私有 snapshot 与取消边界见 [W06 契约](search-git-contract.md)。原始仓库不得直接交 CLI 读取。
