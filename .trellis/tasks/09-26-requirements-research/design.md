# Persistty 首版设计审查稿

## 状态与边界

依据[PRD](prd.md)与[来源](research/requirements-source.md)，仅规划，未批准实施。以下工程方案/默认参数随本稿请求审查，未执行目标机实验。研究的本地file:line证据保留在四个专题，不迁移或删除。

## 架构

浏览器React应用通过VPN内Nginx的HTTP/WS访问Gin；后端普通用户服务负责认证、工作根注册、文件/传输/搜索/Git、SQLite元数据和终端控制。独立tmux systemd服务承载真实任务，与Web不共享销毁cgroup或socket生命周期。高权限保存是独立root-owned执行器，不把Gin进程提升为root。

共享：工作根资源身份、终端身份/名称/状态、上传会话。浏览器本地：当前根、标签/布局/主题、待恢复草稿；不存系统密码或登录token到Zustand/localStorage。认证使用HttpOnly会话Cookie，服务器只持有token hash及有效期/撤销状态。

三个事实域分别管理：真实文件/进程、持久元数据、当前视图。元数据不存在不能擅自清理真实任务；真实进程不存在不能根据标签创建替代进程。

## 数据流与契约

### 认证与部署

- 固定批准origin，Nginx仅绑定VPN地址/受批准VPN网络访问；Gin用同机loopback或Unix socket。公网只开放WireGuard，IPv4/IPv6与转发都验证，不以隐藏URL证明安全。
- `vpn_http`是明确生产部署模式，不自动落入development。该模式无法设置Secure Cookie，但保留HttpOnly、SameSite、host-only cookie、CSRF、严格Origin、会话期限与撤销；HTTP响应敏感内容no-store。未来TLS模式独立启用Secure，不以自制加密替代TLS。
- 登录创建独立会话，不撤销其他设备；默认退出当前浏览器会话，关闭其WS，任务继续。运维更改应用密码默认撤销全部认证会话，但不杀任务；首版不做复杂账号/权限系统。
- 请求和WS副作用都查有效认证及资源身份，终端输入不落日志/不广播观察者；消息队列、连接数、解析与认证并发有界。日志不记录密码、Cookie、输入、文件正文。

### 终端生命周期、单控制端

- 固定私有tmux socket与同一非root UID，Web管理调用禁止隐式启动另一tmux server。tmux版本/前台server策略、socket服务所有权先实验；是否需要keeper不能凭初始规范决定。见[终端证据](research/terminal-lifecycle.md)。
- 每次显式创建有不可复用terminal_id与创建请求幂等键。传入已注册当前工作根身份，服务器验证目录仍有效；创建后改变网页根不修改旧进程cwd。
- 每终端单一PTY输出源广播，各端独立有界队列，慢端可断开后重新同步，不阻塞真实任务或其他观察者。不能让多个观察连接的tmux attach尺寸互相影响PTY。
- 控制状态包含connection_id与递增control_generation；输入/resize携带或服务器绑定当前generation，旧输入拒绝。显式接管串行更新；断开释放控制，新端重连先看真实状态，不自动夺回/重放。
- 当前画面恢复握手分成快照、同步序号和live输出，测试交界不重复/漏掉；tmux display-grid与alternate-screen区别不能用普通文本拼接蒙混。优先复用xterm能力，原始输入不用于日志回放。
- 真实会话查证后标running/ended，管理链路错误标unavailable。整机重启、管理员停tmux后的原进程结束，保留元数据，不命令重跑。

### 终止请求

- `termination_request_id + terminal_id + control_generation + deadline`是唯一待处理请求；控制端明确确认后发起，所有查看端可取消。
- 服务端定时/序列化裁决；取消只有在服务器接受且仍pending时生效，随后广播cancelled。截止执行前复验原会话身份，状态变更只能一次。客户端显示计时不承担终止执行。
- 保守设计：发起控制端失效、控制权改变或Web重启时取消pending，不重启后补执行。Web启动把残留pending标cancelled，不自动重放。此故障取舍随本稿审查。
- 重连/新查看端返回当前pending；重复请求返回已有状态，不建立多计时器。命令执行结果不明时查询真实状态，不重试杀死同名替代任务。

### 工作根与普通文件

