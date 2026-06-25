package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	if connStr == "" {
		return nil, fmt.Errorf("invalid connection string")
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create new pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping DB: %w", err)
	}

	return pool, nil
}
