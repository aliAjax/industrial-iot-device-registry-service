package application

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/example/iot-device-management/internal/platform/apperr"
	clockpkg "github.com/example/iot-device-management/internal/platform/clock"
	"github.com/example/iot-device-management/internal/platform/eventbus"
	"github.com/example/iot-device-management/internal/platform/id"
	"github.com/example/iot-device-management/internal/platform/trace"
	"github.com/example/iot-device-management/internal/rule/domain"
	tsdomain "github.com/example/iot-device-management/internal/timeseries/domain"
)

type ActionExecutor interface {
	Execute(ctx context.Context, rule domain.Rule, execution domain.Execution) error
}

type BusActionExecutor struct {
	bus eventbus.Bus
}

func NewBusActionExecutor(bus eventbus.Bus) *BusActionExecutor {
	return &BusActionExecutor{bus: bus}
}

func (e *BusActionExecutor) Execute(ctx context.Context, rule domain.Rule, execution domain.Execution) error {
	if e.bus == nil {
		return nil
	}
	for _, action := range rule.Actions {
		eventType := "rule.action." + string(action.Type)
		if err := e.bus.Publish(ctx, eventbus.Event{
			ID:         id.New("evt"),
			Type:       eventType,
			Subject:    rule.ID,
			Payload:    map[string]any{"rule": rule, "execution": execution, "action": action},
			OccurredAt: time.Now().UTC(),
			TraceID:    trace.TraceID(ctx),
		}); err != nil {
			return err
		}
	}
	return nil
}

type Engine struct {
	repo     domain.Repository
	bus      eventbus.Bus
	executor ActionExecutor
	clock    clockpkg.Clock
	mu       sync.Mutex
	buffers  map[string][]tsdomain.Point
	cooldown map[string]time.Time
}

func NewEngine(repo domain.Repository, bus eventbus.Bus, executor ActionExecutor, clock clockpkg.Clock) *Engine {
	if clock == nil {
		clock = clockpkg.SystemClock{}
	}
	if executor == nil {
		executor = NewBusActionExecutor(bus)
	}
	return &Engine{
		repo:     repo,
		bus:      bus,
		executor: executor,
		clock:    clock,
		buffers:  make(map[string][]tsdomain.Point),
		cooldown: make(map[string]time.Time),
	}
}

func (e *Engine) CreateRule(ctx context.Context, rule domain.Rule) (domain.Rule, error) {
	if rule.Name == "" || rule.Property == "" {
		return domain.Rule{}, apperr.E(apperr.KindInvalid, "rule.CreateRule", "name and property are required", nil)
	}
	rule = rule.Normalized()
	rule.ID = id.New("rule")
	now := e.clock.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	if err := e.repo.SaveRule(ctx, rule); err != nil {
		return domain.Rule{}, apperr.E(apperr.KindInternal, "rule.CreateRule", "save rule", err)
	}
	_ = e.publish(ctx, "rule.created", rule.ID, rule)
	return rule, nil
}

func (e *Engine) UpdateRule(ctx context.Context, ruleID string, patch RulePatch) (domain.Rule, error) {
	rule, err := e.repo.GetRule(ctx, ruleID)
	if err != nil {
		return domain.Rule{}, apperr.E(apperr.KindNotFound, "rule.UpdateRule", "rule not found", err)
	}
	if patch.Name != nil && *patch.Name != "" {
		rule.Name = *patch.Name
	}
	if patch.Enabled != nil {
		rule.Enabled = *patch.Enabled
	}
	if patch.DeviceID != "" {
		rule.DeviceID = patch.DeviceID
	}
	if patch.Property != "" {
		rule.Property = patch.Property
	}
	if patch.Condition != nil {
		rule.Condition = *patch.Condition
	}
	if patch.Actions != nil {
		rule.Actions = patch.Actions
	}
	if patch.Window > 0 {
		rule.Window = patch.Window
	}
	if patch.Cooldown >= 0 {
		rule.Cooldown = patch.Cooldown
	}
	rule = rule.Normalized()
	rule.UpdatedAt = e.clock.Now()
	if err := e.repo.SaveRule(ctx, rule); err != nil {
		return domain.Rule{}, apperr.E(apperr.KindInternal, "rule.UpdateRule", "save rule", err)
	}
	return rule, nil
}

func (e *Engine) DeleteRule(ctx context.Context, ruleID string) error {
	if err := e.repo.DeleteRule(ctx, ruleID); err != nil {
		return apperr.E(apperr.KindInternal, "rule.DeleteRule", "delete rule", err)
	}
	e.mu.Lock()
	delete(e.buffers, ruleID)
	delete(e.cooldown, ruleID)
	e.mu.Unlock()
	return nil
}

func (e *Engine) ListRules(ctx context.Context, deviceID string, offset, limit int) ([]domain.Rule, int, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return e.repo.ListRules(ctx, deviceID, offset, limit)
}

func (e *Engine) ListExecutions(ctx context.Context, filter domain.ExecutionFilter) ([]domain.Execution, int, error) {
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 50
	}
	return e.repo.ListExecutions(ctx, filter)
}

func (e *Engine) HandleTelemetry(ctx context.Context, point tsdomain.Point) error {
	if point.DeviceID == "" || point.Property == "" {
		return apperr.E(apperr.KindInvalid, "rule.HandleTelemetry", "point device and property are required", nil)
	}
	rules, _, err := e.repo.ListRules(ctx, point.DeviceID, 0, 1000)
	if err != nil {
		return apperr.E(apperr.KindInternal, "rule.HandleTelemetry", "load rules", err)
	}
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		rule = rule.Normalized()
		if rule.DeviceID != "" && rule.DeviceID != point.DeviceID {
			continue
		}
		if rule.Property != "" && rule.Property != point.Property {
			continue
		}
		if err := e.evaluateRule(ctx, rule, point); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) Start(ctx context.Context) {
	if e.bus == nil {
		return
	}
	unsubscribe := e.bus.Subscribe("telemetry.point_ingested", func(ctx context.Context, event eventbus.Event) error {
		point, ok := event.Payload.(tsdomain.Point)
		if !ok {
			return nil
		}
		return e.HandleTelemetry(ctx, point)
	})
	go func() {
		<-ctx.Done()
		unsubscribe()
	}()
}

