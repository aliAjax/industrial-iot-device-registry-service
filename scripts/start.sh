#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."
GOTOOLCHAIN=local go run ./cmd/iotd -config configs/config.yaml
