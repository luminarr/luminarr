package db

import (
	"context"
	"database/sql"
	"fmt"

	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
)

// RunInTx runs fn inside a database transaction. If fn returns an error the
// transaction is rolled back; otherwise it is committed. The *Queries passed
// to fn is bound to the transaction so all operations share the same tx.
func RunInTx(ctx context.Context, sqlDB *sql.DB, fn func(q dbsqlite.Querier) error) error {
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	txQ := dbsqlite.New(sqlDB).WithTx(tx)

	if err := fn(txQ); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
