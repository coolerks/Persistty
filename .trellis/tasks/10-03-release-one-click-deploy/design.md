# 局域网部署设计

首次：现有 Nginx http 块 include /opt/persistty/nginx.conf → 实际 Python install --host 局域网IPv4 → 输入应用密码。后续仍为 sudo /usr/local/sbin/persistty-deploy update。

## 修改归属与边界

- 部署器负责 --host 的 RFC1918 IPv4 校验，阻止配置指令注入；state.origin 持久化为 http://地址，更新校验并读取该值。无需新增第二份地址状态或默认个人IP。
- config.example.yaml 和 nginx/persistty.conf 使用 LAN_IP 占位；render_config/render_nginx 从同一已保存 origin 生成真实目标配置。模板不直接运行。
- 测试使用虚构地址，覆盖三种私有网段、非法/公网/注入输入、持久化后生成配置；Go模式矩阵仅替换测试值，产品行为不变。
- README、规范、任务和日志去除个人 IP，不删除原失败结论。目标机生成的私有配置不进入仓库。

## 保留行为

### Actions 失败诊断

首次运行失败在根 Go 命令，但 stdout JSON 被重定向，runner 的 fail-fast 使原日志检查器未执行；没有上传日志附件，原失败用例无法从现存 job log 恢复。本轮归属为 release.yml 的命令状态处理和 check-go-test-log.py 的诊断输出；拆分根测试/桥接/静态/前端部署步骤，保存 Go 退出码、仍执行检查器、失败时上传指定日志。检查器保留 fail/required skip/空发现拒绝，增加测试与编译上下文。新增隔离回归验证失败详情可见及退出码不会吞掉，不改未证实失败的业务测试。

已有 Nginx/Python 路径、非 root 开发账号、下载校验、SQLite 一致备份、原子版本切换和 tmux 独立生命周期不变。更新保留密码、后端配置、项目和终端任务；不安装依赖或扩展 SSL/VPN/端口/DNS 支持。

## 第二次 Actions 失败边界

真实 run 37100462607（提交 fcf5301）显示 rg 14.1.0 的替换 JSON 缺少 replacement 字段，以及 Git 2.55.0 下历史分页 fixture 的快照冲突。搜索能力判断归 internal/search/native_engine.go：空输入只能验证模式，需额外用固定非空捕获/替换样本验证 JSON 能力，再固定 rg 或 native。native_engine_test.go 保留真实 rg 坐标对照；旧版 rg 无捕获 JSON 时用明确期望验证 native 展开，并新增旧能力下搜索→预览→应用回归。Git 冲突先以同版本隔离工具复现，若由 fixture 后台维护引起，只修 internal/gitview 的测试建仓边界，不改产品快照复验或增加重试。工具仅放忽略的 .cache，不升级目标机依赖；本轮不修改终端、前端、安装和发布流程。

## 第三次 Actions 失败边界

run 37102646682（提交 b007202）与用户附件只剩 scripts/dev/TestLocalStartupPersistence 第二次 restart 失败；搜索/Git修复已在Ubuntu通过。本轮归属 scripts/dev 的进程组释放和 supervisor 控制socket生命周期，以及真实联调诊断。先读取失败分类、以确定性回归复现连续重启竞态，再修改实际归属；不以增加固定等待、自动重试、跳过required测试或关闭race解决。失败输出只含固定分类，不能把私有启动日志、配置或终端正文上传。保留tmux独立、只清理本次自有前后端、禁止按端口杀其他进程；现有部署器/systemd/Nginx流程不改。

以上调试已被用户取消，未宣称重启故障修复，撤回本轮未提交的临时启动器诊断修改。

## 手工包边界（最新请求）

复用 scripts/package-release.sh 的前端构建、CGO_ENABLED=0 Linux编译、许可/迁移manifest、确定性tar与SHA；新增可选架构参数，默认all保持原Actions调用，仅本次选择amd64。deploy/MANUAL.md 归手工安装文档，同时复制为包根README；不新增安装器或下载步骤、不改产品Go、Nginx模板和systemd沙箱。文档明确占位IP/hash/账号/工具路径、root拥有的程序与Nginx配置、私有后端配置、独立tmux、数据保留与停止Web后的备份。验收为实际amd64 ELF、前端资源、必要配置与包内校验，真实Debian/systemd和Nginx仍由目标机执行。

## 普通 HTTP UUID 修复边界

现有 editor-session 的 FileBuffer.id/EditorScope.viewId 与 SearchPanel 定位事件直接调用 crypto.randomUUID；浏览器将其限定安全上下文，loopback本机测试未覆盖真实私有HTTP源。UUID生成归 `web/src/lib/random-uuid.ts`，安全上下文复用原生方法，其余用 getRandomValues 的16随机字节设置v4版本/变体、保持原字符串格式；不使用Math.random、不修改全局crypto、不改认证/API/存储字段。三个调用点统一替换；上传既有getRandomValues和Monaco自带能力检测保持。

回归：生成器单测验两端随机位、原生方法、独立身份；会话回归验无原生UUID时文件初始化与草稿归属。浏览器在拦截的虚构私有HTTP地址保持非安全上下文，静态资源转取实际打包产物、API/WS使用明确fixture；旧包复现失败、新包验工作台/Monaco/搜索定位，不能称目标机后端或真实Nginx复验。交付新路径而不覆盖旧包。
