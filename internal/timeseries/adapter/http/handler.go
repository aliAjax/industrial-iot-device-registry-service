package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/platform/jsonutil"
	"github.com/example/iot-device-management/internal/timeseries/application"
	"github.com/example/iot-device-management/internal/timeseries/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/timeseries", h.query)
	mux.HandleFunc("POST /api/v1/timeseries/batch", h.writeBatch)
	mux.HandleFunc("GET /api/v1/timeseries/aggregate", h.aggregate)
	mux.HandleFunc("GET /api/v1/timeseries/gaps", h.gaps)
}

func (h *Handler) query(w http.ResponseWriter, r *http.Request) {
	query := domain.Query{
		DeviceID:   r.URL.Query().Get("device_id"),
		Property:   r.URL.Query().Get("property"),
		Start:      parseTime(r.URL.Query().Get("start")),
		End:        parseTime(r.URL.Query().Get("end")),
		Limit:      intQuery(r, "limit", 1000),
		Descending: r.URL.Query().Get("order") == "desc",
	}
	points, err := h.service.Query(r.Context(), query)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{"items": points, "count": len(points)})
}

func (h *Handler) writeBatch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Points []domain.Point `json:"points"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, apperr.E(apperr.KindInvalid, "timeseries.writeBatch", "invalid request body", err))
		return
	}
	if err := h.service.WriteBatch(r.Context(), body.Points); err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusCreated, map[string]int{"accepted": len(body.Points)})
}

func (h *Handler) aggregate(w http.ResponseWriter, r *http.Request) {
	bucket, _ := time.ParseDuration(r.URL.Query().Get("bucket"))
	request := domain.AggregateRequest{
		DeviceID: r.URL.Query().Get("device_id"),
		Property: r.URL.Query().Get("property"),
		Start:    parseTime(r.URL.Query().Get("start")),
		End:      parseTime(r.URL.Query().Get("end")),
		Kind:     domain.AggregateKind(r.URL.Query().Get("kind")),
		Bucket:   bucket,
	}
	results, err := h.service.Aggregate(r.Context(), request)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{"items": results, "count": len(results)})
}

func (h *Handler) gaps(w http.ResponseWriter, r *http.Request) {
	interval, _ := time.ParseDuration(r.URL.Query().Get("expected_interval"))
	request := domain.GapRequest{
		DeviceID:         r.URL.Query().Get("device_id"),
		Property:         r.URL.Query().Get("property"),
		Start:            parseTime(r.URL.Query().Get("start")),
		End:              parseTime(r.URL.Query().Get("end")),
		ExpectedInterval: interval,
	}
	gaps, err := h.service.DetectGaps(r.Context(), request)
	if err != nil {
		respondError(w, err)
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{"items": gaps, "count": len(gaps)})
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(unix, 0)
	}
	return time.Time{}
}

func intQuery(r *http.Request, key string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return fallback
	}
	return value
}

func respondError(w http.ResponseWriter, err error) {
	_ = jsonutil.Error(w, apperr.StatusCode(err), string(apperr.KindOf(err)), err.Error())
}
