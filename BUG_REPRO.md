# Bug 复现说明

## Bug 是什么

固件失败回执处理链路中，任务查询错误没有被带回，自动回滚不会启动；任务缺失错误类型和零百分比回滚校验也不符合预期。

## 如何触发

```bash
go test ./q09b -run '^TestGetTaskMissingReturnsNotFoundKind$' -count=1
go test ./q09a -run '^TestCompleteReceiptReturnsTaskLookupError$' -count=1
go test ./q09a -run '^TestValidateRolloutRejectsZeroPercentage$' -count=1
```

## 真实错误信息

```text
--- FAIL: TestGetTaskMissingReturnsNotFoundKind (0.00s)
    repository_test.go:15: expected not_found kind, got internal (task not found)

--- FAIL: TestCompleteReceiptReturnsTaskLookupError (0.00s)
    service_test.go:22: expected task lookup error to be returned
```
