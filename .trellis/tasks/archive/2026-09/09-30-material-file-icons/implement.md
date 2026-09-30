# Material 图标实施清单

## 依赖与开始前

先完成语言子任务的轻量路径 metadata/adapter。用户批准最终父任务规划后，按 inline 激活本任务，使用 trellis-before-dev 读取 PRD/design/本文、研究与 frontend 目录/组件/主题/状态/类型/质量及复用指南。主会话实施/检查，禁用 subagent。

## 实施顺序

1. 以 npm 添加精确 devDependency material-icon-theme 5.38.1，核对锁文件 integrity 与实际 MIT LICENSE，不混用包管理器。
2. 新资源生成脚本运行默认 generateManifest，校验/复制完整映射引用 SVG，生成 typed manifest 和 asset checksum 清单。修改 predev/pretest/prebuild 等统一准备入口，使干净 checkout 可重现；生成目录按规则忽略。
3. 新纯 resolver：小写 basename、隐藏文件、最长后缀、语言与目录/root 开闭、light 覆盖；批量验证所有映射引用存在。保留 patterns/clones 已展开结果。
4. 新装饰图标组件及共享 effective theme 输入，替换 Explorer 根/条目、ProjectWorkbench 文件标签、DirectoryPicker 真实条目，保持中文 aria-label、chevron和现有 shadcn 基础控件。
5. 扩展 collect-licenses.mjs，保证构建输出包含资产 MIT 原文与来源/checksum清单，不只扫描运行依赖。
6. 新增解析案例与全量资源完整性检查；组件测试树/标签一致、目录开闭及主题/加载失败行为；真实浏览器检验清晰度与 cold-cache 本地请求。

## 验证与审查

- web/：npm run lint、npm run typecheck、npm run test、npm run build；Playwright 在显式隔离实例执行 AC-I01..05，与语言任务联合验证实际 token 和图标。
- 浏览器浅色/深色/system、桌面/手机视口、长名字/多标签/左右分组；document 宽高几何断言和截图，模拟手机视口不称为真机验收。
- asset 生成重跑输出一致，全部引用存在，打包路径与 local-only 请求正确；source metadata 与 per-asset hash完整，分发 LICENSE 有版权原文。
- 完成 trellis-check 自审，go test ./...、go vet ./... 按父任务综合门禁记录，可复用同一未变代码状态的合格运行结果；无终端/并发改动不补 race。
- Markdown 本地链接、忽略规则和 git diff --check；将 resolver/版本/许可维护规则更新 frontend theme-assets owner。
- 交父任务集成验收。未经用户要求不提交或归档。

## 回滚点

先生成资源并验证，再接入消费者；消费者回滚保留通用 lucide 图标。最终回滚仅涉及本子任务的生成资源/adapter/组件/依赖/锁文件/ignore/许可配置，不改文件、项目或终端数据。
