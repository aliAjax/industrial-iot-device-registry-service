package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/rule/domain"
	tsdomain "github.com/example/iot-device-management/internal/timeseries/domain"
)

type Repository struct {
	mu         sync.RWMutex
	rules      map[string]domain.Rule
	executions []domain.Execution
}

func NewRepository() *Repository {
	return &Repository{rules: make(map[string]domain.Rule)}
}

func (r *Repository) SaveRule(_ context.Context, rule domain.Rule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules[rule.ID] = cloneRule(rule)
	return nil
}

func (r *Repository) GetRule(_ context.Context, id string) (domain.Rule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rule, exists := r.rules[id]
	if !exists {
		return domain.Rule{}, apperr.E(apperr.KindNotFound, "rule.memory.GetRule", "rule not found", nil)
	}
	return cloneRule(rule), nil
}

func (r *Repository) ListRules(_ context.Context, deviceID string, offset, limit int) ([]domain.Rule, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Rule, 0)
	for _, rule := range r.rules {
		if deviceID != "" && rule.DeviceID != "" && rule.DeviceID != deviceID {
			continue
		}
		items = append(items, cloneRule(rule))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	total := len(items)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return items[offset:end], total, nil
}

func (r *Repository) DeleteRule(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.rules[id]; !exists {
		return apperr.E(apperr.KindNotFound, "rule.memory.DeleteRule", "rule not found", nil)
	}
	delete(r.rules, id)
	return nil
}

func (r *Repository) SaveExecution(_ context.Context, execution domain.Execution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executions = append(r.executions, cloneExecution(execution))
	if len(r.executions) > 5000 {
		r.executions = r.executions[len(r.executions)-5000:]
	}
	return nil
}

func (r *Repository) ListExecutions(_ context.Context, filter domain.ExecutionFilter) ([]domain.Execution, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Execution, 0)
	for _, execution := range r.executions {
		if filter.RuleID != "" && execution.RuleID != filter.RuleID {
			continue
		}
		if filter.DeviceID != "" && execution.DeviceID != filter.DeviceID {
			continue
		}
		items = append(items, cloneExecution(execution))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ExecutedAt.After(items[j].ExecutedAt) })
	total := len(items)
	offset := filter.Offset
	if offset > total {
		offset = total
	}
	end := offset + filter.Limit
	if end > total {
		end = total
	}
	return items[offset:end], total, nil
}

// cloneRule returns a deep copy of rule so callers cannot mutate stored state
// through the Actions slice or per-action Config maps. It mirrors the
// clone-on-write/read convention used by the command and timeseries stores.
func cloneRule(rule domain.Rule) domain.Rule {
	out := rule
	out.Actions = cloneActions(rule.Actions)
	return out
}

func cloneActions(actions []domain.Action) []domain.Action {
	if len(actions) == 0 {
		return nil
	}
	out := make([]domain.Action, len(actions))
	for i, action := range actions {
		out[i] = action
		if action.Config != nil {
			config := make(map[string]any, len(action.Config))
			for key, value := range action.Config {
				config[key] = value
			}
			out[i].Config = config
		}
	}
	return out
}

// cloneExecution returns a deep copy of execution so the stored Inputs and
// Actions slices cannot be mutated through the caller's references.
func cloneExecution(execution domain.Execution) domain.Execution {
	out := execution
	out.Inputs = append([]tsdomain.Point(nil), execution.Inputs...)
	out.Actions = cloneActions(execution.Actions)
	return out
}