- 单独的注册操作接受用户明确选定绝对目录，完成认证/有效性/权限验证后固定真实目录身份并返回opaque workspace_id。后续File API仅workspace_id和相对POSIX路径，不能把所有根设为`/`替代安全设计。
- 切根是浏览器视图事务：读新根信息、计算范围外标签、提示、保存草稿、确认切换，目录不可用或草稿持久化失败不得悄悄丢内容。其他浏览器根不改变。文件资源按实际身份去重，切根不能重建model清空undo。
- Go安全目录句柄I/O；使用实施当时维护版本，若用os.Root.Rename最低API门槛为Go1.25，不凭门槛选择已经停止维护的版本。禁止EvalSymlinks检查后裸路径写入。限制socket/FIFO/device和其他非普通内容，根内symlink行为/目录枚举与ZIP不跟随越界目标，硬链接写入默认拒绝并提示。
- 文件版本以完整内容hash+metadata/identity复验；相同mtime/size但内容变了必须冲突。后端自己按身份串行写入，同目录完整临时文件、权限位保留（不继承setuid/setgid）、fsync与原子发布。明确非协作writer的最终窗口。
- rename/move/copy使用安全父目录句柄与明确不存在/版本前置条件，批次逐项返回；跨文件系统移动采用受控复制完成后删除源，源变动或任一步失败不伪装原子移动。树内拖拽复用同一操作入口，禁止自身/子目录目标。
- delete明确确认，绑定目标身份/预期版本并在执行前复验；检测到确认后目标替换/变化拒绝旧请求并要求重新确认，不承诺非协作writer下绝对CAS。普通权限，无回收站；与编辑器存储状态协调，取消零副作用，删除后自动保存禁止重建。目录变化/部分失败逐项报告，不承诺整树快照或隐式提高权限。
- watcher事件作为失效提示，外部创建/重命名/移动/删除刷新相关目录列表与打开状态，内容变化按干净刷新/脏缓冲区保留冲突处理。监听丢失/溢出、WS重连触发重新列举和版本复验；列表响应不能清空编辑输入，也不能凭旧事件重建已删除路径。

### 编辑与草稿

- `EditorAdapter`共享打开/内容generation/保存状态/语言模式/草稿接口；桌面Monaco、手机基础textarea类视图。基础视图不带diff或完整语言服务。
- 后端内容判定：先普通文件与大小，再有效UTF-8/BOM/二进制特征，未知后缀不拒绝文本；图片类型用内容判断。编码/换行与语言模式分开，保留BOM/CRLF/LF/无末尾换行，不替代字符转码保存。
- 去抖自动保存捕获content_generation，返回只推进对应已保存基线，新的输入继续pending。冲突/权限/路径消失暂停该文件自动提交；不得自动PUT最新版本来绕过用户解决。
- 本地IndexedDB保存内容、原始基线/版本、时间、workspace/path/schema/generation。成功清理只删已提交版本；多标签按实例/修订隔离，不能删另一标签输入。退出保留待恢复草稿，但未认证不展示正文；显式清理站点数据是用户边界。
- 恢复状态独立于正常autosave：先获取最新服务器内容，桌面只读diff，用户确认恢复到buffer，保持autosave暂停直到明确允许写回。手机基线已变仅保留/导出；版本没变也须明确确认。不存在目标不自动重建。
- Monaco不自定义语言服务器，优先内置tokenizer/providers；JS/TS/JSON/CSS等内置诊断逐项禁用，保留所需基础补全。锁定版本和示例行为测试，不把无LSP等同无诊断。

### Git与搜索

- Git读取本地真实仓库/元数据，固定argv无shell，禁分页器、外部diff/textconv/hooks/fsmonitor/隐式lazy fetch与可控配置注入。root外gitdir/worktree元数据必须显式注册验证或准确报不支持，不能裸读取后再复验冒充安全。
- 变更状态区分staged/unstaged/untracked；当前总变化以HEAD到磁盘为默认列表，编辑器色条HEAD到buffer，提交详情指定父到提交。选定tag/branch先解析固定commit id，再和当前工作内容比较；标明buffer未保存与磁盘差异。引用/HEAD变化使缓存失效，不混旧对象。
- 根提交空树；merge默认第一父并标注来源，其他父可作为提交详情选择，不提供完整合并流程。对象缺失报不可用，不联网补取。
- 搜索先形成遵守ignore、排除依赖/.git/二进制且根内安全的允许集合，再匹配；include只收窄。包含未被ignore的隐藏代码文件，附加exclude可收窄，默认依赖排除列表可配置。
- 项目搜索/替换共用单一后端regex方言与快照，不以JS预览和rg执行两种语义混用。优先rg兼容的非回溯单行模式；不支持lookaround/backreference等模式明确提示。capture替换语法在API固定；字节offset转Monaco UTF-16列须测Unicode。
- 预览生成有界preview_id与每文件原版本/新内容hash，确认后每文件再检查，跳过冲突/仍有pending编辑输入的文件并给理由；返回applied/conflict/error/skipped，取消只停止未提交部分，不自动重试成功项。

### 传输、ZIP与预览

- 持久upload_id/源内容身份、块索引/hash/确认状态与完成结果；页面仍有源File时可查询缺块续传。刷新/关页不保证恢复，不把File handle保存等同永久权限。
- source/目标内容hash相同skip，其他目标版本须明确替换确认，发布前再次复验。complete幂等，校验完整文件后同目录发布；临时空间/并发/TTL、取消回收都限制。二进制字节不做换行转换。
- 目录枚举循环读取所有批次，空目录能表示则保留，不能表示必须说明；逐项结果不伪装整批完成。ZIP保持范围内目录结构，跳过/拒绝特殊目标须报告，不follow目录symlink到根外。
- 普通ZIP使用有界任务状态与临时完成产物，全部close/校验成功才提供下载，避免流已返回200却无法表达末尾失败；不是一致快照。遇已检测改动失败提示重试，无法检测全部竞争不承诺同一时点。完成产物限额、取消及TTL回收。
- 图片/SVG用受保护内容入口；SVG采用安全图像模式，不innerHTML、不开放任意外链/脚本；不带Office/PDF渲染器。超编辑/预览上限仍可在传输限额内下载。

