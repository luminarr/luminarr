package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // registers "sqlite" driver

	"github.com/davidfic/luminarr/internal/config"
)

// DB wraps the underlying sql.DB and tracks which driver is in use.
type DB struct {
	SQL    *sql.DB
	Driver string
}

// Open opens a database connection based on the provided configuration.
// The caller is responsible for calling Close when done.
func Open(cfg config.DatabaseConfig) (*DB, error) {
	switch cfg.Driver {
	case "sqlite", "":
		return openSQLite(cfg.Path)
	case "postgres":
		return openPostgres(cfg.DSN.Value())
	default:
		return nil, fmt.Errorf("unsupported database driver: %q (must be sqlite or postgres)", cfg.Driver)
	}
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.SQL.Close()
}

func openSQLite(path string) (*DB, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite path must not be empty")
	}

	// Ensure the parent directory exists.
	if err := ensureDir(path); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=ON&_busy_timeout=5000", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite database: %w", err)
	}

	// SQLite performs best with a single writer and multiple readers via WAL.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("pinging sqlite database: %w", err)
	}

	return &DB{SQL: sqlDB, Driver: "sqlite"}, nil
}

func openPostgres(dsn string) (*DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("postgres DSN must not be empty")
	}

	// pgx/stdlib registers the "pgx" driver name for database/sql compatibility.
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening postgres database: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("pinging postgres database: %w", err)
	}

	return &DB{SQL: sqlDB, Driver: "postgres"}, nil
}
