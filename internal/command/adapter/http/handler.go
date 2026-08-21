package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/example/iot-device-management/internal/command/application"
	"github.com/example/iot-device-management/internal/command/domain"
	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/platform/jsonutil"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/commands", h.enqueue)
	mux.HandleFunc("GET /api/v1/commands", h.list)
	mux.HandleFunc("GET /api/v1/commands/{id}", h.get)
	mux.HandleFunc("POST /api/v1/commands/ack", h.ack)
	mux.HandleFunc("GET /api/v1/devices/{id}/commands/pending", h.pending)
}

func (h *Handler) enqueue(w http.ResponseWriter, r *http.Request) {
	var input domain.EnqueueInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "command.enqueue", "invalid request body", err))
		return
	}
	command, err := h.service.Enqueue(r.Context(), input)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusCreated, command)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	command, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, command)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	filter := domain.CommandFilter{
		DeviceID: r.URL.Query().Get("device_id"),
		Status:   domain.Status(r.URL.Query().Get("status")),
		Query:    r.URL.Query().Get("q"),
	}
	offset, limit := pagination(r)
	items, total, err := h.service.List(r.Context(), filter, offset, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "offset": offset, "limit": limit})
}

func (h *Handler) ack(w http.ResponseWriter, r *http.Request) {
	var input domain.AckInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "command.ack", "invalid request body", err))
		return
	}
	command, err := h.service.Ack(r.Context(), input)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, command)
}

func (h *Handler) pending(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.PendingForDevice(r.Context(), r.PathValue("id"), limit)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
}

func pagination(r *http.Request) (int, int) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return offset, limit
}

func respondError(w http.ResponseWriter, err error) {
	_ = jsonutil.Error(w, apperr.StatusCode(err), string(apperr.KindOf(err)), err.Error())
}
