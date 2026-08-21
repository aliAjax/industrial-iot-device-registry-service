package memory

import (
	"context"
	"testing"

	"github.com/example/iot-device-management/internal/rule/domain"
	tsdomain "github.com/example/iot-device-management/internal/timeseries/domain"
)

func TestSaveExecutionClonesInputs(t *testing.T) {
	repo := NewRepository()
	execution := domain.Execution{
		ID:       "exec-1",
		RuleID:   "rule-1",
		DeviceID: "dev-1",
		Inputs:   []tsdomain.Point{{DeviceID: "dev-1", Property: "temp", Value: 20}},
	}
	if err := repo.SaveExecution(context.Background(), execution); err != nil {
		t.Fatal(err)
	}
	execution.Inputs[0].Value = 99
	items, _, err := repo.ListExecutions(context.Background(), domain.ExecutionFilter{RuleID: "rule-1", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if got := items[0].Inputs[0].Value; got != 20 {
		t.Fatalf("stored execution aliases caller input, got %v", got)
	}
}

func TestSaveRuleClonesActions(t *testing.T) {
	repo := NewRepository()
	rule := domain.Rule{
		ID:      "rule-1",
		Actions: []domain.Action{{Type: domain.ActionAlarm, Config: map[string]any{"severity": "warning"}}},
	}
	if err := repo.SaveRule(context.Background(), rule); err != nil {
		t.Fatal(err)
	}
	rule.Actions[0].Config["severity"] = "critical"
	got, err := repo.GetRule(context.Background(), "rule-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Actions[0].Config["severity"] != "warning" {
		t.Fatal("stored rule aliases caller input")
	}
}