### 单文件提权执行边界

- 用户明确点击权限失败后的提权入口，展示目标绝对路径及一次保存范围；密码字段不进入共享store、草稿或日志。请求绑定session、nonce、root-owned配置允许的target_id、预期文件身份/版本、内容hash、过期时间，不用客户端任意命令或任意绝对路径充当root目标。
- 推荐独立root-owned helper与root-owned允许列表，由受控sudo机制验证系统用户授权，仅允许这个固定执行器。非root服务不能改helper/策略；helper自身重复验证策略、目标父目录/身份、版本、限额和完整内容hash，并维持原所有者/模式，拒绝链接重定向和特殊文件。
- 密码通过受限管道短时用于系统认证，不通过argv、URL、环境变量或一般日志；处理后清理字段/引用，避免代理请求body落盘/调试记录。helper只处理已授权内容快照，不能持续获取后续autosave。
- 一次授权在一次执行尝试后消耗，失败/取消/超时/目标变化需重新授权；网络重试只查询同一请求结果，不再次写入。系统sudo timestamp不能代替应用nonce/单文件验证。
- sudo/PAM是否可无TTY、安全传递内容与认证、helper恢复权限/SELinux/ACL限制、请求状态崩溃一致性均为实施前实验。机制无法满足安全契约则停止此交付重新审查，不能默认放弃首版要求，也不能改成root Web服务。

## 默认参数审查建议

参数均可配置、有合法范围；这是工程建议，不是性能测试结果或用户已单独批准的数值。

| 参数 | 建议初值 | 原因 |
| --- | --- | --- |
| 初次工作根 | 开发用户HOME | 不猜实际项目路径，以后记忆本浏览器选择 |
| 自动保存去抖 | 1秒 | 避免每按键写盘，状态立即更新 |
| 历史保留 | 5000行/终端 | 有界近期历史，tmux与浏览器上限一致 |
| 终止倒计时 | 10秒，允许1..120秒 | 留出跨端取消时间，不允许0秒绕过 |
| 登录绝对有效期 | 7天 | 浏览器重开继续，有明确到期；用户可调整 |
| 可编辑文本 | 8MiB | 限制Monaco/基础编辑与草稿资源，超限仅下载 |
| 图片预览输入 | 16MiB，另限像素 | 压缩字节数不代表解码内存 |
| 单文件/批次上传 | 256MiB/1GiB | 日常项目与图片/文档，非大型模型重点 |
| 全局暂存 | 2GiB | 上传与ZIP共享预算，磁盘不足提前拒绝 |
| 上传/ZIP临时TTL | 24小时/1小时 | 页面断网重试与完成下载后回收 |
| 提权请求TTL | 60秒 | 单文件一次尝试，不形成长授权窗口 |

连接/文件数量、搜索结果、像素、深度、hash/解码/认证并发等其余资源限制按目标机实验确定并在配置文档记录；不能用上述初值证明目标机性能。

## 规范差异与迁移

现有规范只是初始约定，产品尚无用户数据迁移。实施前更新以下owner契约，不改Trellis框架：

| 文件 | 需要更新 |
| --- | --- |
| backend/websocket-protocol.md、terminal-lifecycle.md | 多观察端替代额外attach拒绝、控制generation、恢复边界、终止请求/取消/故障 |
| backend/security-config.md | vpn_http、任意可访问根注册、Cookie差异、提权单次/策略/凭据 |
| backend/filesystem-guidelines.md、http-api.md | 注册根身份、CRUD前置条件、提权独立资源，不放宽全部路径API |
| frontend/editor-terminal-lifecycle.md、hook-guidelines.md | autosave、恢复暂停、桌面/移动adapter与无诊断 |
| frontend/state-management.md、backend/database-guidelines.md | 浏览器本地偏好不全局覆盖，共享资源/上传/终止元数据与迁移 |
| backend/transfer-search-git.md、process-guidelines.md | 共同搜索语义、Git副作用/根外元数据、持久续传与普通ZIP任务 |
| frontend/component-guidelines.md、theme-assets.md | 状态栏、完整CRUD、移动边界、链接、资产许可 |

完整本地file:line锚点见对应研究专题；不要将审查稿的候选类型/字段当作已经实施的API。

## 运行、回滚与验证

Web升级先备份配置/SQLite，迁移失败拒绝Web服务但不停止tmux；回滚兼容版本，不自动清会话/重跑命令。配置/DB/文件备份是运维说明，不内置快照系统。helper安装/策略变更独立审批验证，回滚先禁用提权接口再回退helper，不提升普通权限。

先做高风险目标机实验：独立cgroup/socket与Web重启、TUI恢复、多端尺寸、根安全CLI、sudo/helper凭证管道和失败恢复，再扩展常规界面。实验未执行、硬件/版本未知；实现计划列出验收关卡，任何实质方案变化必须重新评审。
