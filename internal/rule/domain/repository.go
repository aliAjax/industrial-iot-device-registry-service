package domain

import "context"

type Repository interface {
	SaveRule(ctx context.Context, rule Rule) error
	GetRule(ctx context.Context, id string) (Rule, error)
	ListRules(ctx context.Context, deviceID string, offset, limit int) ([]Rule, int, error)
	DeleteRule(ctx context.Context, id string) error

	SaveExecution(ctx context.Context, execution Execution) error
	ListExecutions(ctx context.Context, filter ExecutionFilter) ([]Execution, int, error)
}
