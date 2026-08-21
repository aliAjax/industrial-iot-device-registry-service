.PHONY: build run test tidy lint up down clean

build:
	GOTOOLCHAIN=local go build -o bin/iotd ./cmd/iotd

run:
	GOTOOLCHAIN=local go run ./cmd/iotd -config configs/config.yaml

test:
	GOTOOLCHAIN=local go test ./...

tidy:
	GOTOOLCHAIN=local go mod tidy

up:
	docker compose -f deploy/docker-compose.yml up -d

down:
	docker compose -f deploy/docker-compose.yml down

clean:
	rm -rf bin
