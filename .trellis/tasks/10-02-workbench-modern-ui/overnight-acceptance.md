# 本轮统一验收报告

执行日期：2026-10-02–03。用户授权直接执行剩余 E2E 与验收，阻塞时跳过并记录，不等中途回复。主会话单代理。W05/W06 原延期安排从本轮起撤销；W07 默认禁用及目标机精确授权要求保留。

## 范围和资源边界

W05 编辑/草稿/恢复/图片/字体，W06 搜索替换/只读 Git，W07 非特权后端和浏览器，以及新版工作台 UI。只使用本轮隔离 fixture、数据库、服务、tmux socket。用户5173/8080开发服务与既有5178预览不改；不安装系统组件、不上传未授权目标、不写用户真实文件。源码包含上一轮已验证但未提交的搜索性能修复，本轮在其基础上验收，不覆盖或自动提交。

## 执行计划

1. 单独启动隔离 Vite 和真实后端；mock 浏览器完整覆盖与真实 HTTP 专项分别执行，结果不混淆。
2. Chromium/WebKit 完整相关 E2E；检查 Firefox 可用性，不能启动则记录环境阻塞。
3. Go 全量 test/vet/race、锁定 npm 四门禁、独立探针本地回归、Linux 编译与可用原生环境。
4. 逐条映射 W05/W06/W07 验收标准；补做可运行的真实后端/终端证据。
5. 失败能复现则修复实际 owner、保留失败和复验；缺设备/精确目标授权就跳过，继续其余项目。
6. 汇总证据路径、修改、安全边界、阻塞及清理结果，更新各任务报告，不将未执行项记为通过。

## 本轮结论

本轮可运行的本机验收、E2E、最终质量门禁与资源清理已完成。Chromium/WebKit各51个不同用例均获得最终通过证据，共102个“浏览器×用例”；普通/race Go全量测试、vet、前端四门禁、独立bridgego及Python探针通过。不是一次未经复验的102项全绿：原始失败、修正原因、复验与工具输出全部保留。

W05/W06/W07与新版UI任务继续in_progress：Firefox、实体设备、Debian/systemd和真实提权仍有下文明确阻塞。本轮执行安排已完成，整体环境验收尚未完成，不自动提交、推送或归档。名称搜索在真实三根约5秒，高负载仍出现过15秒超时；这是残余性能限制，不能以本机专项通过掩盖。

## 执行记录

以下按阶段保留首轮失败和复验过程；最终判定以“最终浏览器矩阵”和“最终门禁”两表为准。

### 首批环境与门禁

- 独立5199前端仅代理本轮8101后端；真实 fixture 为新建私有根、SQLite与tmux socket，虚构测试口令，不借用用户认证。
- Go全量普通测试、全量vet、Linux CGO=0全量vet已通过；Linux Web/broker/helper/TTY probe与五包测试共9份产物编译通过，SHA256在忽略目录.cache/overnight/linux/sha256.json；均未上传、安装或在Linux运行。
- Python从总目录discover首次返回0项，不能作为通过；改按实际6个含测试目录显式discover，全套通过，逐套数量/退出码保留probe-local-explicit.log。
- 官方Playwright Firefox155已成功下载，本轮继续尝试启动；不能沿用旧报告的启动失败结论。
- Docker CLI可见，但本地daemon与Colima socket均不存在，Linux原生验证阻塞；未启动/安装系统环境。物理手机、中文输入法、真实触控板无操作条件。目标机精确载荷/安装授权未补齐，按用户指示跳过，不反复请求许可。

### 首轮 Chromium 与修正

31项合成API专项运行5.9分钟：29通过、2失败，没有skip。W05六项编辑恢复、W07七项确认/结果、新UI五项和终端设备/历史多数通过；失败不能写通过。

