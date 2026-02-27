package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/albievan/clarity/clarity-api/internal/config"
)

// DB wraps the standard sql.DB so additional methods can be attached.
type DB struct {
	*sql.DB
}

// Connect opens a database connection.
// Supported drivers: postgres (lib/pq). Wire in additional drivers as needed.
func Connect(cfg config.DBConfig) (*DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("DB_DSN is empty")
	}
	sqlDB, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	return &DB{sqlDB}, nil
}

// TxFn is a function that executes within a transaction.
type TxFn func(tx *sql.Tx) error

// WithTx wraps fn in a transaction, committing on success and rolling back on error.
func (d *DB) WithTx(fn TxFn) error {
	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
