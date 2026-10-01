# W05 保存入口与有界预览契约

## 1. 范围与触发条件

修改 `internal/files/preview.go`、`internal/httpapi/{files,preview,router}.go` 或前端内容分类/保存 decoder 时适用。W04 的安全根、版本和原子保存继续由 [文件契约](workspace-files-contract.md) 持有；W05 不新增 SQL、任意执行接口或文件权限绕过。

## 2. 签名

路径相对于 `/api/v1/projects/:id/folders/:folderId`：

- `PUT /content`，正文 `{project_version,path,expected_version:{identity,mtime,size,etag},content}` → `{version:{identity,mtime,size,etag}}`。
- `GET /inspect?project_version=<int>&path=<relative>` → `{kind:"text"|"image"|"binary",mime,size,width,height,editable,previewable}`。
- `GET /preview?project_version=<int>&path=<relative>` → 图片原字节，非 JSON。

Go owner：`files.Inspect(root, relative)`、`files.ReadPreview(root, relative)`；TS owner：`api.saveContent`、`api.inspect`、`decodeSaveResult`、`decodeInspection`。共享 [fixture](../../../tests/contracts/workspace-files.json) 包含 saved 与三种 inspection。

## 3. 契约

所有端点使用会话鉴权，写入另要求 Origin/CSRF。读取同样绑定配置版本、注册根身份及安全相对路径，不信任扩展名。文本为有效 UTF-8 且不含 NUL，编辑上限 8 MiB；保存以原始字节计算强版本，保留 BOM/换行。HTTP JSON body 上限 `6*(8<<20)+8192` 覆盖合法 JSON 最坏转义；不提高正文上限。

inspect 先 stat，超过 16 MiB 不读取正文；否则有界读取、复验大小/mtime。PNG/JPEG/GIF 读标准 header；WebP 读声明的 RIFF chunk；AVIF 只递归声明的 BMFF meta/iprp/ipco/ispe（深度 8），检查全部尺寸属性，不搜索压缩字节。预览每边 ≤8192，总像素 ≤16,000,000，字节 ≤16 MiB；浏览器另复验 natural dimensions。未知、损坏、超限只能下载。SVG 源码仍可编辑；preview 采用有界 XML 元素/属性白名单，拒绝脚本、事件、foreignObject、外链、use、style、DOCTYPE、非 XML processing instruction、url()，深度 64、token 10,000。仅原生 `<img>`，不插入宿主 DOM/iframe。

preview 返回 MIME、`X-Content-Type-Options: nosniff`、`Cache-Control: no-store`、`Content-Disposition: inline`、`Content-Security-Policy: default-src 'none'; sandbox; frame-ancestors 'none'`。没有新配置/环境变量；8/16 MiB 为当前实现上限，不能引用 bootstrap 的 10/50 MiB 或假设已有 files 配置。

## 4. 验证与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 未认证 | 401；图片 URL 同样拒绝 |
| 缺版本 / 配置、根或文件版本变化 | 428 / 409；不自动重试保存 |
| 非图片、unsafe SVG、像素/字节超限 | inspect 给出 previewable=false；preview 415 |
| 相对路径/符号链接/特殊文件越界 | 使用原安全文件错误映射，fail closed |
| 读取期间变化 | 409，丢弃不完整快照 |
| JSON 转义正文合法且原文 ≤8 MiB | 允许；原文超限仍 413 |

## 5. 正常 / 基础 / 错误用例

正常：PNG 文件名为 `.txt` 仍按内容预览；合法 SVG 预览与源码切换。基础：普通文本只编辑，二进制下载。错误：文件扩展名为 `.svg` 就内联 DOM，或下载超限文件后才检查大小。

## 6. 所需测试

Go preview/HTTP 测试覆盖路径越界、稀疏超限、图像尺寸、SVG 主动内容、格式声明边界、匿名与 headers、JSON 转义大于旧 body 上限的真实落盘。Go/TS 共用 fixture 并拒绝缺字段/未知字段/非法尺寸。`w05-live.spec.ts` 在显式 Debian 私有实例实际落盘、冲突、PNG/SVG/匿名/unsafe SVG；fixture 浏览器不替代真实 Debian。其他完整编解码/真实手机矩阵未执行时须单列。

## 7. 错误与正确示例

错误：`c.File(userPath)` 或 `innerHTML = svg`，跳过根身份、类型和资源限制。

正确：`files.ReadPreview(registeredRoot, relative)` 后仅以确认 MIME 返回有界字节；前端 `FilePreview` 先 inspect，再显示受鉴权原生 img，切换卸载并取消 inspect。
