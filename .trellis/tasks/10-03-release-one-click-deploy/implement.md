# 实施与验证

1. trellis-before-dev：读取需求、设计及部署/配置规范。
2. 部署器新增首次 --host 校验及持久化；模板从同一地址渲染，更新保持单命令。
3. 文档和受版本管理开发资料去除用户实际 IP，测试使用虚构地址。
4. trellis-check：部署隔离回归、Go配置及test/vet、Python/Bash/Actions检查、实际两架构打包验包、链接和diff检查。
5. trellis-update-spec：同步参数与占位契约；记录实际结果，不自动提交、归档或安装目标机。

前轮结果见 [检查报告](check-report.md)，既有无缓存Go shell fixture失败与目标机未执行项保留；本轮验证结果将在该报告追加。

本轮已完成32项部署回归、Go test/vet、模板与开源文件扫描、两架构实际打包验包；文档/规范已同步。目标机和GitHub实测尚未执行，详见检查报告。

## Actions 首次失败跟进

读取真实run/job日志 → 修复Go退出码与诊断输出/失败附件 → 回归实际workflow命令 → 同参数无缓存Go、本机桥接与vet、Linux测试交叉编译 → 更新规范和失败报告。原Ubuntu失败用例因未保留stdout而未知，新工作流尚未推送执行；不以本机通过代替Ubuntu实跑。

## 第二次 Actions 跟进

用户推送后 run 37100462607 已提供具体错误。按设计边界修复 selectEngine 的 JSON 替换能力探测和 Git fixture 的自动维护隔离；用隔离 rg 14/Git 2.55 重现原错误、负向验证回归、运行修复后完整门禁，并同步 owning spec。最新证据与实际本机环境限制见 [工具兼容检查报告](ci-tool-compat-report.md)，不把本机交叉编译写成 Ubuntu 执行。

## 最新交付：手工 Linux amd64 包

用户停止 Actions 跟进，改为本机交叉编译并自行在 Debian 安装。已恢复临时 dev 测试诊断，保留原 CI 失败状态；打包脚本增加单架构选择，默认两架构不变。新增 `deploy/MANUAL.md`，同时置于包根 README 和产物目录，列明文件目标位置、账号/配置替换、现有 Nginx 接入及保留 tmux 的手工升级。已生成实际 amd64 包并验证；结果与限制见 [手工包检查报告](manual-package-report.md)。未提交、发布、安装目标机或归档。

## 已部署后的 HTTP UUID 修复

统一 UUID 生成器替换 editor-session/SearchPanel 的三个直接调用；单测覆盖HTTP分支和草稿隔离。生产旧包在浏览器非安全HTTP源复现相同堆栈，新构建在Chromium/WebKit通过工作台/Monaco/重复搜索定位；交付新版 `manual-20261003-http-uuid`，详见 [HTTP UUID检查报告](http-uuid-report.md)。升级保留实际配置/数据/tmux，不修改Nginx或要求HTTPS。
