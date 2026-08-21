package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/example/iot-device-management/internal/device/domain"
	"github.com/example/iot-device-management/internal/platform/apperr"
	clockpkg "github.com/example/iot-device-management/internal/platform/clock"
	"github.com/example/iot-device-management/internal/platform/eventbus"
	"github.com/example/iot-device-management/internal/platform/id"
)

type PrincipalRegistrar interface {
	RegisterDevice(ctx context.Context, deviceID, productID, credentialType, credential string, scopes []string) error
	RevokeDevice(ctx context.Context, deviceID string) error
}

type Service struct {
	products domain.ProductRepository
	devices  domain.DeviceRepository
	groups   domain.GroupRepository
	bus      eventbus.Bus
	identity PrincipalRegistrar
	clock    clockpkg.Clock
	tokenTTL time.Duration
}

func NewService(products domain.ProductRepository, devices domain.DeviceRepository, groups domain.GroupRepository, bus eventbus.Bus, identity PrincipalRegistrar, clock clockpkg.Clock, tokenTTL time.Duration) *Service {
	if clock == nil {
		clock = clockpkg.SystemClock{}
	}
	if tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour
	}
	return &Service{products: products, devices: devices, groups: groups, bus: bus, identity: identity, clock: clock, tokenTTL: tokenTTL}
}

