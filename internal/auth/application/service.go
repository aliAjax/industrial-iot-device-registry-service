package application

import (
	"context"
	"crypto/subtle"
	"strings"
	"time"

	"github.com/example/iot-device-management/internal/auth/domain"
	"github.com/example/iot-device-management/internal/platform/apperr"
	clockpkg "github.com/example/iot-device-management/internal/platform/clock"
	"github.com/example/iot-device-management/internal/platform/id"
)

type Service struct {
	repo       domain.Repository
	lookup     domain.DeviceLookup
	clock      clockpkg.Clock
	tokenTTL   time.Duration
	rateLimit  int
	rateWindow time.Duration
}

func NewService(repo domain.Repository, lookup domain.DeviceLookup, clock clockpkg.Clock, tokenTTL time.Duration, rateLimit int, rateWindow time.Duration) *Service {
	if clock == nil {
		clock = clockpkg.SystemClock{}
	}
	if tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour
	}
	if rateLimit <= 0 {
		rateLimit = 120
	}
	if rateWindow <= 0 {
		rateWindow = time.Minute
	}
	return &Service{repo: repo, lookup: lookup, clock: clock, tokenTTL: tokenTTL, rateLimit: rateLimit, rateWindow: rateWindow}
}

func (s *Service) RegisterDevice(ctx context.Context, deviceID, productID, credentialType, credential string, scopes []string) error {
	if deviceID == "" || credential == "" {
		return apperr.E(apperr.KindInvalid, "auth.RegisterDevice", "device id and credential are required", nil)
	}
	now := s.clock.Now()
	principal := domain.Principal{
		DeviceID:       deviceID,
		ProductID:      productID,
		CredentialType: domain.CredentialType(credentialType),
		Credential:     credential,
		Scopes:         append([]string(nil), scopes...),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.UpsertPrincipal(ctx, principal); err != nil {
		return apperr.E(apperr.KindInternal, "auth.RegisterDevice", "register principal", err)
	}
	return nil
}

func (s *Service) RevokeDevice(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return apperr.E(apperr.KindInvalid, "auth.RevokeDevice", "device id is required", nil)
	}
	return s.repo.DeletePrincipal(ctx, deviceID)
}

func (s *Service) Authenticate(ctx context.Context, deviceID, credential string, credentialType domain.CredentialType) (domain.Principal, error) {
	record, err := s.repo.IncrementRate(ctx, "auth:"+deviceID, s.rateWindow, s.rateLimit)
	if err != nil {
		return domain.Principal{}, apperr.E(apperr.KindUnavailable, "auth.Authenticate", "rate limit storage unavailable", err)
	}
	if record.Count > record.Limit {
		return domain.Principal{}, apperr.E(apperr.KindRateLimited, "auth.Authenticate", "authentication rate limit exceeded", nil)
	}
	principal, err := s.repo.FindPrincipal(ctx, deviceID)
	if err != nil {
		return domain.Principal{}, apperr.E(apperr.KindUnauthorized, "auth.Authenticate", "device identity not found", err)
	}
	if !s.constantEqual(principal.Credential, credential) {
		return domain.Principal{}, apperr.E(apperr.KindUnauthorized, "auth.Authenticate", "invalid credential", nil)
	}
	if credentialType != "" && principal.CredentialType != credentialType {
		return domain.Principal{}, apperr.E(apperr.KindUnauthorized, "auth.Authenticate", "credential type mismatch", nil)
	}
	if s.lookup != nil {
		identity, lookupErr := s.lookup.FindDevice(ctx, deviceID)
		if lookupErr != nil {
			return domain.Principal{}, apperr.E(apperr.KindUnauthorized, "auth.Authenticate", "device is not available", lookupErr)
		}
		if !identity.Enabled {
			return domain.Principal{}, apperr.E(apperr.KindForbidden, "auth.Authenticate", "device is disabled", nil)
		}
	}
	return principal, nil
}

func (s *Service) Authorize(principal domain.Principal, scope string) domain.Decision {
	if principal.HasScope(scope) {
		return domain.Decision{Allowed: true, DeviceID: principal.DeviceID, Scope: scope}
	}
	return domain.Decision{Allowed: false, DeviceID: principal.DeviceID, Scope: scope, Reason: "scope not granted"}
}

func (s *Service) AuthenticateAndAuthorize(ctx context.Context, deviceID, credential, scope string, credentialType domain.CredentialType) (domain.Principal, error) {
	principal, err := s.Authenticate(ctx, deviceID, credential, credentialType)
	if err != nil {
		return domain.Principal{}, err
	}
	decision := s.Authorize(principal, scope)
	if !decision.Allowed {
		return domain.Principal{}, apperr.E(apperr.KindForbidden, "auth.AuthenticateAndAuthorize", decision.Reason, nil)
	}
	return principal, nil
}

func (s *Service) GenerateToken() string {
	return id.New("token") + id.New("secret")
}

func (s *Service) constantEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func NormalizeCredentialType(value string) domain.CredentialType {
	switch strings.ToLower(value) {
	case "psk":
		return domain.CredentialPSK
	case "cert", "certificate":
		return domain.CredentialCert
	default:
		return domain.CredentialToken
	}
}
