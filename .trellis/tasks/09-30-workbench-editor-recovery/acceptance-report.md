# W05 恢复验收

2026-10-01 用户明确要求恢复 W05/W06 统一验收。最新共同记录以[统一验收进展](../10-01-search-readonly-git/acceptance-report.md)为准；原[检查报告](check-report.md)和[截图反馈报告](feedback-check-report.md)保留历史真实Debian证据。

本轮修复触屏横屏切换到桌面模式的问题，补充六种图片/三主题/字体/横竖屏与未保存输入回归。本机自动化检查通过，当前真实手机、触控板、shell主题及新Debian专项尚未完成，任务保持 `in_progress`、`acceptance=in_progress`。未提交、推送或归档。

## 2026-10-02 再次调整优先级

用户明确要求先推进 [W07](../10-02-privileged-file-edit/prd.md)，本任务剩余统一验收延后。原已通过与未执行结果保持；任务仍为 in_progress，验收执行安排为 deferred，不记最终通过。


## 2026-10-02 本轮统一验收

最新用户授权覆盖此前延期安排，已直接执行可运行的本机门禁、浏览器和真实HTTP/终端专项；结果、首轮失败、复验与环境阻塞集中在[统一验收报告](../10-02-workbench-modern-ui/overnight-acceptance.md)。既有远端/设备证据保留，本轮未运行的Linux原生、真实手机/输入法/触控板与精确特权安装不记通过。任务保持in_progress，不归档或自动提交。


## 2026-10-03 本轮收尾结果

本机可运行验收已完成：Chromium/WebKit各51个不同用例获最终通过证据，共102个“浏览器×用例”；前端四门禁与31文件163单测、全量Go test/vet/race、独立bridgego普通/vet/race、45项Python探针通过。完整首轮失败、复验、真实HTTP/tmux、六种图片/三主题、缺rg搜索与清理证据见[统一验收报告](../10-02-workbench-modern-ui/overnight-acceptance.md)。W07其中7项/浏览器为mock权限服务，不代表真实sudo/PAM。名称搜索真实三根仍约5秒，重负载有15秒超时反例，不宣称即时响应。

Firefox启动、实体设备/IME/触控板、Debian原生/systemd与精确提权安装阻塞按用户要求跳过并记录；本机执行localChecks=completed，任务和整体验收保持in_progress，原pending保留。没有自动提交、推送或归档，没有修改用户真实文件或终止用户终端。
