# W06 实施清单

## 阶段与授权

用户于 2026-10-01 要求建立 W06 优先做功能，W05/后续实机验收等明确指令再统一做。PRD/design/research 已形成，用户随后明确回复“开始实施”，已运行 task.py start，状态 in_progress；主会话 inline。不自动提交、推送、归档或操作 Debian。

## 有序批次

- [x] P01 核对首版 P10/P11、PA11/PA12/PA19、现有 owner 与 CLI 实验，形成 PRD/design/本文；记录 W05 验收延期。
- [x] P02 用户批准最新规划总结；激活 W06，trellis-before-dev 完整加载相关 owner，说明最终文件边界。
- [x] M01 安全 snapshot/私有 staging 与 runner、严格配置和上限；本地临时数据证明 traversal/symlink/父目录替换/特殊文件/配置/辅助程序/取消/容量与源资源零写入。
- [x] M02 搜索 rg 能力探测、ignore skeleton/候选集合、安全文本快照、真实身份去重、JSON/UTF-16 DTO；临时 fixture 对照全部模式与 ignore 限制。
- [x] M03 搜索 API、严格 decoder/client/共享 fixture 与 SearchPanel；活动栏/目录右键/快捷键/单内容手机导航，结果到原编辑 buffer 正确定位。
- [x] M04 替换 preview/同引擎捕获组与原字节拼接、session/TTL/资源额度、逐文件 apply/取消/结果查询；复用 files.Save/WithRegisteredFolder，版本/关联冲突与幂等尝试验证。
- [x] M05 EditorScope 目标保护/迟到 generation 与替换结果协调；逐匹配/逐文件选择、桌面只读 diff/手机前后文本、取消零写入、部分结果/脏文件跳过。
- [x] M06 仓库有界发现/身份隔离、metadata/worktree staging、安全配置/对象验证；真实临时仓库 CLI 对照 clean/modified/staged/untracked/deleted/rename/Unicode/根与 merge/多仓库/嵌套仓库及不支持状态。
- [x] M07 Git API/decoder/fixture 与 GitPanel：状态/refs/log/detail/父选择、staged/unstaged/HEAD 与本地引用比较；范围约束、缺对象零联网、原 HEAD/index/正文零修改。
- [x] M08 所属仓库 HEAD baseline/缓存失效、Monaco 行级标记；输入/保存/HEAD 修改/面板切换/主题及 model/undo 保护。
- [x] M09 trellis-check 主会话全范围检查、owner spec/README/AGENTS 同步、必要自动化门禁、功能交付报告与统一验收待办；不将实机待办写为通过。

每个批次先闭合一条真实链路，再继续下一条；不创建空包或假 API 充数。M01 为所有 CLI API 的安全阻塞门禁。搜索/替换先实现，Git 复用安全输入与比较 owner；若 snapshot 无法保持原 CLI 语义，停止受影响 API 并报告具体差异，不用无保护 fallback 或静默降级。

## 规范加载

实施时执行 trellis-before-dev。读取 backend/index、directory-structure/error-handling/quality-guidelines/security-config/http-api/filesystem-guidelines/workspace-files-contract/editor-preview-contract/transfer-search-git/process-guidelines/logging-guidelines；frontend/index、directory-structure/component-guidelines/state-management/type-safety/quality-guidelines/clients/hook-guidelines/editor-terminal-lifecycle/theme-assets；guides/index 与跨层/复用指南。新的搜索/Git 实施契约由 owner spec 持有，规划稿不冒充现有协议。

## 实际验证入口

按锁定 Go/npm 工程运行，修改 Go 先 gofmt；测试可限制前端并发而不放宽断言：

```sh
go test ./...
go vet ./...
go test -race ./...
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run test -- --maxWorkers=2
npm --prefix web run build
```

本地新 Go 集成测试调用实际 Git/rg 和自有临时树；工具缺失明确 skipped，不记通过。Chromium 开发回归按真实新测试文件限定运行，覆盖搜索/替换/Git 与原 workbench/editor-recovery/terminal 几何必要回归。构建写 public 会触发 Vite 重载，build 与浏览器测试顺序执行；冻结最终源码后报告最终结果。

产品自动化至少验证：

1. 搜索多根/重叠/同名/ignore/include/隐藏文件/UTF-16、无匹配/invalid pattern/tool failure、截断和取消；range 与版本统一。
2. 预览/apply 同引擎与捕获组、BOM/混合换行/EOF/零宽匹配、选择后准确字节；409/配置变化/路径替换、脏/保存中/处理中新输入不丢，部分失败/取消/重试查询零重复写。
3. Git 各状态及普通多仓库/最内层仓库/HEAD→buffer、refs 固定对象、根与 merge 父/二进制/缺对象；原 HEAD/index/config/文件 hash 不变，外部 helper/配置 include/alternates/根外元数据/联网入口拒绝。
4. 后端 auth/Origin/CSRF/session 与 opaque ID 绑定、正文/原始 stderr 不进日志，所有 payload 都严格解码、数量/大小有界。
5. 比较资源释放、原 model/undo/草稿/终端 runtime 保留、设备状态与 schema 兼容、主题及键盘入口。

另检查 Markdown 本地链接、JSON、忽略规则、规范一致性和 git diff --check。规划阶段仅运行文档检查；实施阶段已运行产品门禁，实际范围见交付报告。

## 统一验收队列

- W05 真实手机软键盘/输入法/旋转/前后台、触控板、完整图片/字体平台矩阵：延期，用户明确要求时恢复。
- W06 真实 Debian 工具/安全边界/CLI 与浏览器集成、实机手机交互与性能：功能完成后列为待统一验收，不自动连接目标机。
- W08 真实部署、安全入口、多设备完整首版矩阵仍属后续工作包。

自动化正确性门禁不等于上述实机验收；统一延期不代表删除需求或任务完成。

## 高风险与回滚

安全根/元数据读取、Git filter/对象指针、replace 发布裁决、EditorScope 保存/generation、恢复 schema 为高风险。每批保持可回滚，禁用新入口只清自身 transient cache/staging，不修改用户文件/仓库状态/项目/终端。无数据库迁移，跨设备草稿仍不引入。

## task.py start 前

PRD 已按目标/背景/需求/边界/延期/产物收敛，无阻塞用户产品选择；design/implement 完整，研究限制明确。inline 不创建历史代理 JSONL，不启动子代理；task.py validate 的 skipped 不视为产品验证。最新总结后用户批准才激活并开始产品代码。

## 本轮规划检查（2026-10-01）

主会话核对 7 份相关 Markdown 的 40 个本地链接，全部存在；任务 JSON 可解析，W05=in_progress/acceptance deferred，W06=planning/acceptance deferred；单代理 inline 配置保留。task.py validate 的 JSONL 两项按 inline 跳过，当前指针指向 W06 规划任务。git diff --check 通过，新建文件另检查尾随空白/围栏/占位符。仅修改规划、优先级与项目状态文档，未运行 Go/前端产品门禁或实机验收；临时 CLI 可行性试验的限制见 research.md。

## 功能交付（2026-10-01）

M01–M09 的功能与开发检查已完成，详见[交付报告](implementation-report.md)及新 owner；上述详细风险矩阵仍需后续统一实机验收。任务保持 in_progress/acceptance deferred；规划检查中的 planning/无产品代码仅是此前阶段记录。单代理、未提交/推送/归档，不连接 Debian。
