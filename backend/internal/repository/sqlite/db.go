package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"shale/internal/domain"
)

type DB struct {
	sql *sql.DB
}

func New(conn *sql.DB) *DB { return &DB{sql: conn} }

func (d *DB) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := txFrom(ctx); ok {
		return fn(ctx)
	}
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

type txKey struct{}

func txFrom(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	return tx, ok
}

type runner interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (d *DB) runner(ctx context.Context) runner {
	if tx, ok := txFrom(ctx); ok {
		return tx
	}
	return d.sql
}

func mapConstraint(err error, conflictMessage string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "UNIQUE constraint failed"):
		return domain.NewError(domain.KindConflict, conflictMessage)
	case strings.Contains(msg, "FOREIGN KEY constraint failed"):
		return domain.NewError(domain.KindNotFound, "referenced record does not exist")
	}
	return err
}

func rowsAffected(res sql.Result) (int64, error) {
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return n, nil
}
