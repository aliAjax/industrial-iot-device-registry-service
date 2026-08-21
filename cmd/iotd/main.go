package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	authapp "github.com/example/iot-device-management/internal/auth/application"
	authdomain "github.com/example/iot-device-management/internal/auth/domain"
	authmem "github.com/example/iot-device-management/internal/auth/infrastructure/memory"
	authpg "github.com/example/iot-device-management/internal/auth/infrastructure/postgres"
	commandhttp "github.com/example/iot-device-management/internal/command/adapter/http"
	commandapp "github.com/example/iot-device-management/internal/command/application"
	commanddomain "github.com/example/iot-device-management/internal/command/domain"
	commandmem "github.com/example/iot-device-management/internal/command/infrastructure/memory"
	commandpg "github.com/example/iot-device-management/internal/command/infrastructure/postgres"
	"github.com/example/iot-device-management/internal/config"
	devicehttp "github.com/example/iot-device-management/internal/device/adapter/http"
	deviceapp "github.com/example/iot-device-management/internal/device/application"
	devicedomain "github.com/example/iot-device-management/internal/device/domain"
	devicemem "github.com/example/iot-device-management/internal/device/infrastructure/memory"
	devicepg "github.com/example/iot-device-management/internal/device/infrastructure/postgres"
	firmwarehttp "github.com/example/iot-device-management/internal/firmware/adapter/http"
	firmwareapp "github.com/example/iot-device-management/internal/firmware/application"
	firmwaredomain "github.com/example/iot-device-management/internal/firmware/domain"
	firmwaremem "github.com/example/iot-device-management/internal/firmware/infrastructure/memory"
	firmwarepg "github.com/example/iot-device-management/internal/firmware/infrastructure/postgres"
	gatewaygrpc "github.com/example/iot-device-management/internal/gateway/grpc"
	gatewayhttp "github.com/example/iot-device-management/internal/gateway/http"
	"github.com/example/iot-device-management/internal/platform/clock"
	"github.com/example/iot-device-management/internal/platform/eventbus"
	"github.com/example/iot-device-management/internal/platform/logging"
	"github.com/example/iot-device-management/internal/platform/postgres"
	rulehttp "github.com/example/iot-device-management/internal/rule/adapter/http"
	ruleapp "github.com/example/iot-device-management/internal/rule/application"
	ruledomain "github.com/example/iot-device-management/internal/rule/domain"
	rulemem "github.com/example/iot-device-management/internal/rule/infrastructure/memory"
	rulepg "github.com/example/iot-device-management/internal/rule/infrastructure/postgres"
	telemetryhttp "github.com/example/iot-device-management/internal/telemetry/adapter/http"
	telemetrymqtt "github.com/example/iot-device-management/internal/telemetry/adapter/mqtt"
	telemetryws "github.com/example/iot-device-management/internal/telemetry/adapter/websocket"
	telemetryapp "github.com/example/iot-device-management/internal/telemetry/application"
	timeserieshttp "github.com/example/iot-device-management/internal/timeseries/adapter/http"
	timeseriesapp "github.com/example/iot-device-management/internal/timeseries/application"
	timeseriesdomain "github.com/example/iot-device-management/internal/timeseries/domain"
	timeseriesmem "github.com/example/iot-device-management/internal/timeseries/infrastructure/memory"
	twinhttp "github.com/example/iot-device-management/internal/twin/adapter/http"
	twinapp "github.com/example/iot-device-management/internal/twin/application"
	twindomain "github.com/example/iot-device-management/internal/twin/domain"
	twinmem "github.com/example/iot-device-management/internal/twin/infrastructure/memory"
	twinpg "github.com/example/iot-device-management/internal/twin/infrastructure/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type deviceLookup struct {
	devices *deviceapp.Service
}