1. 91语言检查Lua/TypeSpec固定2秒等待不足。独立真实浏览器使用Monaco公共异步colorize等待provider后，两者正常产生keyword/string token，分别47.9ms/11.7ms，无pageerror；产品加载无需修改。测试改为等待公共异步API后保留91种token断言，不删语言/样例、不增加同步私有加载器。
2. 工作台综合操作30秒总期限到期，trace显示已执行到后段点击/断言，没有独立产品断言失败。该用例包含大量菜单/标签/滚动/重载/手机操作，总期限改为90秒，保留全部行为、几何和零mutation断言，最终复验结果另记。

Python显式6组共45项通过（bridge6/helper5/history6/snapshot4/terminal21/w06 3）；初次0项发现保留为编排错误。

### 真实后端第一批和资源负载

首轮11项真实HTTP专项：8通过、3失败。Git悬浮、双区域、合并父关系、105条历史三页分页、路径对话框、六种图片格式/三主题/字体、触屏横竖屏、替换取消零写入均有实际通过证据。3项失败为旧刷新按钮名称和搜索快捷键自动展开过滤后重复点击收起；按当前可访问名称/可见状态调整后在最终矩阵通过。没有删断言或修改产品行为。

首轮真实终端7项：6通过、1失败。失败的历史用例只执行pwd，无tmux scrollback；界面先显示读取状态后按既有空历史契约返回live，因而找不到“返回实时终端”。改为明确输出100行并断言历史xterm内容，再验证返回live。其余手机快捷键字节、粘贴/链接、多端控制权、关闭重开、上下移动零额外WS、倒计时与暗色确认通过。

W06真实用户三根只读性能在同时运行浏览器与启动集成时失败：有rg名称9.944s，literal达到15s；无rg名称达到15s。不是通过，也不把超时改成空结果。下一轮停止重负载后复验；保留search-performance.log。只读查看系统进程确认另有用户应用持续占CPU，未关闭或操作这些应用。

启动脚本显式真实集成通过44.38s：后台启动、重复start/stop、restart参数恢复、代理终端输入输出、pane PID保留。该测试只用自己的目录/端口/tmux socket，不等于Debian/systemd运行验收。

Firefox155实际启动两次均在打开网页前失败“Could not find profile folder”（普通临时profile与显式绝对私有profile均失败）。记录环境阻塞，跳过Firefox；已下载不记已验收。Linux原生与真实设备/特权安装仍按前述边界跳过。

前端31文件163单测已通过。中间lint发现语言等待循环的初始赋值无用，已移除初始赋值而保留每轮实际token判断，最终lint另记。独立bridgego模块第一次普通测试在200ms合成录制时返回record_incomplete，待空闲复验，不能以根模块通过代替独立模块通过。


### 第二轮真实Chromium（保留失败）

18项（11项真实HTTP+7项终端）：13通过、5失败，6.2分钟。刷新与过滤修正的相关项已推进，但出现其余失效前置条件及慢请求：

- 新仓库UI按用户最新要求只显示分支，不再显示“main · 尚无提交”；测试改为断言真实branch summary及独立“尚无提交历史”，保留无HEAD比较与只读断言。
- 变更按钮现在包含数量徽标，测试折叠定位兼容真实可访问名称中的数量。
- 三端取消终止用例的第三浏览器已经自动恢复全局终止状态弹窗，测试再点被modal覆盖的tab导致总超时；删除这次多余tab点击，直接断言第三端状态弹窗并从第一端取消，保留三端状态同步、关闭重开与控制权断言。
- Git两项在实际请求仍加载时达到5秒等待；搜索打开文件也在文件加载阶段达到5秒等待。trace中保留进行中的响应，不直接认定产品通过。测试后端经多轮累计大量终端，本轮停止并清理全部自有fixture，重新建新根/元数据/socket复验；不调整产品HTTP15秒预算、不关闭用户应用。

本轮新增修改均为既有E2E与任务文档；产品源码继续沿用上一轮搜索修复。中间失败输出与trace完整保留在忽略目录，不纳入提交。


