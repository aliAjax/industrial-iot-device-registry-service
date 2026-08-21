package domain

import (
	"time"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusScheduled Status = "scheduled"
	StatusRunning   Status = "running"
	StatusPaused    Status = "paused"
	StatusFailed    Status = "failed"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
)

type Firmware struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"product_id"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	ArtifactURI string    `json:"artifact_uri"`
	Checksum    string    `json:"checksum"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RolloutPolicy struct {
	Strategy      string   `json:"strategy"`
	Percentage    float64  `json:"percentage"`
	DeviceIDs     []string `json:"device_ids"`
	GroupIDs      []string `json:"group_ids"`
	MaxConcurrent int      `json:"max_concurrent"`
	AutoRollback  bool     `json:"auto_rollback"`
}

type UpgradeTask struct {
	ID          string        `json:"id"`
	FirmwareID  string        `json:"firmware_id"`
	Status      Status        `json:"status"`
	Rollout     RolloutPolicy `json:"rollout"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	StartedAt   *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
}

type DeviceReceipt struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	DeviceID  string    `json:"device_id"`
	Status    Status    `json:"status"`
	Attempt   int       `json:"attempt"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TaskInput struct {
	FirmwareID string        `json:"firmware_id"`
	Rollout    RolloutPolicy `json:"rollout"`
}

type ReceiptInput struct {
	TaskID   string `json:"task_id"`
	DeviceID string `json:"device_id"`
	Success  bool   `json:"success"`
	Message  string `json:"message"`
}

type Filter struct {
	FirmwareID string
	Status     Status
	DeviceID   string
}

func (r RolloutPolicy) Normalized() RolloutPolicy {
	if r.Strategy == "" {
		r.Strategy = "percentage"
	}
	if r.MaxConcurrent <= 0 {
		r.MaxConcurrent = 1
	}
	if r.Percentage < 0 {
		r.Percentage = 0
	}
	if r.Percentage > 100 {
		r.Percentage = 100
	}
	return r
}
