# Bug 复现说明

## Bug 是什么

设备仓储返回内部可变引用，服务层并发更新标签时与仓储读取产生数据竞争，内存缓存被并发读写污染。

## 如何触发

```bash
go test -race ./q03a -run '^TestUpdateTagsDoesNotRaceRepositoryReads$' -count=1
go test -race ./q03a -run '^TestAssignGroupsValidatesAndDeduplicates$' -count=1
go test -race ./q03a -run '^TestCreateGroupValidatesParent$' -count=1
go test -race ./q03b -run '^TestRepositoryGetDeviceCopiesTags$' -count=1
go test -race ./q03b -run '^TestRepositoryListDevicesCopiesItems$' -count=1
```

## 真实错误信息

```text
==================
WARNING: DATA RACE
Write at 0x00c00007cb40 by goroutine 11:
  github.com/example/iot-device-management/q03a.(*Service).UpdateTags()
      q03a/service.go:207
Previous write at 0x00c00007cb40 by goroutine 289:
  github.com/example/iot-device-management/q03a.(*Service).UpdateTags()
      q03a/service.go:207

--- FAIL: TestRepositoryGetDeviceCopiesTags (0.00s)
    repository_test.go:27: GetDevice returned an internal mutable reference
```
