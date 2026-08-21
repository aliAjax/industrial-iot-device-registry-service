package domain

import (
	"time"

	"github.com/example/iot-device-management/internal/timeseries/domain"
)

type Operator string

const (
	OperatorGT  Operator = "gt"
	OperatorGTE Operator = "gte"
	OperatorLT  Operator = "lt"
	OperatorLTE Operator = "lte"
	OperatorEQ  Operator = "eq"
	OperatorNE  Operator = "ne"
)

type AggregateKind string

const (
	AggregateAvg   AggregateKind = "avg"
	AggregateMin   AggregateKind = "min"
	AggregateMax   AggregateKind = "max"
	AggregateSum   AggregateKind = "sum"
	AggregateCount AggregateKind = "count"
	AggregateLast  AggregateKind = "last"
)

type ActionType string

const (
	ActionAlarm   ActionType = "alarm"
	ActionCommand ActionType = "command"
	ActionForward ActionType = "forward"
)

type Rule struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Enabled   bool          `json:"enabled"`
	DeviceID  string        `json:"device_id"`
	Property  string        `json:"property"`
	Condition Condition     `json:"condition"`
	Actions   []Action      `json:"actions"`
	Window    time.Duration `json:"window"`
	Cooldown  time.Duration `json:"cooldown"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type Condition struct {
	Operator   Operator      `json:"operator"`
	Threshold  float64       `json:"threshold"`
	Aggregate  AggregateKind `json:"aggregate"`
	MinSamples int           `json:"min_samples"`
}

type Action struct {
	Type   ActionType     `json:"type"`
	Config map[string]any `json:"config"`
}

type Execution struct {
	ID         string         `json:"id"`
	RuleID     string         `json:"rule_id"`
	RuleName   string         `json:"rule_name"`
	DeviceID   string         `json:"device_id"`
	Property   string         `json:"property"`
	Inputs     []domain.Point `json:"inputs"`
	Aggregate  float64        `json:"aggregate"`
	Triggered  bool           `json:"triggered"`
	Reason     string         `json:"reason"`
	Actions    []Action       `json:"actions"`
	ReplayKey  string         `json:"replay_key"`
	ExecutedAt time.Time      `json:"executed_at"`
}

type ExecutionFilter struct {
	RuleID   string
	DeviceID string
	Offset   int
	Limit    int
}

func (r Rule) Normalized() Rule {
	if r.Window <= 0 {
		r.Window = time.Minute
	}
	if r.Cooldown < 0 {
		r.Cooldown = time.Minute
	}
	if r.Condition.MinSamples <= 0 {
		r.Condition.MinSamples = 1
	}
	if r.Condition.Operator == "" {
		r.Condition.Operator = OperatorGT
	}
	if r.Condition.Aggregate == "" {
		r.Condition.Aggregate = AggregateAvg
	}
	return r
}
