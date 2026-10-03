# Hooks 与异步边界

hook 命名 useX，拥有副作用及其完整清理，依赖数组完整；禁止用 eslint-disable 隐藏 stale closure。组件 render 不发请求、不创建 Monaco/xterm，不向 Zustand 同步写派生值造成循环。
请求使用 AbortController，workspace/path/generation 变化取消旧请求；旧 response 不得覆盖新 workspace。读可重试，写入根据幂等/版本结果处理，不能一般化自动重试所有 POST。server cache 与 UI store 边界见 [状态](state-management.md)。
认证异步响应还必须绑定发起时的本地会话身份；不能只依赖 AbortController。TTL 到期后重新登录时，旧 logout 的迟到完成或旧请求的 401 不得清除新会话。login 先取消当前视图的旧 logout 并等待其 fetch 结算，再发送新登录，避免旧响应删除新 Cookie。取消不保证服务端不执行，也不构成跨标签 Cookie 强一致。认证 owner 集中更新身份并校验世代，测试包括旧 fetch 忽略 abort 且仍 pending 时不能先发 login。

```tsx
useEffect(() => {
  const controller = new AbortController();
  loadSnapshot(workspaceId, controller.signal);
  return () => controller.abort();
}, [workspaceId]);
```

片段仅示意，实际 hook 必须捕获非 AbortError、校验 response generation 并反馈 UI。订阅/window listener/ResizeObserver/WS/timer/addon 对称释放；React StrictMode mount-cleanup-remount 不能创建重复资源或 Close Terminal。
useTerminalConnection 只 attach/reconnect/detach，长期 tmux Create 必须用户 command；useFileDraft 只 IndexedDB，不能自动 PUT。watcher 事件让相关目录/cache 失效；dirty editor 不因外部事件直接 setValue，转入外部修改状态。
测试切换 workspace/取消/卸载/StrictMode/慢请求 out-of-order、401 停重连、effect 清理后无状态污染。


文件名快速打开按 project.id/version、关键词和序列化文件夹ID集合绑定effect；不能依赖轮询响应每次新建的project.folders数组引用，否则15秒刷新会取消同一个慢搜索并重发。响应仍验证项目版本和每项folder_id，改词/关闭/真正配置变化继续abort；FileQuickOpen.test与workbench-modern-ui浏览器专项覆盖跨16秒轮询、请求仍唯一和配置变化取消。
