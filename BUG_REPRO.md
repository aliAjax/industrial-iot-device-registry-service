# Bug 复现说明

## Bug 是什么

孪生更新路径在 defer 中吞掉仓储保存错误，导致保存失败被误判为成功；仓储保存文档还会别名调用方数据。

## 如何触发

```bash
go test ./q06b -run '^TestSaveClonesDocument$' -count=1
go test ./q06a -run '^TestUpdateDesiredReturnsSaveError$' -count=1
go test ./q06a -run '^TestUpdateReportedReturnsSaveError$' -count=1
go test ./q06a -run '^TestComputeDiffFindsNestedProperty$' -count=1
```

## 真实错误信息

```text
--- FAIL: TestUpdateDesiredReturnsSaveError (0.00s)
    service_test.go:29: expected save error to be returned

--- FAIL: TestSaveClonesDocument (0.00s)
    repository_test.go:26: stored document aliases caller input: turbo
```
