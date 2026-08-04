package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gloscai/template-go-vue3-docker/server/config"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func Open(ctx context.Context, cfg config.Database) (*sql.DB, error) {
	driverName := cfg.Driver
	if driverName == "postgres" {
		driverName = "pgx"
	}

	db, err := sql.Open(driverName, cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("opening %s database: %w", cfg.Driver, err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connecting to %s database: %w", cfg.Driver, err)
	}
	return db, nil
}
