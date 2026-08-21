package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root runtime configuration. Values may come from a YAML file
// and are subsequently overridden by environment variables using the IOT_
// prefix. The split between HTTP and gRPC listeners keeps telemetry traffic
// isolated from management traffic in horizontally scaled deployments.
type Config struct {
	Service   ServiceConfig   `yaml:"service"`
	HTTP      HTTPConfig      `yaml:"http"`
	GRPC      GRPCConfig      `yaml:"grpc"`
	WebSocket WebSocketConfig `yaml:"websocket"`
	MQTT      MQTTConfig      `yaml:"mqtt"`
	Storage   StorageConfig   `yaml:"storage"`
	Telemetry TelemetryConfig `yaml:"telemetry"`
	Auth      AuthConfig      `yaml:"auth"`
	Metrics   MetricsConfig   `yaml:"metrics"`
	Logging   LoggingConfig   `yaml:"logging"`
}

type ServiceConfig struct {
	Name            string        `yaml:"name"`
	Environment     string        `yaml:"environment"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type HTTPConfig struct {
	Listen         string        `yaml:"listen"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	IdleTimeout    time.Duration `yaml:"idle_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes"`
	TLS            TLSConfig     `yaml:"tls"`
	CORS           CORSConfig    `yaml:"cors"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
}

type GRPCConfig struct {
	Listen            string        `yaml:"listen"`
	MaxRecvMsgSize    int           `yaml:"max_recv_msg_size"`
	MaxSendMsgSize    int           `yaml:"max_send_msg_size"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
}

type WebSocketConfig struct {
	Path            string        `yaml:"path"`
	ReadLimit       int64         `yaml:"read_limit"`
	WriteWait       time.Duration `yaml:"write_wait"`
	PongWait        time.Duration `yaml:"pong_wait"`
	MaxMessageQueue int           `yaml:"max_message_queue"`
	AllowedOrigin   string        `yaml:"allowed_origin"`
}

type MQTTConfig struct {
	Enabled         bool          `yaml:"enabled"`
	Broker          string        `yaml:"broker"`
	ClientID        string        `yaml:"client_id"`
	Username        string        `yaml:"username"`
	Password        string        `yaml:"password"`
	SubscribeTopics []string      `yaml:"subscribe_topics"`
	QoS             byte          `yaml:"qos"`
	ConnectTimeout  time.Duration `yaml:"connect_timeout"`
	ReconnectDelay  time.Duration `yaml:"reconnect_delay"`
}

type StorageConfig struct {
	Driver          string        `yaml:"driver"`
	PostgresDSN     string        `yaml:"postgres_dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	MigrationDir    string        `yaml:"migration_dir"`
}

type TelemetryConfig struct {
	IngestQueueSize   int           `yaml:"ingest_queue_size"`
	WorkerCount       int           `yaml:"worker_count"`
	MaxBatchSize      int           `yaml:"max_batch_size"`
	MaxClockSkew      time.Duration `yaml:"max_clock_skew"`
	DefaultRetention  time.Duration `yaml:"default_retention"`
	DownsampleBuckets []string      `yaml:"downsample_buckets"`
}

type AuthConfig struct {
	DefaultTokenTTL     time.Duration `yaml:"default_token_ttl"`
	MaxCredentialLength int           `yaml:"max_credential_length"`
	DefaultRateLimit    int           `yaml:"default_rate_limit"`
	DefaultRateWindow   time.Duration `yaml:"default_rate_window"`
	JWTSecret           string        `yaml:"jwt_secret"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		path = "configs/config.yaml"
	}
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return cfg, fmt.Errorf("read config %s: %w", path, err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse config %s: %w", path, err)
		}
	}
	applyEnvironment(&cfg)
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func Default() Config {
	return Config{
		Service: ServiceConfig{Name: "iot-device-management", Environment: "development", ShutdownTimeout: 10 * time.Second},
		HTTP: HTTPConfig{
			Listen:         ":8080",
			ReadTimeout:    15 * time.Second,
			WriteTimeout:   30 * time.Second,
			IdleTimeout:    60 * time.Second,
			MaxHeaderBytes: 1 << 20,
			CORS:           CORSConfig{AllowedOrigins: []string{"*"}, AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}, AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Device-ID", "X-Device-Credential", "X-Request-ID"}},
		},
		GRPC:      GRPCConfig{Listen: ":9090", MaxRecvMsgSize: 4 << 20, MaxSendMsgSize: 4 << 20, ConnectionTimeout: 10 * time.Second},
		WebSocket: WebSocketConfig{Path: "/api/v1/telemetry/ws", ReadLimit: 1 << 20, WriteWait: 10 * time.Second, PongWait: 60 * time.Second, MaxMessageQueue: 1024, AllowedOrigin: "*"},
		MQTT:      MQTTConfig{Enabled: false, Broker: "tcp://localhost:1883", ClientID: "iot-device-management", SubscribeTopics: []string{"devices/+/telemetry", "devices/+/events", "devices/+/attributes"}, QoS: 1, ConnectTimeout: 5 * time.Second, ReconnectDelay: 3 * time.Second},
		Storage:   StorageConfig{Driver: "memory", PostgresDSN: "postgres://iot:iot@localhost:5432/iot?sslmode=disable", MaxOpenConns: 20, MaxIdleConns: 5, ConnMaxLifetime: 30 * time.Minute, MigrationDir: "migrations"},
		Telemetry: TelemetryConfig{IngestQueueSize: 4096, WorkerCount: 4, MaxBatchSize: 512, MaxClockSkew: 5 * time.Minute, DefaultRetention: 24 * time.Hour, DownsampleBuckets: []string{"1m", "5m", "1h"}},
		Auth:      AuthConfig{DefaultTokenTTL: 24 * time.Hour, MaxCredentialLength: 4096, DefaultRateLimit: 120, DefaultRateWindow: time.Minute, JWTSecret: "development-only-secret"},
		Metrics:   MetricsConfig{Enabled: true, Path: "/metrics"},
		Logging:   LoggingConfig{Level: "info", Format: "json"},
	}
}