func (e *Engine) evaluateRule(ctx context.Context, rule domain.Rule, point tsdomain.Point) error {
	e.mu.Lock()
	buffer := append(e.buffers[rule.ID], point)
	cutoff := e.clock.Now().Add(-rule.Window)
	buffer = prunePoints(buffer, cutoff)
	if len(buffer) > 1000 {
		buffer = buffer[len(buffer)-1000:]
	}
	e.buffers[rule.ID] = buffer
	e.mu.Unlock()

	if len(buffer) < rule.Condition.MinSamples {
		return nil
	}
	value, err := aggregate(buffer, rule.Condition.Aggregate)
	if err != nil {
		return err
	}
	triggered, reason := compare(value, rule.Condition.Operator, rule.Condition.Threshold)
	execution := domain.Execution{
		ID:         id.New("rule_exec"),
		RuleID:     rule.ID,
		RuleName:   rule.Name,
		DeviceID:   point.DeviceID,
		Property:   point.Property,
		Inputs:     append([]tsdomain.Point(nil), buffer...),
		Aggregate:  value,
		Triggered:  triggered,
		Reason:     reason,
		Actions:    append([]domain.Action(nil), rule.Actions...),
		ReplayKey:  replayKey(rule.ID, e.clock.Now(), value),
		ExecutedAt: e.clock.Now(),
	}
	if err := e.repo.SaveExecution(ctx, execution); err != nil {
		return apperr.E(apperr.KindInternal, "rule.evaluateRule", "save execution", err)
	}
	if triggered {
		if last, ok := e.cooldown[rule.ID]; ok && e.clock.Now().Before(last.Add(rule.Cooldown)) {
			return nil
		}
		e.cooldown[rule.ID] = e.clock.Now()
		if err := e.executor.Execute(ctx, rule, execution); err != nil {
			return apperr.E(apperr.KindInternal, "rule.evaluateRule", "execute actions", err)
		}
		_ = e.publish(ctx, "rule.triggered", rule.ID, execution)
	}
	return nil
}

func (e *Engine) publish(ctx context.Context, eventType, subject string, payload any) error {
	if e.bus == nil {
		return nil
	}
	return e.bus.Publish(ctx, eventbus.Event{
		ID:         id.New("evt"),
		Type:       eventType,
		Subject:    subject,
		Payload:    payload,
		OccurredAt: e.clock.Now(),
		TraceID:    trace.TraceID(ctx),
	})
}

func aggregate(points []tsdomain.Point, kind domain.AggregateKind) (float64, error) {
	if len(points) == 0 {
		return 0, fmt.Errorf("no points")
	}
	switch kind {
	case domain.AggregateAvg:
		var sum float64
		for _, point := range points {
			sum += point.Value
		}
		return sum / float64(len(points)), nil
	case domain.AggregateMin:
		value := math.MaxFloat64
		for _, point := range points {
			if point.Value < value {
				value = point.Value
			}
		}
		return value, nil
	case domain.AggregateMax:
		value := -math.MaxFloat64
		for _, point := range points {
			if point.Value > value {
				value = point.Value
			}
		}
		return value, nil
	case domain.AggregateSum:
		var sum float64
		for _, point := range points {
			sum += point.Value
		}
		return sum, nil
	case domain.AggregateCount:
		return float64(len(points)), nil
	case domain.AggregateLast:
		return points[len(points)-1].Value, nil
	default:
		return 0, fmt.Errorf("unsupported aggregate %q", kind)
	}
}

func compare(value float64, operator domain.Operator, threshold float64) (bool, string) {
	triggered := false
	switch operator {
	case domain.OperatorGT:
		triggered = value > threshold
	case domain.OperatorGTE:
		triggered = value >= threshold
	case domain.OperatorLT:
		triggered = value < threshold
	case domain.OperatorLTE:
		triggered = value <= threshold
	case domain.OperatorEQ:
		triggered = math.Abs(value-threshold) < 1e-9
	case domain.OperatorNE:
		triggered = math.Abs(value-threshold) >= 1e-9
	}
	reason := fmt.Sprintf("%s %s %f = %v", kindLabel(operator), operator, threshold, triggered)
	return triggered, reason
}

func kindLabel(operator domain.Operator) string {
	return strings.ToUpper(string(operator))
}

func prunePoints(points []tsdomain.Point, cutoff time.Time) []tsdomain.Point {
	idx := sort.Search(len(points), func(i int) bool {
		return !points[i].Timestamp.Before(cutoff)
	})
	return points[idx:]
}

func replayKey(ruleID string, at time.Time, value float64) string {
	return fmt.Sprintf("%s:%d:%.6f", ruleID, at.UnixNano(), value)
}

type RulePatch struct {
	Name      *string           `json:"name,omitempty"`
	Enabled   *bool             `json:"enabled,omitempty"`
	DeviceID  string            `json:"device_id,omitempty"`
	Property  string            `json:"property,omitempty"`
	Condition *domain.Condition `json:"condition,omitempty"`
	Actions   []domain.Action   `json:"actions,omitempty"`
	Window    time.Duration     `json:"window,omitempty"`
	Cooldown  time.Duration     `json:"cooldown,omitempty"`
}
