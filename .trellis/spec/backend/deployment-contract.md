# GitHub 发布与局域网部署契约

## 1. 范围与触发条件

部署为局域网 HTTP，沿用已有手动 Nginx 与其他方式安装的 Python。开源源码、模板、受版本管理文档和记录不保存用户真实 IP；首次 --host 指定地址，仅写入目标机配置。此指令覆盖旧VPN/TLS部署方案；deploy主流程不再配置这些环境或运行apt。后端仍使用非root账号、loopback反向代理和原有鉴权。正式目标机操作与GitHub发布未在本轮执行。

## 2. 签名

```bash
sudo "$(command -v python3)" persistty-deploy.py install --host LAN_IPV4 \
  [--user USER] [--nginx ABSOLUTE_BINARY] [--nginx-config ABSOLUTE_MAIN_CONFIG] \
  [--write-path ABSOLUTE_DIR ...] [--repo OWNER/REPO]
sudo /usr/local/sbin/persistty-deploy update [--version latest|TAG]
```

安装和更新另保留可选root私有`--token-file`认证入口，主文档不引入私有仓库分支。首次user默认SUDO_USER，不能使用root。--host 必填，仅首次安装支持 RFC1918 IPv4 地址（10/8、172.16/12、192.168/16），拒绝公网、loopback、DNS、端口、路径和配置指令。没有SSL/origin/listen参数。后端CLI仍为serve/password，password仅TTY隐藏输入。

## 3. 契约

- 模板使用 LAN_IP 占位；部署器将 --host 规范化为 state.origin=http://地址，从该值同时渲染 Nginx listen/server_name 与后端 public_origin。更新校验并沿用保存的 origin，不需 --host、不重写实际配置；后端127.0.0.1:8080，public_origin严格同源，mode=lan_http。lan_http要求HTTP私有IP origin与loopback后端；普通非Secure的persistty_session仍HttpOnly/SameSite Strict，写请求Origin/CSRF和WS鉴权不变。
- 已有Nginx主配置http块一次include `/opt/persistty/nginx.conf`，用户添加、不自动覆盖。站点root-owned0644，ROOT目录root-owned0755。使用保存的实际nginx程序-t和-s reload，可带原主配置-c，不调用nginx.service或发行版sites-enabled目录。
- Nginx查找PATH/常见安装路径（包括/usr/local/nginx/sbin/nginx），也可首次--nginx指定。tmux/rg/git从实际安装程序查找，保存绝对路径至配置/独立tmux unit，不固定/usr/bin。
- 第一次用实际Python运行；成功安装的`persistty-deploy` shebang使用sys.executable的实际绝对路径，避免sudo PATH切换解释器。只用Python标准库，不安装软件。
- main push/手工main自动发布同提交Linux amd64/arm64包；固定Action SHA、锁定工具链/npm；Go/root+bridgego、前端、部署回归/构建门禁，draft完整上传后Latest。required Go skip/失败/空发现拒绝发布；两个显式本地性能探针例外，不记性能验收。
- 根/桥接Go分别无缓存-count=1运行，保存命令退出码；即使Go失败也执行 `python3 scripts/check-go-test-log.py LOG_JSONL`，检查器显示失败测试或build-fail的最近20个输出事件（每个最多4KiB）、required skip原因。检查器失败或原Go退出非零都阻止发布。Actions failure()上传 `.cache/release-*-tests.jsonl` 为go-test-logs，显式包含隐藏路径；不上传真实后端配置或token。
- 资产名称不变，含bin/web/deploy、version.json实际commit/平台/迁移SHA与许可证、SHA256SUMS。删除本任务TLS模板，不强求归档存在TLS文件。单次Release响应锁定资产ID；GitHub HTTPS、跨域剥离Authorization、SHA/ELF、安全tar校验在停Web前；512MiB下载/1GiB解压/10000成员，拒绝链接/穿越/特殊成员/重复。
- /opt/persistty/releases不可变版本/current原子切换；/etc/persistty私有长期配置、/var/lib/persistty私有DB/tmux/staging。独立root0700部署设置state与备份、root锁。首次拒绝未知同名配置/数据/unit。普通更新不重写密码/配置，不chown项目，不部署W07。
- Nginx-t/reload在停Web前；Web stop→SQLite backup API（含WAL、quick_check、90秒期限）→切换前后端→Web start→active/匿名session401+unauthenticated/代理登录页检查。仅更新Web，tmux不停止/重启；初装启用独立tmux，同版健康无Web重启。
- 失败恢复兼容旧代码，DB migration/checksum不兼容保持Web停止；不自动覆盖DB、不恢复项目或真实进程。SIGKILL/断电不保证Python恢复分支执行。旧通用安装记录不自动接管；当前记录须含合法局域网HTTP origin和nginx路径；已保存合法origin的上一轮局域网安装记录可继续更新，不要求重复传入地址。

