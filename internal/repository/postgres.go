package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sakashimaa/site-monitor/internal/domain"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) SiteRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetAll(ctx context.Context) ([]domain.Site, error) {
	query := `
		SELECT id, name, url
		FROM sites
		ORDER BY id ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sites: %w", err)
	}
	defer rows.Close()

	sites := make([]domain.Site, 0)
	for rows.Next() {
		var s domain.Site
		if err := rows.Scan(&s.ID, &s.Name, &s.URL); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		sites = append(sites, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error in rows: %w", err)
	}

	return sites, nil
}

func (r *PostgresRepository) Create(ctx context.Context, req *domain.Site) error {
	query := `
		INSERT INTO sites (id, name, url, status)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.pool.Exec(ctx, query, req.ID, req.Name, req.URL, domain.StatusPending)
	if err != nil {
		var pgErr *pgconn.PgError
		// код уникальности
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrURLAlreadyExists
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	cmdTag, err := r.pool.Exec(ctx, "DELETE FROM sites WHERE id = $1", id)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrSiteNotFound
	}
	return nil
}

func (r *PostgresRepository) GetStatus(ctx context.Context, id string) (domain.SiteStatus, error) {
	var s domain.SiteStatus

	err := r.pool.QueryRow(ctx, `
		SELECT url, status, response_code, last_check_time, response_time, error 
		FROM sites 
		WHERE id = $1`,
		id,
	).Scan(&s.URL, &s.Status, &s.ResponseCode, &s.LastCheckTime, &s.ResponseTime, &s.Error)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.SiteStatus{}, ErrSiteNotFound
		}
		return domain.SiteStatus{}, err
	}

	return s, nil
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id string, status domain.SiteStatus) error {
	cmdTag, err := r.pool.Exec(ctx, `
		UPDATE sites 
		SET status = $1, response_code = $2, last_check_time = $3, response_time = $4, error = $5 
		WHERE id = $6`,
		status.Status, status.ResponseCode, status.LastCheckTime, status.ResponseTime, status.Error, id,
	)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrSiteNotFound
	}
	return nil
}