func (s *Service) CreateProduct(ctx context.Context, input CreateProductInput) (domain.Product, error) {
	if strings.TrimSpace(input.Name) == "" {
		return domain.Product{}, apperr.E(apperr.KindInvalid, "device.CreateProduct", "product name is required", nil)
	}
	if err := validateAttributeSchemas(input.Attributes); err != nil {
		return domain.Product{}, err
	}
	now := s.clock.Now()
	product := domain.Product{
		ID:          id.New("prod"),
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		Attributes:  append([]domain.AttributeSchema(nil), input.Attributes...),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.products.CreateProduct(ctx, product); err != nil {
		return domain.Product{}, wrap("create product", err)
	}
	return product, nil
}

func (s *Service) CreateGroup(ctx context.Context, input CreateGroupInput) (domain.Group, error) {
	if strings.TrimSpace(input.Name) == "" {
		return domain.Group{}, apperr.E(apperr.KindInvalid, "device.CreateGroup", "group name is required", nil)
	}
	if input.ParentID != "" {
		if _, err := s.groups.GetGroup(ctx, input.ParentID); err != nil {
			return domain.Group{}, apperr.E(apperr.KindInvalid, "device.CreateGroup", "parent group does not exist", err)
		}
	}
	now := s.clock.Now()
	group := domain.Group{
		ID:          id.New("grp"),
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		ParentID:    input.ParentID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.groups.CreateGroup(ctx, group); err != nil {
		return domain.Group{}, wrap("create group", err)
	}
	return group, nil
}

func (s *Service) RegisterDevice(ctx context.Context, input RegisterDeviceInput) (domain.Device, error) {
	if input.ProductID == "" {
		return domain.Device{}, apperr.E(apperr.KindInvalid, "device.RegisterDevice", "product_id is required", nil)
	}
	product, err := s.products.GetProduct(ctx, input.ProductID)
	if err != nil {
		return domain.Device{}, apperr.E(apperr.KindNotFound, "device.RegisterDevice", "product not found", err)
	}
	now := s.clock.Now()
	device := domain.Device{
		ID:           id.New("dev"),
		ProductID:    product.ID,
		SerialNumber: strings.TrimSpace(input.SerialNumber),
		Name:         strings.TrimSpace(input.Name),
		Status:       domain.DevicePending,
		GroupIDs:     append([]string(nil), input.GroupIDs...),
		Tags:         cloneTags(input.Tags),
		Attributes:   cloneAnyMap(input.Attributes),
		Location:     input.Location,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	for _, groupID := range device.GroupIDs {
		if _, err := s.groups.GetGroup(ctx, groupID); err != nil {
			return domain.Device{}, apperr.E(apperr.KindInvalid, "device.RegisterDevice", "group does not exist", err)
		}
	}
	credential, err := s.issueCredential(ctx, device.ID, input.CredentialType, input.CredentialValue, now)
	if err != nil {
		return domain.Device{}, err
	}
	device.Credentials = []domain.Credential{credential}
	if err := s.devices.CreateDevice(ctx, device); err != nil {
		return domain.Device{}, wrap("register device", err)
	}
	if s.identity != nil {
		if err := s.identity.RegisterDevice(ctx, device.ID, product.ID, string(credential.Type), credential.Value, defaultScopes(device.ID)); err != nil {
			_ = s.devices.DeleteDevice(ctx, device.ID)
			return domain.Device{}, wrap("register device identity", err)
		}
	}
	_ = s.publish(ctx, "device.registered", device.ID, map[string]any{"device_id": device.ID, "product_id": product.ID})
	return device, nil
}

func (s *Service) EnableDevice(ctx context.Context, deviceID string) (domain.Device, error) {
	device, err := s.getDeviceForMutation(ctx, deviceID)
	if err != nil {
		return domain.Device{}, err
	}
	if device.Status == domain.DeviceEnabled {
		return device, nil
	}
	device.Status = domain.DeviceEnabled
	device.UpdatedAt = s.clock.Now()
	if err := s.devices.UpdateDevice(ctx, device); err != nil {
		return domain.Device{}, wrap("enable device", err)
	}
	_ = s.publish(ctx, "device.enabled", device.ID, map[string]any{"device_id": device.ID})
	return device, nil
}

func (s *Service) DisableDevice(ctx context.Context, deviceID string) (domain.Device, error) {
	device, err := s.getDeviceForMutation(ctx, deviceID)
	if err != nil {
		return domain.Device{}, err
	}
	device.Status = domain.DeviceDisabled
	device.UpdatedAt = s.clock.Now()
	if err := s.devices.UpdateDevice(ctx, device); err != nil {
		return domain.Device{}, wrap("disable device", err)
	}
	_ = s.publish(ctx, "device.disabled", device.ID, map[string]any{"device_id": device.ID})
	return device, nil
}

func (s *Service) DeregisterDevice(ctx context.Context, deviceID string) error {
	device, err := s.getDeviceForMutation(ctx, deviceID)
	if err != nil {
		return err
	}
	device.Status = domain.DeviceDeregistered
	device.UpdatedAt = s.clock.Now()
	if err := s.devices.UpdateDevice(ctx, device); err != nil {
		return wrap("deregister device", err)
	}
	if s.identity != nil {
		if err := s.identity.RevokeDevice(ctx, deviceID); err != nil {
			return wrap("revoke device identity", err)
		}
	}
	_ = s.publish(ctx, "device.deregistered", deviceID, map[string]any{"device_id": deviceID})
	return nil
}

func (s *Service) RotateCredential(ctx context.Context, deviceID string, credentialType domain.CredentialType) (domain.Credential, error) {
	device, err := s.getDeviceForMutation(ctx, deviceID)
	if err != nil {
		return domain.Credential{}, err
	}
	now := s.clock.Now()
	credential, err := s.issueCredential(ctx, device.ID, credentialType, "", now)
	if err != nil {
		return domain.Credential{}, err
	}
	device.Credentials = append(device.Credentials, credential)
	device.UpdatedAt = now
	if err := s.devices.UpdateDevice(ctx, device); err != nil {
		return domain.Credential{}, wrap("rotate credential", err)
	}
	if s.identity != nil {
		if err := s.identity.RegisterDevice(ctx, device.ID, device.ProductID, string(credential.Type), credential.Value, defaultScopes(device.ID)); err != nil {
			return domain.Credential{}, wrap("rotate credential identity", err)
		}
	}
	return credential, nil
}

func (s *Service) UpdateTags(ctx context.Context, deviceID string, tags map[string]string) (domain.Device, error) {
	device, err := s.getDeviceForMutation(ctx, deviceID)
	if err != nil {
		return domain.Device{}, err
	}
	device.Tags = cloneTags(tags)
	device.UpdatedAt = s.clock.Now()
	if err := s.devices.UpdateDevice(ctx, device); err != nil {
		return domain.Device{}, wrap("update tags", err)
	}
	return device, nil
}

func (s *Service) AssignGroups(ctx context.Context, deviceID string, groupIDs []string) (domain.Device, error) {
	device, err := s.getDeviceForMutation(ctx, deviceID)
	if err != nil {
		return domain.Device{}, err
	}
	device.GroupIDs = append([]string(nil), groupIDs...)
	device.UpdatedAt = s.clock.Now()
	if err := s.devices.UpdateDevice(ctx, device); err != nil {
		return domain.Device{}, wrap("assign groups", err)
	}
	return device, nil
}

func (s *Service) UpdateLocation(ctx context.Context, deviceID string, location *domain.GeoLocation) (domain.Device, error) {
	if location == nil {
		return domain.Device{}, apperr.E(apperr.KindInvalid, "device.UpdateLocation", "location is required", nil)
	}
	if location.Latitude < -90 || location.Latitude > 90 || location.Longitude < -180 || location.Longitude > 180 {
		return domain.Device{}, apperr.E(apperr.KindInvalid, "device.UpdateLocation", "latitude or longitude out of range", nil)
	}
	device, err := s.getDeviceForMutation(ctx, deviceID)
	if err != nil {
		return domain.Device{}, err
	}
	device.Location = location
	device.UpdatedAt = s.clock.Now()
	if err := s.devices.UpdateDevice(ctx, device); err != nil {
		return domain.Device{}, wrap("update location", err)
	}
	return device, nil
}

func (s *Service) GetDevice(ctx context.Context, deviceID string) (domain.Device, error) {
	device, err := s.devices.GetDevice(ctx, deviceID)
	if err != nil {
		return domain.Device{}, apperr.E(apperr.KindNotFound, "device.GetDevice", "device not found", err)
	}
	return device, nil
}

func (s *Service) ListDevices(ctx context.Context, filter domain.DeviceFilter, offset, limit int) ([]domain.Device, int, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items, total, err := s.devices.ListDevices(ctx, filter, offset, limit)
	if err != nil {
		return nil, 0, wrap("list devices", err)
	}
	return items, total, nil
}

func (s *Service) GetProduct(ctx context.Context, productID string) (domain.Product, error) {
	product, err := s.products.GetProduct(ctx, productID)
	if err != nil {
		return domain.Product{}, apperr.E(apperr.KindNotFound, "device.GetProduct", "product not found", err)
	}
	return product, nil
}

func (s *Service) ListProducts(ctx context.Context, offset, limit int) ([]domain.Product, int, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items, total, err := s.products.ListProducts(ctx, offset, limit)
	if err != nil {
		return nil, 0, wrap("list products", err)
	}
	return items, total, nil
}

func (s *Service) ListGroups(ctx context.Context, offset, limit int) ([]domain.Group, int, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items, total, err := s.groups.ListGroups(ctx, offset, limit)
	if err != nil {
		return nil, 0, wrap("list groups", err)
	}
	return items, total, nil
}

func (s *Service) getDeviceForMutation(ctx context.Context, deviceID string) (domain.Device, error) {
	device, err := s.devices.GetDevice(ctx, deviceID)
	if err != nil {
		return domain.Device{}, apperr.E(apperr.KindNotFound, "device.getDeviceForMutation", "device not found", err)
	}
	if device.Status == domain.DeviceDeregistered {
		return domain.Device{}, apperr.E(apperr.KindConflict, "device.getDeviceForMutation", "device is deregistered", nil)
	}
	return device, nil
}

func (s *Service) issueCredential(_ context.Context, deviceID string, credentialType domain.CredentialType, value string, now time.Time) (domain.Credential, error) {
	if credentialType == "" {
		credentialType = domain.CredentialToken
	}
	if credentialType != domain.CredentialToken && credentialType != domain.CredentialPSK && credentialType != domain.CredentialCert {
		return domain.Credential{}, apperr.E(apperr.KindInvalid, "device.issueCredential", "unsupported credential type", nil)
	}
	if value == "" {
		value = id.New("secret") + id.New("part")
	}
	credential := domain.Credential{
		ID:        id.New("cred"),
		Type:      credentialType,
		Value:     value,
		CreatedAt: now,
	}
	if credentialType == domain.CredentialCert {
		sum := sha256.Sum256([]byte(value))
		credential.Fingerprint = hex.EncodeToString(sum[:])
	}
	expires := now.Add(s.tokenTTL)
	credential.ExpiresAt = &expires
	_ = deviceID
	return credential, nil
}

func (s *Service) publish(ctx context.Context, eventType, subject string, payload any) error {
	if s.bus == nil {
		return nil
	}
	return s.bus.Publish(ctx, eventbus.Event{
		ID:         id.New("evt"),
		Type:       eventType,
		Subject:    subject,
		Payload:    payload,
		OccurredAt: s.clock.Now(),
	})
}

func wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	if apperr.KindOf(err) != apperr.KindInternal {
		return err
	}
	return apperr.E(apperr.KindInternal, op, "operation failed", err)
}

func validateAttributeSchemas(items []domain.AttributeSchema) error {
	keys := map[string]struct{}{}
	for _, item := range items {
		if strings.TrimSpace(item.Key) == "" {
			return apperr.E(apperr.KindInvalid, "device.validateAttributeSchemas", "attribute key is required", nil)
		}
		if _, ok := keys[item.Key]; ok {
			return apperr.E(apperr.KindInvalid, "device.validateAttributeSchemas", "duplicate attribute key", nil)
		}
		keys[item.Key] = struct{}{}
		switch item.Type {
		case "string", "number", "boolean", "object", "array":
		default:
			return apperr.E(apperr.KindInvalid, "device.validateAttributeSchemas", "invalid attribute type", nil)
		}
	}
	return nil
}

func defaultScopes(deviceID string) []string {
	return []string{"device:" + deviceID + ":telemetry", "device:" + deviceID + ":commands", "device:" + deviceID + ":attributes"}
}

func cloneTags(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

type CreateProductInput struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Attributes  []domain.AttributeSchema `json:"attributes"`
}

type CreateGroupInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ParentID    string `json:"parent_id"`
}

type RegisterDeviceInput struct {
	ProductID       string                `json:"product_id"`
	SerialNumber    string                `json:"serial_number"`
	Name            string                `json:"name"`
	GroupIDs        []string              `json:"group_ids"`
	Tags            map[string]string     `json:"tags"`
	Attributes      map[string]any        `json:"attributes"`
	Location        *domain.GeoLocation   `json:"location"`
	CredentialType  domain.CredentialType `json:"credential_type"`
	CredentialValue string                `json:"credential_value"`
}
