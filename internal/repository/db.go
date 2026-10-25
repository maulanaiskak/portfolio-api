package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

// DBTX is satisfied by both *pgxpool.Pool and pgx.Tx, so repositories run
// unchanged inside or outside a transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txKey struct{}

// conn returns the transaction stored in ctx by Transactor, or the pool.
func conn(ctx context.Context, pool *pgxpool.Pool) DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

// Transactor lets usecases group repository calls into one transaction
// without depending on pgx directly.
type Transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type transactor struct {
	pool *pgxpool.Pool
}

func (t *transactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				err = errors.Join(err, fmt.Errorf("rollback: %w", rbErr))
			}
		}
	}()
	if err = fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func NewTransactor(pool *pgxpool.Pool) Transactor {
	return &transactor{pool: pool}
}

// nonNil keeps JSONB array columns as [] instead of null.
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// Postgres error codes mapped to entity errors.
const (
	pgUniqueViolation     = "23505"
	pgCheckViolation      = "23514"
	pgForeignKeyViolation = "23503"
)

// mapWriteError converts constraint violations into client-safe entity
// errors; everything else is returned unchanged (and treated as internal).
func mapWriteError(err error, what string) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case pgUniqueViolation:
		return entity.NewConflictError("%s with this slug already exists", what)
	case pgCheckViolation:
		return entity.NewValidationError("%s has an invalid field (%s)", what, pgErr.ConstraintName)
	case pgForeignKeyViolation:
		return entity.NewNotFoundError("referenced %s not found", what)
	}
	return err
}

// nullIfEmpty stores "" as NULL for nullable text columns.
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
