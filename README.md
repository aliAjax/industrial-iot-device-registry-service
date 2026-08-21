# IoT 设备管理与遥测平台

纯 Go 实现的物联网设备管理、数据治理、指令下发与固件升级平台。项目不包含前端页面，提供 REST API、gRPC、HTTP/WebSocket 和 MQTT 接入端。

## 核心能力

- 设备注册：产品类型、设备、分组、标签、属性模型、地理位置，支持启用、禁用、注销和凭证轮换。
- 认证接入：token、PSK、证书指纹校验，服务端校验设备身份、订阅范围和消息频率。
- 遥测摄取：HTTP、WebSocket、MQTT 三种入口，统一协议解码、规范化、时间戳校准和背压队列。
- 时序存储：面向设备、属性、时间窗口的原始查询、聚合、降采样和断点检测。
- 数字孪生：期望状态和上报状态分离，上报状态不会直接覆盖服务端权威期望状态，支持差异计算和合并。
- 指令下发：异步命令队列、设备确认、超时、失败回执、重试、幂等键和审计记录。
- 规则引擎：条件、窗口聚合和动作，支持告警、命令和事件转发，规则执行记录可重放。
- 固件升级：版本、升级任务、灰度策略、设备回执、失败回滚和同设备并发升级限制。

## 项目结构

```text
cmd/iotd                        服务入口
api/iot/v1                      API 类型和 gRPC proto
internal/device                设备领域
internal/auth                  认证领域
internal/telemetry             遥测领域
internal/timeseries            时序领域
internal/twin                  数字孪生
internal/command               指令领域
internal/rule                  规则引擎
internal/firmware              固件升级
internal/gateway               HTTP、gRPC 中间件与适配器
internal/platform              日志、事件总线、限流、背压、PostgreSQL 等基础设施
configs                        示例配置
migrations                     PostgreSQL 初始化迁移
deploy                         Docker Compose、Dockerfile、MQTT 配置
examples                       设备、WebSocket、gRPC、MQTT 客户端示例
scripts                        启动和冒烟脚本
```

## 快速启动

### 内存模式

无需外部依赖，适合本地验证：

```bash
make build
IOT_HTTP_LISTEN=:8080 IOT_GRPC_LISTEN=:9090 ./bin/iotd -config configs/config.yaml
```

如果本机 `8080`、`9090` 已被占用，可修改环境变量为其他端口。

### PostgreSQL + MQTT 模式

```bash
docker compose -f deploy/docker-compose.yml up -d
IOT_STORAGE_DRIVER=postgres \
IOT_STORAGE_POSTGRES_DSN='postgres://iot:iot@localhost:5432/iot?sslmode=disable' \
IOT_MQTT_ENABLED=true \
IOT_HTTP_LISTEN=:8080 \
IOT_GRPC_LISTEN=:9090 \
./bin/iotd -config configs/config.yaml
```

服务启动时如果使用 PostgreSQL 驱动，会自动执行 `internal/platform/postgres/schema.go` 中的 `CREATE TABLE IF NOT EXISTS`。生产部署建议先使用 `migrations/0001_init.sql` 和迁移工具管理版本。

## 配置项

配置默认位于 `configs/config.yaml`，所有字段均可用 `IOT_` 前缀环境变量覆盖，例如：

| 环境变量 | 说明 |
| --- | --- |
| `IOT_HTTP_LISTEN` | HTTP 监听地址 |
| `IOT_GRPC_LISTEN` | gRPC 监听地址 |
| `IOT_STORAGE_DRIVER` | `memory` 或 `postgres` |
| `IOT_STORAGE_POSTGRES_DSN` | PostgreSQL DSN |
| `IOT_MQTT_ENABLED` | 是否启用 MQTT 接入 |
| `IOT_MQTT_BROKER` | MQTT broker 地址 |
| `IOT_TELEMETRY_WORKER_COUNT` | 遥测处理 worker 数 |
| `IOT_AUTH_DEFAULT_RATE_LIMIT` | 默认认证速率限制 |

## HTTP API 示例

### 健康、就绪和指标

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/metrics
```

### 创建产品和设备

```bash
PRODUCT_ID=$(curl -sS -X POST http://localhost:8080/api/v1/products \
  -H 'Content-Type: application/json' \
  -d '{"name":"Demo Product","attributes":[{"key":"temperature","type":"number","unit":"celsius"}]}' | jq -r '.id')

DEVICE_JSON=$(curl -sS -X POST http://localhost:8080/api/v1/devices \
  -H 'Content-Type: application/json' \
  -d "{\"product_id\":\"$PRODUCT_ID\",\"name\":\"Demo Device\",\"credential_type\":\"token\"}")

DEVICE_ID=$(printf '%s' "$DEVICE_JSON" | jq -r '.id')
CREDENTIAL=$(printf '%s' "$DEVICE_JSON" | jq -r '.credentials[0].value')

curl -sS -X POST http://localhost:8080/api/v1/devices/$DEVICE_ID/enable
```

### HTTP 遥测接入

```bash
curl -sS -X POST http://localhost:8080/api/v1/telemetry/ingest \
  -H 'Content-Type: application/json' \
  -H "X-Device-ID: $DEVICE_ID" \
  -H "X-Device-Credential: $CREDENTIAL" \
  -d '{"kind":"telemetry","timestamp":"2026-08-19T15:00:00Z","data":{"temperature":23.5}}'

sleep 1
curl "http://localhost:8080/api/v1/timeseries?device_id=$DEVICE_ID&property=temperature"
```

### WebSocket 遥测

连接 `ws://localhost:8080/api/v1/telemetry/ws`，握手请求携带 `X-Device-ID` 和 `X-Device-Credential`，随后发送与 HTTP 相同的 JSON 消息。示例见 `examples/websocket_client`。

### MQTT 遥测

启用 MQTT 后向主题 `devices/{device_id}/telemetry`、`devices/{device_id}/events`、`devices/{device_id}/attributes` 发布 JSON。示例见 `examples/mqtt_publisher`。

### 指令、孪生和规则

```bash
curl -X POST http://localhost:8080/api/v1/commands \
  -H 'Content-Type: application/json' \
  -d "{\"device_id\":\"$DEVICE_ID\",\"name\":\"reboot\",\"idempotency_key\":\"cmd-1\"}"

curl -X PUT http://localhost:8080/api/v1/twins/$DEVICE_ID/desired \
  -H 'Content-Type: application/json' -d '{"mode":"eco"}'

curl -X POST http://localhost:8080/api/v1/rules \
  -H 'Content-Type: application/json' \
  -d '{"name":"high temp","enabled":true,"device_id":"'$DEVICE_ID'","property":"temperature","condition":{"operator":"gt","threshold":20,"aggregate":"avg","min_samples":1},"actions":[{"type":"alarm","config":{"severity":"warning"}}],"window":60000000000,"cooldown":60000000000}'
```

## gRPC 验证

服务使用 JSON codec 暴露 `iot.v1.IoTService`。示例客户端：

```bash
go run ./examples/grpc_client -addr localhost:9090
```

## 构建与自检

```bash
make tidy
make build
make test
```

运行完整冒烟脚本：

```bash
scripts/smoke.sh
```

项目使用 Go 1.22，`go build ./...` 已通过验证。
