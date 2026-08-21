FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/iotd ./cmd/iotd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/iotd /iotd
COPY configs/config.yaml /configs/config.yaml
EXPOSE 8080 9090
ENTRYPOINT ["/iotd", "-config", "/configs/config.yaml"]
