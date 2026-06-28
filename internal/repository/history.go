package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sakashimaa/site-monitor/internal/domain"
)

type CheckHistoryRepository interface {
	Create(ctx context.Context, h *domain.CheckHistory) error
	GetLatest(ctx context.Context, siteID string) (*domain.CheckHistory, error)
	GetHistory(ctx context.Context, siteID string) ([]domain.CheckHistory, error)
}

type CheckHistoryPGRepo struct {
	db *pgxpool.Pool
}

func NewCheckHistoryRepo(db *pgxpool.Pool) CheckHistoryRepository {
	return &CheckHistoryPGRepo{db: db}
}

func (c *CheckHistoryPGRepo) Create(ctx context.Context, h *domain.CheckHistory) error {
	query := `
		INSERT INTO site_checks (id, site_id, status, response_code, response_time, error)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	if _, err := c.db.Exec(
		ctx,
		query,
		h.ID,
		h.SiteID,
		h.Status,
		h.ResponseCode,
		h.ResponseTime,
		h.Error,
	); err != nil {
		return fmt.Errorf("failed to insert history: %w", err)
	}

	return nil
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

func (c *CheckHistoryPGRepo) GetHistory(ctx context.Context, siteID string) ([]domain.CheckHistory, error) {
	query := `
		SELECT id, site_id, status, response_code, response_time, error, created_at
		FROM site_checks
		WHERE site_id = $1
		ORDER BY created_at DESC
	`

	rows, err := c.db.Query(
		ctx,
		query,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query site check history: %w", err)
	}

	defer rows.Close()

	var res []domain.CheckHistory
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