全量race曾与浏览器并行启动，但auth Argon2在race下耗时50.52秒，出现本轮合成API总期限失败。已主动停止这次编排，不把中断记通过；后续门禁与浏览器改为完全串行，保留go-race.jsonl和stderr。与用户应用的负载无关的产品断言仍继续复验。


## 验收标准与证据归属

以下映射覆盖本轮本机可运行项；“本机证据”不是全部环境验收完成。最终矩阵结果另表列出，失败及未执行优先于历史通过。

| 任务标准 | 本轮证据与检查范围 | 剩余边界 |
| --- | --- | --- |
| W05 E01/E02 | editor-session单测、editor-recovery真实Monaco/合成API、HTTP保存/冲突Go测试；共享model/undo、BOM换行、提交期间新输入 | 最终浏览器复验结果以矩阵为准，真实手机软键盘仍缺 |
| W05 E03/E04 | 真实IndexedDB草稿、刷新零隐式PUT、最后标签关闭保护、存储失败、手机基线变化仅保留/导出 | Chromium中间总期限失败保留，独立复验通过；模拟视图不替代手机 |
| W05 E05 | watcher/session/配置身份Go测试、打开缓冲区的旧响应与版本保护单测 | Debian原生文件身份/系统级行为本轮未运行 |
| W05 P01/A01 | 真HTTP六种格式PNG/JPEG/GIF/WebP/AVIF/SVG实际解码3×2、响应隔离、401、hash、三主题font；SVG/容量Go回归 | Firefox启动阻塞；iOS/Android实际解码/字体未运行 |
| W05 L01–L03 | 自动化四组/排序/上下移动/model/undo、终端同ID/零多余WS、恢复、触屏390×844及844×390 | 真实触控板惯性、触摸滚动与前后台/IME未运行 |
| W06 S01–S04 | 全量search/files单测及边界/ignore/重叠根/取消/Unicode/预算；真实HTTP过滤定位；用户三根只读性能专项 | 串行六次最终复测通过，高负载超时保留；Linux原生安全句柄未执行 |
| W06 R01–R03 | 原子保存/替换强版本/部分结果Go回归、实际预览/冲突/应用与磁盘字节验证、传输等待取消零写入 | 本轮不向用户真实文件应用替换；实际手机diff和IME未执行 |
| W06 G01–G04 | 临时真实Git/CLI对照、合并父关系、三页冻结HEAD、hover真实统计、慢状态/零轮询、本地仓库只读性能、HEAD/index/config hash不变 | 源码与fixture由服务用户权限读；本轮未连接Debian |
| W06 I01/新版UI | 面板toggle/选中同步、微圆角与分隔器、顶部文件名搜索跨15s轮询、标签/状态栏/深浅色/窄屏、目录弹窗 | 物理输入/Firefox阻塞；任何失败按矩阵保留 |
| W07 E01–E03/E06/E07/E08 | 非特权Go broker/helper/storage/router回归；7个浏览器mock用例覆盖一次冻结保存、密码清空、未知结果只查状态、新输入及undo、取消与认证失败 | mock响应不是sudo/PAM认证，普通权限helper测试不是root保护验收 |
| W07 E04/E05/E09 | Linux helper/broker/Web/probe交叉编译与SHA256、当前平台可用的策略/协议/故障单测 | root-owned策略、UID/GID/ACL/xattr/安全标签、sudo/PAM/systemd/NNP/cgroup/Nginx真实安装均跳过，默认禁用 |

## 阻塞与跳过（不计通过）

