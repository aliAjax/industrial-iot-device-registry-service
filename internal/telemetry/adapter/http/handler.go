package http

import (
	"encoding/json"
	"net/http"

	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/platform/jsonutil"
	"github.com/example/iot-device-management/internal/telemetry/application"
	"github.com/example/iot-device-management/internal/telemetry/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/telemetry/ingest", h.ingest)
	mux.HandleFunc("POST /api/v1/telemetry/batch", h.batch)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v1/telemetry/ingest":
		h.ingest(w, r)
	case "/api/v1/telemetry/batch":
		h.batch(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) ingest(w http.ResponseWriter, r *http.Request) {
	var message domain.Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "telemetry.ingest", "invalid request body", err))
		return
	}
	if message.DeviceID == "" {
		message.DeviceID = r.Header.Get("X-Device-ID")
	}
	result, err := h.service.Ingest(r.Context(), message)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusAccepted, result)
}

func (h *Handler) batch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Messages []domain.Message `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "telemetry.batch", "invalid request body", err))
		return
	}
	deviceID := r.Header.Get("X-Device-ID")
	for i := range body.Messages {
		if body.Messages[i].DeviceID == "" {
			body.Messages[i].DeviceID = deviceID
		}
	}
	results, err := h.service.IngestBatch(r.Context(), body.Messages)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusAccepted, map[string]any{"items": results, "count": len(results)})
}

func respondError(w http.ResponseWriter, err error) {
	_ = jsonutil.Error(w, apperr.StatusCode(err), string(apperr.KindOf(err)), err.Error())
}
