package db

import (
	"database/sql"
	"fmt"

	"shadow_api/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

func Open(cfg *config.Config) (*sql.DB, error) {
	switch cfg.Database.Driver {
	case "sqlite":
		db, err := sql.Open("sqlite", cfg.Database.SQLite.Path)
		if err != nil {
			return nil, err
		}
		if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
			return nil, fmt.Errorf("enable foreign keys: %w", err)
		}
		return db, nil
	case "postgres":
		return sql.Open("pgx", cfg.Database.Postgres.DSN)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Database.Driver)
	}
}