func applyEnvironment(cfg *Config) {
	setString := func(name string, target *string) {
		if value, ok := os.LookupEnv("IOT_" + name); ok {
			*target = value
		}
	}
	setBool := func(name string, target *bool) {
		if value, ok := os.LookupEnv("IOT_" + name); ok {
			if parsed, err := strconv.ParseBool(value); err == nil {
				*target = parsed
			}
		}
	}
	setInt := func(name string, target *int) {
		if value, ok := os.LookupEnv("IOT_" + name); ok {
			if parsed, err := strconv.Atoi(value); err == nil {
				*target = parsed
			}
		}
	}
	setDuration := func(name string, target *time.Duration) {
		if value, ok := os.LookupEnv("IOT_" + name); ok {
			if parsed, err := time.ParseDuration(value); err == nil {
				*target = parsed
			}
		}
	}

	setString("SERVICE_NAME", &cfg.Service.Name)
	setString("SERVICE_ENVIRONMENT", &cfg.Service.Environment)
	setDuration("SERVICE_SHUTDOWN_TIMEOUT", &cfg.Service.ShutdownTimeout)
	setString("HTTP_LISTEN", &cfg.HTTP.Listen)
	setDuration("HTTP_READ_TIMEOUT", &cfg.HTTP.ReadTimeout)
	setDuration("HTTP_WRITE_TIMEOUT", &cfg.HTTP.WriteTimeout)
	setDuration("HTTP_IDLE_TIMEOUT", &cfg.HTTP.IdleTimeout)
	setInt("HTTP_MAX_HEADER_BYTES", &cfg.HTTP.MaxHeaderBytes)
	setBool("HTTP_TLS_ENABLED", &cfg.HTTP.TLS.Enabled)
	setString("HTTP_TLS_CERT_FILE", &cfg.HTTP.TLS.CertFile)
	setString("HTTP_TLS_KEY_FILE", &cfg.HTTP.TLS.KeyFile)
	setString("GRPC_LISTEN", &cfg.GRPC.Listen)
	setInt("GRPC_MAX_RECV_MSG_SIZE", &cfg.GRPC.MaxRecvMsgSize)
	setInt("GRPC_MAX_SEND_MSG_SIZE", &cfg.GRPC.MaxSendMsgSize)
	setDuration("GRPC_CONNECTION_TIMEOUT", &cfg.GRPC.ConnectionTimeout)
	setString("WEBSOCKET_PATH", &cfg.WebSocket.Path)
	setInt64 := func(name string, target *int64) {
		if value, ok := os.LookupEnv("IOT_" + name); ok {
			if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
				*target = parsed
			}
		}
	}
	setInt64("WEBSOCKET_READ_LIMIT", &cfg.WebSocket.ReadLimit)
	setDuration("WEBSOCKET_WRITE_WAIT", &cfg.WebSocket.WriteWait)
	setDuration("WEBSOCKET_PONG_WAIT", &cfg.WebSocket.PongWait)
	setInt("WEBSOCKET_MAX_MESSAGE_QUEUE", &cfg.WebSocket.MaxMessageQueue)
	setString("WEBSOCKET_ALLOWED_ORIGIN", &cfg.WebSocket.AllowedOrigin)
	setBool("MQTT_ENABLED", &cfg.MQTT.Enabled)
	setString("MQTT_BROKER", &cfg.MQTT.Broker)
	setString("MQTT_CLIENT_ID", &cfg.MQTT.ClientID)
	setString("MQTT_USERNAME", &cfg.MQTT.Username)
	setString("MQTT_PASSWORD", &cfg.MQTT.Password)
	if value, ok := os.LookupEnv("IOT_MQTT_SUBSCRIBE_TOPICS"); ok {
		cfg.MQTT.SubscribeTopics = strings.Split(value, ",")
	}
	if value, ok := os.LookupEnv("IOT_MQTT_QOS"); ok {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 && parsed <= 2 {
			cfg.MQTT.QoS = byte(parsed)
		}
	}
	setDuration("MQTT_CONNECT_TIMEOUT", &cfg.MQTT.ConnectTimeout)
	setDuration("MQTT_RECONNECT_DELAY", &cfg.MQTT.ReconnectDelay)
	setString("STORAGE_DRIVER", &cfg.Storage.Driver)
	setString("STORAGE_POSTGRES_DSN", &cfg.Storage.PostgresDSN)
	setInt("STORAGE_MAX_OPEN_CONNS", &cfg.Storage.MaxOpenConns)
	setInt("STORAGE_MAX_IDLE_CONNS", &cfg.Storage.MaxIdleConns)
	setDuration("STORAGE_CONN_MAX_LIFETIME", &cfg.Storage.ConnMaxLifetime)
	setString("STORAGE_MIGRATION_DIR", &cfg.Storage.MigrationDir)
	setInt("TELEMETRY_INGEST_QUEUE_SIZE", &cfg.Telemetry.IngestQueueSize)
	setInt("TELEMETRY_WORKER_COUNT", &cfg.Telemetry.WorkerCount)
	setInt("TELEMETRY_MAX_BATCH_SIZE", &cfg.Telemetry.MaxBatchSize)
	setDuration("TELEMETRY_MAX_CLOCK_SKEW", &cfg.Telemetry.MaxClockSkew)
	setDuration("TELEMETRY_DEFAULT_RETENTION", &cfg.Telemetry.DefaultRetention)
	if value, ok := os.LookupEnv("IOT_TELEMETRY_DOWNSAMPLE_BUCKETS"); ok {
		cfg.Telemetry.DownsampleBuckets = strings.Split(value, ",")
	}
	setDuration("AUTH_DEFAULT_TOKEN_TTL", &cfg.Auth.DefaultTokenTTL)
	setInt("AUTH_MAX_CREDENTIAL_LENGTH", &cfg.Auth.MaxCredentialLength)
	setInt("AUTH_DEFAULT_RATE_LIMIT", &cfg.Auth.DefaultRateLimit)
	setDuration("AUTH_DEFAULT_RATE_WINDOW", &cfg.Auth.DefaultRateWindow)
	setString("AUTH_JWT_SECRET", &cfg.Auth.JWTSecret)
	setBool("METRICS_ENABLED", &cfg.Metrics.Enabled)
	setString("METRICS_PATH", &cfg.Metrics.Path)
	setString("LOGGING_LEVEL", &cfg.Logging.Level)
	setString("LOGGING_FORMAT", &cfg.Logging.Format)
}

func (c Config) Validate() error {
	if c.Service.Name == "" {
		return fmt.Errorf("service.name is required")
	}
	if c.HTTP.Listen == "" {
		return fmt.Errorf("http.listen is required")
	}
	if c.GRPC.Listen == "" {
		return fmt.Errorf("grpc.listen is required")
	}
	if c.Storage.Driver != "memory" && c.Storage.Driver != "postgres" {
		return fmt.Errorf("storage.driver must be memory or postgres")
	}
	if c.Telemetry.WorkerCount <= 0 {
		return fmt.Errorf("telemetry.worker_count must be positive")
	}
	if c.Telemetry.IngestQueueSize <= 0 {
		return fmt.Errorf("telemetry.ingest_queue_size must be positive")
	}
	if c.Auth.DefaultRateWindow <= 0 {
		return fmt.Errorf("auth.default_rate_window must be positive")
	}
	return nil
}
