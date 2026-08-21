package trace

import (
	"context"
	"net/http"

	"github.com/example/iot-device-management/internal/platform/id"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	traceIDKey   contextKey = "trace_id"
)

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	if value == "" {
		return "unknown"
	}
	return value
}

func TraceID(ctx context.Context) string {
	value, _ := ctx.Value(traceIDKey).(string)
	if value == "" {
		return RequestID(ctx)
	}
	return value
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = id.New("req")
		}
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = id.New("trace")
		}
		ctx := WithRequestID(r.Context(), requestID)
		ctx = WithTraceID(ctx, traceID)
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Trace-ID", traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
