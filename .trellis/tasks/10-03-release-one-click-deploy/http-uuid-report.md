# 普通 HTTP UUID 故障修复

## 根因与修改

用户在Debian部署旧手工包后，浏览器报 `crypto.randomUUID is not a function`。三个业务调用位于 FileBuffer.id、EditorScope.viewId 和搜索匹配定位事件；仅loopback的本机测试掩盖了私有HTTP源的非安全上下文差异。

[Web Crypto规范](https://w3c.github.io/webcrypto/#crypto-interface)中randomUUID限定SecureContext，getRandomValues不受该限制。新增 `web/src/lib/random-uuid.ts`：可用时复用原生UUID，否则16随机字节设置v4版本/变体，再按标准格式输出。三个调用统一引用，无Math.random、全局polyfill、存储schema或认证变化；原上传随机batch与Monaco能力检测保持。

## 验证证据

- 生成器4项回归：原生调用、全0/全255随机值的版本/变体、1000个独立格式正确ID。会话1项回归：没有randomUUID时scope和buffer初始化、草稿归属与视图隔离。
- 前端lint/typecheck/test/build实际通过，32个测试文件、168项测试。既有jsdom canvas及构建大chunk提示仍保留，命令退出0。
- `lan-http.spec.ts` 使用虚构私有HTTP源，静态文件转取本机隔离服务，API/WS拦截为fixture。明确断言 `isSecureContext=false`、原生randomUUID缺失、getRandomValues存在，覆盖工作台/Monaco和重复搜索定位。
- 旧包负向回归在上述源重现相同 `index-2B5M867b.js:19:12378` 堆栈，工作台不出现。首次静态服务根目录错误与后续Node环境启动失败仅为harness问题，纠正后才取得产品复现证据，不将这些早期失败归于产品。
- 新包对应生产前端在Chromium和WebKit各1项通过。验证包内全部前端文件与浏览器实际读取的 `web/dist` 逐字节一致，不以dev server替代发布产物。
- 实际amd64包SHA、安全解压/manifest、ELF64架构和静态链接、完整模板/许可证、包根README检查通过；旁置文档与包根相同。
- `go test -count=1 ./...` / `go vet ./...` 本轮再次通过，GOTMPDIR使用短路径；未开启已取消跟进的dev启动集成，不记Actions成功。文档本地链接/围栏及diff空白检查通过，两个本机静态服务已停止。

## 新交付

目录 `dist/debian-amd64-http-uuid-20261003/`（忽略的本地产物）：`persistty-linux-amd64.tar.gz`、`SHA256SUMS`、`部署说明.md`，原独立下载器保留但手工部署无需使用。

版本 `manual-20261003-http-uuid`，tar SHA256：`a82798f2042222e6104fbd0650cc2b9c0dcdd04fd57626398b85a6b7449a5ac3`。基线HEAD仍为b007202，修复与文档来自未提交工作树；不声称该提交已含修复。旧包保留供对照，不覆盖旧交付。

按[手工部署说明](../../../deploy/MANUAL.md)第7节升级，保留实际后端/Nginx配置、数据库和独立tmux。浏览器强制刷新，加载新hash资源；无需修改HTTP方式。用户已报告实际Debian部署到浏览器阶段，本轮修复后的目标机升级/登录/WS尚未执行，模拟接口回归不替代真实后端或Nginx验收。原Actions dev重启失败仍未修复，任务保持in_progress，不自动提交或归档。
