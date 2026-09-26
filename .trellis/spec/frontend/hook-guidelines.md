# Hooks 与异步边界

hook 命名 useX，拥有副作用及其完整清理，依赖数组完整；禁止用 eslint-disable 隐藏 stale closure。组件 render 不发请求、不创建 Monaco/xterm，不向 Zustand 同步写派生值造成循环。
请求使用 AbortController，workspace/path/generation 变化取消旧请求；旧 response 不得覆盖新 workspace。读可重试，写入根据幂等/版本结果处理，不能一般化自动重试所有 POST。server cache 与 UI store 边界见 [状态](state-management.md)。

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
