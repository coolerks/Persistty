# W01 前端实施报告

## 边界与交付
仅新增 `web/**` 与本报告，没有修改 Go、共同 fixture、部署或规范，没有启动第二个服务器、提交或分派代理。读取本任务 PRD/design/implement、全部 implement.jsonl 入口、父需求/设计和前端开发前检查清单的具体规范，并读取共享与跨层指南；遵守新版项目/主题/关闭契约而非旧 bootstrap 示例。

- 真正同源登录、退出及 session 探测；Cookie 由浏览器管理，CSRF 仅内存，密码不进入 URL、日志或本地存储。登录 POST 无自动重试，429 尊重 Retry-After，失败可重新输入。
- React Router 提供 `/projects`、`/projects/:projectId`、`/terminals`、`/terminals/:terminalId`，登录返回地址白名单；直接访问无重复打开提示，项目列表打开可选当前/新标签页且使用 noopener。
- 项目、主文件夹及终端列表均读取真实 API，空态、网络失败与不存在分开；失效项目保留项目面板和终端入口。没有演示数据，没有项目创建、PTY、文件、Git、搜索或提权伪接口与死按钮。
- 三主题在同一语义 token 源切换；system 监听并清理系统变化，显式模式不被覆盖，偏好版本化本地保存，失败可见。入口在 React 渲染前应用主题，HTML 无内联脚本，适合 script-src self。
- 所有外部 JSON 视为 unknown，统一严格 decoder 检查 envelope、已实现 DTO、主 folder、safe integer、UTC、null、列表数量及唯一 ID；同一个 `tests/contracts/foundation.json` 被 Go 与 TS 使用。
- 有取消和 stale response 保护的局部请求 hook，无全应用万能 store；布局仅建立水平组、设备独立 key 与四种类型化命令的模型及测试，未宣称真实拆分/终端倒计时已实现。

## 工具链与官方组件
实际运行 Node `24.19.0`、npm `11.17.0`，`packageManager` 固定 npm，Node engine 为 24 系列，直接依赖精确版本与唯一 package-lock。React/React DOM `19.3.0`、React Router `8.4.0`、Vite `8.3.1`、TypeScript `6.0.3`、Tailwind `4.3.3`、Vitest `5.0.2`、lucide-react `1.48.0`、radix-ui `1.6.7`。Node 类型与 Node 24 对齐。初次安装经过机器已有 npmmirror registry，保留 lock 实际来源，不修改全局配置；随后对 React、Vite、React Router 显式使用 registry.npmjs.org 查询 version/dist.integrity，与 lock 逐项一致。CLI/docs 使用官方来源，不能把镜像安装称为全部官方直连。

复用查询顺序：确认初始不存在 web/src/components/ui，加载本地 shadcn skill，然后执行官方 CLI `shadcn@4.21.0 search @shadcn -q button`、`docs button input field dialog select alert empty skeleton badge tooltip`、`add ... --yes` 和 `info --json`。查阅 [Vite 官方入口](https://vite.dev/guide/)、[React Router 声明式安装](https://reactrouter.com/start/declarative/installation)、[shadcn Vite 安装](https://ui.shadcn.com/docs/installation/vite)及 CLI 返回的 `https://ui.shadcn.com/docs/components/radix/<component>` 文档。info 确认 `new-york`/`radix`/Tailwind v4/lucide，生成 12 个 UI 文件并逐个读取复核。

实际业务使用 Button、Input、Field/FieldGroup/FieldLabel/FieldError、Dialog、Select/SelectGroup、Alert、Empty、Skeleton、Badge、Tooltip；Field 内部复用 Label/Separator。未手写这些基础控件。生成器产生的 `cn` 包 import 改为配置指定本地 utils，移除不再使用的 cn 依赖；Dialog 的 Close 文案中文化，移除负字距，其他保留官方组成。仅应用布局、业务列表和认证协调为定制范围。

真实许可证来源（主会话统筹分发 notice）：`web/node_modules/lucide-react/LICENSE` 为 ISC，同时包含 Feather 派生图标的 MIT/Cole Bemis 授权；`react/LICENSE`、`react-dom/LICENSE`、`react-router/LICENSE.md`、`radix-ui/LICENSE`、`clsx/license`、`tailwind-merge/LICENSE.md` 为 MIT；`class-variance-authority/LICENSE` 为 Apache-2.0。shadcn/ui 源组件来源为官方 registry，项目上游 MIT 来源 `https://github.com/shadcn-ui/ui/blob/main/LICENSE.md`。本包不导入 Nerd Font 或 Material 文件图标，不能把这些后续资产当已验收。

## 验证结果
新增 `web/scripts/collect-licenses.mjs` 在每次 build 前结构化遍历 `npm ls --omit=dev --all --long --json`，按安装路径去重，读取实际 LICENSE/NOTICE，生成 public/third-party-licenses.txt 并自动进入 dist。未知缺许可证依赖使构建失败。react-remove-scroll-bar 2.3.8 发布物声明 MIT 但未打包 LICENSE；精确 npm gitHead 的许可证路径404，补充来自上游作者明确新增授权的2025-05-26提交，不伪称精确版本文件。来源和真实文本保存在 `web/licenses/`；复制的 shadcn 组件 MIT 同样随包保留。

- `npm run lint` 通过。
- `npm run typecheck` 通过（strict/noUncheckedIndexedAccess/exactOptionalPropertyTypes，无 any 或网络强转）。
- `npm run test` 通过：5 个测试文件、26 项，包含共同 fixture/非法 DTO/认证和204/429、匿名书签登录恢复与404、失败非空态、项目打开弹窗取消和位置、旧请求 abort 与迟到响应、未知终端、主题和监听清理、类型化关闭/不可下移/设备 key。
- `npm run build` 通过，生产 JS 约425kB、gzip134kB；`npm ls --depth=0` 无缺失依赖，`git diff --check` 通过。
- 主会话反馈真实服务浏览器登录/退出后的401、项目404、桌面1280x577与手机390x844无横向溢出、三主题与重新加载成功，浏览器错误为空。本代理没有重复启动或替代主会话截图验证。

## 未交付与风险
这不是完整首版验收。SQLite 中目前真实无项目/终端时正确显示空列表；项目配置、真实文件/PTY、Monaco/xterm、草稿、左右拆分、应用级倒计时、上传/search/Git/helper仍按后续包实施。真实 Android/iOS、Debian/systemd、Nginx生产 CSP仍由相应验收负责；本地 viewport 不能替代真机。新资产与依赖更新须继续核实授权，当前补充文件不覆盖其他版本。组件源可用性不等于鉴权安全，后端独立执行权限检查。
