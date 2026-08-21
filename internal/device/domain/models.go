package domain

import (
	"encoding/json"
	"time"
)

type DeviceStatus string

const (
	DevicePending      DeviceStatus = "pending"
	DeviceEnabled      DeviceStatus = "enabled"
	DeviceDisabled     DeviceStatus = "disabled"
	DeviceDeregistered DeviceStatus = "deregistered"
)

type CredentialType string

const (
	CredentialToken CredentialType = "token"
	CredentialPSK   CredentialType = "psk"
	CredentialCert  CredentialType = "cert"
)

type Product struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Attributes  []AttributeSchema `json:"attributes"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type AttributeSchema struct {
	Key         string   `json:"key"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Description string   `json:"description"`
	Unit        string   `json:"unit,omitempty"`
	Minimum     *float64 `json:"minimum,omitempty"`
	Maximum     *float64 `json:"maximum,omitempty"`
}

type Device struct {
	ID           string            `json:"id"`
	ProductID    string            `json:"product_id"`
	SerialNumber string            `json:"serial_number"`
	Name         string            `json:"name"`
	Status       DeviceStatus      `json:"status"`
	GroupIDs     []string          `json:"group_ids"`
	Tags         map[string]string `json:"tags"`
	Attributes   map[string]any    `json:"attributes"`
	Location     *GeoLocation      `json:"location,omitempty"`
	Credentials  []Credential      `json:"credentials,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type Credential struct {
	ID          string         `json:"id"`
	Type        CredentialType `json:"type"`
	Value       string         `json:"value,omitempty"`
	Fingerprint string         `json:"fingerprint,omitempty"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	RevokedAt   *time.Time     `json:"revoked_at,omitempty"`
}

type Group struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ParentID    string    `json:"parent_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type GeoLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude,omitempty"`
	Accuracy  float64 `json:"accuracy,omitempty"`
}

func (d *Device) MarshalJSON() ([]byte, error) {
	type alias Device
	return json.Marshal((*alias)(d))
}

func (d *Device) IsActive() bool {
	return d.Status == DeviceEnabled
}

func (d *Device) HasCredential(credentialID string) bool {
	for _, credential := range d.Credentials {
		if credential.ID == credentialID && credential.RevokedAt == nil {
			if credential.ExpiresAt == nil || credential.ExpiresAt.After(time.Now().UTC()) {
				return true
			}
		}
	}
	return false
}