## 4. 验证与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 缺 --host 或非法地址/配置注入 | 参数拒绝，不生成站点或后端配置 |
| Go测试/编译失败或required skip | 显示失败上下文，上传日志附件，阻止构建/发布 |
| 缺现有程序/无有效非root账号 | 明确失败，不安装依赖 |
| 未include或错误站点响应 | HTTP检查失败，不声称部署成功；修正后重试update |
| 原Nginx-t/reload失败 | 停Web前失败，不操作nginx.service |
| 下载/校验/归档错误 | 原Web不停止 |
| 私有配置/锁权限错误、已有未知资源 | 拒绝读取/覆盖 |
| 同版本健康 | 无Web重启；原Nginx按已保存路径检查/reload |
| 备份失败 | 不切换，尝试恢复原Web，删除不完整备份 |
| 新版失败且DB兼容 | 原代码恢复并检查，命令仍失败 |
| DB不兼容 | 保留数据，Web停止，给出备份路径 |

## 5. 正常 / 基础 / 错误用例

正常：现有nginx.conf加include→实际Python install --host 局域网IPv4→局域网浏览器登录；后续update自动下载同提交包，tmux任务持续。

基础：自定义/opt/python与/usr/local/nginx，无需系统包布局；缺省SUDO_USER选择现有开发账号。

错误：把站点写进服务UID可替换的私有配置目录、用nginx.service重启手动Nginx、依赖/usr/bin/python3、忽略已提交WAL或启动失败时自动覆盖DB。

## 6. 所需测试

Go模式矩阵：虚构私有LAN源合法，公网IP/非HTTP/非loopback后端/路径拒绝；Cookie flags正确，旧模式兼容。

Python部署回归：三种私有IPv4网段、非法/公网/配置注入拒绝、state持久化后Nginx与后端地址一致、占位全部替换，实际LAN模板、自定义工具和systemd路径、原Nginx程序/主配置-t/-s reload、无nginx.service/apt、实际Python shebang、保留长期配置；原下载/安全tar/SHA/架构/锁/WAL备份/失败恢复/同版幂等保留。实际两架构包验包、工程门禁、文档链接/diff；目标机Nginx/systemd/HTTP/WS运行另验，未执行不记通过。

CI回归直接执行workflow根/桥接run块，替换为隔离Go fixture：正常、Go非零但JSON有pass、失败测试、required skip；验证退出码保留和详情输出。检查器另外覆盖build-output/build-fail。Linux交叉编译不作为Linux运行证据；旧run日志被重定向且未保存时，如实保留根因未知，待新workflow实跑，不猜测或跳过疑似测试。

## 7. 错误与正确示例

错误：把用户真实IP硬编码进源码/模板/开发记录，或`apt install nginx python3`替换用户现有环境，或`systemctl restart nginx`假定发行版service。

正确：用户主配置http块include `/opt/persistty/nginx.conf`，保存实际nginx路径与可选主配置，执行它的-t/-s reload；`sudo /usr/local/sbin/persistty-deploy update`使用首次实际Python，不改账号/密码或结束tmux。操作步骤归[部署文档](../../../deploy/README.md)。