| 项目 | 实际条件/证据 | 本轮处理与后续所需 |
| --- | --- | --- |
| Firefox | 官方155下载完成；两种profile均在网页前退出，错误Could not find profile folder | 跳过Firefox浏览器矩阵，需修复本机Playwright Firefox运行环境 |
| Linux原生及Linux-only测试 | macOS主机；Docker/Colima daemon socket均缺 | 仅交叉编译/vet，不等同Linux执行；需可用原生Debian或授权VM |
| Debian/systemd/远端验收 | 最新消息没有精确目标/载荷授权；先前上传审批拒绝证据仍保留 | 本轮不SSH/SCP，不绕过；后续精确目标和载荷授权后执行 |
| W07真实提权 | 无root-owned helper/broker/sudoers/PAM/systemd精确安装授权与真实系统认证条件 | 保留默认禁用，不索取/存储系统密码，不执行系统变更 |
| 真机手机与IME | 无可操作的iOS/Android实体设备/软键盘/输入法条件 | 模拟触屏仅记录viewport/hasTouch；中文输入组合、前后台与实际旋转待实机 |
| 触控板和shell主题 | 无物理触控板/目标Debian shell主题操作条件 | xterm合成wheel与sh真实输出照常测，惯性和主题不记通过 |
| 仅远端及旧E2E入口 | w05-live/w05-feedback-live/restart/force-restart明确要求远端私有根，部分旧/terminals入口已被最新欢迎页设计移除 | 本轮不伪造REMOTE_ROOT或绕过防护。另workbench-ui旧入口5项未执行，依赖专用extra/TUI根且含旧路由；当前专项覆盖相关本机行为，不将旧套件算通过。真实本地terminal、platform-matrix和启动持久性补证；SIGKILL/systemd完整A/B/C仍待远端 |


### Chromium最终本机结果

- 合成API全套31项最终运行：30通过、1个409草稿用例总期限失败。停止并行race后该用例单独原断言复验1/1通过14.0秒。按不同用例归一：31项均有本轮最终通过证据；不把中间失败删除，完整日志为chromium-final-mocked.log及chromium-draft-recheck.log。
- 全新真实HTTP+终端18项最终全部通过（chromium-clean-live.log，套件额外含下述图标专项，因此总汇报18通过/1失败，3.4分钟）。搜索UTF-16定位与clipboard/模型、冲突拒绝与重新预览明确落盘、取消零写入、Git比较/冻结HEAD/分页/合并/hover/无轮询/无HEAD、六格式图片与触屏自动保存，以及7项真实tmux终端行为均通过。只用隔离文件，Git HEAD/index/config hash保持。
- 额外editor-workbench真实图标/语言检查在首次Monaco动态模块5秒等待失败；改为有界15秒等待实际编辑器出现，断言与HTTP预算不变，独立复验1/1通过（用例25.7秒、整次27.1秒）。真实后端综合E2E使用90秒总期限，产品各请求15秒预算没有修改。
- 中间Git两项的加载失败在全新fixture、串行执行下通过；仅作资源条件关联，不声称证明某一个进程是唯一根因。用户运行应用与服务保持原状。


### WebKit与确定性时序

WebKit完整31项首轮29通过/2失败，最终两项时序复验通过。91语言公共异步加载、自动保存与BOM/换行、409/IndexedDB恢复、多组终端恢复已通过。发现两项原测试依赖操作快于1秒：最后标签保护与灰点宽度；WebKit中实际autosave已先发生，trace分别记录PUT与已销毁dot，不直接当作产品关闭时隐式保存或错误CSS。

两项改为公开Playwright受控时钟：编辑器加载完成后暂停，再编辑/关闭或检查灰点；关闭失败后推进1200ms仍必须零PUT，灰点检查后明确手动保存且恢复时钟。原数据、model、8px及零写入断言保留，真实autosave用例继续运行真实时间；产品1秒策略没有改变。修改后的两项在Chromium/WebKit均已复验通过，结果如下。

额外真实文件树/图标/语言/窄屏专项Chromium复验1/1通过25.7秒，无额外content写入、终端连接、外部请求或pageerror；此前首次动态模块等待失败保留。至此Chromium本轮已获31个合成API+19个真实/图标用例通过证据，两项时钟更新和缺rg专项最终结果见下表。


时序修正最终复验：Chromium2/2通过7.6秒，WebKit2/2通过8.7秒；关闭失败后实际推进1200ms仍零PUT，灰点实际8px且手动保存一次。证据为chromium-timing-recheck.log、webkit-timing-recheck.log和timing-results.json。WebKit合成API全套首轮29通过/2失败6.5分钟，其两项最终已通过；归一31项均有最终通过证据，不删除首轮失败。


