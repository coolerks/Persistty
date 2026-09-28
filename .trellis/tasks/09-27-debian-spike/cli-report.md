# D09 受限 CLI 实测

2026-09-28 在已授权 Debian 13.4 用户隔离环境运行交叉构建 Linux/amd64 探针。只使用本次随机 0700 `mktemp` 目录与固定合成数据；私有 `.env` 仅经既有受限 parser 读取，输出是脱敏布尔判定与退出码，不保存原始 CLI stdout/stderr。远端目录删除已验证。

九项真实判定全部通过：根内文件可读、Unicode/前导短横线路径可读、根外直接目标拒绝、根内 symlink 指向根外拒绝、子目录替换为根外 symlink 后拒绝、固定 `rg --follow` 内部标记命中而外部标记未出现、只读 Git log 成功、未知命令失败关闭、阻塞探针取消后回收。`rg` 因故意拒绝的链接退出 2，此退出码不当作搜索本身成功；内部标记与外部哨兵分别核对。`git` 和 `rg` 在目标机均存在。Landlock 规则初始化失败时探针固定非零退出，不允许无隔离 fallback。根 Go test/race/vet、Linux/amd64 build/vet 和远端执行通过。

**边界：** 直接越界目标由词法检查拒绝，symlink/目录替换由实际 Landlock 规则拒绝。动态工具要求 `/usr`、库目录等系统读取，`/dev/null` 允许写入，故本探针没有证明“除项目根外无文件可读”；已打开 fd、特殊文件、系统白名单链接、恶意 Git 配置/外部 helper、全竞态、长期资源上限仍需 W04 设计和实测。不能把本实验二进制作为产品沙箱。

依据：[Linux Landlock 文档](https://docs.kernel.org/6.12/userspace-api/landlock.html)、[no_new_privs 文档](https://docs.kernel.org/userspace-api/no_new_privs.html)。
