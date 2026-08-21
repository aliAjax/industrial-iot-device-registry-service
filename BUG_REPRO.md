# Bug 复现说明

## Bug 是什么

事件总线退订后旧处理器仍会收到消息，历史消息返回的 payload 会别名发布方数据；限流器没有正确保留配置的突发额度。

## 如何触发

```bash
go test -race ./q08a -run '^TestUnsubscribeStopsHandler$' -count=1
go test -race ./q08a -run '^TestHistoryReturnsClonedPayload$' -count=1
go test -race ./q08b -run '^TestLimiterAllowsConfiguredBurst$' -count=1
```

## 真实错误信息

```text
--- FAIL: TestUnsubscribeStopsHandler (0.00s)
    eventbus_test.go:34: handler called after unsubscribe: 8

--- FAIL: TestLimiterAllowsConfiguredBurst (0.00s)
    ratelimit_test.go:13: request 3 within configured burst should be allowed
```