WebKit真实19项首轮：18通过、1失败2.5分钟。其中真实图标/语言/窄屏和18项主专项的17项已通过。失败的粘贴/外链用例尚未进入终端操作，停在第11次快速登录：真实服务器429且界面按60秒冷却禁用登录。认证保护正常，不修改限流。终端openProject测试已按既有workbench-ui策略识别429、等待界面重新启用后只重试一次并强断言200，仅该分支扩展总期限；WebKit/Chromium原粘贴与外链断言各独立1/1复验通过（4.0秒/2.6秒）。没有把请求失败伪装登录成功。


## 最终浏览器矩阵（按不同用例归一）

| 范围 | Chromium | WebKit | 原始完整运行与复验 |
| --- | --- | --- | --- |
| 合成API/真实Monaco/IndexedDB 31项 | 31项均通过 | 31项均通过 | Chromium全套30/31+409独立1/1；WebKit全套29/31+时序2/2，时序改动Chromium亦2/2 |
| 真实HTTP/真tmux 18项+真实图标/语言1项 | 19项均通过 | 19项均通过 | 两边全套均18/19；Chromium图标独立1/1，WebKit粘贴独立1/1；认证等待改动Chromium粘贴亦1/1 |
| 完全缺rg的真实全文搜索1项 | 1/1通过4.3s | 1/1通过4.7s | 实际服务配置/nonexistent/rg；过滤ignore、UTF-16定位、轮询保持、预览零写入、明确替换后磁盘字节均断言 |
| 合计 | 51个不同用例通过 | 51个不同用例通过 | 共102个“浏览器×用例”，不把独立重跑累加为更多用例；保留每次失败/复验日志与trace |

完整E2E不是Firefox/物理设备/远端旧入口全部完成；以上是本轮实际可运行范围。W07其中7项/浏览器仍为mock权限服务响应，不宣称系统提权已工作。服务日志中测试上下文关闭及fixture重建产生的WS ECONNRESET/EPIPE不计产品断言失败；实际测试仍断言pageerror、请求和资源边界。

## 自有资源清理

四个本轮私有fixture根均由退出时先kill精确专属tmux socket再删除目录。5199 Vite与8101后端均停止，Firefox临时profile不存在；cleanup-result.json共7项实际检查全部通过。没有停止用户5173/8080、已有5178/8098预览或其他应用。原始日志、截图、trace和9份Linux编译产物保留在忽略目录.cache/overnight，运行配置/SQLite/测试文件已随私有根删除。


## 真实大项目性能与限制

停止所有本轮浏览器、Vite、后端、tmux后单独只读复验通过，仍用默认15秒请求预算，不改超时配置、不建立缓存索引，不向源根或真实元数据写入。根来自用户报告项目的3个文件夹；临时SQLite仅创建隔离测试项目。最终search-performance-final.log共40.27秒，六次请求全部通过：

| 工具条件 | 名称project | literal Backend | regex Back[a-z]+ |
| --- | --- | --- | --- |
| 有rg | 5.170s，22项，无截断 | 7.498s，122文件/319匹配 | 9.034s，900文件/5000匹配，按配额截断 |
| /nonexistent/rg | 4.960s，22项，无截断 | 7.512s，122文件/319匹配 | 6.084s，900文件/5000匹配，按配额截断 |

跳过统计：literal179项、regex89项；保留过滤/字节/匹配配额语义，不把配额截断称为完整结果。首轮与其他验收同时运行时的15秒超时仍是有效性能反例；当前方案改善了重复扫描/工具启动开销，但5秒名称查询仍是秒级，不承诺即时响应或任意负载下不超时。后续若需要接近VS Code的即时大项目定位，应另规划有失效边界的索引/缓存，不能本轮绕过文件身份和权限复验。

