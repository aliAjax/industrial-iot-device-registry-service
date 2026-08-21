# Bug 复现说明

## Bug 是什么

服务启动后持续接入遥测数据，队列关闭路径与并发入队路径不同步，会触发 `panic: send on closed channel`。

## 如何触发

```bash
go test -race ./q01a -run '^TestQueueCloseRejectsConcurrentEnqueue$' -count=1
go test -race ./q01b -run '^TestStartDoesNotCloseQueueBeforeIngest$' -count=1
go test -race ./q01a -run '^TestClosedQueueRejectsEnqueue$' -count=1
go test -race ./q01a -run '^TestClosedQueueTryEnqueueReturnsFalse$' -count=1
```

## 真实错误信息

```text
panic: send on closed channel

goroutine 20 [running]:
github.com/example/iot-device-management/q01a.(*Queue[...]).Enqueue(...)
    q01a/backpressure.go:25
github.com/example/iot-device-management/q01a.TestQueueCloseRejectsConcurrentEnqueue.func2()
    q01a/queue_test.go:22

--- FAIL: TestStartDoesNotCloseQueueBeforeIngest (0.00s)
    service_test.go:14: unexpected panic after Start: send on closed channel
```
