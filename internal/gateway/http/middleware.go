package http

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/example/iot-device-management/internal/config"
	"github.com/example/iot-device-management/internal/platform/jsonutil"
	"github.com/example/iot-device-management/internal/platform/trace"
)

func recoverer(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered", "panic", recovered, "stack", string(debug.Stack()), "request_id", trace.RequestID(r.Context()))
				_ = jsonutil.Error(w, http.StatusInternalServerError, "internal", "unexpected server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"request_id", trace.RequestID(r.Context()),
			"trace_id", trace.TraceID(r.Context()),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func cors(cfg config.CORSConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowsOrigin(cfg.AllowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowsOrigin(allowed []string, origin string) bool {
	for _, candidate := range allowed {
		if candidate == "*" || candidate == origin {
			return true
		}
	}
	return false
}

func contentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			contentType := r.Header.Get("Content-Type")
			if contentType != "" && !strings.Contains(contentType, "application/json") {
				_ = jsonutil.Error(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
