package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/example/iot-device-management/internal/platform/apperr"
	"github.com/example/iot-device-management/internal/rule/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) SaveRule(ctx context.Context, rule domain.Rule) error {
	data, err := json.Marshal(rule)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, "INSERT INTO rules (id, device_id, data, updated_at) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET device_id = EXCLUDED.device_id, data = EXCLUDED.data, updated_at = EXCLUDED.updated_at", rule.ID, rule.DeviceID, data, time.Now().UTC())
	return err
}

func (r *Repository) GetRule(ctx context.Context, id string) (domain.Rule, error) {
	row := r.pool.QueryRow(ctx, "SELECT data FROM rules WHERE id = $1", id)
	var data []byte
	if err := row.Scan(&data); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Rule{}, apperr.E(apperr.KindNotFound, "rule.postgres.GetRule", "rule not found", err)
		}
		return domain.Rule{}, err
	}
	var rule domain.Rule
	if err := json.Unmarshal(data, &rule); err != nil {
		return domain.Rule{}, err
	}
	return rule, nil
}

func (r *Repository) ListRules(ctx context.Context, deviceID string, offset, limit int) ([]domain.Rule, int, error) {
	rows, err := r.pool.Query(ctx, "SELECT data FROM rules ORDER BY data->>'created_at' DESC")
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	all := make([]domain.Rule, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, 0, err
		}
		var rule domain.Rule
		if err := json.Unmarshal(data, &rule); err != nil {
			return nil, 0, err
		}
		if deviceID == "" || rule.DeviceID == "" || rule.DeviceID == deviceID {
			all = append(all, rule)
		}
	}
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (r *Repository) DeleteRule(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM rules WHERE id = $1", id)
	return err
}

func (r *Repository) SaveExecution(ctx context.Context, execution domain.Execution) error {
	data, err := json.Marshal(execution)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, "INSERT INTO rule_executions (id, rule_id, device_id, data, executed_at) VALUES ($1, $2, $3, $4, $5)", execution.ID, execution.RuleID, execution.DeviceID, data, execution.ExecutedAt)
	return err
}

func (r *Repository) ListExecutions(ctx context.Context, filter domain.ExecutionFilter) ([]domain.Execution, int, error) {
	rows, err := r.pool.Query(ctx, "SELECT data FROM rule_executions ORDER BY executed_at DESC")
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	all := make([]domain.Execution, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, 0, err
		}
		var execution domain.Execution
		if err := json.Unmarshal(data, &execution); err != nil {
			return nil, 0, err
		}
		if filter.RuleID != "" && execution.RuleID != filter.RuleID {
			continue
		}
		if filter.DeviceID != "" && execution.DeviceID != filter.DeviceID {
			continue
		}
		all = append(all, execution)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ExecutedAt.After(all[j].ExecutedAt) })
	total := len(all)
	offset := filter.Offset
	if offset > total {
		offset = total
	}
	end := offset + filter.Limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}
