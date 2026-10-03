# 首次 Actions 失败跟进

日期：2026-10-03。真实失败运行：[构建并发布 Persistty #1](https://github.com/coolerks/Persistty/actions/runs/37099594114)，commit `ec2fb384ce4f217f5613a5754ca5d1bced69d916`，Ubuntu 24.04，Go 1.26.8。

## 已确认

- build job 的工程门禁失败；安装依赖成功，打包与publish均未执行。
- 该步骤第一条Go命令stdout被重定向到JSON文件；Go非零退出触发bash -e，检查器未执行。完整job日志也只有依赖下载和退出码，无法恢复失败测试或编译详情。
- 该run的artifact API返回total_count=0，没有可下载的测试JSON。这不是已确认的依赖下载故障，也不能据此指定某个测试为根因。

## 修改

- release.yml拆分根Go/桥接/静态/前端部署检查；根和桥接无缓存-count=1；Go失败时仍运行日志检查器，原非零退出不得被JSON中的pass掩盖。
- check-go-test-log.py打印fail、required skip以及Go build-output/build-fail的有界上下文，拒绝规则保留。
- failure()上传go-test-logs，路径仅.cache/release-*-tests.jsonl；include-hidden-files=true确保隐藏目录里的日志能上传。
- 部署文档说明新workflow须提交推送触发，旧run的Re-run仍使用旧workflow。

## 本机验证

- 与CI相同开启PERSISTTY_DEV_INTEGRATION的无缓存串行根Go测试：退出0，198个Go pass事件，required无skip；包括真实本机启动/终端持久性验收。
- bridgego无缓存串行：退出0，29个pass事件，required无skip；root/bridge vet通过。
- 全部根包Linux amd64测试交叉编译通过，使用true执行器，**未执行Linux测试**；本机无可用Docker daemon，不冒充Ubuntu运行。
- 34项部署/日志回归通过，其中直接运行实际workflow根/桥接run块的8种隔离情况，确认正常、原Go非零、失败详情与required skip行为。
- actionlint/Python语法、Markdown链接和diff检查通过。前端/产品Go源码未修改，不重复与本变更无关的UI验收。

日志在忽略目录.cache/release-go-ci-repro.jsonl、release-linux-compile.jsonl、release-bridge-ci-check.jsonl、release-ci-diagnostics-tests.log。原macOS shell fixture清理竞态历史保留，本次无缓存通过不证明该竞态已修复。

## 仍待验证

新workflow未提交/推送或在GitHub执行，原Ubuntu失败的具体测试仍未知。此次修复的是失败可观测性，不能声称已消除原Ubuntu测试失败。下一次运行将直接显示详情并保留附件，才能继续针对实际失败修复。未重跑旧run、发布Release或操作目标部署机器；任务保持in_progress。
