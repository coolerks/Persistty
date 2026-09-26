# TypeScript 与 DTO

strict=true，启用 noUncheckedIndexedAccess、exactOptionalPropertyTypes；空值必须设计，不用非空断言躲检查。禁止 any、as unknown as、@ts-ignore 处理网络契约；JSON.parse/fetch JSON/WS 数据视为 unknown，在 lib/api/lib/ws owner 用 runtime decoder 验证（可选 Zod，安装前任务说明必要性）。decode 一次，组件消费 typed domain data。

区分 backend snake_case DTO 和 UI model；转换函数单一归属。Backend [HTTP](../backend/http-api.md)、[WS](../backend/websocket-protocol.md)、[文件版本](../backend/filesystem-guidelines.md) 是跨层权威。shared fixture 检查 nullable/ID/版本/UTF-16 column，不能仅编译 TS 类型就认为 Go JSON 一致。

```ts
type TerminalObservation =
  | { state: 'running' }
  | { state: 'terminated'; reason: string }
  | { state: 'unavailable'; code: string };
```

switch 判别联合穷尽处理；UI connection type 独立，不能把 unavailable 当 terminated。ApiError.details 按 code 解码，不能强转 current_version。二进制 Terminal bytes 保持 Uint8Array；Monaco 行列、rg byte offset、UTC 时间、chunk index/size 都显式转换和范围校验。

测试非法/missing/null/未知 discriminator/巨大数字/Unicode；断言 decoder 拒绝并出现可恢复错误。fixture 没有真实密码/session。必要第三方声明不完备时把 workaround 限定在 adapter 并注释原因及类型测试。
