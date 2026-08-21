# Bug 复现说明

## Bug 是什么

时间序列查询复用底层数组并原地 append，导致历史序列被查询结果污染；批量写入和元数据保存也会别名调用方数据，缺省时间窗口处理不正确。

## 如何触发

```bash
go test ./q05b -run '^TestQueryReturnsIndependentSnapshot$' -count=1
go test ./q05a -run '^TestWriteBatchDoesNotAliasCallerMetadata$' -count=1
go test ./q05a -run '^TestDetectGapsDefaultsWindowForward$' -count=1
```

## 真实错误信息

```text
--- FAIL: TestQueryReturnsIndependentSnapshot (0.00s)
    store_test.go:41: stored series was mutated by query

--- FAIL: TestDetectGapsDefaultsWindowForward (0.00s)
    service_test.go:50: detect gaps with default window failed: timeseries.DetectGaps: end must be after start
```
