package domain

import (
	"time"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusAcked    Status = "acked"
	StatusFailed   Status = "failed"
	StatusExpired  Status = "expired"
	StatusRetrying Status = "retrying"
)

type Command struct {
	ID             string         `json:"id"`
	DeviceID       string         `json:"device_id"`
	Name           string         `json:"name"`
	Payload        map[string]any `json:"payload"`
	IdempotencyKey string         `json:"idempotency_key"`
	Status         Status         `json:"status"`
	RetryCount     int            `json:"retry_count"`
	MaxRetries     int            `json:"max_retries"`
	Timeout        time.Duration  `json:"timeout"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	ExpiresAt      time.Time      `json:"expires_at"`
	AckedAt        *time.Time     `json:"acked_at,omitempty"`
	FailureReason  string         `json:"failure_reason,omitempty"`
	Audit          []AuditEntry   `json:"audit"`
}

type AuditEntry struct {
	At      time.Time `json:"at"`
	Status  Status    `json:"status"`
	Message string    `json:"message"`
	TraceID string    `json:"trace_id"`
}

type EnqueueInput struct {
	DeviceID       string         `json:"device_id"`
	Name           string         `json:"name"`
	Payload        map[string]any `json:"payload"`
	IdempotencyKey string         `json:"idempotency_key"`
	MaxRetries     int            `json:"max_retries"`
	Timeout        time.Duration  `json:"timeout"`
}

type AckInput struct {
	DeviceID  string `json:"device_id"`
	CommandID string `json:"command_id"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
}

type Delivery struct {
	Command Command `json:"command"`
	Attempt int     `json:"attempt"`
}

func (c Command) IsTerminal() bool {
	return c.Status == StatusAcked || c.Status == StatusFailed || c.Status == StatusExpired
}
