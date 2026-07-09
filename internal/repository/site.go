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

var (
	ErrURLAlreadyExists = errors.New("URL already exists")
)

type SiteRepository interface {
	GetById(ctx context.Context, id string) (*domain.Site, error)
	GetAll(ctx context.Context) ([]domain.Site, error)
	Create(ctx context.Context, req *domain.Site) error
	CreateTx(ctx context.Context, tx pgx.Tx, req *domain.Site) error
	Update(ctx context.Context, req *domain.Site) error
	Delete(ctx context.Context, id string) error
	DeleteTx(ctx context.Context, tx pgx.Tx, id string) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) SiteRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Update(ctx context.Context, req *domain.Site) error {
	cmdTag, err := r.pool.Exec(ctx, `
		UPDATE sites
		SET name = $1, url = $2
		WHERE id = $3
	`, req.Name, req.URL, req.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrURLAlreadyExists
		}
		return fmt.Errorf("failed to update site: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrSiteNotFound
	}

	return nil
}

func (r *PostgresRepository) GetById(ctx context.Context, id string) (*domain.Site, error) {
	var s domain.Site
	err := r.pool.QueryRow(ctx,
		`
		SELECT id, name, url
		FROM sites
		WHERE id = $1
	`, id).Scan(&s.ID, &s.Name, &s.URL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSiteNotFound
		}

		return nil, fmt.Errorf("failed to query site: %w", err)
	}

	return &s, nil
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

func (r *PostgresRepository) create(ctx context.Context, q querier, req *domain.Site) error {
	query := `
		INSERT INTO sites (id, name, url)
		VALUES ($1, $2, $3)
	`
	_, err := q.Exec(ctx, query, req.ID, req.Name, req.URL)
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

func (r *PostgresRepository) Create(ctx context.Context, req *domain.Site) error {
	return r.create(ctx, r.pool, req)
}

func (r *PostgresRepository) CreateTx(ctx context.Context, tx pgx.Tx, req *domain.Site) error {
	return r.create(ctx, tx, req)
}

func (r *PostgresRepository) delete(ctx context.Context, q querier, id string) error {
	cmdTag, err := q.Exec(ctx, "DELETE FROM sites WHERE id = $1", id)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrSiteNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	return r.delete(ctx, r.pool, id)
}

func (r *PostgresRepository) DeleteTx(ctx context.Context, tx pgx.Tx, id string) error {
	return r.delete(ctx, tx, id)
}