当前仓库真实Git只读专项9.56秒通过：发现295ms，status4.470s（42项），baseline957ms，HEAD比较916ms，历史924ms（50条），详情947ms（30文件），提交比较1.025s；验证源HEAD/index/config hash未改变。证据git-performance.log；不是Debian或所有大仓库的性能承诺。


## 最终门禁（2026-10-03收尾）

| 检查/实际命令 | 最终结果 | 原始证据（均相对仓库.cache/overnight） |
| --- | --- | --- |
| `go test -count=1 -json ./...` | 15个有测试包通过，6个包无测试；197个test/subtest通过事件，无fail | go-test-fresh.jsonl/.stderr |
| `go test -race -count=1 -p1 -json ./...` | 15个有测试包通过，6个包无测试；197个test/subtest通过事件，无fail；串行完整完成，前次主动中断不计通过 | go-race-final.jsonl/.stderr，前次go-race.jsonl/.stderr保留 |
| `go vet ./...` | 全量通过 | go-vet.log |
| Linux CGO=0 vet与交叉编译 | 全量vet通过，Web/broker/helper/TTY probe及5包测试共9份产物编译通过；仅编译不等于运行 | linux-vet.log、linux-build.log、linux/sha256.json |
| 独立bridgego `go test -count=1 ./...` / `go vet ./...` / `go test -race -count=1 ./...` | 16个顶层测试（含子例）所在独立模块普通/race/vet均通过；普通1.249秒、race2.523秒 | bridgego-test-final-network.log、bridgego-vet-final.log、bridgego-race-final.log |
| `npm run lint` / `npm run typecheck` | 当前全部源码/E2E通过 | frontend-lint-final2.log、frontend-typecheck-final.log |
| `npm test` | 31文件/163项通过，5.93秒 | frontend-test-final.log |
| `npm run build` | 字体SHA256/等宽/glyph与37项许可证验证、tsc、Vite构建通过（Vite2.27秒） | frontend-build-final.log |
| Python显式6个目录的unittest discover | 45项通过；总目录discover的0项错误编排不作为通过 | probe-local-explicit.log、probe-local.log |
| 三项Go opt-in集成 | 普通/race默认跳过的SearchPerformance、RepositoryPerformance、StartupPersistence均已另行显式真实执行，通过 | search-performance-final.log、git-performance.log、local-startup-persistence.log |
| 格式、文档与状态 | 15个新增/修改Go文件gofmt干净；99个本地Markdown链接有效、4个task JSON可解析且状态一致、53项relatedFiles与git实际修改完全一致；git diff --check通过；日志/截图/trace/构建结果受ignore保护 | static-checks-final.json |

独立bridgego先前200ms合成录制在并行负载下record_incomplete；最终空闲运行原代码/原断言通过，没有调长录制或改探针。恢复后的第一次沙箱内复验因httptest监听权限被拒而失败，授权的隔离本机监听复验通过；保留bridgego-test.log与bridgego-test-final.log两份失败。该权限错误不是产品断言通过，也没有修改系统配置。

Vitest仍输出jsdom缺少Canvas实现的既有提示，实际图片/终端浏览器专项另有真实验证；Vite仍有大于500kB的既有chunk警告，没有关闭或抑制。构建通过不代表首次加载体积问题已优化。

静态核对脚本初次使用git check-ignore -q检查多个路径、随后以文本比较中文路径转义均失败；改用--stdin/-z保留原路径后4项实际证据均确认受ignore保护。只是核对脚本修正，没有修改gitignore或删除文件来制造通过。

## 本轮修改与审查

产品修复沿用本轮开始前已实施的安全单遍搜索/目录ignore剪枝、literal matcher与有界批量regex，以及FileQuickOpen稳定folder ID依赖；API/DB字段、15秒HTTP预算、认证、替换强版本和终端生命周期没有另行放宽。全量test/race及对照回归验证控制文件/身份冲突、ignore优先级、重叠根、Unicode/BOM/换行、LF文件名、取消与配额。名称结果打开仍由原文件接口重新鉴权。

