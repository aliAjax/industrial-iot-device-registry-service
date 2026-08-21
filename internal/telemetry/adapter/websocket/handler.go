package websocket

import (
	"encoding/json"
	"net/http"
	"time"

	authhttp "github.com/example/iot-device-management/internal/auth/adapter/http"
	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/telemetry/application"
	"github.com/example/iot-device-management/internal/telemetry/domain"
	"github.com/gorilla/websocket"
)

type Handler struct {
	service   *application.Service
	upgrader  websocket.Upgrader
	readLimit int64
	writeWait time.Duration
	pongWait  time.Duration
	queueSize int
}

func NewHandler(service *application.Service, allowedOrigin string, readLimit int64, writeWait, pongWait time.Duration, queueSize int) *Handler {
	if readLimit <= 0 {
		readLimit = 1 << 20
	}
	if writeWait <= 0 {
		writeWait = 10 * time.Second
	}
	if pongWait <= 0 {
		pongWait = 60 * time.Second
	}
	if queueSize <= 0 {
		queueSize = 1024
	}
	return &Handler{
		service: service,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return allowedOrigin == "*" || allowedOrigin == "" || r.Header.Get("Origin") == allowedOrigin
			},
		},
		readLimit: readLimit,
		writeWait: writeWait,
		pongWait:  pongWait,
		queueSize: queueSize,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	principal, ok := authhttp.Principal(r.Context())
	if !ok {
		http.Error(w, "device authentication required", http.StatusUnauthorized)
		return
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(h.readLimit)
	_ = conn.SetReadDeadline(time.Now().Add(h.pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(h.pongWait))
	})
	send := make(chan []byte, h.queueSize)
	go h.writeLoop(conn, send)
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			continue
		}
		var message domain.Message
		if err := json.Unmarshal(payload, &message); err != nil {
			_ = conn.WriteJSON(map[string]any{"error": "invalid message", "message": err.Error()})
			continue
		}
		message.DeviceID = principal.DeviceID
		result, ingestErr := h.service.Ingest(r.Context(), message)
		if ingestErr != nil {
			send <- mustJSON(map[string]any{"error": string(apperr.KindOf(ingestErr)), "message": ingestErr.Error()})
			continue
		}
		send <- mustJSON(result)
	}
	close(send)
}

func (h *Handler) writeLoop(conn *websocket.Conn, send <-chan []byte) {
	ticker := time.NewTicker(h.writeWait * 9 / 10)
	defer ticker.Stop()
	for {
		select {
		case payload, ok := <-send:
			_ = conn.SetWriteDeadline(time.Now().Add(h.writeWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(h.writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func mustJSON(value any) []byte {
	payload, _ := json.Marshal(value)
	return payload
}
