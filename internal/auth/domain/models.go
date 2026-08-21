package domain

import (
	"time"
)

type CredentialType string

const (
	CredentialToken CredentialType = "token"
	CredentialPSK   CredentialType = "psk"
	CredentialCert  CredentialType = "cert"
)

type Principal struct {
	DeviceID       string         `json:"device_id"`
	ProductID      string         `json:"product_id"`
	CredentialType CredentialType `json:"credential_type"`
	Credential     string         `json:"-"`
	Scopes         []string       `json:"scopes"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type Decision struct {
	Allowed  bool   `json:"allowed"`
	Reason   string `json:"reason,omitempty"`
	DeviceID string `json:"device_id,omitempty"`
	Scope    string `json:"scope,omitempty"`
}

type RateRecord struct {
	Key    string    `json:"key"`
	Window time.Time `json:"window"`
	Count  int       `json:"count"`
	Limit  int       `json:"limit"`
}

func (p Principal) HasScope(scope string) bool {
	for _, candidate := range p.Scopes {
		if candidate == scope || candidate == "*" {
			return true
		}
	}
	return false
}

func (p Principal) IsRevoked() bool {
	return p.Credential == "" || p.DeviceID == ""
}