func (l *deviceLookup) FindDevice(ctx context.Context, deviceID string) (authdomain.DeviceIdentity, error) {
	if l.devices == nil {
		return authdomain.DeviceIdentity{}, fmt.Errorf("device service is not initialized")
	}
	device, err := l.devices.GetDevice(ctx, deviceID)
	if err != nil {
		return authdomain.DeviceIdentity{}, err
	}
	return authdomain.DeviceIdentity{
		ID:        device.ID,
		ProductID: device.ProductID,
		Enabled:   device.Status == "enabled",
	}, nil
}

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to YAML configuration")
	flag.Parse()
	if err := run(*configPath); err != nil {
		slog.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	logger := logging.New(cfg.Logging)
	slog.SetDefault(logger)

	rootCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	clock := clock.SystemClock{}
	bus := eventbus.NewMemoryBus(2000)
	registry := prometheus.NewRegistry()

	var deviceProducts devicedomain.ProductRepository
	var deviceDevices devicedomain.DeviceRepository
	var deviceGroups devicedomain.GroupRepository
	var authRepo authdomain.Repository
	var twinRepo twindomain.Repository
	var commandRepo commanddomain.Repository
	var ruleRepo ruledomain.Repository
	var firmwareRepo firmwaredomain.Repository
	var timeSeriesStore timeseriesdomain.Repository
	var pool *pgxpool.Pool

	if cfg.Storage.Driver == "postgres" {
		pool, err = postgres.OpenPool(rootCtx, postgres.PoolConfig{
			DSN:          cfg.Storage.PostgresDSN,
			MaxOpenConns: cfg.Storage.MaxOpenConns,
			MaxIdleConns: cfg.Storage.MaxIdleConns,
			MaxLifetime:  cfg.Storage.ConnMaxLifetime,
		})
		if err != nil {
			return fmt.Errorf("open postgres pool: %w", err)
		}
		defer pool.Close()
		if err := postgres.EnsureSchema(rootCtx, pool); err != nil {
			return fmt.Errorf("ensure postgres schema: %w", err)
		}
		devicePG := devicepg.NewRepository(pool)
		deviceProducts, deviceDevices, deviceGroups = devicePG, devicePG, devicePG
		authRepo = authpg.NewRepository(pool)
		twinRepo = twinpg.NewRepository(pool)
		commandRepo = commandpg.NewRepository(pool)
		ruleRepo = rulepg.NewRepository(pool)
		firmwareRepo = firmwarepg.NewRepository(pool)
		timeSeriesStore = timeseriesmem.NewStore(20000)
	} else {
		deviceMem := devicemem.NewRepository()
		deviceProducts, deviceDevices, deviceGroups = deviceMem, deviceMem, deviceMem
		authRepo = authmem.NewRepository()
		twinRepo = twinmem.NewRepository()
		commandRepo = commandmem.NewRepository()
		ruleRepo = rulemem.NewRepository()
		firmwareRepo = firmwaremem.NewRepository()
		timeSeriesStore = timeseriesmem.NewStore(20000)
	}

	timeSeriesService := timeseriesapp.NewService(timeSeriesStore, clock)
	twinService := twinapp.NewService(twinRepo, clock)

	lookup := &deviceLookup{}
	authService := authapp.NewService(authRepo, lookup, clock, cfg.Auth.DefaultTokenTTL, cfg.Auth.DefaultRateLimit, cfg.Auth.DefaultRateWindow)
	deviceService := deviceapp.NewService(deviceProducts, deviceDevices, deviceGroups, bus, authService, clock, cfg.Auth.DefaultTokenTTL)
	lookup.devices = deviceService

	telemetryService := telemetryapp.NewService(telemetryapp.Config{
		QueueSize:        cfg.Telemetry.IngestQueueSize,
		WorkerCount:      cfg.Telemetry.WorkerCount,
		MaxBatchSize:     cfg.Telemetry.MaxBatchSize,
		MaxClockSkew:     cfg.Telemetry.MaxClockSkew,
		DefaultRetention: cfg.Telemetry.DefaultRetention,
	}, timeSeriesService, twinService, bus, clock, logger)

	commandService := commandapp.NewService(commandRepo, bus, clock, 5*time.Second)
	ruleEngine := ruleapp.NewEngine(ruleRepo, bus, ruleapp.NewBusActionExecutor(bus), clock)
	firmwareService := firmwareapp.NewService(firmwareRepo, bus, clock)

	deviceHandler := devicehttp.NewHandler(deviceService)
	telemetryHandler := telemetryhttp.NewHandler(telemetryService)
	timeSeriesHandler := timeserieshttp.NewHandler(timeSeriesService)
	twinHandler := twinhttp.NewHandler(twinService)
	commandHandler := commandhttp.NewHandler(commandService)
	ruleHandler := rulehttp.NewHandler(ruleEngine)
	firmwareHandler := firmwarehttp.NewHandler(firmwareService)
	webSocketHandler := telemetryws.NewHandler(
		telemetryService,
		cfg.WebSocket.AllowedOrigin,
		cfg.WebSocket.ReadLimit,
		cfg.WebSocket.WriteWait,
		cfg.WebSocket.PongWait,
		cfg.WebSocket.MaxMessageQueue,
	)

	httpServer := gatewayhttp.NewServer(cfg, gatewayhttp.Dependencies{
		Logger:           logger,
		Device:           deviceHandler,
		Auth:             authService,
		Telemetry:        telemetryHandler,
		WebSocketHandler: webSocketHandler,
		TimeSeries:       timeSeriesHandler,
		Twin:             twinHandler,
		Command:          commandHandler,
		Rule:             ruleHandler,
		Firmware:         firmwareHandler,
		MetricsRegistry:  registry,
		Readiness: func(ctx context.Context) error {
			if deviceDevices == nil {
				return fmt.Errorf("device repository unavailable")
			}
			return nil
		},
	})

	grpcService := gatewaygrpc.NewService(deviceService, telemetryService, commandService, twinService, timeSeriesService)
	grpcServer := gatewaygrpc.NewServer(cfg.GRPC, grpcService)

	telemetryService.Start(rootCtx)
	go commandService.StartReaper(rootCtx)
	ruleEngine.Start(rootCtx)

	mqttAdapter := telemetrymqtt.NewAdapter(cfg.MQTT, telemetryService, logger)
	mqttAdapter.Start(rootCtx)
	defer mqttAdapter.Close()

	errCh := make(chan error, 2)
	go func() {
		if err := httpServer.Start(rootCtx); err != nil {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()
	go func() {
		if err := grpcServer.Start(rootCtx); err != nil {
			errCh <- fmt.Errorf("grpc server: %w", err)
		}
	}()

	logger.Info("iot platform started",
		"service", cfg.Service.Name,
		"environment", cfg.Service.Environment,
		"http", cfg.HTTP.Listen,
		"grpc", cfg.GRPC.Listen,
		"storage", cfg.Storage.Driver,
	)

	select {
	case <-rootCtx.Done():
		logger.Info("shutdown requested")
	case err := <-errCh:
		cancel()
		return err
	}
	<-time.After(cfg.Service.ShutdownTimeout)
	return nil
}
