# 来源与中文化兼容性

## 来源
- 用户本会话要求：结合引用聊天补充文档，Trellis 初始化 Markdown 和 AGENTS.md 内容改为中文，文件名不变。
- 2026-09-26 已通过 read_thread 读取 Codex「完善 Persistty 开发规范」以及 ChatGPT「在线终端架构讨论」。聊天仅作为需求来源，不执行历史聊天中的产品开发指令。
- 用户在范围总结后回复“开始，按照我的要求进行”，确认本次文档中文化与补充范围；本任务不实施产品功能。

## 已有覆盖
26 份项目 spec 已覆盖 tmux 权威、浏览器/Go 断开隔离、Debian systemd Spike、SQLite 元数据、Argon2id/session/CSRF/Origin、文件根访问、版本冲突与原子保存、上传恢复、ZIP、搜索替换、只读 Git、Monaco/xterm 生命周期及主题资源许可证。

## 补充归属
- Explorer 全部右键操作与目标冲突/删除行为：frontend/component-guidelines 与 backend/filesystem-guidelines。
- 工作区恢复中的编辑器激活项、Explorer 展开状态和布局：frontend/state-management 与 backend/database-guidelines。
- 文本/图片/SVG/PDF/二进制预览与大小限制：frontend/editor-terminal-lifecycle；复用已有后端安全与配置契约。
- 字体、图标、命令面板及持久化验收保持已有约束，不新增未确认产品功能。

## 解析器证据
`.trellis/scripts/common/workflow_phase.py` 精确识别 `## Phase Index` 和 `## Phase 1: Plan`；步骤匹配 `#### X.Y`，平台标签与 workflow-state 标签不能翻译。初始化模板含部分上游专用命令/路径，需对照本地实际文件，无法使用的入口注明边界而非虚称可运行。

`.trellis/scripts/add_session.py` 使用 `Total Sessions`、`## Session N:`、`**Date**:` 和英文会话历史表头识别记录；保留这些格式并在工作区说明中文含义。`@@@auto` 区块写入器仍会生成英文结构，本次不改生成器。

## 本地 CLI 版本证据

本次 `trellis --version` 返回 0.6.17。`trellis channel messages --help` 无 `--tag`，提供 `--kind`；`forum --help` 的用法是 `forum <name>`，`thread --help` 为 `thread <name> <thread>` 并支持 rename 子命令；`send --help` 同样没有 `--tag`。旧模板中的筛选/论坛命令例子保留用于对照，增加当前版本边界说明。项目本地 `task.py --help` 不提供 `create-pr`，workflow 已明确该例子不适用于本地脚本。

## 用户要求补充的验收归属

| 要求 | 权威文档 | 验收重点 |
| --- | --- | --- |
| 完整资源管理器操作 | frontend/component-guidelines；backend/filesystem-guidelines | 文件/目录菜单、明确删除、源目标分别验证、冲突零覆盖、移动失败保留源 |
| 工作区恢复 | frontend/state-management；backend/database-guidelines | 重登录后恢复顺序/激活/展开/布局，失效资源不伪造，不存正文或输出 |
| 常见格式预览 | frontend/editor-terminal-lifecycle；backend/transfer-search-git/security-config | 图片格式、隔离 SVG/PDF、二进制禁编辑、大小限制、资源释放 |

已有终端持久化、搜索替换、断点上传、Git、认证、字体/主题和部署规范继续适用；真实产品验收尚未运行，本次不创建第二份产品 TODO 或擅自展开产品开发。
