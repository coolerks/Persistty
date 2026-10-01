# W05 检查与验收报告

日期：2026-10-01。执行方式：主会话单代理 inline；未提交、推送、归档或改动用户现有服务。

## 已实施

- 桌面 Monaco 与手机基础编辑共用 FileBuffer；原始 UTF-8/BOM/混合换行保留，默认 1 秒自动保存，完整强版本与 generation 保护。
- IndexedDB 草稿按项目/真实身份/独立 view 修订保护；409 暂停，桌面只读比较后明确保存，恢复零隐式 PUT；手机基线变化只保留/导出。存储失败阻止最后标签关闭、显式链接离开和登出，并保留内存输入。
- watcher/轮询/focus/online 复验；文件移动与配置移除先保护，剩余覆盖根复验完整版本后重绑；失效项目草稿导出。
- inspect/preview 复用安全根和鉴权，按内容分类；8 MiB 文本、16 MiB 图片、单边 8192/总 1600 万像素；SVG 白名单与隔离响应。未知/PDF/超限只下载。
- 上下最多四个水平组，文件/终端标签移动排序、设备独立视图、尺寸/折叠/展开恢复；复用既有 terminal runtime。
- 自托管 Nerd Fonts v3.5.1 JetBrainsMonoNL NFM Regular 原始字体、hash/name/glyph/等宽验证，字体加载后 Monaco/xterm 尺寸复测，原始 OFL 随生产构建分发。

## 质量门禁

| 检查 | 实际结果 |
| --- | --- |
| gofmt（修改 Go 文件） | 已执行 |
| `go test ./...` | 通过；使用隔离 GOCACHE；HTTP 测试在允许本机端口的执行环境运行 |
| `go vet ./...` | 通过 |
| `go test -race ./...` | 通过 |
| `npm run lint` / `typecheck` | 通过 |
| `npm run test` | 21 个文件，128 项通过 |
| `npm run build` | 通过；37 项第三方许可、字体校验通过 |
| 最终 Chromium 专项 | 9 项通过（50.4 秒）：editor-assets 2、editor-recovery 5、terminal-geometry 1、workbench-interactions 1 |
| Markdown 本地链接 / `git diff --check` | 通过；报告更新后再次复核 |
| 忽略与分发检查 | node_modules/dist/test-results/生成 Material 资源受忽略；生产字体 SHA-256 与原许可核对通过 |

jsdom 给出未安装 canvas 的提示，生产构建给出大 chunk 提示，均无失败；不将 jsdom 当真实字体/Monaco 着色证据。JSONL 在 inline 工作流中按规范跳过，产物/spec 已由主会话直接读取；task.py validate 的 skipped 不等于新增产品验收。

## 真实 Debian 专项

使用现有 browser-start 私有 harness、独立 Linux runtime-probe 与临时目录/服务，通过已批准连接建立本机测试隧道。`w05-live.spec.ts` 显式测试专属 W03 隔离项目：真实键盘编辑并原子保存 BOM/混合换行；外部改写冲突、只读 diff 后明确保存；三组刷新；伪装文本后缀 PNG、安全 SVG 原生图片渲染；匿名 401、主动 SVG 415。1 项通过（9.4 秒），不是 fixture HTTP。

去敏启动记录 `/private/tmp/persistty-w05-debian-start.json`；清理记录 `/private/tmp/persistty-w05-debian-cleanup.json`：units_inactive=true、pane_processes_gone=true、root_removed=true。测试隧道及专属 Vite 已停止。记录不含连接凭据，不读取或修改用户真实项目。此轮不声称重新执行完整 W03 systemd 持久性矩阵。

Debian 专项之后的迟到读取/关闭缓存修正由最新单测与 Chromium 回归覆盖；AVIF 多尺寸属性防绕过由最终 Go 单测覆盖，未在 Debian 浏览器重新跑 AVIF 格式矩阵。

## 缺陷与规范同步

1. 保存中输入按提交 generation 推进 base，迟到 refresh 必须复验发起时 base/epoch，避免倒退刚保存版本。
2. DiffEditor 清理显式取消 viewModel，再释放 editor/models，避免异步 worker “no diff result available”。
3. xterm 字体 ready 后更换正确 fontFamily，触发字符服务重新测量，不能只 fit 旧字符缓存；controller/observer 门禁保留。
4. 最后干净标签关闭清除 buffer 缓存，重开重新读取；未保护脏输入仍保留并暂停。
5. 恢复校验 schema/路径/数量/尺寸；旧双组迁移与桌面/手机隔离，不从本地标签推断真实资源存在。
6. 构建许可证写入 public 会触发 Vite 页面重载，浏览器回归与 build 必须顺序运行；最终以冻结源文件、构建结束后的结果为准。

owner 更新：backend/editor-preview-contract、workspace-files-contract、HTTP/index；frontend 状态、生命周期、主题字体/index；README 与 AGENTS 当前状态同步。

## 尚未验收

- 真实手机软键盘、输入法、旋转及浏览器前后台恢复；视口模拟仅覆盖基础文本、设备记录与草稿限制。
- JPEG/GIF/WebP/AVIF 完整文件解码及多浏览器组合；Go 包含内容/容器/尺寸边界检查，真实 Debian 图片渲染仅 PNG/SVG。
- 真实 Oh My Zsh/Powerline 输出、中英文及实机字体矩阵；字体表和 Chromium 几何验证不代表全部 shell 主题运行时通过。
- W08 正式部署与发布矩阵、W06 搜索/Git、W07 提权均不属于本轮实现。

任务保持进行中。可从这些实际验收边界继续，不重做已通过的检查，不将待验收项勾为完成。
