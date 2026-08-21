# Bug 复现说明

## Bug 是什么

设备身份未登记时，认证接口返回内部错误而不是未授权；设备注册时的凭证类型和权限范围归一化也不符合预期，主存储复制还会别名调用方数据。

## 如何触发

```bash
go test ./q02a -run '^TestAuthenticateUnknownDeviceYieldsUnauthorized$' -count=1
go test ./q02a -run '^TestRegisterDeviceNormalizesCredentialType$' -count=1
go test ./q02a -run '^TestRegisterDeviceNormalizesScopes$' -count=1
go test ./q02b -run '^TestUpsertPrincipalClonesScopes$' -count=1
```

## 真实错误信息

```text
--- FAIL: TestAuthenticateUnknownDeviceYieldsUnauthorized (0.00s)
    service_test.go:17: expected unauthorized, got internal (auth.Authenticate: device identity not found: principal not found)

--- FAIL: TestRegisterDeviceNormalizesCredentialType (0.00s)
    service_test.go:32: expected normalized psk credential type, got PSK
```
