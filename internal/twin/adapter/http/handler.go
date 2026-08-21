package http

import (
	"encoding/json"
	"net/http"

	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/platform/jsonutil"
	"github.com/example/iot-device-management/internal/twin/application"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/twins/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/twins/{id}/desired", h.updateDesired)
	mux.HandleFunc("PUT /api/v1/twins/{id}/reported", h.updateReported)
	mux.HandleFunc("GET /api/v1/twins/{id}/merge", h.merge)
	mux.HandleFunc("DELETE /api/v1/twins/{id}", h.delete)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	document, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, document)
}

func (h *Handler) updateDesired(w http.ResponseWriter, r *http.Request) {
	patch := map[string]any{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "twin.updateDesired", "invalid request body", err))
		return
	}
	if err := h.service.ValidatePatch(patch, 10); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "twin.updateDesired", err.Error(), nil))
		return
	}
	result, err := h.service.UpdateDesired(r.Context(), r.PathValue("id"), patch)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) updateReported(w http.ResponseWriter, r *http.Request) {
	patch := map[string]any{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "twin.updateReported", "invalid request body", err))
		return
	}
	if err := h.service.ValidatePatch(patch, 10); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "twin.updateReported", err.Error(), nil))
		return
	}
	result, err := h.service.UpdateReported(r.Context(), r.PathValue("id"), patch)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) merge(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Merge(r.Context(), r.PathValue("id"))
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("id")); err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func respondError(w http.ResponseWriter, err error) {
	_ = jsonutil.Error(w, apperr.StatusCode(err), string(apperr.KindOf(err)), err.Error())
}
