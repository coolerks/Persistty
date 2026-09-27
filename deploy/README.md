# 部署示例（W01）

这里只提供基础Web服务示例，不运行安装命令，不配置WireGuard、不更改公网防火墙、不安装tmux/helper。示例必须由运维按实际路径、VPN地址和现有非root开发UID审查后再应用，不能直接作为已经安全部署的证据。

## 配置与构建

`config.example.yaml`中的`password_hash`必须使用`persistty password`交互生成的Argon2id PHC替换。命令从TTY隐藏输入，不支持把明文密码放参数。生产配置文件0600、父目录0700并归服务UID，数据库及WAL/SHM也保持owner-only权限。`public_origin`须与用户浏览器地址的scheme/host/port严格一致，不带路径；所有写请求检查Origin。

```bash
go build -o persistty ./cmd/persistty
npm --prefix web ci
npm --prefix web run build
./persistty password
./persistty serve --config /etc/persistty/config.yaml
```

生产使用`vpn_http`且Nginx只绑定实际WireGuard地址，后端仅loopback。`tls`由可信Nginx终结HTTPS，Cookie带Secure；`vpn_http`Cookie没有Secure，因此不能暴露到VPN隧道之外，不声称HTTP本身加密。可信代理只列同机真实代理，禁止任意`X-Forwarded-*`扩大信任。

## Nginx页面分流

将前端`web/dist`发布至示例`/opt/persistty/web`，Nginx服务块见[配置](nginx/persistty.conf)。仅已定义客户端页面路由回退index.html，API永不回退HTML、缺失assets返回404。登录请求体限制在内存缓冲上限内，关闭请求体文件和代理缓存，不配置记录Cookie/请求体/查询参数的日志。该限制只用于W01小JSON；后续上传/WS包必须按实际有界协议更新，不能只提高全局大小。

运维应用前必须执行目标机`nginx -t`，分别验证页面书签直达、未知API JSON错误、缺失JS 404；验证IPv4/IPv6、VPN路由与公网不可达。只绑定VPN地址不代表防火墙/转发检查已经通过。当前Mac本地没有Nginx实机验证，不标通过。

## systemd边界

[unit](systemd/persistty.service)使用现有非root账号，`developer`是必须替换的示例，不创建第二个账号。状态目录由该账号拥有；只允许写`/var/lib/persistty`是W01限制。后续文件工具需要审核root句柄与实际目录写访问，W07受限sudo/helper与NoNewPrivileges的兼容性必须独立实验，不能静默移除限制或声称提权已可用。

Web unit无PartOf/BindsTo/tmux依赖、ExecStop不杀终端。终端长期任务须由未来W02/W03独立unit承载；此W01 unit不提供持久PTY。升级只重启Web，不重启未来tmux unit；Web自动重启不是任务复活机制。DB迁移失败拒绝服务，回滚前核查schema兼容性；不复制在线.db忽略WAL，不靠Git回滚真实文件/进程。

官方配置依据：[Nginx try_files](https://nginx.org/en/docs/http/ngx_http_core_module.html#try_files)、[Nginx请求缓存](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_request_buffering)。systemd目标Debian版本及限制仍需W02实机确认，未把本地无法访问的远程文档当作验证证据。