验收期间改动集中在测试实际前提与当前入口：可访问名称/数量徽标、过滤区已展开后不重复收起、全局终止Dialog、真实scrollback、Monaco公共异步加载、自动保存前受控时钟、429冷却后单次重试与私有根防护。保留输入/model/undo、磁盘字节、零隐式写入、外链及进程边界断言；没有强制点击遮罩下元素、删除失败断言、修改真实限流或将真实API换成mock。

已人工查看最终浅色工作台、Git树与分段tab、缺rg搜索面板、项目欢迎页截图：面板微框线/空隙、顶栏搜索、图标动作、选中tab与背景区分、底部状态文字及左/右底边保持当前统一样式；几何/深浅色/窄屏的量化断言见102项矩阵。截图是fixture表现，不替代用户真实长文件/物理设备复核。

## 证据索引与复查

所有原始输出放在仓库`.cache/overnight/`，被gitignore排除，不纳入提交；可提交的需求、规范与结论为本报告及各任务报告。不要将忽略目录清理为“修复失败”。

| 范围 | 最终完整运行与必要复验 |
| --- | --- |
| Chromium合成API31项 | chromium-final-mocked.log、chromium-draft-recheck.log、chromium-timing-recheck.log |
| WebKit合成API31项 | webkit-clean-mocked.log、webkit-timing-recheck.log |
| Chromium真HTTP/tmux/图标19项 | chromium-clean-live.log、chromium-clean-icons.log、chromium-paste-recheck.log |
| WebKit真HTTP/tmux/图标19项 | webkit-clean-live.log、webkit-paste-recheck.log |
| 缺rg真实搜索 | chromium-norg.log、webkit-norg.log、norg-results.json；私有配置的Search.Binary=/nonexistent/rg |
| 时序复验汇总 | timing-results.json（两浏览器exit0） |
| 失败trace与截图 | 对应chromium-*/webkit-*目录；所有trace.zip及test-failed截图保留 |
| 环境阻塞 | firefox-install.log、firefox-probe.log、firefox-profile-probe.log；Linux与设备条件见阻塞表 |
| 原始重负载搜索反例 | search-performance.log；不能用final日志覆盖 |
| 清理 | cleanup-result.json：7项全true；cleanup-roots.json仅记录本轮私有根 |

复现浏览器时使用新隔离项目/后端/端口、每浏览器独立输出目录并串行执行。mock组为editor-assets/editor-recovery/projects-modern-ui/terminal-device-attributes/terminal-geometry/terminal-scrolling/w07-elevation/workbench-interactions/workbench-modern-ui；真服务组为editor-workbench/git-hover/git-layout/git-pagination/git-requests/screenshot-adjustments/terminal/w05-platform-matrix/w06-local。原始编排脚本browser-command.py/timing-recheck.py、临时fixture源码local-fixture.go/local-fixture-norg.go也保留在忽略目录，仅作本次隔离证据，服务已经关闭。

## 后续待办和任务状态

1. 名称查询仍约5秒，高负载有15秒超时反例；需要进一步性能目标时规划具备失效/权限/身份边界的索引或缓存，当前实时扫描不承诺即时响应。
2. 修复Firefox启动环境后补该浏览器矩阵；实体iOS/Android软键盘/IME/旋转/前后台、触控板惯性和目标shell主题须实机执行。
3. 精确Debian目标/载荷授权和原生运行条件具备后，执行Linux-only安全文件/提权、完整终端SIGKILL与systemd重启专项；W07仅在精确安装差异授权后接root helper/sudo/PAM/systemd/Nginx。默认禁用保持。

四个任务均保留in_progress；meta中本机执行localChecks=completed，pending保留缺环境/精确安装项，统一报告指向本文件。任务不因本轮可运行范围结束被完成或归档。提交方案已整理在[commit-plan.md](commit-plan.md)，本轮没有git add/commit/push，也没有触碰用户服务或真实文件。
