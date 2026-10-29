package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"chit-fund-api/internal/config"
)

type PostgresPool struct {
	*pgxpool.Pool
}

func NewPostgresPool(cfg config.DatabaseConfig) (*PostgresPool, error) {
	poolConfig, err := pgxpool.ParseConfig(
		fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Info().Msg("Database connection pool established")
	return &PostgresPool{Pool: pool}, nil
}

// WithCompanyTx starts a transaction and sets the RLS session variable
// app.current_company_id using SET LOCAL (transaction-scoped, safe for pooling).
// All 10 payment tables have RLS enabled — every query touching them must use this.
func (p *PostgresPool) WithCompanyTx(ctx context.Context, companyID int, fn func(pgx.Tx) error) error {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx, "SET LOCAL app.current_company_id = $1", companyID); err != nil {
		return fmt.Errorf("set RLS company: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ExecWithCompany runs a single non-transactional query that needs the RLS variable.
// Uses a short-lived transaction for safety. Use WithCompanyTx for multi-step operations.
func (p *PostgresPool) ExecWithCompany(ctx context.Context, companyID int,
	fn func(pgx.Tx) error) error {
	return p.WithCompanyTx(ctx, companyID, fn)
}
