#!/usr/bin/env sh
set -eu

BASE="${BASE:-http://localhost:8080}"
curl -fsS "$BASE/healthz"
curl -fsS "$BASE/readyz"
curl -fsS "$BASE/api/v1/info"

PRODUCT=$(curl -fsS -X POST "$BASE/api/v1/products" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Demo Product","description":"smoke product","attributes":[{"key":"temperature","type":"number","unit":"celsius"}]}')
PRODUCT_ID=$(printf '%s' "$PRODUCT" | jq -r '.id')

DEVICE=$(curl -fsS -X POST "$BASE/api/v1/devices" \
  -H 'Content-Type: application/json' \
  -d "{\"product_id\":\"$PRODUCT_ID\",\"name\":\"Smoke Device\",\"credential_type\":\"token\"}")
DEVICE_ID=$(printf '%s' "$DEVICE" | jq -r '.id')
CREDENTIAL=$(printf '%s' "$DEVICE" | jq -r '.credentials[0].value')

curl -fsS -X POST "$BASE/api/v1/devices/$DEVICE_ID/enable"

curl -fsS -X POST "$BASE/api/v1/telemetry/ingest" \
  -H 'Content-Type: application/json' \
  -H "X-Device-ID: $DEVICE_ID" \
  -H "X-Device-Credential: $CREDENTIAL" \
  -d '{"kind":"telemetry","timestamp":"2026-08-19T15:00:00Z","data":{"temperature":23.5}}'

sleep 1
curl -fsS "$BASE/api/v1/timeseries?device_id=$DEVICE_ID&property=temperature"
