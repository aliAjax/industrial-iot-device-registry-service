package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authhttp "github.com/example/iot-device-management/internal/auth/adapter/http"
	authapp "github.com/example/iot-device-management/internal/auth/application"
	commandhttp "github.com/example/iot-device-management/internal/command/adapter/http"
	"github.com/example/iot-device-management/internal/config"
	devicehttp "github.com/example/iot-device-management/internal/device/adapter/http"
	firmwarehttp "github.com/example/iot-device-management/internal/firmware/adapter/http"
	"github.com/example/iot-device-management/internal/platform/jsonutil"
	"github.com/example/iot-device-management/internal/platform/trace"
	rulehttp "github.com/example/iot-device-management/internal/rule/adapter/http"
	telemetryhttp "github.com/example/iot-device-management/internal/telemetry/adapter/http"
	timeserieshttp "github.com/example/iot-device-management/internal/timeseries/adapter/http"
	twinhttp "github.com/example/iot-device-management/internal/twin/adapter/http"
	"github.com/prometheus/client_golang/prometheus"
)

type Dependencies struct {
	Logger           *slog.Logger
	Device           *devicehttp.Handler
	Auth             *authapp.Service
	Telemetry        *telemetryhttp.Handler
	WebSocketHandler http.Handler
	TimeSeries       *timeserieshttp.Handler
	Twin             *twinhttp.Handler
	Command          *commandhttp.Handler
	Rule             *rulehttp.Handler
	Firmware         *firmwarehttp.Handler
	MetricsRegistry  *prometheus.Registry
	Readiness        func(context.Context) error
}

type Server struct {
	cfg     config.HTTPConfig
	cfgAll  config.Config
	deps    Dependencies
	metrics *Metrics
	server  *http.Server
}

func NewServer(cfg config.Config, deps Dependencies) *Server {
	return &Server{cfg: cfg.HTTP, cfgAll: cfg, deps: deps}
}

func (s *Server) Handler() http.Handler {
	registry := s.deps.MetricsRegistry
	if registry == nil {
		registry = prometheus.NewRegistry()
	}
	s.metrics = NewMetrics(registry)
	mux := http.NewServeMux()
	s.registerSystemRoutes(mux, registry)
	s.deps.Device.Register(mux)
	s.deps.TimeSeries.Register(mux)
	s.deps.Twin.Register(mux)
	s.deps.Command.Register(mux)
	s.deps.Rule.Register(mux)
	s.deps.Firmware.Register(mux)

	deviceAuth := authhttp.DeviceAuth(s.deps.Auth)
	mux.Handle("POST /api/v1/telemetry/ingest", deviceAuth(http.HandlerFunc(s.deps.Telemetry.ServeHTTP)))
	mux.Handle("POST /api/v1/telemetry/batch", deviceAuth(http.HandlerFunc(s.deps.Telemetry.ServeHTTP)))
	mux.Handle("GET /api/v1/telemetry/ws", deviceAuth(s.deps.WebSocketHandler))

	var handler http.Handler = mux
	handler = contentTypeJSON(handler)
	handler = recoverer(s.deps.Logger, handler)
	handler = requestLogger(s.deps.Logger, handler)
	handler = cors(s.cfg.CORS, handler)
	handler = trace.Middleware(handler)
	handler = s.metrics.Middleware(handler)
	return handler
}

func (s *Server) Start(ctx context.Context) error {
	handler := s.Handler()
	s.server = &http.Server{
		Addr:           s.cfg.Listen,
		Handler:        handler,
		ReadTimeout:    s.cfg.ReadTimeout,
		WriteTimeout:   s.cfg.WriteTimeout,
		IdleTimeout:    s.cfg.IdleTimeout,
		MaxHeaderBytes: s.cfg.MaxHeaderBytes,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfgAll.Service.ShutdownTimeout)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
	}()
	s.deps.Logger.Info("http server listening", "addr", s.cfg.Listen)
	if s.cfg.TLS.Enabled {
		return s.server.ListenAndServeTLS(s.cfg.TLS.CertFile, s.cfg.TLS.KeyFile)
	}
	return s.server.ListenAndServe()
}

func (s *Server) registerSystemRoutes(mux *http.ServeMux, registry *prometheus.Registry) {
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /readyz", s.readyz)
	if s.cfgAll.Metrics.Enabled {
		mux.Handle("GET "+s.cfgAll.Metrics.Path, s.metrics.Handler())
	}
	mux.HandleFunc("GET /api/v1/info", s.info)
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": s.cfgAll.Service.Name})
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	if s.deps.Readiness != nil {
		if err := s.deps.Readiness(r.Context()); err != nil {
			_ = jsonutil.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "reason": err.Error()})
			return
		}
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) info(w http.ResponseWriter, r *http.Request) {
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{
		"service":     s.cfgAll.Service.Name,
		"environment": s.cfgAll.Service.Environment,
		"version":     "0.1.0",
		"time":        time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) WaitForSignal(ctx context.Context) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
	case sig := <-signals:
		s.deps.Logger.Info("signal received", "signal", sig.String())
	}
}

func WriteJSONForTest(w http.ResponseWriter, value any) {
	_ = json.NewEncoder(w).Encode(value)
}

func IsHealthEndpoint(path string) bool {
	return path == "/healthz" || path == "/readyz"
}

func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func ServerAddr(listen string) string {
	if listen == "" {
		return ":8080"
	}
	if listen[0] == ':' {
		return fmt.Sprintf("localhost%s", listen)
	}
	return listen
}
