# Bug 复现说明

## Bug 是什么

规则冷却状态跨 goroutine 未同步，锁范围不足导致并发消息在冷却窗口内重复触发动作；规则执行和保存还会别名调用方输入。

## 如何触发

```bash
go test -race ./q07a -run '^TestCooldownPreventsDuplicateActions$' -count=1
go test -race ./q07b -run '^TestSaveExecutionClonesInputs$' -count=1
go test -race ./q07b -run '^TestSaveRuleClonesActions$' -count=1
```

## 真实错误信息

```text
==================
WARNING: DATA RACE
Read at 0x00c000094ab0 by goroutine 11:
  github.com/example/iot-device-management/q07a.(*Engine).evaluateRule()
      q07a/engine.go:244
Previous write at 0x00c000094ab0 by goroutine 20:
  github.com/example/iot-device-management/q07a.(*Engine).evaluateRule()
      q07a/engine.go:244
```
