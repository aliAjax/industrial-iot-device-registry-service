# Bug 复现说明

## Bug 是什么

gRPC 服务入口没有检查已取消的请求上下文，导致请求取消后仍继续访问底层服务；HTTP readyz 也不理会取消信号。

## 如何触发

```bash
go test ./q10a -run '^TestGetTwinReturnsCanceledForCanceledContext$' -count=1
go test ./q10a -run '^TestGetDeviceReturnsCanceledForCanceledContext$' -count=1
go test ./q10a -run '^TestQueryTimeSeriesReturnsCanceledForCanceledContext$' -count=1
go test ./q10b -run '^TestReadyzHonorsCanceledRequestContext$' -count=1
```

## 真实错误信息

```text
--- FAIL: TestGetTwinReturnsCanceledForCanceledContext (0.00s)
    service_test.go:22: expected canceled code, got OK (<nil>)

--- FAIL: TestGetDeviceReturnsCanceledForCanceledContext (0.00s)
panic: runtime error: invalid memory address or nil pointer dereference
```
