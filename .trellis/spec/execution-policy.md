# 单代理执行契约

## 1. 范围与决策
用户明确要求仅在 Persistty 项目禁用 subagent，原因是过度分派导致流程和资源开销。适用于本项目研究、实现、审查、测试、规范维护及 Trellis 工作流；全部由当前主会话执行。不改变产品需求或降低质量门禁，不影响其他项目或全局 Codex 设置。

## 2. 配置与入口
项目 `.trellis/config.yaml`：
```yaml
codex:
  dispatch_mode: inline
```
项目根 AGENTS.md 保留禁用指令。通过 `get_context.py --mode phase --step 2.1 --platform codex` 验证内联分支。不修改用户全局 AGENTS.md/config.toml、已安装 Trellis 包或删除历史代理文件；不在项目配置中写不受支持的全局 feature 开关。这是项目执行约束与 Trellis 分派配置，不宣称宿主工具已从全局移除。

## 3. 执行约束
禁止 spawn_agent/spawn_subagent/trellis_subagent 及等价分派；不得以 channel worker、独立 AI 进程或其他会话绕过。主会话执行 `trellis-before-dev`、编辑、`trellis-check` 技能、实际测试、规范更新和提交流程。容量不足时缩小批次或报告限制，不自动分派。只有用户明确解除禁用才能重新配置，普通继续/检查/调研不算解除。

## 4. 行为矩阵
| 情况 | 行为 |
| --- | --- |
| 研究/实现/检查请求 | 主会话读取上下文并直接执行 |
| 旧计划、JSONL 或报告提到子代理 | 保留历史资料，不视为当前分派授权 |
| 自动模板提示分派 | 检查 inline 配置与注入；不通过另开 worker 绕过 |
| 配置更新/工具重启 | 核对项目 inline 配置与项目约束未被覆盖 |
| 用户明确要求解除 | 按新的明确授权更新配置；否则保持禁用 |

## 5. 用例
正常：主会话实施后重新阅读 diff，自行运行测试并按检查技能逐项评审。基础：仅文档变更做链接、配置和 diff 检查。错误：为“独立评审”启动 check Agent，或替换为后台 channel worker。

## 6. 验证
检查 Trellis CLI/parser 识别 inline、每轮提示采用 in_progress-inline、内联阶段不要求启动子代理、所有项目配置仍可解析、本地链接与 git diff --check 通过。全局 feature 状态不作为本项目门禁。既有进程不因写配置自动结束；主会话自检不称为独立代理复核。

## 7. 正反例
错误：任务复杂 → 自动 spawn implement/check，或让其他 AI 进程处理。正确：读取 PRD/设计/规范 → 主会话按小批次实现 → 运行真实门禁 → 自行检查并如实报告未运行项。
