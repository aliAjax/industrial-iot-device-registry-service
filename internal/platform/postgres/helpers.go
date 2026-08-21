package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func UpsertJSON(ctx context.Context, pool *pgxpool.Pool, table, id string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", table, err)
	}
	_, err = pool.Exec(ctx, "INSERT INTO "+table+" (id, data, updated_at) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data, updated_at = EXCLUDED.updated_at", id, data, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("upsert %s: %w", table, err)
	}
	return nil
}

func GetJSON(ctx context.Context, pool *pgxpool.Pool, table, id string, out any) error {
	row := pool.QueryRow(ctx, "SELECT data FROM "+table+" WHERE id = $1", id)
	var data []byte
	if err := row.Scan(&data); err != nil {
		return err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unmarshal %s: %w", table, err)
	}
	return nil
}

func DeleteByID(ctx context.Context, pool *pgxpool.Pool, table, id string) error {
	tag, err := pool.Exec(ctx, "DELETE FROM "+table+" WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("row not found")
	}
	return nil
}

func ListJSON[T any](ctx context.Context, pool *pgxpool.Pool, table, where string, args []any, order string, offset, limit int) ([]T, error) {
	query := "SELECT data FROM " + table
	if where != "" {
		query += " WHERE " + where
	}
	if order == "" {
		order = "updated_at DESC"
	}
	query += " ORDER BY " + order + " LIMIT $" + fmt.Sprint(len(args)+1) + " OFFSET $" + fmt.Sprint(len(args)+2)
	params := append(append([]any{}, args...), limit, offset)
	rows, err := pool.Query(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]T, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var item T
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
