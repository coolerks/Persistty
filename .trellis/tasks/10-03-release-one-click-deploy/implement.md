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
