package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/example/iot-device-management/internal/config"
	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/telemetry/application"
	"github.com/example/iot-device-management/internal/telemetry/domain"
)

type Adapter struct {
	cfg     config.MQTTConfig
	service *application.Service
	logger  *slog.Logger
	client  mqtt.Client
}

func NewAdapter(cfg config.MQTTConfig, service *application.Service, logger *slog.Logger) *Adapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &Adapter{cfg: cfg, service: service, logger: logger}
}

func (a *Adapter) Start(ctx context.Context) {
	if !a.cfg.Enabled {
		a.logger.Info("mqtt adapter disabled")
		return
	}
	opts := mqtt.NewClientOptions().
		AddBroker(a.cfg.Broker).
		SetClientID(a.cfg.ClientID).
		SetUsername(a.cfg.Username).
		SetPassword(a.cfg.Password).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(a.cfg.ReconnectDelay).
		SetOnConnectHandler(a.onConnect).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) {
			a.logger.Warn("mqtt connection lost", "error", err)
		})
	a.client = mqtt.NewClient(opts)
	go func() {
		for {
			token := a.client.Connect()
			if token.WaitTimeout(a.cfg.ConnectTimeout) && token.Error() == nil {
				a.logger.Info("mqtt adapter connected", "broker", a.cfg.Broker)
				<-ctx.Done()
				a.client.Disconnect(250)
				return
			}
			if token.Error() != nil {
				a.logger.Warn("mqtt connect failed", "error", token.Error())
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(a.cfg.ReconnectDelay):
			}
		}
	}()
}

func (a *Adapter) onConnect(client mqtt.Client) {
	for _, topic := range a.cfg.SubscribeTopics {
		if token := client.Subscribe(topic, a.cfg.QoS, a.handle); token.Wait() && token.Error() != nil {
			a.logger.Error("mqtt subscribe failed", "topic", topic, "error", token.Error())
		}
	}
}

func (a *Adapter) handle(_ mqtt.Client, msg mqtt.Message) {
	deviceID, kind, err := parseTopic(msg.Topic())
	if err != nil {
		a.logger.Warn("ignoring mqtt message", "topic", msg.Topic(), "error", err)
		return
	}
	var message domain.Message
	if err := json.Unmarshal(msg.Payload(), &message); err != nil {
		a.logger.Warn("invalid mqtt payload", "topic", msg.Topic(), "error", err)
		return
	}
	message.DeviceID = deviceID
	if message.Kind == "" {
		message.Kind = kind
	}
	if _, err := a.service.Ingest(context.Background(), message); err != nil {
		a.logger.Error("mqtt ingest failed", "device_id", deviceID, "error", err)
	}
}

func (a *Adapter) Close() {
	if a.client != nil && a.client.IsConnected() {
		a.client.Disconnect(250)
	}
}

func parseTopic(topic string) (string, domain.Kind, error) {
	parts := strings.Split(strings.TrimPrefix(topic, "/"), "/")
	if len(parts) < 3 {
		return "", "", apperr.E(apperr.KindInvalid, "mqtt.parseTopic", "invalid topic", nil)
	}
	deviceID := parts[1]
	switch parts[2] {
	case "telemetry":
		return deviceID, domain.KindTelemetry, nil
	case "events":
		return deviceID, domain.KindEvent, nil
	case "attributes":
		return deviceID, domain.KindAttributeChange, nil
	default:
		return "", "", fmt.Errorf("unsupported topic type %q", parts[2])
	}
}
