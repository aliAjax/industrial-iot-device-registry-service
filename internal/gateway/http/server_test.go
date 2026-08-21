package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadyzHonorsCanceledRequestContext(t *testing.T) {
	s := &Server{
		deps: Dependencies{
			Readiness: func(ctx context.Context) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				return nil
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	s.readyz(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for canceled context, got %d", rec.Code)
	}
}
