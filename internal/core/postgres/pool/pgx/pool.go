package pgx

import (
	"context"
	"errors"
	"fmt"
	"time"

	driver "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/makmanu/golang_todoapp/internal/core/postgres/pool"
)

type Pool struct {
	inner   *pgxpool.Pool
	timeout time.Duration
}

var _ pool.Pool = (*Pool)(nil)

func New(ctx context.Context, dsn string, timeout time.Duration, maxConns int32) (*Pool, error) {
	if timeout <= 0 || maxConns < 1 {
		return nil, fmt.Errorf("pool timeout and max connections must be positive")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}
	cfg.MaxConns = maxConns
	cfg.ConnConfig.ConnectTimeout = timeout
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := p.Ping(pingCtx); err != nil {
		p.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Pool{inner: p, timeout: timeout}, nil
}
func (p *Pool) Close()                   { p.inner.Close() }
func (p *Pool) OpTimeout() time.Duration { return p.timeout }
func (p *Pool) Query(ctx context.Context, sql string, args ...any) (pool.Rows, error) {
	rows, err := p.inner.Query(ctx, sql, args...)
	if err != nil {
		return nil, mapErrors(err)
	}
	return pgxRows{rows}, nil
}
func (p *Pool) QueryRow(ctx context.Context, sql string, args ...any) pool.Row {
	return pgxRow{p.inner.QueryRow(ctx, sql, args...)}
}
func (p *Pool) Exec(ctx context.Context, sql string, args ...any) (pool.CommandTag, error) {
	tag, err := p.inner.Exec(ctx, sql, args...)
	return pgxCommandTag{tag}, mapErrors(err)
}

type pgxRow struct{ driver.Row }

func (r pgxRow) Scan(dest ...any) error { return mapErrors(r.Row.Scan(dest...)) }

type pgxRows struct{ driver.Rows }

func (r pgxRows) Scan(dest ...any) error { return mapErrors(r.Rows.Scan(dest...)) }
func (r pgxRows) Err() error             { return mapErrors(r.Rows.Err()) }

type pgxCommandTag struct{ pgconn.CommandTag }

func mapErrors(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, driver.ErrNoRows) {
		return fmt.Errorf("%w: %w", pool.ErrNoRows, err)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		var sentinel error
		switch pgErr.Code {
		case "23503":
			sentinel = pool.ErrViolatesForeignKey
		case "23505":
			sentinel = pool.ErrUniqueViolation
		case "23514", "23502", "22001":
			sentinel = pool.ErrCheckViolation
		}
		if sentinel != nil {
			return fmt.Errorf("%w: %w", sentinel, err)
		}
	}
	return err
}
