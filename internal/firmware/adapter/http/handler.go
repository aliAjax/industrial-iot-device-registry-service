package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/example/iot-device-management/internal/firmware/application"
	"github.com/example/iot-device-management/internal/firmware/domain"
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
	mux.HandleFunc("POST /api/v1/firmware", h.createFirmware)
	mux.HandleFunc("GET /api/v1/firmware", h.listFirmware)
	mux.HandleFunc("POST /api/v1/firmware/tasks", h.createTask)
	mux.HandleFunc("GET /api/v1/firmware/tasks", h.listTasks)
	mux.HandleFunc("POST /api/v1/firmware/tasks/{id}/transition", h.transitionTask)
	mux.HandleFunc("POST /api/v1/firmware/receipts", h.receive)
	mux.HandleFunc("POST /api/v1/firmware/receipts/{id}/complete", h.complete)
	mux.HandleFunc("GET /api/v1/firmware/devices/{id}/receipts", h.deviceReceipts)
}

func (h *Handler) createFirmware(w http.ResponseWriter, r *http.Request) {
	var firmware domain.Firmware
	if err := json.NewDecoder(r.Body).Decode(&firmware); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "firmware.createFirmware", "invalid request body", err))
		return
	}
	created, err := h.service.CreateFirmware(r.Context(), firmware)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusCreated, created)
}

func (h *Handler) listFirmware(w http.ResponseWriter, r *http.Request) {
	offset, limit := pagination(r)
	items, total, err := h.service.ListFirmware(r.Context(), r.URL.Query().Get("product_id"), offset, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "offset": offset, "limit": limit})
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	var input domain.TaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "firmware.createTask", "invalid request body", err))
		return
	}
	if err := h.service.ValidateRollout(input.Rollout); err != nil {
		respondError(w, err)
		return
	}
	task, err := h.service.CreateTask(r.Context(), input)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusCreated, task)
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	filter := domain.Filter{
		FirmwareID: r.URL.Query().Get("firmware_id"),
		Status:     domain.Status(r.URL.Query().Get("status")),
		DeviceID:   r.URL.Query().Get("device_id"),
	}
	offset, limit := pagination(r)
	items, total, err := h.service.ListTasks(r.Context(), filter, offset, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "offset": offset, "limit": limit})
}

func (h *Handler) transitionTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status domain.Status `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "firmware.transitionTask", "invalid request body", err))
		return
	}
	task, err := h.service.TransitionTask(r.Context(), r.PathValue("id"), body.Status)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, task)
}

func (h *Handler) receive(w http.ResponseWriter, r *http.Request) {
	var input domain.ReceiptInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "firmware.receive", "invalid request body", err))
		return
	}
	receipt, err := h.service.Receive(r.Context(), input)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusCreated, receipt)
}

func (h *Handler) complete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "firmware.complete", "invalid request body", err))
		return
	}
	receipt, err := h.service.CompleteReceipt(r.Context(), r.PathValue("id"), body.Success, body.Message)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, receipt)
}

func (h *Handler) deviceReceipts(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.ReceiptsForDevice(r.Context(), r.PathValue("id"), limit)
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
