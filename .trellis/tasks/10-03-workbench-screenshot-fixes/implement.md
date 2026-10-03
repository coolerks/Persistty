# 执行计划

1. 保存用户已批准的四项范围与验收，启动任务，按 trellis-before-dev 读取对应前端规范。
2. 修复标签背景衔接、移除 tracked HEAD，复用原滚动及基线。
3. 添加终端最大化/恢复、顶部拖动吸附和常规布局持久化保护；保留所有宿主。
4. 重写登录页视觉组合，保留认证状态处理；补充用户行为及浏览器几何回归。
5. 执行 npm lint/typecheck/test/build 与 Chromium/WebKit 适用专项；Go test/vet、链接与 git diff --check。
6. 以 trellis-check 检查范围、复用、异步与生命周期，更新 owner spec 与检查报告。无自动提交、部署或其他任务归档。

7. 实施追加反馈：全屏反向拖动、原生滚动条覆盖与文字居中、历史已绘制切换和实时尺寸保留；补充真实指针及历史切换/刷新/尺寸变化回归，重新执行门禁及Chromium/WebKit专项。
