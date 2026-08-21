package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/platform/jsonutil"
	"github.com/example/iot-device-management/internal/rule/application"
	"github.com/example/iot-device-management/internal/rule/domain"
)

type Handler struct {
	engine *application.Engine
}

func NewHandler(engine *application.Engine) *Handler {
	return &Handler{engine: engine}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/rules", h.create)
	mux.HandleFunc("GET /api/v1/rules", h.list)
	mux.HandleFunc("PATCH /api/v1/rules/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/rules/{id}", h.delete)
	mux.HandleFunc("GET /api/v1/rules/executions", h.executions)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var rule domain.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "rule.create", "invalid request body", err))
		return
	}
	created, err := h.engine.CreateRule(r.Context(), rule)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusCreated, created)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	offset, limit := pagination(r)
	items, total, err := h.engine.ListRules(r.Context(), r.URL.Query().Get("device_id"), offset, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "offset": offset, "limit": limit})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var patch application.RulePatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "rule.update", "invalid request body", err))
		return
	}
	rule, err := h.engine.UpdateRule(r.Context(), r.PathValue("id"), patch)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, rule)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.engine.DeleteRule(r.Context(), r.PathValue("id")); err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) executions(w http.ResponseWriter, r *http.Request) {
	filter := domain.ExecutionFilter{
		RuleID:   r.URL.Query().Get("rule_id"),
		DeviceID: r.URL.Query().Get("device_id"),
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	filter.Offset = offset
	filter.Limit = limit
	items, total, err := h.engine.ListExecutions(r.Context(), filter)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "offset": offset, "limit": limit})
}

func (h *Handler) DebugEvaluate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DeviceID  string    `json:"device_id"`
		Property  string    `json:"property"`
		Value     float64   `json:"value"`
		Timestamp time.Time `json:"timestamp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "rule.DebugEvaluate", "invalid request body", err))
		return
	}
	point := struct {
		DeviceID  string    `json:"device_id"`
		Property  string    `json:"property"`
		Value     float64   `json:"value"`
		Timestamp time.Time `json:"timestamp"`
	}{body.DeviceID, body.Property, body.Value, body.Timestamp}
	_ = point
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]bool{"accepted": true})
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
