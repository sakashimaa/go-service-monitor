package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sakashimaa/site-monitor/internal/domain"
)

type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type CheckHistoryRepository interface {
	Create(ctx context.Context, h *domain.CheckHistory) error
	CreateTx(ctx context.Context, tx pgx.Tx, h *domain.CheckHistory) error
	GetLatest(ctx context.Context, siteID string) (*domain.CheckHistory, error)
	GetHistory(ctx context.Context, siteID string, limit int, cursor *time.Time) ([]domain.CheckHistory, error)
	DeleteBySiteId(ctx context.Context, siteID string) error
	DeleteBySiteIdTx(ctx context.Context, tx pgx.Tx, siteID string) error
}

type CheckHistoryPGRepo struct {
	db *pgxpool.Pool
}

func NewCheckHistoryRepo(db *pgxpool.Pool) CheckHistoryRepository {
	return &CheckHistoryPGRepo{db: db}
}

func (c *CheckHistoryPGRepo) deleteById(ctx context.Context, q querier, siteID string) error {
	if _, err := q.Exec(ctx, `DELETE FROM site_checks WHERE id = $1`, siteID); err != nil {
		return fmt.Errorf("failed to delete site history: %w", err)
	}

	return nil
}

func (c *CheckHistoryPGRepo) create(ctx context.Context, q querier, req *domain.CheckHistory) error {
	query := `
		INSERT INTO site_checks (id, site_id, status, response_code, response_time, error)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	if _, err := q.Exec(
		ctx,
		query,
		req.ID,
		req.SiteID,
		req.Status,
		req.ResponseCode,
		req.ResponseTime,
		req.Error,
	); err != nil {
		return fmt.Errorf("failed to insert history: %w", err)
	}

	return nil
}

func (c *CheckHistoryPGRepo) DeleteBySiteId(ctx context.Context, siteID string) error {
	return c.deleteById(ctx, c.db, siteID)
}

func (c *CheckHistoryPGRepo) DeleteBySiteIdTx(ctx context.Context, tx pgx.Tx, siteID string) error {
	return c.deleteById(ctx, tx, siteID)
}

func (c *CheckHistoryPGRepo) CreateTx(ctx context.Context, tx pgx.Tx, h *domain.CheckHistory) error {
	return c.create(ctx, tx, h)
}

func (c *CheckHistoryPGRepo) Create(ctx context.Context, h *domain.CheckHistory) error {
	return c.create(ctx, c.db, h)
}

func (c *CheckHistoryPGRepo) GetLatest(ctx context.Context, siteID string) (*domain.CheckHistory, error) {
	query := `
		SELECT id, site_id, status, response_code, response_time, error, created_at
		FROM site_checks
		WHERE site_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var res domain.CheckHistory
	if err := c.db.QueryRow(
		ctx,
		query,
		siteID,
	).Scan(
		&res.ID,
		&res.SiteID,
		&res.Status,
		&res.ResponseCode,
		&res.ResponseTime,
		&res.Error,
		&res.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSiteHistoryNotFound
		}

		return nil, fmt.Errorf("failed to query latest check history: %w", err)
	}

	return &res, nil
}

func (c *CheckHistoryPGRepo) GetHistory(ctx context.Context, siteID string, limit int, cursor *time.Time) ([]domain.CheckHistory, error) {
	var query string
	var args []any

	if cursor == nil {
		query = `
			SELECT id, site_id, status, response_code, response_time, error, created_at
			FROM site_checks
			WHERE site_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		args = []any{siteID, limit}
	} else {
		query = `
			SELECT id, site_id, status, response_code, response_time, error, created_at
			FROM site_checks
			WHERE site_id = $1 AND created_at < $2
			ORDER BY created_at DESC
			LIMIT $3
		`
		args = []any{siteID, cursor, limit}
	}

	rows, err := c.db.Query(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query site check history: %w", err)
	}

	defer rows.Close()

	res := make([]domain.CheckHistory, 0)
	for rows.Next() {
		var h domain.CheckHistory
		if err := rows.Scan(
			&h.ID,
			&h.SiteID,
			&h.Status,
			&h.ResponseCode,
			&h.ResponseTime,
			&h.Error,
			&h.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		res = append(res, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return res, nil
}
