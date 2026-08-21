# Bug 复现说明

## Bug 是什么

命令重试的后台回收任务没有观察取消信号，服务关闭时取消不传播，命令状态仍会被继续修改；命令入队和仓储保存还会别名调用方传入的 payload。

## 如何触发

```bash
go test ./q04a -run '^TestReapExpiredObservesCancellation$' -count=1
go test ./q04a -run '^TestEnqueueDeepClonesPayload$' -count=1
go test ./q04b -run '^TestSaveCopiesCommandPayload$' -count=1
```

## 真实错误信息

```text
--- FAIL: TestReapExpiredObservesCancellation (0.00s)
    service_test.go:31: expected pending after canceled context, got retrying

--- FAIL: TestSaveCopiesCommandPayload (0.00s)
    repository_test.go:25: stored command aliases caller input: turbo
```
