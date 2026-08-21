package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/example/iot-device-management/internal/auth/application"
	authdomain "github.com/example/iot-device-management/internal/auth/domain"
	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/platform/jsonutil"
)

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal authdomain.Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func Principal(ctx context.Context) (authdomain.Principal, bool) {
	value, ok := ctx.Value(principalContextKey{}).(authdomain.Principal)
	return value, ok
}

func DeviceAuth(service *application.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			deviceID := strings.TrimSpace(r.Header.Get("X-Device-ID"))
			credential := strings.TrimSpace(r.Header.Get("X-Device-Credential"))
			if deviceID == "" || credential == "" {
				_ = jsonutil.Error(w, http.StatusUnauthorized, string(apperr.KindUnauthorized), "device authentication headers are required")
				return
			}
			scope := deviceScopeForPath(deviceID, r.URL.Path)
			principal, err := service.AuthenticateAndAuthorize(r.Context(), deviceID, credential, scope, "")
			if err != nil {
				_ = jsonutil.Error(w, apperr.StatusCode(err), string(apperr.KindOf(err)), err.Error())
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
}

func deviceScopeForPath(deviceID, path string) string {
	switch {
	case strings.Contains(path, "/telemetry"):
		return "device:" + deviceID + ":telemetry"
	case strings.Contains(path, "/commands"):
		return "device:" + deviceID + ":commands"
	case strings.Contains(path, "/attributes") || strings.Contains(path, "/twin"):
		return "device:" + deviceID + ":attributes"
	default:
		return "device:" + deviceID + ":telemetry"
	}
}
