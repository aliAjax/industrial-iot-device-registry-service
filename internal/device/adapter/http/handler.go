package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/iot-device-management/internal/device/application"
	"github.com/example/iot-device-management/internal/device/domain"
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
	mux.HandleFunc("GET /api/v1/products", h.listProducts)
	mux.HandleFunc("POST /api/v1/products", h.createProduct)
	mux.HandleFunc("GET /api/v1/products/{id}", h.getProduct)
	mux.HandleFunc("POST /api/v1/groups", h.createGroup)
	mux.HandleFunc("GET /api/v1/groups", h.listGroups)
	mux.HandleFunc("POST /api/v1/devices", h.registerDevice)
	mux.HandleFunc("GET /api/v1/devices", h.listDevices)
	mux.HandleFunc("GET /api/v1/devices/{id}", h.getDevice)
	mux.HandleFunc("POST /api/v1/devices/{id}/enable", h.enableDevice)
	mux.HandleFunc("POST /api/v1/devices/{id}/disable", h.disableDevice)
	mux.HandleFunc("DELETE /api/v1/devices/{id}", h.deregisterDevice)
	mux.HandleFunc("POST /api/v1/devices/{id}/credentials/rotate", h.rotateCredential)
	mux.HandleFunc("PUT /api/v1/devices/{id}/tags", h.updateTags)
	mux.HandleFunc("PUT /api/v1/devices/{id}/groups", h.assignGroups)
	mux.HandleFunc("PUT /api/v1/devices/{id}/location", h.updateLocation)
}

func (h *Handler) createProduct(w http.ResponseWriter, r *http.Request) {
	var input application.CreateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "http.createProduct", "invalid request body", err))
		return
	}
	product, err := h.service.CreateProduct(r.Context(), input)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusCreated, product)
}

func (h *Handler) getProduct(w http.ResponseWriter, r *http.Request) {
	product, err := h.service.GetProduct(r.Context(), r.PathValue("id"))
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, product)
}

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	offset, limit := pagination(r)
	items, total, err := h.service.ListProducts(r.Context(), offset, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, pageResponse(items, total, offset, limit))
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	var input application.CreateGroupInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "http.createGroup", "invalid request body", err))
		return
	}
	group, err := h.service.CreateGroup(r.Context(), input)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusCreated, group)
}

func (h *Handler) listGroups(w http.ResponseWriter, r *http.Request) {
	offset, limit := pagination(r)
	items, total, err := h.service.ListGroups(r.Context(), offset, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, pageResponse(items, total, offset, limit))
}

func (h *Handler) registerDevice(w http.ResponseWriter, r *http.Request) {
	var input application.RegisterDeviceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "http.registerDevice", "invalid request body", err))
		return
	}
	device, err := h.service.RegisterDevice(r.Context(), input)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusCreated, device)
}

func (h *Handler) getDevice(w http.ResponseWriter, r *http.Request) {
	device, err := h.service.GetDevice(r.Context(), r.PathValue("id"))
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, device)
}

func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request) {
	filter := domain.DeviceFilter{
		ProductID: r.URL.Query().Get("product_id"),
		Status:    domain.DeviceStatus(r.URL.Query().Get("status")),
		GroupID:   r.URL.Query().Get("group_id"),
		Query:     r.URL.Query().Get("q"),
	}
	if tags := r.URL.Query().Get("tags"); tags != "" {
		filter.Tags = map[string]string{}
		for _, pair := range strings.Split(tags, ",") {
			parts := strings.SplitN(pair, ":", 2)
			if len(parts) == 2 {
				filter.Tags[parts[0]] = parts[1]
			}
		}
	}
	offset, limit := pagination(r)
	items, total, err := h.service.ListDevices(r.Context(), filter, offset, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, pageResponse(items, total, offset, limit))
}

func (h *Handler) enableDevice(w http.ResponseWriter, r *http.Request) {
	device, err := h.service.EnableDevice(r.Context(), r.PathValue("id"))
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, device)
}

func (h *Handler) disableDevice(w http.ResponseWriter, r *http.Request) {
	device, err := h.service.DisableDevice(r.Context(), r.PathValue("id"))
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, device)
}

func (h *Handler) deregisterDevice(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeregisterDevice(r.Context(), r.PathValue("id")); err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) rotateCredential(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type domain.CredentialType `json:"type"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	credential, err := h.service.RotateCredential(r.Context(), r.PathValue("id"), body.Type)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusCreated, credential)
}

func (h *Handler) updateTags(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Tags map[string]string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "http.updateTags", "invalid request body", err))
		return
	}
	device, err := h.service.UpdateTags(r.Context(), r.PathValue("id"), body.Tags)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, device)
}

func (h *Handler) assignGroups(w http.ResponseWriter, r *http.Request) {
	var body struct {
		GroupIDs []string `json:"group_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "http.assignGroups", "invalid request body", err))
		return
	}
	device, err := h.service.AssignGroups(r.Context(), r.PathValue("id"), body.GroupIDs)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, device)
}

func (h *Handler) updateLocation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Location *domain.GeoLocation `json:"location"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "http.updateLocation", "invalid request body", err))
		return
	}
	device, err := h.service.UpdateLocation(r.Context(), r.PathValue("id"), body.Location)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, device)
}

func pagination(r *http.Request) (int, int) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return offset, limit
}

func pageResponse(items any, total, offset, limit int) map[string]any {
	return map[string]any{
		"items":  items,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	}
}

func respondError(w http.ResponseWriter, err error) {
	_ = jsonutil.Error(w, apperr.StatusCode(err), string(apperr.KindOf(err)), err.Error())
}
